package resource

import (
	"context"
	"fmt"
	"github.com/wireflowio/wireflow-controller/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"time"
	"wireflow/internal"

	wireflowv1alpha1 "github.com/wireflowio/wireflow-controller/pkg/apis/wireflowcontroller/v1alpha1"
	"github.com/wireflowio/wireflow-controller/pkg/controller"
	listers "github.com/wireflowio/wireflow-controller/pkg/generated/listers/wireflowcontroller/v1alpha1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

type NetworkEventHandler struct {
	ctx        context.Context
	informer   cache.SharedIndexInformer
	nodeLister listers.NodeLister
	wt         *internal.WatchManager
	queue      workqueue.TypedRateLimitingInterface[controller.WorkerItem]
}

func (n *NetworkEventHandler) AddFunc(obj interface{})         {}
func (n *NetworkEventHandler) UpdateFunc(old, new interface{}) {}
func (n *NetworkEventHandler) DeleteFunc(obj interface{})      {}
func (n *NetworkEventHandler) EventType() EventType {
	return NetworkType
}

func NewNetworkEventHandler(ctx context.Context, informer cache.SharedIndexInformer, wt *internal.WatchManager, queue workqueue.TypedRateLimitingInterface[controller.WorkerItem]) *NetworkEventHandler {
	h := &NetworkEventHandler{
		ctx:      ctx,
		informer: informer,
		wt:       wt,
		queue:    queue,
	}

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			network := obj.(*wireflowv1alpha1.Network)
			if time.Since(network.CreationTimestamp.Time) > 5*time.Minute {
				klog.V(4).Infof("Skipping old network during initial sync: %s", network.Name)
				return
			}
			EnqueueItem(controller.AddEvent, nil, obj, h.queue)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldNetwork, newNetwork := oldObj.(*wireflowv1alpha1.Network), newObj.(*wireflowv1alpha1.Network)
			if oldNetwork.ResourceVersion == newNetwork.ResourceVersion {
				return
			}
			EnqueueItem(controller.UpdateEvent, oldObj, newObj, h.queue)
		},
		DeleteFunc: func(obj interface{}) {
			EnqueueItem(controller.DeleteEvent, nil, obj, h.queue)
		},
	})

	return h
}

func (n *NetworkEventHandler) Informer() cache.SharedIndexInformer {
	return n.informer
}

func (n *NetworkEventHandler) RunWorker(ctx context.Context) {
	for n.ProcessNextItem(ctx) {
	}
}

func (n *NetworkEventHandler) ProcessNextItem(ctx context.Context) bool {
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

func (n *NetworkEventHandler) syncHandler(ctx context.Context, item controller.WorkerItem) error {
	// Get the Node resource with this namespace/name
	namespace, name := item.Key.Namespace, item.Key.Name
	logger := klog.FromContext(ctx)

	switch item.EventType {
	case controller.AddEvent:
		logger.Info("Add event", "namespace", namespace, "name", name, "resource type", "Network")
	case controller.UpdateEvent:
		oldNet, newNet := item.OldObject.(*wireflowv1alpha1.Network), item.NewObject.(*wireflowv1alpha1.Network)
		logger.Info("Update event", "namespace", namespace, "name", name, "resource type", "Network")

		adds, removes := utils.Differences(oldNet.Spec.Nodes, newNet.Spec.Nodes)
		// need seed to client to update node's dst nodes.
		if len(removes) > 0 {
			err := handleNetworkNodes(ctx, "remove", removes, n, namespace)
			if err != nil {
				return err
			}
		}

		if len(adds) > 0 {
			// add node for current node list
			err := handleNetworkNodes(ctx, "add", adds, n, namespace)
			if err != nil {
				return err
			}
		}

	case controller.DeleteEvent:
		logger.Info("Delete event", "namespace", namespace, "name", name, "resource type", "Network")
	}

	return nil
}

func handleNetworkNodes(ctx context.Context, event string, handleNodes []string, n *NetworkEventHandler, namespace string) error {
	nodes := make([]*wireflowv1alpha1.Node, 0)
	allNodes := make([]*internal.Node, 0)
	for _, nodeName := range handleNodes {
		// remove node from old node list
		node, err := n.nodeLister.Nodes(namespace).Get(nodeName)
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return fmt.Errorf("failed to get node %s: %v", nodeName, err)
		}
		handleNode := new(internal.Node)
		handleNode.Address = node.Spec.Address
		handleNode.PublicKey = node.Spec.PublicKey
		handleNode.PrivateKey = node.Spec.ClientId
		allNodes = append(allNodes, handleNode)
		nodes = append(nodes, node)
	}

	for _, node := range nodes {
		n.handelNodeType(ctx, event, node, allNodes)
	}
	return nil
}

func (n *NetworkEventHandler) WorkQueue() workqueue.TypedRateLimitingInterface[controller.WorkerItem] {
	return n.queue
}

func (n *NetworkEventHandler) handelNodeType(ctx context.Context, event string, node *wireflowv1alpha1.Node, removed []*internal.Node) {
	logger := klog.FromContext(ctx)
	msg := new(internal.Message)
	switch event {
	case event:
	case "remove":
		msg.EventType = internal.EventTypeNodeRemove
	case "add":
		msg.EventType = internal.EventTypeNodeAdd
	}
	msg.Current.Address = node.Spec.Address
	msg.Current.PublicKey = node.Spec.PublicKey
	msg.Current.PrivateKey = node.Spec.ClientId

	msg.Network = new(internal.Network)
	msg.Network.Nodes = append(msg.Network.Nodes, removed...)
	n.wt.Send(msg.Current.PublicKey, msg)
	logger.Info("Node IP address send to client success", "address", node.Spec.Address)
}
