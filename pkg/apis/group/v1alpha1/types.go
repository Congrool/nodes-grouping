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

// NodeGroupSpec defines the desired state of NodeGroup
type NodeGroupSpec struct {
	// Nodes contains names of all the nodes in the nodegroup.
	Nodes []string `json:"nodes"`

	// MatchLabels match the nodes that have the labels
	MatchLabels map[string]string `json:"matchLables,omitempty"`
}

// NodeGroupStatus defines the observed state of NodeGroup
type NodeGroupStatus struct {
	// ContainedNodes represents names of all nodes the nodegroup contains.
	// +optional
	ContainedNodes []string `json:"containedNodes,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// NodeGroup is the Schema for the nodegroups API
type NodeGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents the specification of the desired behavior of member nodegroup.
	Spec NodeGroupSpec `json:"spec"`

	// Status represents the status of member nodegroup.
	// +optional
	Status NodeGroupStatus `json:"status,omitempty"`
}
