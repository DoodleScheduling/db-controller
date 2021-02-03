# KUBEDB

Kubernetes Controller that sets up databases, credentials and permissions in Doodle databases.

Build with [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder).

Work in progress.

Name "kubedb" clashes with an open-source project: https://github.com/kubedb
We're going to have naming "clashes" (not technical, but on human level) if we ever decide to use that one.

TODO: Write proper README file.

Config options:

Argument | Default value | Example | Purpose |
---------|---------------|---------|---------|
--metrics-addr | :8080 | :8080 | Metrics port |
--enable-leader-election | false | true | Enable leader election |
--namespaces | "" | devops,default |  Namespaces to watch. Default: watch all namespaces |
--max-concurrent-reconciles | 1 | 5 | Maximum concurrent reconciles. This config covers all controllers. TODO maybe have a separate flag for each controller? |




