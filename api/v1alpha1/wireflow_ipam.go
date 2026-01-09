package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:object:root=true

type WireflowGlobalIPPool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              WireflowGlobalIPPoolSpec   `json:"spec"`
	Status            WireflowGlobalIPPoolStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type WireflowGlobalIPPoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
}

// WireflowGlobalIPPoolSpec define global ip pool
type WireflowGlobalIPPoolSpec struct {
	CIDR       string `json:"cidr"`       // 例如 "10.0.0.0/8"
	SubnetMask int    `json:"subnetMask"` // 每个 Network 分配多大，例如 24
}

type WireflowGlobalIPPoolStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:resource:shortName=wfsubnet

// WireflowSubnetAllocation for store / search a network's cidr
// 它的 Name 格式定为: subnet-<hex-ip> (例如 subnet-0a0a0100)
type WireflowSubnetAllocation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              WireflowSubnetAllocationSpec `json:"spec"`
}

// +kubebuilder:object:root=true

type WireflowSubnetAllocationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
}

type WireflowSubnetAllocationSpec struct {
	NetworkName string `json:"networkName"` // 归属于哪个 Network
	CIDR        string `json:"cidr"`        // 实际分配的段，如 10.10.1.0/24
}

func init() {
	SchemeBuilder.Register(&WireflowGlobalIPPool{}, &WireflowGlobalIPPoolList{})
	SchemeBuilder.Register(&WireflowSubnetAllocation{}, &WireflowSubnetAllocationList{})
}
