module github.com/Congrool/nodes-grouping

go 1.16

require (
	github.com/gorilla/mux v1.8.0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.15.0
	k8s.io/api v0.22.3
	k8s.io/apimachinery v0.22.3
	k8s.io/client-go v0.22.3
	k8s.io/klog/v2 v2.9.0
	k8s.io/kube-scheduler v0.22.3
	sigs.k8s.io/controller-runtime v0.10.2
)

replace k8s.io/code-generator => k8s.io/code-generator v0.21.3
