// Copyright 2025 The Wireflow Authors, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
	"wireflow/internal/infra"
	"wireflow/internal/ipam"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"wireflow/api/v1alpha1"
)

// PeerReconciler reconciles a WireflowPeer object
type PeerReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	IPAM          *ipam.IPAM
	generator     *Generator
	SnapshotCache map[types.NamespacedName]*PeerStateSnapshot

	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=wireflowcontroller.wireflow.run,resources=wireflowpeers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=wireflowcontroller.wireflow.run,resources=wireflowpeers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=wireflowcontroller.wireflow.run,resources=wireflowpeers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the WireflowPeer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
func (r *PeerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciling WireflowPeer", "namespace", req.NamespacedName, "node", req.Name)

	var (
		err  error
		node v1alpha1.WireflowPeer
	)

	if err = r.Get(ctx, req.NamespacedName, &node); err != nil {
		if errors.IsNotFound(err) {
			log.Info("WireflowPeer resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}

		log.Error(err, "Failed to get WireflowPeer")
		return ctrl.Result{}, err
	}

	// Shadow peers are managed exclusively by NetworkPeeringReconciler /
	// ClusterPeeringReconciler; skip them here to avoid conflicting updates.
	if node.GetLabels()[LabelShadow] == "true" {
		log.Info("Skipping shadow peer", "namespace", req.Namespace, "name", req.Name)
		return ctrl.Result{}, nil
	}

	action, err := r.determineAction(ctx, &node)
	if err != nil {
		return ctrl.Result{}, err
	}
	switch action {
	case NodeJoinNetwork:
		log.Info("Handing join network", "namespace", req.Namespace, "name", req.Name)
		return r.reconcileJoinNetwork(ctx, &node, req)
	case NodeLeaveNetwork:
		log.Info("Handing leave network", "namespace", req.Namespace, "name", req.Name)
		return r.reconcileLeaveNetwork(ctx, &node, req)
	default:
		log.Info("No action to handle", "namespace", req.Namespace, "name", req.Name)
		return r.lastReconcile(ctx, &node, req)
	}

}

type Action string

const (
	NodeJoinNetwork  Action = "joinNetwork"
	NodeLeaveNetwork Action = "leaveNetwork"
	ActionNone       Action = "none"
)

// reconcileJoinNetwork handles the join-network flow in at most 2 reconcile
// rounds (down from 4) by relying on updateSpec/updateStatus syncing the
// patched ResourceVersion back into the caller's pointer so intermediate
// r.Get() calls and early returns after status-only patches are unnecessary.
//
// Round 1 – first-time join (keys not yet generated):
//
//	ensurePendingPhase (status patch, no early return) →
//	ensurePeerSpec (generates keys / sets labels, spec changes → requeue)
//
// Round 2 – keys already present:
//
//	ensurePendingPhase (no-op if already Pending) →
//	ensurePeerSpec (no-op) → AllocateIP → updateStatus(Ready) → lastReconcile
func (r *PeerReconciler) reconcileJoinNetwork(ctx context.Context, peer *v1alpha1.WireflowPeer, request ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Join network", "namespace", request.Namespace, "name", request.Name)

	// Fail fast: check network readiness before any writes.
	if peer.Spec.Network == nil {
		return ctrl.Result{}, fmt.Errorf("spec.network is empty")
	}
	var network v1alpha1.WireflowNetwork
	if err := r.Get(ctx, types.NamespacedName{Namespace: peer.Namespace, Name: *peer.Spec.Network}, &network); err != nil {
		return ctrl.Result{}, err
	}
	// NetworkReconciler sets ActiveCIDR via status patch (doesn't change generation).
	// networkReadyPredicate in SetupWithManager re-enqueues peers when it transitions.
	if network.Status.ActiveCIDR == "" {
		log.Info("Network not ready yet, waiting for ActiveCIDR", "network", network.Name)
		return ctrl.Result{RequeueAfter: 3 * time.Second}, nil
	}

	// Round 1a: mark Pending — updateStatus syncs ResourceVersion back so the
	// subsequent spec patch doesn't conflict without an extra r.Get().
	if err := r.ensurePendingPhase(ctx, peer); err != nil {
		return ctrl.Result{}, err
	}

	// Round 1b: apply label / key changes.
	// Labels don't change Generation, so we must requeue explicitly when spec changed.
	changed, err := r.ensurePeerSpec(ctx, peer)
	if err != nil {
		return ctrl.Result{}, err
	}
	if changed {
		return ctrl.Result{RequeueAfter: time.Millisecond * 100}, nil
	}

	// Round 2: allocate IP, mark Ready, write config.
	address, err := r.IPAM.AllocateIP(ctx, &network, peer)
	if err != nil {
		return ctrl.Result{}, err
	}
	log.Info("get allocated address", "address", address)

	if _, err = r.updateStatus(ctx, peer, func(node *v1alpha1.WireflowPeer) {
		node.Status.Phase = v1alpha1.NodePhaseReady
		node.Status.AllocatedAddress = &address
		node.Status.ActiveNetwork = node.Spec.Network
	}); err != nil {
		return ctrl.Result{}, err
	}

	return r.lastReconcile(ctx, peer, request)
}

// ensurePendingPhase sets Status.Phase to Pending when it isn't already.
// It syncs the updated ResourceVersion back into peer so callers can
// immediately issue further patches without a conflict.
func (r *PeerReconciler) ensurePendingPhase(ctx context.Context, peer *v1alpha1.WireflowPeer) error {
	if peer.Status.Phase == v1alpha1.NodePhasePending {
		return nil
	}
	_, err := r.updateStatus(ctx, peer, func(node *v1alpha1.WireflowPeer) {
		node.Status.Phase = v1alpha1.NodePhasePending
	})
	return err
}

// ensurePeerSpec sets the network label and generates WireGuard keys when
// they are absent. Returns (true, nil) when a patch was written so the caller
// can requeue; (false, nil) when the peer spec was already up-to-date.
func (r *PeerReconciler) ensurePeerSpec(ctx context.Context, peer *v1alpha1.WireflowPeer) (bool, error) {
	return r.updateSpec(ctx, peer, func(node *v1alpha1.WireflowPeer) error {
		network, err := r.getNetwork(ctx, node)
		if err != nil {
			return err
		}

		lbls := node.GetLabels()
		if lbls == nil {
			lbls = make(map[string]string)
		}
		// 切换网络时，先删除所有旧的 network label，再添加新的，避免 peer 同时出现在多个网络
		for label := range lbls {
			if strings.HasPrefix(label, "wireflow.run/network-") {
				delete(lbls, label)
			}
		}
		lbls[fmt.Sprintf("wireflow.run/network-%s", network.Name)] = "true"
		node.SetLabels(lbls)

		if node.Spec.PrivateKey == "" {
			key, err := wgtypes.GeneratePrivateKey()
			if err != nil {
				return err
			}
			node.Spec.PrivateKey = key.String()
			node.Spec.PublicKey = key.PublicKey().String()
			node.Spec.PeerId = fmt.Sprintf("%d", infra.FromKey(key.PublicKey()).ToUint64())
		}
		return nil
	})
}

// lastReconcile create or update the configmap
func (r *PeerReconciler) lastReconcile(ctx context.Context, peer *v1alpha1.WireflowPeer, request ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)
	logger.Info("Last reconciling", "name", request.NamespacedName)

	configMapName := fmt.Sprintf("%s-config", peer.Name)

	// 1) 每次都重新构建 snapshot（不再做 changes 检查）
	snapshot := r.getPeerStateSnapshot(ctx, peer, request)

	// 2) 用 WireflowPolicy 计算 computedPeers / computedRules，并生成最终 message
	message, err := r.generator.generate(ctx, peer, snapshot, r.generator.generateConfigVersion())
	if err != nil {
		return ctrl.Result{}, err
	}

	var newHash string
	newHash, err = computeMessageHash(message)
	if err != nil {
		return ctrl.Result{}, err
	}

	desiredConfigMap := r.newConfigmap(peer.Namespace, configMapName, message.String(), newHash)
	if err = controllerutil.SetControllerReference(peer, desiredConfigMap, r.Scheme); err != nil {
		logger.Error(err, "Failed to set owner reference on configmap")
		return ctrl.Result{}, err
	}

	// 3) 获取当前 CM，看 hash 是否一致；不一致才更新
	var found corev1.ConfigMap
	err = r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: peer.Namespace}, &found)
	if err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		logger.Info("Creating configmap", "name", configMapName, "hash", newHash)
		manager := client.FieldOwner("wireflow-controller-manager")
		if err := r.Patch(ctx, desiredConfigMap, client.Apply, manager); err != nil {
			logger.Error(err, "Failed to create configmap")
			return ctrl.Result{}, err
		}

		// Also persist hash in status so the next reconcile (triggered by the
		// ConfigMap Create event) sees CurrentHash == newHash and skips cleanly.
		if _, err := r.updateStatus(ctx, peer, func(node *v1alpha1.WireflowPeer) {
			node.Status.CurrentHash = newHash
		}); err != nil {
			return ctrl.Result{}, err
		}

		if r.Recorder != nil {
			r.Recorder.Eventf(peer, corev1.EventTypeNormal, "ConfigMapCreated",
				"configmap=%s hash=%s version=%s", configMapName, newHash, message.ConfigVersion)
		}
		return ctrl.Result{}, nil
	}

	// Fix oldHash using peer status CurrentHash
	oldHash := peer.Status.CurrentHash

	if oldHash == newHash {
		logger.Info("Configmap unchanged by hash, skipping update", "name", configMapName, "hash", newHash)
		return ctrl.Result{}, nil
	}

	logger.Info("Updating configmap by hash", "namespace", peer.Namespace, "name", configMapName, "oldHash", oldHash, "newHash", newHash)
	manager := client.FieldOwner("wireflow-controller-manager")
	if err := r.Patch(ctx, desiredConfigMap, client.Apply, manager); err != nil {
		logger.Error(err, "Failed to update configmap")
		return ctrl.Result{}, err
	}

	ok, err := r.updateStatus(ctx, peer, func(node *v1alpha1.WireflowPeer) {
		node.Status.CurrentHash = newHash
	})

	if err != nil {
		return ctrl.Result{}, err
	}

	if ok {
		return ctrl.Result{}, nil
	}

	if r.Recorder != nil {
		r.Recorder.Eventf(peer, corev1.EventTypeNormal, "ConfigMapUpdated",
			"configmap=%s oldHash=%s newHash=%s version=%s", configMapName, oldHash, newHash, message.ConfigVersion)
	}
	return ctrl.Result{}, nil
}

