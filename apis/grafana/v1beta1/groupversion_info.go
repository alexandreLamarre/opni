// // +kubebuilder:object:generate=true
// // +groupName=nfd.opni.io
package v1

<<<<<<<< HEAD:apis/grafana/v1beta1/groupversion_info.go
import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "grafana.opni.io", Version: "v1beta1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
========
//
//import (
//	"k8s.io/apimachinery/pkg/runtime/schema"
//	"sigs.k8s.io/controller-runtime/pkg/scheme"
//)
//
//var (
//	// GroupVersion is group version used to register these objects
//	GroupVersion = schema.GroupVersion{Group: "grafana.opni.io", Version: "v1alpha1"}
//
//	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
//	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
//
//	// AddToScheme adds the types in this group-version to the given scheme.
//	AddToScheme = SchemeBuilder.AddToScheme
//)
>>>>>>>> 27b28a40c (add grafanav1beta1 to scheme):apis/grafana/v1alpha1/groupversion_info.go
