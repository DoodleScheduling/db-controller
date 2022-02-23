# Database controller

[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/4787/badge)](https://bestpractices.coreinfrastructure.org/projects/4787)
[![e2e](https://github.com/DoodleScheduling/k8sdb-controller/workflows/e2e/badge.svg)](https://github.com/DoodleScheduling/k8sdb-controller/actions)
[![report](https://goreportcard.com/badge/github.com/DoodleScheduling/k8sdb-controller)](https://goreportcard.com/report/github.com/DoodleScheduling/k8sdb-controller)
[![license](https://img.shields.io/github/license/DoodleScheduling/k8sdb-controller.svg)](https://github.com/DoodleScheduling/k8sdb-controller/blob/main/LICENSE)
[![release](https://img.shields.io/github/release/DoodleScheduling/k8sdb-controller/all.svg)](https://github.com/DoodleScheduling/k8sdb-controller/releases)

Kubernetes Controller for database and user provisioning.

**Note**: This controller does not deploy database servers. It is meant for managing existing ones either deployed by other controllers or manually.

## Example for PostgreSQL

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: postgresql-admin-credentials
  namespace: default
data:
  password: MTIzNA==
  username: MTIzNA==
---
apiVersion: dbprovisioning.infra.doodle.com/v1beta1
kind: PostgreSQLDatabase
metadata:
  name: my-app
  namespace: default
spec:
  address: "postgres://localhost:5432"
  rootSecret:
    name: postgresql-admin-credentials
---
apiVersion: dbprovisioning.infra.doodle.com/v1beta1
kind: PostgreSQLUser
metadata:
  name: my-app
  namespace: default
spec:
  database:
    name: my-app
  credentials:
    name: my-app-postgresql-credentials
---
apiVersion: v1
kind: Secret
metadata:
  name: my-app-postgresql-credentials
  namespace: default
data:
  password: MTIzNA==
  username: MTIzNA==
```

## Example for MongoDB
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: mongodb-admin-credentials
  namespace: default
data:
  password: MTIzNA==
  username: MTIzNA==
---
apiVersion: dbprovisioning.infra.doodle.com/v1beta1
kind: MongoDBDatabase
metadata:
  name: my-app
  namespace: default
spec:
  address: "mongodb://localhost:27017"
  rootSecret:
    name: mongodb-admin-credentials
---
apiVersion: dbprovisioning.infra.doodle.com/v1beta1
kind: MongoDBUser
metadata:
  name: my-app
  namespace: default
spec:
  database:
    name: my-app-mongodb-credentials
  credentials:
    name: my-app-mongodb
  roles:
    - name: readWrite
---
apiVersion: v1
kind: Secret
metadata:
  name: my-app-mongodb-credentials
  namespace: default
data:
  password: MTIzNA==
  username: MTIzNA==
```

## Helm chart

Please see [chart/k8sdb-controller](https://github.com/DoodleScheduling/k8sdb-controller) for the helm chart docs.

## Profiling
To profile controller, access web server on #profilerPort (default 6060). 

In Kubernetes, port-forward to this port, and open the `/debug/pprof` URL in browser. For example, if you port-forward 6060 from container to 6060 on your machine, access:
```
http://localhost:6060/debug/pprof/
```

## Limitations

Currently there is no garbage collection implemented, meaning all the things created are not removed.
This will be at least implemented for user provisioning. Discussion will stay open for databases.

## Configure the controller

ENV Variable | Argument | Default value | Example | Purpose |
-------------|----------|---------------|---------|---------|
METRICS_ADDR | --metrics-addr | :8080 | :8080 | Metrics port |
ENABLE_LEADER_ELECTION | --enable-leader-election | false | true | Enable leader election |
LEADER_ELECTION_NAMESPACE | --leader-election-namespace | "" | devops | Leader election namespace. Default is the same as controller.
NAMESPACES | --namespaces | "" | devops,default |  Namespaces to watch. Default: watch all namespaces |
MAX_CONCURRENT_RECONCILES | --max-concurrent-reconciles | 1 | 5 | Maximum concurrent reconciles per controller. This config covers all controllers. |
