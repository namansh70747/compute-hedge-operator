// Package v1alpha1 contains the ComputePosition API types.
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// GroupVersion is the API group and version for this package.
var GroupVersion = schema.GroupVersion{Group: "computehedge.dev", Version: "v1alpha1"}

// SchemeBuilder registers the types with a runtime scheme.
var SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

// AddToScheme adds the types in this group to a scheme.
var AddToScheme = SchemeBuilder.AddToScheme

func init() {
	SchemeBuilder.Register(&ComputePosition{}, &ComputePositionList{})
}
