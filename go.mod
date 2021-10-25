module github.com/Congrool/nodes-grouping

go 1.16

require (
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	k8s.io/api v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v0.21.2
	sigs.k8s.io/controller-runtime v0.9.2
	k8s.io/code-generator v0.21.3
)

replace (
	k8s.io/code-generator => k8s.io/code-generator v0.21.3
)
