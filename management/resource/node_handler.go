package resource

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"
	"wireflow/internal"

	"github.com/wireflowio/wireflow-controller/pkg/generated/informers/externalversions/wireflowcontroller/v1alpha1"
	listers "github.com/wireflowio/wireflow-controller/pkg/generated/listers/wireflowcontroller/v1alpha1"
	"github.com/wireflowio/wireflow-controller/pkg/utils"

	wireflowv1alpha1 "github.com/wireflowio/wireflow-controller/pkg/apis/wireflowcontroller/v1alpha1"
	"github.com/wireflowio/wireflow-controller/pkg/controller"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

type NodeEventHandler struct {
	ctx             context.Context
	informer        v1alpha1.NodeInformer
	wt              *internal.WatchManager
	queue           workqueue.TypedRateLimitingInterface[controller.WorkerItem]
	lastPushedState *StateCache
	hashMu          sync.RWMutex
	networkLister   listers.NetworkLister
}

type StateCache struct {
	states map[string]string
	sync.RWMutex
}

func NewNodeEventHandler(
	ctx context.Context,
	informer v1alpha1.NodeInformer,
	wt *internal.WatchManager,
	queue workqueue.TypedRateLimitingInterface[controller.WorkerItem]) *NodeEventHandler {
	h := &NodeEventHandler{
		ctx:      ctx,
		informer: informer,
		wt:       wt,
		queue:    queue,
		lastPushedState: &StateCache{
			states: make(map[string]string),
		},
	}

	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			node := obj.(*wireflowv1alpha1.Node)
			if time.Since(node.CreationTimestamp.Time) > 5*time.Minute {
				klog.V(4).Infof("Skipping old node during initial sync: %s", node.Name)
				return
			}
			EnqueueItem(controller.AddEvent, nil, obj, h.queue)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldNode, newNode := oldObj.(*wireflowv1alpha1.Node), newObj.(*wireflowv1alpha1.Node)
			if oldNode.ResourceVersion == newNode.ResourceVersion {
				return
			}

			// ready的时候才会推送
			if newNode.Status.Phase == wireflowv1alpha1.NodeReady {
				EnqueueItem(controller.UpdateEvent, oldObj, newObj, h.queue)
			}
		},
		DeleteFunc: func(obj interface{}) {
			EnqueueItem(controller.DeleteEvent, nil, obj, h.queue)
		},
	})

	return h
}

func (n *NodeEventHandler) EventType() EventType {
	return NodeType
}

func (n *NodeEventHandler) RunWorker(ctx context.Context) {
	for n.ProcessNextItem(ctx) {
	}
}

func (n *NodeEventHandler) syncHandler(ctx context.Context, item controller.WorkerItem) error {
	// Get the Node resource with this namespace/name
	namespace, name := item.Key.Namespace, item.Key.Name
	logger := klog.FromContext(ctx)
	switch item.EventType {
	case controller.AddEvent:
		// 新节点加入
		node := item.NewObject.(*wireflowv1alpha1.Node)
		logger.Info("Node Add event", "namespace", namespace, "name", name, "appId", node.Spec.AppId)

		// 检查节点状态，决定是否推送
		return n.reconcileNodeAdd(ctx, node)

	case controller.UpdateEvent:
		oldNode := item.OldObject.(*wireflowv1alpha1.Node)
		newNode := item.NewObject.(*wireflowv1alpha1.Node)
		logger.Info("Node Update event", "namespace", namespace, "name", name, "appId", newNode.Spec.AppId)

		// 分析变化类型，执行相应的推送
		return n.reconcileNodeUpdate(ctx, oldNode, newNode)

	case controller.DeleteEvent:
		// 节点删除
		node := item.NewObject.(*wireflowv1alpha1.Node)
		logger.Info("Node Delete event", "namespace", namespace, "name", name, "appId", node.Spec.AppId)

		return n.reconcileNodeDelete(ctx, node)
	}

	return nil
}

