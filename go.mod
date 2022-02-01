module github.com/doodlescheduling/k8sdb-controller

go 1.16

require (
	github.com/go-logr/logr v0.4.0
	github.com/golang/snappy v0.0.2 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20190430165422-3e4dfb77656c // indirect
	github.com/jackc/pgx/v4 v4.10.1
	github.com/mongodb-forks/digest v1.0.2
	github.com/onsi/ginkgo/v2 v2.0.0
	github.com/onsi/gomega v1.17.0
	github.com/prometheus/common v0.26.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.0
	github.com/testcontainers/testcontainers-go v0.12.0
	go.mongodb.org/atlas v0.11.0
	go.mongodb.org/mongo-driver v1.7.1
	k8s.io/api v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.2
	sigs.k8s.io/controller-runtime v0.10.2
)
