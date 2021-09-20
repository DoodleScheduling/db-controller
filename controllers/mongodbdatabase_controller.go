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
	"fmt"

	"github.com/doodlescheduling/k8sdb-controller/common/database"
	"github.com/go-logr/logr"
	"github.com/prometheus/common/log"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	infrav1beta1 "github.com/doodlescheduling/k8sdb-controller/api/v1beta1"
)

// MongoDBDatabaseReconciler reconciles a MongoDBDatabase object
type MongoDBDatabaseReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func (r *MongoDBDatabaseReconciler) SetupWithManager(mgr ctrl.Manager, maxConcurrentReconciles int) error {
	// Index the MongoDBDatabase by the Secret references they point at
	if err := mgr.GetFieldIndexer().IndexField(context.TODO(), &infrav1beta1.MongoDBDatabase{}, secretIndexKey,
		func(o client.Object) []string {
			db := o.(*infrav1beta1.MongoDBDatabase)
			return []string{
				fmt.Sprintf("%s/%s", db.GetNamespace(), db.Spec.RootSecret.Name),
			}
		},
	); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1beta1.MongoDBDatabase{}).
		Watches(
			&source.Kind{Type: &corev1.Secret{}},
			handler.EnqueueRequestsFromMapFunc(r.requestsForSecretChange),
		).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxConcurrentReconciles}).
		Complete(r)
}

func (r *MongoDBDatabaseReconciler) requestsForSecretChange(o client.Object) []reconcile.Request {
	s, ok := o.(*corev1.Secret)
	if !ok {
		panic(fmt.Sprintf("expected a Secret, got %T", o))
	}

	ctx := context.Background()
	var list infrav1beta1.MongoDBDatabaseList
	if err := r.List(ctx, &list, client.MatchingFields{
		secretIndexKey: objectKey(s).String(),
	}); err != nil {
		return nil
	}

	var reqs []reconcile.Request
	for _, i := range list.Items {
		r.Log.Info("referenced secret from a MongoDBDatabase changed detected", "namespace", i.GetNamespace(), "name", i.GetName())
		reqs = append(reqs, reconcile.Request{NamespacedName: objectKey(&i)})
	}

	return reqs
}

// +kubebuilder:rbac:groups=infra.doodle.com,resources=mongodbdatabases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infra.doodle.com,resources=mongodbdatabases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *MongoDBDatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("mongodbdatabase", req.NamespacedName)
	logger.Info("reconciling MongoDBDatabase")

	// get database resource by namespaced name
	var db infrav1beta1.MongoDBDatabase
	if err := r.Get(ctx, req.NamespacedName, &db); err != nil {
		if apierrors.IsNotFound(err) {
			// resource no longer present. Consider dropping a database? What about data, it will be lost.. Probably acceptable for devboxes
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	db.SetDefaults()

	// set finalizer
	if err := db.SetFinalizer(func() error {
		return r.Update(ctx, &db)
	}); err != nil {
		return reconcile.Result{}, err
	}

	// finalize
	if finalized, err := db.Finalize(func() error {
		return r.Update(ctx, &db)
	}, func() error {
		return nil
		//return gc.CleanFromSpec(&database)
	}); err != nil {
		return reconcile.Result{}, err
	} else if finalized {
		return reconcile.Result{}, nil
	}

	db, err := r.reconcileGenericDatabase(ctx, db, r.Recorder)

	if err != nil {
		r.Recorder.Event(&db, "Normal", "error", err.Error())
		return ctrl.Result{Requeue: true}, nil
	}

	msg := "Database successfully provisioned"
	r.Recorder.Event(&db, "Normal", "info", msg)
	infrav1beta1.DatabaseReadyCondition(&db, infrav1beta1.DatabaseProvisiningSuccessfulReason, msg)

	// Update status after reconciliation.
	if err := r.patchStatus(ctx, &db); err != nil {
		log.Error(err, "unable to update status after reconciliation")
		return ctrl.Result{Requeue: true}, err
	}

	return ctrl.Result{}, nil
}

func (r *MongoDBDatabaseReconciler) reconcileGenericDatabase(ctx context.Context, db infrav1beta1.MongoDBDatabase, recorder record.EventRecorder) (infrav1beta1.MongoDBDatabase, error) {
	usr, pw, err := getSecret(ctx, r.Client, db.GetRootSecret())

	if err != nil {
		infrav1beta1.DatabaseNotReadyCondition(&db, infrav1beta1.CredentialsNotFoundReason, err.Error())
		return db, err
	}

	if db.MigrationRequired() {
		dbHandler, err := setupMongoDB(ctx, db, usr, pw)

		if err != nil {
			infrav1beta1.DatabaseNotReadyCondition(&db, infrav1beta1.ConnectionFailedReason, err.Error())
			return db, err
		}

		srcUsername := usr
		srcPassword := pw

		if db.GetMigrationRootSecret() != nil {
			srcUsername, srcPassword, err = getSecret(ctx, r.Client, db.GetMigrationRootSecret())

			if err != nil {
				infrav1beta1.DatabaseNotReadyCondition(&db, infrav1beta1.CredentialsNotFoundReason, err.Error())
				return db, err
			}
		}

		srcOpts := database.MongoDBOptions{
			URI:          db.GetMigrationAddress(),
			DatabaseName: db.GetMigrationDatabaseName(),
			Username:     srcUsername,
			Password:     srcPassword,
		}

		workloads, err := downscaleWorkloads(ctx, r.Client, db.GetMigrationWorkloads())
		db.Spec.Migration.Workloads = workloads

		if err != nil {
			err = fmt.Errorf("Failed to scale down referenced workloads: %w", err)
			infrav1beta1.DatabaseNotReadyCondition(&db, infrav1beta1.ConnectionFailedReason, err.Error())
			return db, err
		}

		err = dbHandler.RestoreDatabaseFrom(ctx, srcOpts)

		if err != nil {
			err = fmt.Errorf("Failed to migrate database: %w", err)
			infrav1beta1.DatabaseNotReadyCondition(&db, infrav1beta1.MigrationFailedReason, err.Error())
			return db, err
		}

		err = upscaleWorkloads(ctx, r.Client, db.GetMigrationWorkloads())
		if err != nil {
			err = fmt.Errorf("Failed to scale up referenced workloads: %w", err)
			infrav1beta1.DatabaseNotReadyCondition(&db, infrav1beta1.ConnectionFailedReason, err.Error())
			return db, err
		}
	}

	return db, nil
}

func (r *MongoDBDatabaseReconciler) reconcileAtlasDatabase(ctx context.Context, db infrav1beta1.MongoDBDatabase, recorder record.EventRecorder) (infrav1beta1.MongoDBDatabase, error) {
	pubKey, privKey, err := getSecret(ctx, r.Client, db.GetRootSecret())

	if err != nil {
		infrav1beta1.DatabaseNotReadyCondition(&db, infrav1beta1.CredentialsNotFoundReason, err.Error())
		return db, err
	}

	_, err = setupAtlas(ctx, db, pubKey, privKey)

	if err != nil {
		infrav1beta1.DatabaseNotReadyCondition(&db, infrav1beta1.ConnectionFailedReason, err.Error())
		return db, err
	}

	return db, nil
}

func (r *MongoDBDatabaseReconciler) patchStatus(ctx context.Context, database *infrav1beta1.MongoDBDatabase) error {
	key := client.ObjectKeyFromObject(database)
	latest := &infrav1beta1.MongoDBDatabase{}
	if err := r.Client.Get(ctx, key, latest); err != nil {
		return err
	}

	return r.Client.Status().Patch(ctx, database, client.MergeFrom(latest))
}
