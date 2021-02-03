# KUBEDB

Kubernetes Controller that sets up databases, credentials and permissions in Doodle databases.

Build with [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder).

Work in progress.

Name "kubedb" clashes with an open-source project: https://github.com/kubedb
We're going to have naming "clashes" (not technical, but on human level) if we ever decide to use that one.

TODO: Write proper README file.

Config options:

ENV Variable | Argument | Default value | Example | Purpose |
-------------|----------|---------------|---------|---------|
METRICS_ADDR | --metrics-addr | :8080 | :8080 | Metrics port |
ENABLE_LEADER_ELECTION | --enable-leader-election | false | true | Enable leader election |
LEADER_ELECTION_NAMESPACE | --leader-election-namespace | "" | devops | Leader election namespace. Default is the same as controller.
NAMESPACES | --namespaces | "" | devops,default |  Namespaces to watch. Default: watch all namespaces |
MAX_CONCURRENT_RECONCILES | --max-concurrent-reconciles | 1 | 5 | Maximum concurrent reconciles per controller. This config covers all controllers. TODO maybe have a separate flag for each controller? |