// 如果你想看具体哪里变了
//func diffConfig(oldData, newData string) {
//	// 假设是 JSON 格式字符串，先反序列化
//	var oldMap, newMap map[string]interface{}
//	json.Unmarshal([]byte(oldData), &oldMap)
//	json.Unmarshal([]byte(newData), &newMap)
//
//	// 使用 go-cmp 打印详细差异
//	fmt.Println(cmp.Diff(oldMap, newMap))
//}

func (r *PeerReconciler) newConfigmap(namespace, configMapName, message, hash string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "wireflow-controller",
			},
			Annotations: map[string]string{
				"wireflow.run/config-hash": hash,
			},
		},
		Data: map[string]string{
			"config.json": message,
		},
	}
}

func computeMessageHash(msg *infra.Message) (string, error) {
	// 排除 ConfigVersion 和 Timestamp，避免这两个字段每次变化导致 hash 不稳定
	tmp := struct {
		*infra.Message
		ConfigVersion interface{} `json:"configVersion,omitempty"` // 覆盖排除
		Timestamp     interface{} `json:"timestamp,omitempty"`     // 覆盖排除
	}{
		Message:       msg,
		ConfigVersion: nil,
		Timestamp:     nil,
	}

	b, err := json.Marshal(tmp)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(b)), nil
}

