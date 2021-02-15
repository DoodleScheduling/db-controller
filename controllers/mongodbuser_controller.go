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

	"github.com/doodlescheduling/kubedb/api/v1beta1"
	infrav1beta1 "github.com/doodlescheduling/kubedb/api/v1beta1"
	mongodbAPI "github.com/doodlescheduling/kubedb/common/db/mongodb"
	"github.com/go-logr/logr"
	"github.com/prometheus/common/log"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// secretIndexKey is the key used for indexing MongoDBUsers based on
	// their secrets.
	credentialsIndexKey string = ".metadata.credentials"
	dbIndexKey          string = ".metadata.database"
)

// MongoDBUserReconciler reconciles a MongoDBUser object
type MongoDBUserReconciler struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	Recorder   record.EventRecorder
	ClientPool *mongodbAPI.ClientPool
}

func (r *MongoDBUserReconciler) SetupWithManager(mgr ctrl.Manager, maxConcurrentReconciles int) error {
	// Index the MongoDBUser by the Credentials references they point at
	if err := mgr.GetFieldIndexer().IndexField(context.TODO(), &infrav1beta1.MongoDBUser{}, credentialsIndexKey,
		func(o client.Object) []string {
			usr := o.(*infrav1beta1.MongoDBUser)
			return []string{
				fmt.Sprintf("%s/%s", usr.GetNamespace(), usr.Spec.Credentials),
			}
		},
	); err != nil {
		return err
	}

	// Index the MongoDBUser by the Database references they point at
	if err := mgr.GetFieldIndexer().IndexField(context.TODO(), &infrav1beta1.MongoDBUser{}, dbIndexKey,
		func(o client.Object) []string {
			usr := o.(*infrav1beta1.MongoDBUser)
			return []string{
				fmt.Sprintf("%s/%s", usr.GetNamespace(), usr.Spec.Database),
			}
		},
	); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1beta1.MongoDBUser{}).
		Watches(
			&source.Kind{Type: &corev1.Secret{}},
			handler.EnqueueRequestsFromMapFunc(r.requestsForSecretChange),
		).
		Watches(
			&source.Kind{Type: &infrav1beta1.MongoDBDatabase{}},
			handler.EnqueueRequestsFromMapFunc(r.requestsForDatabaseChange),
		).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxConcurrentReconciles}).
		Complete(r)
}

func (r *MongoDBUserReconciler) requestsForSecretChange(o client.Object) []reconcile.Request {
	s, ok := o.(*corev1.Secret)
	if !ok {
		panic(fmt.Sprintf("expected a Secret, got %T", o))
	}

	ctx := context.Background()
	var list infrav1beta1.MongoDBUserList
	if err := r.List(ctx, &list, client.MatchingFields{
		secretIndexKey: objectKey(s).String(),
	}); err != nil {
		return nil
	}

	var reqs []reconcile.Request
	for _, i := range list.Items {
		r.Log.Info("referenced secret from a mongodbuser change detected, reconcile binding", "namespace", i.GetNamespace(), "name", i.GetName())
		reqs = append(reqs, reconcile.Request{NamespacedName: objectKey(&i)})
	}

	return reqs
}

func (r *MongoDBUserReconciler) requestsForDatabaseChange(o client.Object) []reconcile.Request {
	s, ok := o.(*infrav1beta1.MongoDBDatabase)
	if !ok {
		panic(fmt.Sprintf("expected a MongoDBDatabase, got %T", o))
	}

	ctx := context.Background()
	var list infrav1beta1.MongoDBUserList
	if err := r.List(ctx, &list, client.MatchingFields{
		secretIndexKey: objectKey(s).String(),
	}); err != nil {
		return nil
	}

	var reqs []reconcile.Request
	for _, i := range list.Items {
		r.Log.Info("referenced database from a mongodbuser change detected, reconcile binding", "namespace", i.GetNamespace(), "name", i.GetName())
		reqs = append(reqs, reconcile.Request{NamespacedName: objectKey(&i)})
	}

	return reqs
}

// +kubebuilder:rbac:groups=infra.doodle.com,resources=MongoDBUsers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infra.doodle.com,resources=MongoDBUsers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *MongoDBUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("MongoDBUser", req.NamespacedName)

	// common controller functions
	//cw := NewControllerWrapper(*r, &ctx)

	// garbage collector
	//gc := NewMongoDBGarbageCollector(r, cw, &logger)

	// get database resource by namespaced name
	var database infrav1beta1.MongoDBUser
	if err := r.Get(ctx, req.NamespacedName, &database); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// set finalizer
	/*if err := database.SetFinalizer(func() error {
		return r.Update(ctx, &database)
	}); err != nil {
		return reconcile.Result{}, err
	}

	// finalize
	if finalized, err := database.Finalize(func() error {
		return r.Update(ctx, &database)
	}, func() error {
		//return gc.CleanFromSpec(&database)
	}); err != nil {
		return reconcile.Result{}, err
	} else if finalized {
		return reconcile.Result{}, nil
	}*/

	// Garbage Collection. If errors occur, log and proceed with reconciliation.
	/*if err := gc.CleanFromStatus(&database); err != nil {
		log.Info("Error while cleaning garbage", "error", err)
	}*/

	database, result, reconcileErr := r.reconcile(ctx, database, logger)

	// Update status after reconciliation.
	if err := r.patchStatus(ctx, &database); err != nil {
		log.Error(err, "unable to update status after reconciliation")
		return ctrl.Result{Requeue: true}, err
	}

	return result, reconcileErr
}

