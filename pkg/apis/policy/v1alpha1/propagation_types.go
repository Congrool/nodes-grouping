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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PropagationSpec represents the desired behavior of PropagationPolicy.
type PropagationSpec struct {
	// ResourceSelectors used to select resources.
	// +required
	ResourceSelectors []ResourceSelector `json:"resourceSelectors"`

	// Placement represents the rule for select clusters to propagate resources.
	// +optional
	Placement ClusterPreferences `json:"placement,omitempty"`
}

// PropagationPolicyStatus defines the observed state of PropagationPolicy
type PropagationPolicyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:resource:shortName=pp

// PropagationPolicy represents the policy that propagates a group of resources to one or more clusters.
type PropagationPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents the desired behavior of PropagationPolicy.
	// +required
	Spec PropagationSpec `json:"spec"`
}

//+kubebuilder:object:root=true

// PropagationPolicyList contains a list of PropagationPolicy
type PropagationPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PropagationPolicy `json:"items"`
}

// ResourceSelector the resources will be selected.
type ResourceSelector struct {
	// APIVersion represents the API version of the target resources.
	// +required
	APIVersion string `json:"apiVersion"`

	// Kind represents the Kind of the target resources.
	// +required
	Kind string `json:"kind"`

	// Namespace of the target resource.
	// Default is empty, which means inherit from the parent object scope.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Name of the target resource.
	// Default is empty, which means selecting all resources.
	// +optional
	Name string `json:"name,omitempty"`

	// A label query over a set of resources.
	// If name is not empty, labelSelector will be ignored.
	// +optional
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:shortName=rsp
// +kubebuilder:deprecatedversion

// ClusterPreferences describes weight for each cluster or for each group of cluster.
type ClusterPreferences struct {
	// StaticWeightList defines the static cluster weight.
	// +required
	StaticWeightList []StaticClusterWeight `json:"staticWeightList"`
}

// StaticClusterWeight defines the static cluster weight.
type StaticClusterWeight struct {
	// ClusterNames specifies clusters with names.
	// +required
	ClusterNames []string `json:"clusterNames"`

	// Weight expressing the preference to the cluster(s) specified by 'TargetCluster'.
	// +kubebuilder:validation:Minimum=1
	// +required
	Weight int64 `json:"weight"`
}