// reconcileLeaveNetwork handles the leave-network flow without intermediate
// requeues after status-only patches.  updateSpec/updateStatus sync the
// ResourceVersion back so no extra r.Get() is needed between calls.
//
// Flow (single reconcile round):
//
//	ensurePendingPhase (status patch, no early return) →
//	updateSpec (Network=nil + remove labels; Generation change queues next reconcile) →
//	updateStatus (clear ActiveNetwork/AllocatedAddress, set Ready) →
//	lastReconcile
func (r *PeerReconciler) reconcileLeaveNetwork(ctx context.Context, peer *v1alpha1.WireflowPeer, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Leaving network", "namespace", req.Namespace, "name", req.Name)

	// Set Pending; syncs ResourceVersion back so the spec patch below doesn't conflict.
	if err := r.ensurePendingPhase(ctx, peer); err != nil {
		return ctrl.Result{}, err
	}

	// Remove network labels and clear Spec.Network.
	// Setting Network=nil increments Generation, which will trigger a re-enqueue
	// via GenerationChangedPredicate; that follow-up reconcile will see
	// ActionNone (both Spec and Status cleared) and go straight to lastReconcile.
	if _, err := r.updateSpec(ctx, peer, func(node *v1alpha1.WireflowPeer) error {
		lbls := node.GetLabels()
		for label := range lbls {
			if strings.HasPrefix(label, "wireflow.run/network-") {
				delete(lbls, label)
			}
		}
		node.SetLabels(lbls)
		node.Spec.Network = nil
		return nil
	}); err != nil {
		return ctrl.Result{}, err
	}

	// Clear status so determineAction returns ActionNone on the next reconcile.
	if _, err := r.updateStatus(ctx, peer, func(node *v1alpha1.WireflowPeer) {
		node.Status.ActiveNetwork = nil
		node.Status.AllocatedAddress = nil
		node.Status.Phase = v1alpha1.NodePhaseReady
	}); err != nil {
		return ctrl.Result{}, err
	}

	return r.lastReconcile(ctx, peer, req)
}