func (r *MongoDBUserReconciler) reconcile(ctx context.Context, user infrav1beta1.MongoDBUser, logger logr.Logger) (infrav1beta1.MongoDBUser, ctrl.Result, error) {
	// Fetch referencing database
	database := &infrav1beta1.MongoDBDatabase{}
	databaseName := types.NamespacedName{
		Namespace: user.GetNamespace(),
		Name:      user.Spec.Database.Name,
	}
	err := r.Client.Get(context.TODO(), databaseName, database)

	// Failed to fetch referenced database, requeue immediately
	if err != nil {
		msg := fmt.Sprintf("Referencing database was not found: %s", err.Error())
		r.Recorder.Event(&user, "Normal", "error", msg)
		infrav1beta1.DatabaseNotReadyCondition(&user, v1beta1.DatabaseNotFoundReason, msg)
		return user, ctrl.Result{Requeue: true}, err
	}

	// Fetch referencing root secret
	secret := &corev1.Secret{}
	secretName := types.NamespacedName{
		Namespace: database.GetNamespace(),
		Name:      database.Spec.RootSecret.Name,
	}
	err = r.Client.Get(context.TODO(), secretName, secret)

	// Failed to fetch referenced secret, requeue immediately
	if err != nil {
		msg := fmt.Sprintf("Referencing root secret was not found: %s", err.Error())
		r.Recorder.Event(&user, "Normal", "error", msg)
		infrav1beta1.DatabaseNotReadyCondition(&user, v1beta1.SecretNotFoundReason, msg)
		return user, ctrl.Result{Requeue: true}, err
	}

	usr, pw, err := extractCredentials(database.Spec.RootSecret, secret)

	if err != nil {
		msg := fmt.Sprintf("Credentials field not found in referenced rootSecret: %s", err.Error())
		r.Recorder.Event(&user, "Normal", "error", msg)
		infrav1beta1.DatabaseNotReadyCondition(&user, infrav1beta1.CredentialsNotFoundReason, msg)
		return user, ctrl.Result{Requeue: true}, err
	}

	dbHandler, err := r.ClientPool.FromURI(context.TODO(), database.Spec.Address, usr, pw)

	if err != nil {
		msg := fmt.Sprintf("Failed to setup connection to database server: %s", err.Error())
		r.Recorder.Event(&user, "Normal", "error", msg)
		infrav1beta1.DatabaseNotReadyCondition(&user, infrav1beta1.ConnectionFailedReason, msg)
		return user, ctrl.Result{Requeue: true}, err
	}

	// Fetch referencing credentials secret
	credentials := &corev1.Secret{}
	credentialsName := types.NamespacedName{
		Namespace: user.GetNamespace(),
		Name:      user.Spec.Credentials.Name,
	}

	err = r.Client.Get(context.TODO(), credentialsName, credentials)
	usr, pw, err = extractCredentials(user.Spec.Credentials, credentials)

	if err != nil {
		msg := fmt.Sprintf("No credentials found to provision user account: %s", err.Error())
		r.Recorder.Event(&user, "Normal", "error", msg)
		infrav1beta1.UserNotReadyCondition(&user, infrav1beta1.CredentialsNotFoundReason, msg)
		return user, ctrl.Result{Requeue: true}, err
	}

	dbName := database.GetName()
	if database.Spec.DatabaseName != "" {
		dbName = database.Spec.DatabaseName
	}

	err = dbHandler.SetupUser(dbName, usr, pw)
	if err != nil {
		msg := fmt.Sprintf("Failed to provison user account: %s", err.Error())
		r.Recorder.Event(&user, "Normal", "error", msg)
		infrav1beta1.UserNotReadyCondition(&user, infrav1beta1.ConnectionFailedReason, msg)
		return user, ctrl.Result{Requeue: true}, err
	}

	msg := "User successfully provisioned"
	r.Recorder.Event(&user, "Normal", "info", msg)
	v1beta1.UserReadyCondition(&user, v1beta1.UserProvisioningSuccessfulReason, msg)
	return user, ctrl.Result{}, err
}

func (r *MongoDBUserReconciler) patchStatus(ctx context.Context, database *infrav1beta1.MongoDBUser) error {
	key := client.ObjectKeyFromObject(database)
	latest := &infrav1beta1.MongoDBUser{}
	if err := r.Client.Get(ctx, key, latest); err != nil {
		return err
	}

	return r.Client.Status().Patch(ctx, database, client.MergeFrom(latest))
}
