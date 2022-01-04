package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeGroupSchedulingArgs holds arguments used to configure NodeGroupScheduling plugin.
type NodeGroupSchedulingArgs struct {
	metav1.TypeMeta

	KubeConfigPath string
}
