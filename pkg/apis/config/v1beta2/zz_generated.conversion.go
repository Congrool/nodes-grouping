//go:build !ignore_autogenerated
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
// Code generated by conversion-gen. DO NOT EDIT.

package v1beta2

import (
	unsafe "unsafe"

	config "github.com/Congrool/nodes-grouping/pkg/apis/config"
	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func init() {
	localSchemeBuilder.Register(RegisterConversions)
}

// RegisterConversions adds conversion functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterConversions(s *runtime.Scheme) error {
	if err := s.AddGeneratedConversionFunc((*NodeGroupSchedulingArgs)(nil), (*config.NodeGroupSchedulingArgs)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1beta2_NodeGroupSchedulingArgs_To_config_NodeGroupSchedulingArgs(a.(*NodeGroupSchedulingArgs), b.(*config.NodeGroupSchedulingArgs), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*config.NodeGroupSchedulingArgs)(nil), (*NodeGroupSchedulingArgs)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_config_NodeGroupSchedulingArgs_To_v1beta2_NodeGroupSchedulingArgs(a.(*config.NodeGroupSchedulingArgs), b.(*NodeGroupSchedulingArgs), scope)
	}); err != nil {
		return err
	}
	return nil
}

func autoConvert_v1beta2_NodeGroupSchedulingArgs_To_config_NodeGroupSchedulingArgs(in *NodeGroupSchedulingArgs, out *config.NodeGroupSchedulingArgs, s conversion.Scope) error {
	out.KubeConfigPath = (*string)(unsafe.Pointer(in.KubeConfigPath))
	return nil
}

// Convert_v1beta2_NodeGroupSchedulingArgs_To_config_NodeGroupSchedulingArgs is an autogenerated conversion function.
func Convert_v1beta2_NodeGroupSchedulingArgs_To_config_NodeGroupSchedulingArgs(in *NodeGroupSchedulingArgs, out *config.NodeGroupSchedulingArgs, s conversion.Scope) error {
	return autoConvert_v1beta2_NodeGroupSchedulingArgs_To_config_NodeGroupSchedulingArgs(in, out, s)
}

func autoConvert_config_NodeGroupSchedulingArgs_To_v1beta2_NodeGroupSchedulingArgs(in *config.NodeGroupSchedulingArgs, out *NodeGroupSchedulingArgs, s conversion.Scope) error {
	out.KubeConfigPath = (*string)(unsafe.Pointer(in.KubeConfigPath))
	return nil
}

// Convert_config_NodeGroupSchedulingArgs_To_v1beta2_NodeGroupSchedulingArgs is an autogenerated conversion function.
func Convert_config_NodeGroupSchedulingArgs_To_v1beta2_NodeGroupSchedulingArgs(in *config.NodeGroupSchedulingArgs, out *NodeGroupSchedulingArgs, s conversion.Scope) error {
	return autoConvert_config_NodeGroupSchedulingArgs_To_v1beta2_NodeGroupSchedulingArgs(in, out, s)
}