// updateSpec applies updateFunc to a deep copy of node and patches only when
// something actually changed. Returns (true, nil) when a patch was written so
// the caller can exit early and let the next reconcile continue the flow.
func (r *PeerReconciler) updateSpec(ctx context.Context, node *v1alpha1.WireflowPeer, updateFunc func(node *v1alpha1.WireflowPeer) error) (bool, error) {
	log := logf.FromContext(ctx)

	nodeCopy := node.DeepCopy()
	if err := updateFunc(nodeCopy); err != nil {
		log.Error(err, "Failed to build WireflowPeer Spec update")
		return false, err
	}

	// Check before patching — skip empty API calls that could spuriously
	// update ResourceVersion and trigger further watch events.
	if reflect.DeepEqual(nodeCopy.Spec, node.Spec) &&
		reflect.DeepEqual(nodeCopy.Labels, node.Labels) &&
		reflect.DeepEqual(nodeCopy.Annotations, node.Annotations) {
		return false, nil
	}

	if err := r.Patch(ctx, nodeCopy, client.MergeFrom(node)); err != nil {
		if errors.IsConflict(err) {
			log.Info("Conflict detected during WireflowPeer Spec patch, will retry on next reconcile.")
			return false, nil
		}
		log.Error(err, "Failed to patch WireflowPeer Spec")
		return false, err
	}

	// Sync patched state back into the caller's pointer so subsequent patches
	// in the same reconcile use the updated ResourceVersion (avoids conflicts).
	node.Spec = nodeCopy.Spec
	node.Labels = nodeCopy.Labels
	node.Annotations = nodeCopy.Annotations
	node.ResourceVersion = nodeCopy.ResourceVersion

	log.Info("WireflowPeer Spec/Metadata patched")
	return true, nil
}

// updateStatus applies updateFunc to a deep copy of node and patches the status
// subresource only when something actually changed.
func (r *PeerReconciler) updateStatus(ctx context.Context, node *v1alpha1.WireflowPeer, updateFunc func(node *v1alpha1.WireflowPeer)) (bool, error) {
	log := logf.FromContext(ctx)

	nodeCopy := node.DeepCopy()
	updateFunc(nodeCopy)

	// Check before patching — status patches that produce no diff still
	// update ResourceVersion on some API server versions.
	if reflect.DeepEqual(nodeCopy.Status, node.Status) {
		return false, nil
	}

	if err := r.Status().Patch(ctx, nodeCopy, client.MergeFrom(node)); err != nil {
		if errors.IsConflict(err) {
			log.Info("Conflict detected during WireflowPeer Status patch, will retry on next reconcile.")
			return false, nil
		}
		log.Error(err, "Failed to patch WireflowPeer Status")
		return false, err
	}

	// Sync patched state back so the next patch in the same reconcile sees the
	// updated ResourceVersion without an extra r.Get().
	node.Status = nodeCopy.Status
	node.ResourceVersion = nodeCopy.ResourceVersion

	log.Info("WireflowPeer Status patched")
	return true, nil
}