// reconcileNodeAdd 处理节点新增
func (n *NodeEventHandler) reconcileNodeAdd(ctx context.Context, node *wireflowv1alpha1.Node) error {
	logger := klog.FromContext(ctx)
	logger.Info("Node Add event", "node", node)

	// Node is not ready yet, skip
	if node.Status.Phase != wireflowv1alpha1.NodeReady {
		logger.Info("Node not ready, skip")
		return nil
	}

	if len(node.Spec.Network) == 0 {
		logger.Info("Node has no network, skip")
		return nil
	}

	msg := n.buildNodeConfig(ctx, node)

	// 5. 推送配置到节点
	return n.pushToNode(ctx, node, msg, internal.EventTypeNodeAdd)
}

func (n *NodeEventHandler) buildNodeConfig(ctx context.Context, node *wireflowv1alpha1.Node) *internal.Message {
	logger := klog.FromContext(ctx)
	logger.Info("Build node config", "node", node)

	return &internal.Message{
		EventType: internal.EventTypeIPChange,
		Current: &internal.Node{
			Address:    node.Spec.Address,
			AppID:      node.Spec.AppId,
			PublicKey:  node.Spec.PublicKey,
			PrivateKey: node.Spec.PrivateKey,
		},
		Network: &internal.Network{
			Nodes: make([]*internal.Node, 0),
		},
	}
}

func (n *NodeEventHandler) buildMessage(ctx context.Context, event internal.EventType, node *internal.Node, network *internal.Network) *internal.Message {
	return &internal.Message{
		EventType: event,
		Current:   node,
		Network:   network,
	}
}

func (n *NodeEventHandler) initNode(ctx context.Context, node *wireflowv1alpha1.Node) error {
	logger := klog.FromContext(ctx)
	logger.Info("Node init event", "node", node)

	return nil
}

func (n *NodeEventHandler) reconcileNodeUpdate(ctx context.Context, oldNode, newNode *wireflowv1alpha1.Node) error {
	logger := klog.FromContext(ctx)
	logger.Info("Node Update event", "oldNode", oldNode, "newNode", newNode)

	if newNode.Status.Phase != wireflowv1alpha1.NodeReady {
		logger.Info("Node not ready, skip")
		return nil
	}

	// when node is ready, push full configuration
	if len(newNode.Spec.Network) == 0 {
		logger.Info("Node has no network, skip")
		return nil
	}

	msg := n.buildNodeConfig(ctx, newNode)

	return n.pushToNode(ctx, newNode, msg, internal.EventTypeNodeUpdate)
}

func (n *NodeEventHandler) analyzeEvent(oldNode, newNode *wireflowv1alpha1.Node) (internal.EventType, error) {
	logger := klog.FromContext(context.Background())
	logger.Info("Node changed", "node", newNode.Name, "status", newNode.Status.Status)

	//新节点
	if oldNode == nil {
		logger.V(4).Info("Node is new, skip push", "appId", newNode.Spec.AppId, "status", newNode.Status.Status)
		return internal.EventTypeNone, nil
	}

	//节点还未reconcile就绪
	if newNode.Status.Phase != wireflowv1alpha1.NodeReady {
		logger.V(4).Info("Node not ready, skip push", "appId", newNode.Spec.AppId, "status", newNode.Status.Status)
		return internal.EventTypeNone, nil
	}

	//节点不在线
	if newNode.Status.Status != wireflowv1alpha1.Active {
		logger.V(4).Info("Node not active, skip push", "appId", newNode.Spec.AppId, "status", newNode.Status.Status)
		return internal.EventTypeNone, nil
	}

	// node private key changed
	if n.SpecEquals(oldNode, newNode) {
		logger.V(4).Info("Node unchanged, skip push", "appId", newNode.Spec.AppId)
		return internal.EventTypeNodeUpdate, nil
	}

	if len(newNode.Spec.Network) == 0 {
		logger.V(4).Info("Node has no network, skip push", "appId", newNode.Spec.AppId)
		return internal.EventTypeNone, nil
	}

	//网络配置了， IP地址变化
	if oldNode.Spec.Address != newNode.Spec.Address {
		logger.V(4).Info("Node address changed", "old", oldNode.Spec.Address, "new", newNode.Spec.Address)
		return internal.EventTypeIPChange, nil
	}

	// 主network 变化了
	if oldNode.Spec.Network[0] != newNode.Spec.Network[0] {
		logger.V(4).Info("Node network changed", "old", oldNode.Spec.Network, "new", newNode.Spec.Network)
		return internal.EventTypeNetworkChanged, nil
	}

	//策略变化
	if oldNode.Status.Phase == wireflowv1alpha1.NodeUpdatingPolicy && newNode.Status.Phase == wireflowv1alpha1.NodeReady {
		logger.V(4).Info("Node policy changed", "old", oldNode.Status.Phase, "new", newNode.Status.Phase)
		return internal.EventTypePolicyChanged, nil
	}

	adds, removes := utils.Differences(newNode.Spec.Network, oldNode.Spec.Network)
	if adds == nil && removes == nil {
		logger.V(4).Info("Node network unchanged", "adds", adds, "removes", removes)
		return internal.EventTypeNone, nil
	} else {
		logger.V(4).Info("Node network changed", "adds", adds, "removes", removes)
		return internal.EventTypeNodeUpdate, nil
	}

	return internal.EventTypeNone, nil
}

