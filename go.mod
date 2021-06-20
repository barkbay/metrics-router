module github.com/barkbay/custom-metrics-router

go 1.16

require (
	github.com/kubernetes-sigs/custom-metrics-apiserver v0.0.0-20210603131538-559674576232
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.7.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.22.0-alpha.3
	k8s.io/apimachinery v0.22.0-alpha.3
	k8s.io/apiserver v0.22.0-alpha.3 // indirect
	k8s.io/client-go v0.22.0-alpha.3
	k8s.io/klog v1.0.0
	k8s.io/metrics v0.21.1
	sigs.k8s.io/controller-runtime v0.9.0
)
