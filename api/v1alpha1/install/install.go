package install

import (
	natsv1alpha1 "github.com/katallaxie/natz-operator/api/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

// Install registers the API group and adds types to a scheme
func Install(scheme *runtime.Scheme) {
	utilruntime.Must(natsv1alpha1.AddToScheme(scheme))
	utilruntime.Must(scheme.SetVersionPriority(natsv1alpha1.SchemeGroupVersion))
}