func (n *NodeEventHandler) reconcileNodeDelete(ctx context.Context, node *wireflowv1alpha1.Node) error {
	logger := klog.FromContext(ctx)
	logger.Info("Node Delete event", "node", node)
	return nil
}

// pushToNode 推送消息到节点（带去重检查）
func (h *NodeEventHandler) pushToNode(ctx context.Context, node *wireflowv1alpha1.Node, msg *internal.Message, eventType internal.EventType) error {
	logger := klog.FromContext(ctx)

	msg.EventType = eventType

	// 1. 计算消息哈希
	msgHash := h.computeMessageHash(msg)

	// 2. 检查是否与上次推送相同
	h.lastPushedState.RLock()
	lastHash, exists := h.lastPushedState.states[node.Spec.AppId]
	h.lastPushedState.RUnlock()

	if exists && lastHash == msgHash {
		logger.V(4).Info("Message unchanged, skipping push", "appId", node.Spec.AppId)
		return nil
	}

	// 3. 推送消息
	if err := h.wt.Send(node.Spec.AppId, msg); err != nil {
		return fmt.Errorf("failed to send message to node %s: %v", node.Spec.AppId, err)
	}

	// 4. 更新缓存
	h.lastPushedState.Lock()
	h.lastPushedState.states[node.Spec.AppId] = msgHash
	h.lastPushedState.Unlock()

	// 5. 记录日志
	b, _ := json.Marshal(msg)
	logger.Info("Pushed message to node",
		"appId", node.Spec.AppId,
		"eventType", eventType,
		"dataSize", len(b))

	return nil
}

