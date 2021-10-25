/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {
	// Region represents the region of the member cluster locate in.
	// +optional
	Region string `json:"region,omitempty"`

	// Zone represents the zone of the member cluster locate in.
	// +optional
	Zone string `json:"zone,omitempty"`

	// Taints attached to the member cluster.
	// Taints on the cluster have the "effect" on
	// any resource that does not tolerate the Taint.
	// +optional
	Taints []corev1.Taint `json:"taints,omitempty"`

	// Nodes contains names of all the nodes in the cluster.
	Nodes []string `json:"nodes"`

	// MatchLabels match the nodes that have the labels
	MatchLabels []string `json:"nodes"`
}

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	// NodeSummary represents the summary of nodes status in the member cluster.
	// +optional
	NodeSummary *NodeSummary `json:"nodeSummary,omitempty"`

	// ResourceSummary represents the summary of resources in the member cluster.
	// +optional
	ResourceSummary *ResourceSummary `json:"resourceSummary,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion


// Cluster is the Schema for the clusters API
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents the specification of the desired behavior of member cluster.
	Spec ClusterSpec `json:"spec"`

	// Status represents the status of member cluster.
	// +optional
	Status ClusterStatus `json:"status,omitempty"`
}

// NodeSummary represents the summary of nodes status in a specific cluster.
type NodeSummary struct {
	// TotalNum is the total number of nodes in the cluster.
	// +optional
	TotalNum int32 `json:"totalNum,omitempty"`
	// ReadyNum is the number of ready nodes in the cluster.
	// +optional
	ReadyNum int32 `json:"readyNum,omitempty"`
}

// ResourceSummary represents the summary of resources in the member cluster.
type ResourceSummary struct {
	// Allocatable represents the resources of a cluster that are available for scheduling.
	// Total amount of allocatable resources on all nodes.
	// +optional
	Allocatable corev1.ResourceList `json:"allocatable,omitempty"`
	// Allocating represents the resources of a cluster that are pending for scheduling.
	// Total amount of required resources of all Pods that are waiting for scheduling.
	// +optional
	Allocating corev1.ResourceList `json:"allocating,omitempty"`
	// Allocated represents the resources of a cluster that have been scheduled.
	// Total amount of required resources of all Pods that have been scheduled to nodes.
	// +optional
	Allocated corev1.ResourceList `json:"allocated,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

//func init() {
//	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
//}
