module github.com/monimesl/zookeeper-operator

go 1.16

require (
	cloud.google.com/go v0.56.0 // indirect
	github.com/go-zookeeper/zk v1.0.2
	github.com/monimesl/operator-helper v0.11.1
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.48.1
	k8s.io/api v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v0.21.1
	sigs.k8s.io/controller-runtime v0.9.0
)
