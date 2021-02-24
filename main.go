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
	"net/http"
	_ "net/http/pprof"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"os"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	infrav1beta1 "github.com/doodlescheduling/k8sdb-controller/api/v1beta1"
	"github.com/doodlescheduling/k8sdb-controller/common/db"
	"github.com/doodlescheduling/k8sdb-controller/controllers"
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
	ProfilerAddr            = "profiler-addr"
	EnableLeaderElection    = "enable-leader-election"
	LeaderElectionNamespace = "leader-election-namespace"
	Namespaces              = "namespaces"
	MaxConcurrentReconciles = "max-concurrent-reconciles"
)

// config variables & defaults
var (
	metricsAddr             = ":8080"
	probesAddr              = ":9558"
	profilerAddr            = ":6060"
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
	flag.StringVar(&profilerAddr, ProfilerAddr, ":6060", "The address of the profiler endpoints bind to.")
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
	profilerAddr = viper.GetString(ProfilerAddr)

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

	// Profiler
	go func() {
		setupLog.Info("Starting profiler...")
		if err := http.ListenAndServe(profilerAddr, nil); err != nil {
			setupLog.Error(err, "Profiler failed to start")
		}
	}()

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

	// MongoDBDatabase setup
	if err = (&controllers.MongoDBDatabaseReconciler{
		Client:     mgr.GetClient(),
		Log:        ctrl.Log.WithName("controllers").WithName("MongoDBDatabase"),
		Scheme:     mgr.GetScheme(),
		ClientPool: db.NewClientPool(),
		Recorder:   mgr.GetEventRecorderFor("MongoDBDatabase"),
	}).SetupWithManager(mgr, maxConcurrentReconciles); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MongoDBDatabase")
		os.Exit(1)
	}

	// MongoDBUser setup
	if err = (&controllers.MongoDBUserReconciler{
		Client:     mgr.GetClient(),
		Log:        ctrl.Log.WithName("controllers").WithName("MongoDBUser"),
		Scheme:     mgr.GetScheme(),
		ClientPool: db.NewClientPool(),
		Recorder:   mgr.GetEventRecorderFor("MongoDBUser"),
	}).SetupWithManager(mgr, maxConcurrentReconciles); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MongoDBUser")
		os.Exit(1)
	}

	// PostgreSQLDatabase setup
	if err = (&controllers.PostgreSQLDatabaseReconciler{
		Client:     mgr.GetClient(),
		Log:        ctrl.Log.WithName("controllers").WithName("PostgreSQLDatabase"),
		Scheme:     mgr.GetScheme(),
		ClientPool: db.NewClientPool(),
		Recorder:   mgr.GetEventRecorderFor("PostgreSQLDatabase"),
	}).SetupWithManager(mgr, maxConcurrentReconciles); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "PostgreSQLDatabase")
		os.Exit(1)
	}

	// PostgreSQLUser setup
	if err = (&controllers.PostgreSQLUserReconciler{
		Client:     mgr.GetClient(),
		Log:        ctrl.Log.WithName("controllers").WithName("PostgreSQLUser"),
		Scheme:     mgr.GetScheme(),
		ClientPool: db.NewClientPool(),
		Recorder:   mgr.GetEventRecorderFor("PostgreSQLUser"),
	}).SetupWithManager(mgr, maxConcurrentReconciles); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "PostgreSQLUser")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
