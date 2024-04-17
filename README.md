# Database controller

[![release](https://img.shields.io/github/release/DoodleScheduling/db-controller/all.svg)](https://github.com/DoodleScheduling/db-controller/releases)
[![release](https://github.com/doodlescheduling/db-controller/actions/workflows/release.yaml/badge.svg)](https://github.com/doodlescheduling/db-controller/actions/workflows/release.yaml)
[![report](https://goreportcard.com/badge/github.com/DoodleScheduling/db-controller)](https://goreportcard.com/report/github.com/DoodleScheduling/db-controller)
[![Coverage Status](https://coveralls.io/repos/github/DoodleScheduling/db-controller/badge.svg?branch=master)](https://coveralls.io/github/DoodleScheduling/db-controller?branch=master)
[![license](https://img.shields.io/github/license/DoodleScheduling/db-controller.svg)](https://github.com/DoodleScheduling/db-controller/blob/master/LICENSE)

Kubernetes Controller for database and user provisioning.
Currently the controller supports Postgres and MongoDB (as well as MongoDB Atlas).
Using the controller you can deploy databases and users defined as code on top of kubernetes.
How to deploy database servers is out of scope of this project.

## Example for PostgreSQL

Example of how to deploy a Postgres database called my-app as well as a user to the server localhost:5432.

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

Example of how to deploy a MongoDB database called my-app as well as a user to the server localhost:5432.

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

## Setup

### Helm chart

Please see [chart/db-controller](https://github.com/DoodleScheduling/db-controller) for the helm chart docs.

### Manifests/kustomize

Alternatively you may get the bundled manifests in each release to deploy it using kustomize or use them directly.

## Configure the controller

The controller can be configured using cmd args:
```
--concurrent int                            The number of concurrent reconciles. (default 4)
--enable-leader-election                    Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.
--graceful-shutdown-timeout duration        The duration given to the reconciler to finish before forcibly stopping. (default 10m0s)
--health-addr string                        The address the health endpoint binds to. (default ":9557")
--insecure-kubeconfig-exec                  Allow use of the user.exec section in kubeconfigs provided for remote apply.
--insecure-kubeconfig-tls                   Allow that kubeconfigs provided for remote apply can disable TLS verification.
--kube-api-burst int                        The maximum burst queries-per-second of requests sent to the Kubernetes API. (default 300)
--kube-api-qps float32                      The maximum queries-per-second of requests sent to the Kubernetes API. (default 50)
--leader-election-lease-duration duration   Interval at which non-leader candidates will wait to force acquire leadership (duration string). (default 35s)
--leader-election-release-on-cancel         Defines if the leader should step down voluntarily on controller manager shutdown. (default true)
--leader-election-renew-deadline duration   Duration that the leading controller manager will retry refreshing leadership before giving up (duration string). (default 30s)
--leader-election-retry-period duration     Duration the LeaderElector clients should wait between tries of actions (duration string). (default 5s)
--log-encoding string                       Log encoding format. Can be 'json' or 'console'. (default "json")
--log-level string                          Log verbosity level. Can be one of 'trace', 'debug', 'info', 'error'. (default "info")
--max-retry-delay duration                  The maximum amount of time for which an object being reconciled will have to wait before a retry. (default 15m0s)
--metrics-addr string                       The address the metric endpoint binds to. (default ":9556")
--min-retry-delay duration                  The minimum amount of time for which an object being reconciled will have to wait before a retry. (default 750ms)
--watch-all-namespaces                      Watch for resources in all namespaces, if set to false it will only watch the runtime namespace. (default true)
--watch-label-selector string               Watch for resources with matching labels e.g. 'sharding.fluxcd.io/shard=shard1'.
```