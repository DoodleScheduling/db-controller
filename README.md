# Database controller

[![release](https://img.shields.io/github/release/DoodleScheduling/k8sdb-controller/all.svg)](https://github.com/DoodleScheduling/k8sdb-controller/releases)
[![release](https://github.com/doodlescheduling/k8sdb-controller/actions/workflows/release.yaml/badge.svg)](https://github.com/doodlescheduling/k8sdb-controller/actions/workflows/release.yaml)
[![report](https://goreportcard.com/badge/github.com/DoodleScheduling/k8sdb-controller)](https://goreportcard.com/report/github.com/DoodleScheduling/k8sdb-controller)
[![Coverage Status](https://coveralls.io/repos/github/DoodleScheduling/k8sdb-controller/badge.svg?branch=master)](https://coveralls.io/github/DoodleScheduling/k8sdb-controller?branch=master)
[![license](https://img.shields.io/github/license/DoodleScheduling/k8sdb-controller.svg)](https://github.com/DoodleScheduling/k8sdb-controller/blob/master/LICENSE)

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

## Installation

### Helm

Please see [chart/k8sdb-controller](https://github.com/DoodleScheduling/k8sdb-controller/tree/master/chart/k8sdb-controller) for the helm chart docs.

### Manifests/kustomize

Alternatively you may get the bundled manifests in each release to deploy it using kustomize or use them directly.

## Limitations

By design there is no garbage collection implemented for databases. Meaning a database does not get dropped if the kubernetes resources is removed.
However this is not the case for users. Users will be removed from the corresponding databases if the referenced kubernetes resource gets removed.
We might reconsider this in the future.

## Profiling
To profile controller, access web server on #profilerPort (default 6060).

In Kubernetes, port-forward to this port, and open the `/debug/pprof` URL in browser. For example, if you port-forward 6060 from container to 6060 on your machine, access:
```
http://localhost:6060/debug/pprof/
```

## Configure the controller

You may change base settings for the controller using env variables (or alternatively command line arguments).
Available env variables:

| Name  | Description | Default |
|-------|-------------| --------|
| `METRICS_ADDR` | The address of the metric endpoint binds to. | `:9556` |
| `PROBE_ADDR` | The address of the probe endpoints binds to. | `:9557` |
| `ENABLE_LEADER_ELECTION` | Enable leader election for controller manager. | `false` |
| `LEADER_ELECTION_NAMESPACE` | Change the leader election namespace. This is by default the same where the controller is deployed. | `` |
| `NAMESPACES` | The controller listens by default for all namespaces. This may be limited to a comma delimited list of dedicated namespaces. | `` |
| `CONCURRENT` | The number of concurrent reconcile workers.  | `1` |
