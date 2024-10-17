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
	"fmt"
	"os"
	"time"

	infrav1beta1 "github.com/doodlescheduling/db-controller/api/v1beta1"
	"github.com/doodlescheduling/db-controller/internal/controllers"
	"github.com/fluxcd/pkg/runtime/client"
	helper "github.com/fluxcd/pkg/runtime/controller"
	"github.com/fluxcd/pkg/runtime/leaderelection"
	"github.com/fluxcd/pkg/runtime/logger"
	flag "github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlcache "sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	// +kubebuilder:scaffold:imports
)

const controllerName = "db-controller"

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = corev1.AddToScheme(scheme)
	_ = infrav1beta1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

var (
	metricsAddr             string
	healthAddr              string
	concurrent              int
	gracefulShutdownTimeout time.Duration
	clientOptions           client.Options
	kubeConfigOpts          client.KubeConfigOptions
	logOptions              logger.Options
	leaderElectionOptions   leaderelection.Options
	rateLimiterOptions      helper.RateLimiterOptions
	watchOptions            helper.WatchOptions
)

func main() {
	flag.StringVar(&metricsAddr, "metrics-addr", ":9556",
		"The address the metric endpoint binds to.")
	flag.StringVar(&healthAddr, "health-addr", ":9557",
		"The address the health endpoint binds to.")
	flag.IntVar(&concurrent, "concurrent", 4,
		"The number of concurrent reconciles.")
	flag.DurationVar(&gracefulShutdownTimeout, "graceful-shutdown-timeout", 600*time.Second,
		"The duration given to the reconciler to finish before forcibly stopping.")

	clientOptions.BindFlags(flag.CommandLine)
	logOptions.BindFlags(flag.CommandLine)
	leaderElectionOptions.BindFlags(flag.CommandLine)
	rateLimiterOptions.BindFlags(flag.CommandLine)
	kubeConfigOpts.BindFlags(flag.CommandLine)
	watchOptions.BindFlags(flag.CommandLine)

	flag.Parse()
	logger.SetLogger(logger.NewLogger(logOptions))

	leaderElectionId := fmt.Sprintf("%s-%s", controllerName, "leader-election")
	if watchOptions.LabelSelector != "" {
		leaderElectionId = leaderelection.GenerateID(leaderElectionId, watchOptions.LabelSelector)
	}

	watchSelector, err := helper.GetWatchSelector(watchOptions)
	if err != nil {
		setupLog.Error(err, "unable to configure watch label selector for manager")
		os.Exit(1)
	}

	opts := ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress:        healthAddr,
		LeaderElection:                leaderElectionOptions.Enable,
		LeaderElectionReleaseOnCancel: leaderElectionOptions.ReleaseOnCancel,
		LeaseDuration:                 &leaderElectionOptions.LeaseDuration,
		RenewDeadline:                 &leaderElectionOptions.RenewDeadline,
		RetryPeriod:                   &leaderElectionOptions.RetryPeriod,
		GracefulShutdownTimeout:       &gracefulShutdownTimeout,
		LeaderElectionID:              leaderElectionId,
		Cache: ctrlcache.Options{
			ByObject: map[ctrlclient.Object]ctrlcache.ByObject{
				&infrav1beta1.MongoDBDatabase{}:    {Label: watchSelector},
				&infrav1beta1.MongoDBUser{}:        {Label: watchSelector},
				&infrav1beta1.PostgreSQLDatabase{}: {Label: watchSelector},
				&infrav1beta1.PostgreSQLUser{}:     {Label: watchSelector},
			},
		},
	}

	if !watchOptions.AllNamespaces {
		opts.Cache.DefaultNamespaces = make(map[string]ctrlcache.Config)
		opts.Cache.DefaultNamespaces[os.Getenv("RUNTIME_NAMESPACE")] = ctrlcache.Config{}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), opts)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Add liveness probe
	err = mgr.AddHealthzCheck("healthz", healthz.Ping)
	if err != nil {
		setupLog.Error(err, "Could not add liveness probe")
		os.Exit(1)
	}

	// Add readiness probe
	err = mgr.AddReadyzCheck("readyz", healthz.Ping)
	if err != nil {
		setupLog.Error(err, "Could not add readiness probe")
		os.Exit(1)
	}

	// MongoDBDatabase setup
	if err = (&controllers.MongoDBDatabaseReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("MongoDBDatabase"),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("MongoDBDatabase"),
	}).SetupWithManager(mgr, concurrent); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MongoDBDatabase")
		os.Exit(1)
	}

	// MongoDBUser setup
	if err = (&controllers.MongoDBUserReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("MongoDBUser"),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("MongoDBUser"),
	}).SetupWithManager(mgr, concurrent); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MongoDBUser")
		os.Exit(1)
	}

	// PostgreSQLDatabase setup
	if err = (&controllers.PostgreSQLDatabaseReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("PostgreSQLDatabase"),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("PostgreSQLDatabase"),
	}).SetupWithManager(mgr, concurrent); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "PostgreSQLDatabase")
		os.Exit(1)
	}

	// PostgreSQLUser setup
	if err = (&controllers.PostgreSQLUserReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("PostgreSQLUser"),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("PostgreSQLUser"),
	}).SetupWithManager(mgr, concurrent); err != nil {
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
