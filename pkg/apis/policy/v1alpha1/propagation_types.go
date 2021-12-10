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

// PropagationPolicySpec represents the desired behavior of PropagationPolicy.
type PropagationPolicySpec struct {
	// ResourceSelectors used to select resources.
	// +required
	ResourceSelectors []ResourceSelector `json:"resourceSelectors"`

	// Placement represents the rule for select nodegroups to propagate resources.
	// +optional
	Placement NodeGroupPreferences `json:"placement,omitempty"`
}

// PropagationPolicyStatus defines the observed state of PropagationPolicy
type PropagationPolicyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of nodegroup
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:resource:shortName=pp

// PropagationPolicy represents the policy that propagates a group of resources to one or more nodegroups.
type PropagationPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents the desired behavior of PropagationPolicy.
	// +required
	Spec PropagationPolicySpec `json:"spec"`
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

// NodeGroupPreferences describes weight for each nodegroups or for each group of nodegroup.
type NodeGroupPreferences struct {
	// StaticWeightList defines the static nodegroup weight.
	// +required
	StaticWeightList []StaticNodeGroupWeight `json:"staticWeightList"`
}

// StaticNodeGroupWeight defines the static NodeGroup weight.
type StaticNodeGroupWeight struct {
	// NodeGroupNames specifies nodegroups with names.
	// +required
	NodeGroupNames []string `json:"nodeGroupNames"`

	// Weight expressing the preference to the nodegroup(s) specified by 'TargetNodeGroup'.
	// +kubebuilder:validation:Minimum=1
	// +required
	Weight int64 `json:"weight"`
}
