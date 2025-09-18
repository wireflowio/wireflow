package resource

import (
	"context"
	"wireflow/internal"
	"wireflow/management/dto"
	"wireflow/management/entity"

	wireflowcontrollerv1alpha1 "github.com/wireflowio/wireflow-controller/pkg/apis/wireflowcontroller/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

type NodeResource struct {
	controller *Controller
}

func NewNodeResource(controller *Controller) *NodeResource {
	return &NodeResource{
		controller: controller,
	}
}

func (n *NodeResource) Register(ctx context.Context, e *dto.NodeDto) (*entity.Node, error) {
	_, err := n.controller.Clientset.WireflowcontrollerV1alpha1().Nodes("default").Create(ctx, &wireflowcontrollerv1alpha1.Node{
		TypeMeta: v1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1alpha1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name: e.PublicKey,
		},
		Spec: wireflowcontrollerv1alpha1.NodeSpec{
			NodeName: e.Name,
			ClientId: e.PublicKey,
		},

		Status: wireflowcontrollerv1alpha1.NodeStatus{
			Status: "Active",
		},
	}, v1.CreateOptions{})

	return nil, err
}

// UpdateNodeState
func (n *NodeResource) UpdateNodeState(clientId string, status internal.Status) error {
	node, err := n.controller.Clientset.WireflowcontrollerV1alpha1().Nodes("default").Get(context.Background(), clientId, v1.GetOptions{})
	if err != nil {
		klog.Errorf("get node %s failed: %v", clientId, err)
		return err
	}

	updateNode := node.DeepCopy()
	switch status {
	case internal.Active:
		updateNode.Status.Status = "Active"
	case internal.Inactive:
		updateNode.Status.Status = "Inactive"
		break
	}

	_, err = n.controller.Clientset.WireflowcontrollerV1alpha1().Nodes("default").Update(context.Background(), updateNode, v1.UpdateOptions{})
	if err != nil {
		klog.Errorf("update node %s failed: %v", clientId, err)
		return err
	}
	klog.Infof("update node %s status to %s successfully", clientId, status)

	return nil
}
