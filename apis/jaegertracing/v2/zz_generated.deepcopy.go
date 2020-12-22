// +build !ignore_autogenerated

// Copyright The Jaeger Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by controller-gen. DO NOT EDIT.

package v2

import (
	"k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AutoScaleSpec) DeepCopyInto(out *AutoScaleSpec) {
	*out = *in
	if in.Autoscale != nil {
		in, out := &in.Autoscale, &out.Autoscale
		*out = new(bool)
		**out = **in
	}
	if in.MinReplicas != nil {
		in, out := &in.MinReplicas, &out.MinReplicas
		*out = new(int32)
		**out = **in
	}
	if in.MaxReplicas != nil {
		in, out := &in.MaxReplicas, &out.MaxReplicas
		*out = new(int32)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AutoScaleSpec.
func (in *AutoScaleSpec) DeepCopy() *AutoScaleSpec {
	if in == nil {
		return nil
	}
	out := new(AutoScaleSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Jaeger) DeepCopyInto(out *Jaeger) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Jaeger.
func (in *Jaeger) DeepCopy() *Jaeger {
	if in == nil {
		return nil
	}
	out := new(Jaeger)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Jaeger) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JaegerAgentSpec) DeepCopyInto(out *JaegerAgentSpec) {
	*out = *in
	if in.ImagePullSecrets != nil {
		in, out := &in.ImagePullSecrets, &out.ImagePullSecrets
		*out = make([]v1.LocalObjectReference, len(*in))
		copy(*out, *in)
	}
	in.JaegerCommonSpec.DeepCopyInto(&out.JaegerCommonSpec)
	if in.SidecarSecurityContext != nil {
		in, out := &in.SidecarSecurityContext, &out.SidecarSecurityContext
		*out = new(v1.SecurityContext)
		(*in).DeepCopyInto(*out)
	}
	if in.HostNetwork != nil {
		in, out := &in.HostNetwork, &out.HostNetwork
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JaegerAgentSpec.
func (in *JaegerAgentSpec) DeepCopy() *JaegerAgentSpec {
	if in == nil {
		return nil
	}
	out := new(JaegerAgentSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JaegerAllInOneSpec) DeepCopyInto(out *JaegerAllInOneSpec) {
	*out = *in
	in.JaegerCommonSpec.DeepCopyInto(&out.JaegerCommonSpec)
	if in.TracingEnabled != nil {
		in, out := &in.TracingEnabled, &out.TracingEnabled
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JaegerAllInOneSpec.
func (in *JaegerAllInOneSpec) DeepCopy() *JaegerAllInOneSpec {
	if in == nil {
		return nil
	}
	out := new(JaegerAllInOneSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JaegerCollectorSpec) DeepCopyInto(out *JaegerCollectorSpec) {
	*out = *in
	in.AutoScaleSpec.DeepCopyInto(&out.AutoScaleSpec)
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
	in.JaegerCommonSpec.DeepCopyInto(&out.JaegerCommonSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JaegerCollectorSpec.
func (in *JaegerCollectorSpec) DeepCopy() *JaegerCollectorSpec {
	if in == nil {
		return nil
	}
	out := new(JaegerCollectorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JaegerCommonSpec) DeepCopyInto(out *JaegerCommonSpec) {
	*out = *in
	if in.Volumes != nil {
		in, out := &in.Volumes, &out.Volumes
		*out = make([]v1.Volume, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.VolumeMounts != nil {
		in, out := &in.VolumeMounts, &out.VolumeMounts
		*out = make([]v1.VolumeMount, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	in.Resources.DeepCopyInto(&out.Resources)
	if in.Affinity != nil {
		in, out := &in.Affinity, &out.Affinity
		*out = new(v1.Affinity)
		(*in).DeepCopyInto(*out)
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.SecurityContext != nil {
		in, out := &in.SecurityContext, &out.SecurityContext
		*out = new(v1.PodSecurityContext)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JaegerCommonSpec.
func (in *JaegerCommonSpec) DeepCopy() *JaegerCommonSpec {
	if in == nil {
		return nil
	}
	out := new(JaegerCommonSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JaegerIngesterSpec) DeepCopyInto(out *JaegerIngesterSpec) {
	*out = *in
	in.AutoScaleSpec.DeepCopyInto(&out.AutoScaleSpec)
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
	in.JaegerCommonSpec.DeepCopyInto(&out.JaegerCommonSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JaegerIngesterSpec.
func (in *JaegerIngesterSpec) DeepCopy() *JaegerIngesterSpec {
	if in == nil {
		return nil
	}
	out := new(JaegerIngesterSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JaegerIngressSpec) DeepCopyInto(out *JaegerIngressSpec) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
	if in.Hosts != nil {
		in, out := &in.Hosts, &out.Hosts
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.TLS != nil {
		in, out := &in.TLS, &out.TLS
		*out = make([]JaegerIngressTLSSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.JaegerCommonSpec.DeepCopyInto(&out.JaegerCommonSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JaegerIngressSpec.
func (in *JaegerIngressSpec) DeepCopy() *JaegerIngressSpec {
	if in == nil {
		return nil
	}
	out := new(JaegerIngressSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JaegerIngressTLSSpec) DeepCopyInto(out *JaegerIngressTLSSpec) {
	*out = *in
	if in.Hosts != nil {
		in, out := &in.Hosts, &out.Hosts
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JaegerIngressTLSSpec.
func (in *JaegerIngressTLSSpec) DeepCopy() *JaegerIngressTLSSpec {
	if in == nil {
		return nil
	}
	out := new(JaegerIngressTLSSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JaegerList) DeepCopyInto(out *JaegerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Jaeger, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JaegerList.
func (in *JaegerList) DeepCopy() *JaegerList {
	if in == nil {
		return nil
	}
	out := new(JaegerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *JaegerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JaegerQuerySpec) DeepCopyInto(out *JaegerQuerySpec) {
	*out = *in
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
	in.JaegerCommonSpec.DeepCopyInto(&out.JaegerCommonSpec)
	if in.TracingEnabled != nil {
		in, out := &in.TracingEnabled, &out.TracingEnabled
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JaegerQuerySpec.
func (in *JaegerQuerySpec) DeepCopy() *JaegerQuerySpec {
	if in == nil {
		return nil
	}
	out := new(JaegerQuerySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JaegerSpec) DeepCopyInto(out *JaegerSpec) {
	*out = *in
	in.AllInOne.DeepCopyInto(&out.AllInOne)
	in.Query.DeepCopyInto(&out.Query)
	in.Collector.DeepCopyInto(&out.Collector)
	in.Ingester.DeepCopyInto(&out.Ingester)
	in.Agent.DeepCopyInto(&out.Agent)
	in.Ingress.DeepCopyInto(&out.Ingress)
	in.JaegerCommonSpec.DeepCopyInto(&out.JaegerCommonSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JaegerSpec.
func (in *JaegerSpec) DeepCopy() *JaegerSpec {
	if in == nil {
		return nil
	}
	out := new(JaegerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JaegerStatus) DeepCopyInto(out *JaegerStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JaegerStatus.
func (in *JaegerStatus) DeepCopy() *JaegerStatus {
	if in == nil {
		return nil
	}
	out := new(JaegerStatus)
	in.DeepCopyInto(out)
	return out
}