func (r *PeerReconciler) determineAction(ctx context.Context, node *v1alpha1.WireflowPeer) (Action, error) {
	log := logf.FromContext(ctx)
	log.Info("Determine action for node", "namespace", node.Namespace, "name", node.Name)
	specNet, activeNet := node.Spec.Network, node.Status.ActiveNetwork

	if specNet == nil {
		if activeNet == nil {
			return ActionNone, nil
		} else {
			return NodeLeaveNetwork, nil
		}
	} else {
		if activeNet == nil {
			return NodeJoinNetwork, nil
		}

		if *specNet == *activeNet {
			return ActionNone, nil
		}

		return NodeJoinNetwork, nil
	}

}

// SetupWithManager sets up the controller with the Manager.
func (r *PeerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Recorder == nil {
		r.Recorder = mgr.GetEventRecorderFor("wireflow-peer-controller")
	}

	if r.generator == nil {
		r.generator = NewGenerator(mgr.GetClient())
	}

	// 定义一个只管更新的过滤器
	onlyUpdatePredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			// 启动时的全量加载会触发这里，返回 false 拦截掉
			return false
		},
	}

	ownedCMPredicate := predicate.NewPredicateFuncs(func(obj client.Object) bool {
		return obj.GetLabels()["app.kubernetes.io/managed-by"] == "wireflow-controller"
	})

	configMapPredicate := predicate.Funcs{
		// 不响应 Create：控制器自己 Create configmap 后无需再 enqueue peer（
		// Status.CurrentHash 已写入，下次 reconcile 会直接跳过）
		CreateFunc: func(e event.CreateEvent) bool { return false },
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldCm := e.ObjectOld.(*corev1.ConfigMap)
			newCm := e.ObjectNew.(*corev1.ConfigMap)
			// 只有 Data 内容真正变了，才触发 map 函数（外部修改才需要自愈）
			return !reflect.DeepEqual(oldCm.Data, newCm.Data)
		},
	}
	// 监听 WireflowNetwork 的 spec 变化（generation changed）以及 ActiveCIDR 从空变非空（status patch）。
	// 使用自定义 predicate 是因为 GenerationChangedPredicate 只检测 spec 变化，
	// 而 NetworkReconciler 分配 ActiveCIDR 是 status patch，不改变 generation。
	networkReadyPredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool { return false },
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldNet, ok1 := e.ObjectOld.(*v1alpha1.WireflowNetwork)
			newNet, ok2 := e.ObjectNew.(*v1alpha1.WireflowNetwork)
			if !ok1 || !ok2 {
				return false
			}
			// spec 变化 或 ActiveCIDR 刚被分配 → 触发 peer reconcile
			return oldNet.Generation != newNet.Generation ||
				(oldNet.Status.ActiveCIDR == "" && newNet.Status.ActiveCIDR != "")
		},
		DeleteFunc:  func(e event.DeleteEvent) bool { return false },
		GenericFunc: func(e event.GenericEvent) bool { return false },
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.WireflowPeer{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(&v1alpha1.WireflowNetwork{},
			handler.EnqueueRequestsFromMapFunc(r.mapNetworkForNodes),
			builder.WithPredicates(networkReadyPredicate)).
		Watches(&v1alpha1.WireflowEndpoint{},
			handler.EnqueueRequestsFromMapFunc(r.mapEndpointForNodes),
			builder.WithPredicates(predicate.And(onlyUpdatePredicate, predicate.GenerationChangedPredicate{}))).
		Watches(&corev1.ConfigMap{},
			handler.EnqueueRequestsFromMapFunc(r.mapConfigMapForNodes),
			builder.WithPredicates(predicate.And(configMapPredicate, ownedCMPredicate))).
		Watches(&v1alpha1.WireflowPolicy{},
			handler.EnqueueRequestsFromMapFunc(r.mapPolicyForNodes),
			// 不加 onlyUpdatePredicate：新建策略（Create）必须触发 peer reconcile 才能下发配置；
			// GenerationChangedPredicate 默认放行 Create/Delete，只过滤 generation 未变的 Update。
			builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Named("node").WithOptions(controller.Options{
		MaxConcurrentReconciles: 5,
	}).Complete(r)
}

// mapNetworkForNodes returns a list of Reconcile Requests for Peers that should be updated based on the given WireflowNetwork.
func (r *PeerReconciler) mapNetworkForNodes(ctx context.Context, obj client.Object) []reconcile.Request {
	network := obj.(*v1alpha1.WireflowNetwork)
	var requests []reconcile.Request

	// 只获取真正加入了该网络的 WireflowPeer（通过网络标签），避免空 PeerSelector 匹配所有 peer
	networkLabel := fmt.Sprintf("wireflow.run/network-%s", network.Name)
	nodeList := &v1alpha1.WireflowPeerList{}
	if err := r.List(ctx, nodeList, client.InNamespace(network.Namespace), client.MatchingLabels(map[string]string{networkLabel: "true"})); err != nil {
		return nil
	}

	// 2. 将所有匹配的 WireflowPeer 加入请求队列
	for _, node := range nodeList.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: node.Namespace,
				Name:      node.Name,
			},
		})
	}
	return requests
}

