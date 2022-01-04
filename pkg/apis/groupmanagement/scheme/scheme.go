package scheme

import (
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	groupmanagementv1alpha1 "github.com/Congrool/nodes-grouping/pkg/apis/groupmanagement/v1alpha1"
	"github.com/Congrool/nodes-grouping/pkg/generated/clientset/versioned/scheme"
)

func init() {
	AddToScheme(scheme.Scheme)
}

func AddToScheme(scheme *runtime.Scheme) {
	utilruntime.Must(groupmanagementv1alpha1.AddToScheme(scheme))
}