func (h *NodeEventHandler) computeMessageHash(msg *internal.Message) string {
	data, _ := json.Marshal(msg)
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func (n *NodeEventHandler) ProcessNextItem(ctx context.Context) bool {

	item, shutdown := n.queue.Get()
	logger := klog.FromContext(ctx)

	if shutdown {
		return false
	}

	// We call Done at the end of this func so the nodeQueue knows we have
	// finished processing this item. We also must remember to call Forget
	// if we do not want this work item being re-queued. For example, we do
	// not call Forget if a transient error occurs, instead the item is
	// put back on the nodeQueue and attempted again after a back-off
	// period.
	defer n.queue.Done(item)

	// Run the syncHandler, passing it the structured reference to the object to be synced.
	err := n.syncHandler(ctx, item)
	if err == nil {
		// If no error occurs then we Forget this item so it does not
		// get queued again until another change happens.
		n.queue.Forget(item)
		logger.Info("Successfully synced", "objectName", item)
		return true
	}
	// there was a failure so be sure to report it.  This method allows for
	// pluggable error handling which can be used for things like
	// cluster-monitoring.
	utilruntime.HandleErrorWithContext(ctx, err, "Error syncing; requeuing for later retry", "objectReference", item)
	// since we failed, we should requeue the item to work on later.  This
	// method will add a backoff to avoid hotlooping on particular items
	// (they're probably still not going to work right away) and overall
	// controller protection (everything I've done is broken, this controller
	// needs to calm down or it can starve other useful work) cases.
	n.queue.AddRateLimited(item)

	return true
}

func (n *NodeEventHandler) Informer() cache.SharedIndexInformer {
	return n.informer.Informer()
}

func (n *NodeEventHandler) WorkQueue() workqueue.TypedRateLimitingInterface[controller.WorkerItem] {
	return n.queue
}

func (n *NodeEventHandler) pushIPChange(node *wireflowv1alpha1.Node) {
	msg := &internal.Message{
		EventType: internal.EventTypeIPChange,
		Current: &internal.Node{
			Name:       node.Name,
			AppID:      node.Spec.AppId,
			Address:    node.Spec.Address,
			PublicKey:  node.Spec.PublicKey,
			PrivateKey: node.Spec.PrivateKey,
		},
	}

	n.pushMessage(node.Spec.AppId, msg)
}

func (n *NodeEventHandler) pushMessage(appId string, msg *internal.Message) {
	logger := klog.Background()

	// 计算消息哈希
	msgHash := n.computeMessageHash(msg)

	// 检查是否与上次推送相同
	n.hashMu.RLock()
	lastHash, exists := n.lastPushedState.states[appId]
	n.hashMu.RUnlock()

	if exists && lastHash == msgHash {
		logger.V(4).Info("Message unchanged, skip push", "appId", appId)
		return
	}

	// 推送
	if err := n.wt.Send(appId, msg); err != nil {
		logger.Error(err, "Failed to push message", "appId", appId)
		return
	}

	// 更新哈希缓存
	n.hashMu.Lock()
	n.lastPushedState.states[appId] = msgHash
	n.hashMu.Unlock()

	b, _ := json.Marshal(msg)
	logger.Info("Pushed message", "appId", appId, "eventType", msg.EventType, "size", len(b))
}

// handleNodeDelete 处理 Node 删除
func (n *NodeEventHandler) handleNodeDelete(node *wireflowv1alpha1.Node) {
	logger := klog.Background()
	logger.Info("Node deleted", "node", node.Name)

	// 通知网络中的其他节点
	if len(node.Spec.Network) > 0 {
		for _, networkName := range node.Spec.Network {
			network, err := n.networkLister.Networks(node.Namespace).Get(networkName)
			if err != nil {
				logger.Error(err, "Failed to get network", "network", networkName)
				continue
			}

			n.notifyPeersNodeRemoved(network, node)
		}
	}

	// 清理哈希缓存
	n.hashMu.Lock()
	delete(n.lastPushedState.states, node.Spec.AppId)
	n.hashMu.Unlock()
}

func (n *NodeEventHandler) notifyPeersNodeRemoved(network *wireflowv1alpha1.Network, node *wireflowv1alpha1.Node) {
	logger := klog.Background()
	logger.Info("Notify peers node removed", "network", network.Name, "node", node.Name)

	msg := &internal.Message{
		EventType: internal.EventTypeNodeRemove,
		Current: &internal.Node{
			Name:       node.Name,
			AppID:      node.Spec.AppId,
			Address:    node.Spec.Address,
			PublicKey:  node.Spec.PublicKey,
			PrivateKey: node.Spec.PrivateKey,
		},
	}

	objs, err := n.informer.Informer().GetIndexer().ByIndex("network", network.Name)
	if err != nil {
		logger.Error(err, "Failed to get network nodes", "network", network.Name)
		return
	}

	for _, obj := range objs {
		node := obj.(*wireflowv1alpha1.Node)
		if node.Name == node.Name {
			continue
		}

		n.pushMessage(node.Spec.AppId, msg)
	}
}

func (n *NodeEventHandler) SpecEquals(old, new *wireflowv1alpha1.Node) bool {
	if old.Spec.PrivateKey != new.Spec.PrivateKey {
		return false
	}

	return true
}

func (n *NodeEventHandler) StatusEquals(old, new *wireflowv1alpha1.Node) bool {
	if old.Status.Status != new.Status.Status {
		return false
	}

	return true
}
