package scheme

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	kubeschedulerscheme "k8s.io/kubernetes/pkg/scheduler/apis/config/scheme"

	config "github.com/Congrool/nodes-grouping/pkg/apis/config"
	configv1beta1 "github.com/Congrool/nodes-grouping/pkg/apis/config/v1beta1"
	configv1beta2 "github.com/Congrool/nodes-grouping/pkg/apis/config/v1beta2"
)

var (
	// Re-use the in-tree Scheme
	Scheme = kubeschedulerscheme.Scheme

	// Codecs provides access to encoding and decoding for the scheme
	Codecs = serializer.NewCodecFactory(Scheme, serializer.EnableStrict)
)

func init() {
	AddToScheme(Scheme)
}

func AddToScheme(scheme *runtime.Scheme) {
	utilruntime.Must(config.AddToScheme(scheme))
	utilruntime.Must(configv1beta1.AddToScheme(scheme))
	utilruntime.Must(configv1beta2.AddToScheme(scheme))
}
