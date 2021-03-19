module github.com/doodlescheduling/k8sdb-controller

go 1.15

require (
	github.com/go-logr/logr v0.3.0
	github.com/golang/snappy v0.0.2 // indirect
	github.com/jackc/pgx/v4 v4.10.1
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/prometheus/common v0.10.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.0
	github.com/xdg/stringprep v1.0.0 // indirect
	go.mongodb.org/mongo-driver v1.4.4
	golang.org/x/tools v0.1.0 // indirect
	k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver v0.20.2 // indirect
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	mvdan.cc/interfacer v0.0.0-20180901003855-c20040233aed // indirect
	mvdan.cc/lint v0.0.0-20170908181259-adc824a0674b // indirect
	mvdan.cc/unparam v0.0.0-20210104141923-aac4ce9116a7 // indirect
	sigs.k8s.io/controller-runtime v0.8.0
)