func (r *PeerReconciler) mapConfigMapForNodes(ctx context.Context, obj client.Object) []reconcile.Request {
	cm := obj.(*corev1.ConfigMap)
	var requests []reconcile.Request

	// 1. 获取所有 WireflowPeer (或只获取匹配 WireflowNetwork.Spec.PeerSelector 的 WireflowPeer)
	var node v1alpha1.WireflowPeer
	name := strings.TrimSuffix(cm.Name, "-config")
	if err := r.Get(ctx, types.NamespacedName{Namespace: cm.Namespace, Name: name}, &node); err != nil {
		return nil
	}

	// 2. 将所有匹配的 WireflowPeer 加入请求队列
	requests = append(requests, reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: node.Namespace,
			Name:      node.Name,
		},
	})
	return requests
}

func (r *PeerReconciler) mapEndpointForNodes(ctx context.Context, obj client.Object) []reconcile.Request {
	endpoint := obj.(*v1alpha1.WireflowEndpoint)
	var requests []reconcile.Request

	//获取所有nsName下的WireflowPeer
	peerList := &v1alpha1.WireflowPeerList{}

	// 使用 ListOptions 锁定命名空间
	listOpts := []client.ListOption{
		client.InNamespace(endpoint.Namespace),
	}

	if err := r.List(ctx, peerList, listOpts...); err != nil {
		return nil
	}

	// 2. 将所有匹配的 WireflowPeer 加入请求队列
	for _, item := range peerList.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: item.Namespace,
				Name:      item.Name,
			},
		})
	}

	return requests
}

// mapPolicyForNodes returns reconcile requests for peers affected by a policy change.
// Peers are scoped to the policy's network to avoid triggering peers from other
// networks in the same namespace (an empty PeerSelector would otherwise match all).
func (r *PeerReconciler) mapPolicyForNodes(ctx context.Context, obj client.Object) []reconcile.Request {
	policy := obj.(*v1alpha1.WireflowPolicy)

	selector, err := metav1.LabelSelectorAsSelector(&policy.Spec.PeerSelector)
	if err != nil {
		return nil
	}

	listOpts := []client.ListOption{
		client.InNamespace(policy.Namespace),
		client.MatchingLabelsSelector{Selector: selector},
	}
	// Scope to the policy's network so that an empty PeerSelector only reaches
	// peers that are actually part of this network, not all peers in the namespace.
	if policy.Spec.Network != "" {
		listOpts = append(listOpts, client.MatchingLabels{
			fmt.Sprintf("wireflow.run/network-%s", policy.Spec.Network): "true",
		})
	}

	var nodeList v1alpha1.WireflowPeerList
	if err = r.List(ctx, &nodeList, listOpts...); err != nil {
		return nil
	}

	requests := make([]reconcile.Request, 0, len(nodeList.Items))
	for _, node := range nodeList.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: node.Namespace,
				Name:      node.Name,
			},
		})
	}
	return requests
}

