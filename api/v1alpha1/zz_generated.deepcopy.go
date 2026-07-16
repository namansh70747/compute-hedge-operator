package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto copies the receiver into out.
func (in *WorkloadRef) DeepCopyInto(out *WorkloadRef) {
	*out = *in
}

// DeepCopy returns a deep copy of WorkloadRef.
func (in *WorkloadRef) DeepCopy() *WorkloadRef {
	if in == nil {
		return nil
	}
	out := new(WorkloadRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies the receiver into out.
func (in *ComputePositionSpec) DeepCopyInto(out *ComputePositionSpec) {
	*out = *in
	if in.WorkloadRef != nil {
		in, out := &in.WorkloadRef, &out.WorkloadRef
		*out = new(WorkloadRef)
		**out = **in
	}
}

// DeepCopy returns a deep copy of ComputePositionSpec.
func (in *ComputePositionSpec) DeepCopy() *ComputePositionSpec {
	if in == nil {
		return nil
	}
	out := new(ComputePositionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies the receiver into out.
func (in *ComputePositionStatus) DeepCopyInto(out *ComputePositionStatus) {
	*out = *in
	if in.IdleSince != nil {
		in, out := &in.IdleSince, &out.IdleSince
		*out = (*in).DeepCopy()
	}
	if in.OriginalReplicas != nil {
		in, out := &in.OriginalReplicas, &out.OriginalReplicas
		*out = new(int32)
		**out = **in
	}
	if in.LastUpdated != nil {
		in, out := &in.LastUpdated, &out.LastUpdated
		*out = (*in).DeepCopy()
	}
}

// DeepCopy returns a deep copy of ComputePositionStatus.
func (in *ComputePositionStatus) DeepCopy() *ComputePositionStatus {
	if in == nil {
		return nil
	}
	out := new(ComputePositionStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies the receiver into out.
func (in *ComputePosition) DeepCopyInto(out *ComputePosition) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy returns a deep copy of ComputePosition.
func (in *ComputePosition) DeepCopy() *ComputePosition {
	if in == nil {
		return nil
	}
	out := new(ComputePosition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a runtime.Object copy of ComputePosition.
func (in *ComputePosition) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto copies the receiver into out.
func (in *ComputePositionList) DeepCopyInto(out *ComputePositionList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		l := make([]ComputePosition, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&l[i])
		}
		out.Items = l
	}
}

// DeepCopy returns a deep copy of ComputePositionList.
func (in *ComputePositionList) DeepCopy() *ComputePositionList {
	if in == nil {
		return nil
	}
	out := new(ComputePositionList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a runtime.Object copy of ComputePositionList.
func (in *ComputePositionList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
