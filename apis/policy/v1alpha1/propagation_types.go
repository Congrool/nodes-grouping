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

// FieldSelector is a field filter.
type FieldSelector struct {
	// A list of field selector requirements.
	MatchExpressions []corev1.NodeSelectorRequirement `json:"matchExpressions,omitempty"`
}

// Placement represents the rule for select clusters.
type Placement struct {
	// ClusterAffinity represents scheduling restrictions to a certain set of clusters.
	// If not set, any cluster can be scheduling candidate.
	// +optional
	ClusterAffinity *ClusterAffinity `json:"clusterAffinity,omitempty"`

	// ClusterTolerations represents the tolerations.
	// +optional
	ClusterTolerations []corev1.Toleration `json:"clusterTolerations,omitempty"`

	// SpreadConstraints represents a list of the scheduling constraints.
	// +optional
	SpreadConstraints []SpreadConstraint `json:"spreadConstraints,omitempty"`

	// ReplicaScheduling represents the scheduling policy on dealing with the number of replicas
	// when propagating resources that have replicas in spec (e.g. deployments, statefulsets) to member clusters.
	// +optional
	ReplicaScheduling *ReplicaSchedulingStrategy `json:"replicaScheduling,omitempty"`
}

// SpreadFieldValue is the type to define valid values for SpreadConstraint.SpreadByField
type SpreadFieldValue string

// Available fields for spreading are: cluster, region, zone, and provider.
const (
	SpreadByFieldCluster  SpreadFieldValue = "cluster"
	SpreadByFieldRegion   SpreadFieldValue = "region"
	SpreadByFieldZone     SpreadFieldValue = "zone"
	SpreadByFieldProvider SpreadFieldValue = "provider"
)

// SpreadConstraint represents the spread constraints on resources.
type SpreadConstraint struct {
	// SpreadByField represents the fields on Karmada cluster API used for
	// dynamically grouping member clusters into different groups.
	// Resources will be spread among different cluster groups.
	// Available fields for spreading are: cluster, region, zone, and provider.
	// SpreadByField should not co-exist with SpreadByLabel.
	// If both SpreadByField and SpreadByLabel are empty, SpreadByField will be set to "cluster" by system.
	// +kubebuilder:validation:Enum=cluster;region;zone;provider
	// +optional
	SpreadByField SpreadFieldValue `json:"spreadByField,omitempty"`

	// SpreadByLabel represents the label key used for
	// grouping member clusters into different groups.
	// Resources will be spread among different cluster groups.
	// SpreadByLabel should not co-exist with SpreadByField.
	// +optional
	SpreadByLabel string `json:"spreadByLabel,omitempty"`

	// MaxGroups restricts the maximum number of cluster groups to be selected.
	// +optional
	MaxGroups int `json:"maxGroups,omitempty"`

	// MinGroups restricts the minimum number of cluster groups to be selected.
	// Defaults to 1.
	// +optional
	MinGroups int `json:"minGroups,omitempty"`
}

// ClusterAffinity represents the filter to select clusters.
type ClusterAffinity struct {
	// LabelSelector is a filter to select member clusters by labels.
	// If non-nil and non-empty, only the clusters match this filter will be selected.
	// +optional
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`

	// FieldSelector is a filter to select member clusters by fields.
	// If non-nil and non-empty, only the clusters match this filter will be selected.
	// +optional
	FieldSelector *FieldSelector `json:"fieldSelector,omitempty"`

	// ClusterNames is the list of clusters to be selected.
	// +optional
	ClusterNames []string `json:"clusterNames,omitempty"`

	// ExcludedClusters is the list of clusters to be ignored.
	// +optional
	ExcludeClusters []string `json:"exclude,omitempty"`
}

// ReplicaSchedulingType describes scheduling methods for the "replicas" in a resouce.
type ReplicaSchedulingType string

const (
	// ReplicaSchedulingTypeDuplicated means when propagating a resource,
	// each candidate member cluster will directly apply the original replicas.
	ReplicaSchedulingTypeDuplicated ReplicaSchedulingType = "Duplicated"
	// ReplicaSchedulingTypeDivided means when propagating a resource,
	// each candidate member cluster will get only a part of original replicas.
	ReplicaSchedulingTypeDivided ReplicaSchedulingType = "Divided"
)

// ReplicaDivisionPreference describes options of how replicas can be scheduled.
type ReplicaDivisionPreference string

const (
	// ReplicaDivisionPreferenceAggregated divides replicas into clusters as few as possible,
	// while respecting clusters' resource availabilities during the division.
	ReplicaDivisionPreferenceAggregated ReplicaDivisionPreference = "Aggregated"
	// ReplicaDivisionPreferenceWeighted divides replicas by weight according to WeightPreference.
	ReplicaDivisionPreferenceWeighted ReplicaDivisionPreference = "Weighted"
)

// ReplicaSchedulingStrategy represents the assignment strategy of replicas.
type ReplicaSchedulingStrategy struct {
	// ReplicaSchedulingType determines how the replicas is scheduled when karmada propagating
	// a resource. Valid options are Duplicated and Divided.
	// "Duplicated" duplicates the same replicas to each candidate member cluster from resource.
	// "Divided" divides replicas into parts according to number of valid candidate member
	// clusters, and exact replicas for each cluster are determined by ReplicaDivisionPreference.
	// +kubebuilder:validation:Enum=Duplicated;Divided
	// +optional
	ReplicaSchedulingType ReplicaSchedulingType `json:"replicaSchedulingType,omitempty"`

	// ReplicaDivisionPreference determines how the replicas is divided
	// when ReplicaSchedulingType is "Divided". Valid options are Aggregated and Weighted.
	// "Aggregated" divides replicas into clusters as few as possible,
	// while respecting clusters' resource availabilities during the division.
	// "Weighted" divides replicas by weight according to WeightPreference.
	// +kubebuilder:validation:Enum=Aggregated;Weighted
	// +optional
	ReplicaDivisionPreference ReplicaDivisionPreference `json:"replicaDivisionPreference,omitempty"`

	// WeightPreference describes weight for each cluster or for each group of cluster
	// If ReplicaDivisionPreference is set to "Weighted", and WeightPreference is not set, scheduler will weight all clusters the same.
	// +optional
	WeightPreference *ClusterPreferences `json:"weightPreference,omitempty"`
}

//func init() {
//	SchemeBuilder.Register(&PropagationPolicy{}, &PropagationPolicyList{})
//}
