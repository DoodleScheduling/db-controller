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

package controllers

import (
	"context"
	infrav1beta1 "github.com/doodlescheduling/kubedb/api/v1beta1"
	mongodbAPI "github.com/doodlescheduling/kubedb/common/db/mongodb"
	vaultAPI "github.com/doodlescheduling/kubedb/common/vault"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// MongoDBDatabaseReconciler reconciles a MongoDBDatabase object
type MongoDBDatabaseReconciler struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	ServerCache *mongodbAPI.Cache
	VaultCache  *vaultAPI.Cache
}

// +kubebuilder:rbac:groups=infra.doodle.com,resources=mongodbdatabases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infra.doodle.com,resources=mongodbdatabases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *MongoDBDatabaseReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("mongodbdatabase", req.NamespacedName)

	// common controller functions
	cw := NewControllerWrapper(*r, &ctx)
	// garbage collector
	gc := NewMongoDBGarbageCollector(r, cw, &log)

	// get database resource by namespaced name
	var database infrav1beta1.MongoDBDatabase
	if err := r.Get(ctx, req.NamespacedName, &database); err != nil {
		if apierrors.IsNotFound(err) {
			// resource no longer present. Consider dropping a database? What about data, it will be lost.. Probably acceptable for devboxes
			// How to do it, though? Resource doesn't exist anymore, so we need to list all databases and all manifests and compare?
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// set finalizer
	if err := database.SetFinalizer(func() error {
		return r.Update(ctx, &database)
	}); err != nil {
		return reconcile.Result{}, err
	}

	// finalize
	if finalized, err := database.Finalize(func() error {
		return r.Update(ctx, &database)
	}, func() error {
		return gc.CleanFromSpec(&database)
	}); err != nil {
		return reconcile.Result{}, err
	} else if finalized {
		return reconcile.Result{}, nil
	}

	// set spec defaults. Does not mutate the spec, since we are not updating resource
	if err := database.SetDefaults(); err != nil {
		database.Status.DatabaseStatus.SetDatabaseStatus(infrav1beta1.Unavailable, err.Error(), "", "")
		database.Status.CredentialsStatus = make(infrav1beta1.CredentialsStatus, 0)
		return r.updateAndReturn(&ctx, &database, &log)
	}

	// Garbage Collection. If errors occur, log and proceed with reconciliation.
	if err := gc.CleanFromStatus(&database); err != nil {
		log.Info("Error while cleaning garbage", "error", err)
	}

	// get root database password from k8s secret
	rootPassword, err := cw.GetRootPassword(database.Spec.RootSecretLookup.Name, database.Spec.RootSecretLookup.Namespace, database.Spec.RootSecretLookup.Field)
	if err != nil {
		database.Status.DatabaseStatus.SetDatabaseStatus(infrav1beta1.Unavailable, err.Error(), "", "")
		database.Status.CredentialsStatus = make(infrav1beta1.CredentialsStatus, 0)
		return r.updateAndReturn(&ctx, &database, &log)
	}

	// mongoDB connection to spec host, cached
	mongoDBServer, err := r.ServerCache.Get(database.Spec.HostName, database.Spec.RootUsername, rootPassword, database.Spec.RootAuthenticationDatabase)
	if err != nil {
		database.Status.DatabaseStatus.SetDatabaseStatus(infrav1beta1.Unavailable, err.Error(), "", database.Spec.HostName).
			WithUsername(database.Spec.RootUsername).
			WithAuthDatabase(database.Spec.RootAuthenticationDatabase).
			WithRootSecretLookup(database.Spec.RootSecretLookup.Name, database.Spec.RootSecretLookup.Namespace, database.Spec.RootSecretLookup.Field)
		database.Status.CredentialsStatus = make(infrav1beta1.CredentialsStatus, 0)
		return r.updateAndReturn(&ctx, &database, &log)
	}

	// vault connection, cached
	vault, err := r.VaultCache.Get(database.Spec.RootSecretLookup.Name)
	if err != nil {
		database.Status.DatabaseStatus.SetDatabaseStatus(infrav1beta1.Unavailable, err.Error(), "", database.Spec.HostName).
			WithUsername(database.Spec.RootUsername).
			WithAuthDatabase(database.Spec.RootAuthenticationDatabase).
			WithRootSecretLookup(database.Spec.RootSecretLookup.Name, database.Spec.RootSecretLookup.Namespace, database.Spec.RootSecretLookup.Field)
		database.Status.CredentialsStatus = make(infrav1beta1.CredentialsStatus, 0)
		return r.updateAndReturn(&ctx, &database, &log)
	}

	// setup credentials as per spec
	for _, credential := range database.Spec.Credentials {
		username := credential.UserName
		mongodbCredentialStatus := database.Status.CredentialsStatus.FindOrCreate(username, func(status *infrav1beta1.CredentialStatus) bool {
			return status != nil && status.Username == username
		})
		// get user credentials from vault
		vaultResponse, err := vault.Get(vaultAPI.VaultRequest{})
		if err != nil {
			mongodbCredentialStatus.SetCredentialsStatus(infrav1beta1.Unavailable, err.Error())
			continue
		}
		password := vaultResponse.Secret
		if err := mongoDBServer.SetupUser(database.Spec.DatabaseName, username, password); err != nil {
			mongodbCredentialStatus.SetCredentialsStatus(infrav1beta1.Unavailable, err.Error())
		} else {
			mongodbCredentialStatus.SetCredentialsStatus(infrav1beta1.Available, "Credentials up.")
		}
	}

	// setup database status - database is automatically set up with credentials
	database.Status.DatabaseStatus.SetDatabaseStatus(infrav1beta1.Available, "Database up.", database.Spec.DatabaseName, database.Spec.HostName).
		WithUsername(database.Spec.RootUsername).
		WithAuthDatabase(database.Spec.RootAuthenticationDatabase).
		WithRootSecretLookup(database.Spec.RootSecretLookup.Name, database.Spec.RootSecretLookup.Namespace, database.Spec.RootSecretLookup.Field)

	return r.updateAndReturn(&ctx, &database, &log)
}

func (r *MongoDBDatabaseReconciler) SetupWithManager(mgr ctrl.Manager, maxConcurrentReconciles int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1beta1.MongoDBDatabase{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: maxConcurrentReconciles,
		}).Complete(r)
}

func (r *MongoDBDatabaseReconciler) updateAndReturn(ctx *context.Context, database *infrav1beta1.MongoDBDatabase, log *logr.Logger) (ctrl.Result, error) {
	now := metav1.Now()
	database.Status.LastUpdateTime = &now
	if err := r.Status().Update(*ctx, database); err != nil {
		(*log).Error(err, "unable to update MongoDBDatabase status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
