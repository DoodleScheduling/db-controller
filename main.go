/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"

	mongodbAPI "github.com/doodlescheduling/kubedb/common/db/mongodb"
	"github.com/doodlescheduling/kubedb/common/vault"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"os"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	postgresqlAPI "github.com/doodlescheduling/kubedb/common/db/postgresql"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	infrav1beta1 "github.com/doodlescheduling/kubedb/api/v1beta1"
	"github.com/doodlescheduling/kubedb/controllers"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

// flags
const (
	MetricAddr              = "metrics-addr"
	ProbeAddr               = "probe-addr"
	EnableLeaderElection    = "enable-leader-election"
	LeaderElectionNamespace = "leader-election-namespace"
	Namespaces              = "namespaces"
	MaxConcurrentReconciles = "max-concurrent-reconciles"
)

// config variables & defaults
var (
	metricsAddr             = ":8080"
	probesAddr              = ":9558"
	enableLeaderElection    = false
	leaderElectionNamespace = ""
	namespacesConfig        = ""
	maxConcurrentReconciles = 1
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = infrav1beta1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	flag.StringVar(&metricsAddr, MetricAddr, ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probesAddr, ProbeAddr, ":9558", "The address of the probe endpoints bind to.")
	flag.BoolVar(&enableLeaderElection, EnableLeaderElection, false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&leaderElectionNamespace, LeaderElectionNamespace, "", "Leader election namespace. Default is the same namespace as controller.")
	flag.StringVar(&namespacesConfig, Namespaces, "", "Comma-separated list of namespaces to watch. If not set, all namespaces will be watched.")
	flag.IntVar(&maxConcurrentReconciles, MaxConcurrentReconciles, 1, "Maximum number of concurrent reconciles. Default is 1.")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		setupLog.Error(err, "Failed parsing command line arguments")
		os.Exit(1)
	}
	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()

	metricsAddr = viper.GetString(MetricAddr)
	enableLeaderElection = viper.GetBool(EnableLeaderElection)
	leaderElectionNamespace = viper.GetString(LeaderElectionNamespace)
	probesAddr = viper.GetString(ProbeAddr)
	namespacesConfig = viper.GetString(Namespaces)
	maxConcurrentReconciles = viper.GetInt(MaxConcurrentReconciles)

	namespaces := strings.Split(namespacesConfig, ",")
	options := ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		HealthProbeBindAddress: probesAddr,
		Port:                   9443,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "99a96989.doodle.com",
	}
	if len(namespaces) > 0 && namespaces[0] != "" {
		options.NewCache = cache.MultiNamespacedCacheBuilder(namespaces)
		setupLog.Info("watching configured namespaces", "namespaces", namespaces)
	} else {
		setupLog.Info("watching all namespaces")
	}
	if leaderElectionNamespace != "" {
		options.LeaderElectionNamespace = leaderElectionNamespace
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Liveness probe
	err = mgr.AddHealthzCheck("healthz", healthz.Ping)
	if err != nil {
		os.Exit(1)
	}

	// Readiness probe
	err = mgr.AddReadyzCheck("readyz", healthz.Ping)
	if err != nil {
		os.Exit(1)
	}

	// General setup
	vaultCache := vault.NewCache()

	// MongoDBDatabase setup
	mongoDBServerCache := mongodbAPI.NewCache()
	if err = (&controllers.MongoDBDatabaseReconciler{
		Client:      mgr.GetClient(),
		Log:         ctrl.Log.WithName("controllers").WithName("MongoDBDatabase"),
		Scheme:      mgr.GetScheme(),
		ServerCache: mongoDBServerCache,
		VaultCache:  vaultCache,
	}).SetupWithManager(mgr, maxConcurrentReconciles); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MongoDBDatabase")
		os.Exit(1)
	}

	// PostgreSQLDatabase setup
	postgreSQLServerCache := postgresqlAPI.NewCache()
	if err = (&controllers.PostgreSQLDatabaseReconciler{
		Client:      mgr.GetClient(),
		Log:         ctrl.Log.WithName("controllers").WithName("PostgreSQLDatabase"),
		Scheme:      mgr.GetScheme(),
		ServerCache: postgreSQLServerCache,
		VaultCache:  vaultCache,
	}).SetupWithManager(mgr, maxConcurrentReconciles); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "PostgreSQLDatabase")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
