// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeGroupPreferences) DeepCopyInto(out *NodeGroupPreferences) {
	*out = *in
	if in.StaticWeightList != nil {
		in, out := &in.StaticWeightList, &out.StaticWeightList
		*out = make([]StaticNodeGroupWeight, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeGroupPreferences.
func (in *NodeGroupPreferences) DeepCopy() *NodeGroupPreferences {
	if in == nil {
		return nil
	}
	out := new(NodeGroupPreferences)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PropagationPolicy) DeepCopyInto(out *PropagationPolicy) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PropagationPolicy.
func (in *PropagationPolicy) DeepCopy() *PropagationPolicy {
	if in == nil {
		return nil
	}
	out := new(PropagationPolicy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PropagationPolicy) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PropagationPolicyList) DeepCopyInto(out *PropagationPolicyList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]PropagationPolicy, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PropagationPolicyList.
func (in *PropagationPolicyList) DeepCopy() *PropagationPolicyList {
	if in == nil {
		return nil
	}
	out := new(PropagationPolicyList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PropagationPolicyList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PropagationPolicySpec) DeepCopyInto(out *PropagationPolicySpec) {
	*out = *in
	if in.ResourceSelectors != nil {
		in, out := &in.ResourceSelectors, &out.ResourceSelectors
		*out = make([]ResourceSelector, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.Placement.DeepCopyInto(&out.Placement)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PropagationPolicySpec.
func (in *PropagationPolicySpec) DeepCopy() *PropagationPolicySpec {
	if in == nil {
		return nil
	}
	out := new(PropagationPolicySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PropagationPolicyStatus) DeepCopyInto(out *PropagationPolicyStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PropagationPolicyStatus.
func (in *PropagationPolicyStatus) DeepCopy() *PropagationPolicyStatus {
	if in == nil {
		return nil
	}
	out := new(PropagationPolicyStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceSelector) DeepCopyInto(out *ResourceSelector) {
	*out = *in
	if in.LabelSelector != nil {
		in, out := &in.LabelSelector, &out.LabelSelector
		*out = new(v1.LabelSelector)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceSelector.
func (in *ResourceSelector) DeepCopy() *ResourceSelector {
	if in == nil {
		return nil
	}
	out := new(ResourceSelector)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StaticNodeGroupWeight) DeepCopyInto(out *StaticNodeGroupWeight) {
	*out = *in
	if in.NodeGroupNames != nil {
		in, out := &in.NodeGroupNames, &out.NodeGroupNames
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StaticNodeGroupWeight.
func (in *StaticNodeGroupWeight) DeepCopy() *StaticNodeGroupWeight {
	if in == nil {
		return nil
	}
	out := new(StaticNodeGroupWeight)
	in.DeepCopyInto(out)
	return out
}