// getNetwork 会获取所有的Networks，正向声明的或者反向声明的都包含
func (r *PeerReconciler) getNetwork(ctx context.Context, peer *v1alpha1.WireflowPeer) (*v1alpha1.WireflowNetwork, error) {

	// 1. 获取所有 WireflowNetwork 资源 (用于反向检查)
	var network v1alpha1.WireflowNetwork
	if err := r.Get(ctx, types.NamespacedName{Namespace: peer.Namespace, Name: *peer.Spec.Network}, &network); err != nil {
		return nil, fmt.Errorf("failed to get joined network: %w", err)
	}

	return &network, nil
}

func (r *PeerReconciler) getPeerStateSnapshot(ctx context.Context, current *v1alpha1.WireflowPeer, req ctrl.Request) *PeerStateSnapshot {
	var (
		err error
	)
	snapshot := &PeerStateSnapshot{
		Peer:   current,
		Labels: current.GetLabels(),
	}

	// 获取网络信息
	if current.Spec.Network != nil {
		networkName := *current.Spec.Network
		var network v1alpha1.WireflowNetwork
		if err = r.Get(ctx, types.NamespacedName{
			Namespace: req.Namespace, Name: networkName,
		}, &network); err != nil {
			return snapshot
		}
		snapshot.Network = &network

		var peerList *v1alpha1.WireflowPeerList
		peerList, err = r.findPeersByNetwork(ctx, &network)
		if err != nil {
			return snapshot
		}
		for _, item := range peerList.Items {
			snapshot.Peers = append(snapshot.Peers, &item)
		}
	}

	//获取网络策略
	policyList, err := r.filterPoliciesForNode(ctx, snapshot.Peer)
	if err != nil {
		return snapshot
	}

	snapshot.Policies = policyList

	return snapshot
}

func (r *PeerReconciler) findPeersByNetwork(ctx context.Context, network *v1alpha1.WireflowNetwork) (*v1alpha1.WireflowPeerList, error) {
	labels := map[string]string{
		fmt.Sprintf("wireflow.run/network-%s", network.Name): "true",
	}

	var peers v1alpha1.WireflowPeerList
	if err := r.List(ctx, &peers, client.InNamespace(network.Namespace), client.MatchingLabels(labels)); err != nil {
		return nil, err
	}

	return &peers, nil
}

func (r *PeerReconciler) filterPoliciesForNode(ctx context.Context, peer *v1alpha1.WireflowPeer) ([]*v1alpha1.WireflowPolicy, error) {
	var policyList v1alpha1.WireflowPolicyList
	if err := r.List(ctx, &policyList, client.InNamespace(peer.Namespace)); err != nil {
		return nil, err
	}

	matched := make([]*v1alpha1.WireflowPolicy, 0)
	nodeLabelSet := labels.Set(peer.Labels)

	peerNetwork := ""
	if peer.Spec.Network != nil {
		peerNetwork = *peer.Spec.Network
	}

	for _, policy := range policyList.Items {
		// 只匹配同一网络的 policy，避免其他网络的 policy 影响当前 peer 的 hash
		if policy.Spec.Network != peerNetwork {
			continue
		}

		selector, err := metav1.LabelSelectorAsSelector(&policy.Spec.PeerSelector)
		if err != nil {
			// selector 写错时：你可以选择忽略该 policy 或直接返回错误
			return nil, fmt.Errorf("invalid nodeSelector in policy %s/%s: %w", policy.Namespace, policy.Name, err)
		}

		// 空 selector {} 会匹配所有对象（这点要注意）
		if selector.Matches(nodeLabelSet) {
			matched = append(matched, &policy)
		}
	}

	return matched, nil
}
