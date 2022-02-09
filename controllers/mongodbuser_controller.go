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

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/doodlescheduling/k8sdb-controller/api/v1beta1"
	infrav1beta1 "github.com/doodlescheduling/k8sdb-controller/api/v1beta1"
	"github.com/doodlescheduling/k8sdb-controller/common/stringutils"
)

// +kubebuilder:rbac:groups=dbprovisioning.infra.doodle.com,resources=mongodbusers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dbprovisioning.infra.doodle.com,resources=mongodbusers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// MongoDBUserReconciler reconciles a MongoDBUser object
type MongoDBUserReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func (r *MongoDBUserReconciler) SetupWithManager(mgr ctrl.Manager, maxConcurrentReconciles int) error {
	// Index the MongoDBUser by the Credentials references they point at
	if err := mgr.GetFieldIndexer().IndexField(context.TODO(), &infrav1beta1.MongoDBUser{}, credentialsIndexKey,
		func(o client.Object) []string {
			usr := o.(*infrav1beta1.MongoDBUser)
			return []string{
				fmt.Sprintf("%s/%s", usr.GetNamespace(), usr.Spec.Credentials.Name),
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
				fmt.Sprintf("%s/%s", usr.GetNamespace(), usr.Spec.Database.Name),
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
		credentialsIndexKey: objectKey(s).String(),
	}); err != nil {
		return nil
	}

	var reqs []reconcile.Request
	for _, i := range list.Items {
		r.Log.Info("referenced secret from a mongodbuser change detected", "namespace", i.GetNamespace(), "name", i.GetName())
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
		dbIndexKey: objectKey(s).String(),
	}); err != nil {
		return nil
	}

	var reqs []reconcile.Request
	for _, i := range list.Items {
		r.Log.Info("referenced database from a mongodbuser change detected, reconcile", "namespace", i.GetNamespace(), "name", i.GetName())
		reqs = append(reqs, reconcile.Request{NamespacedName: objectKey(&i)})
	}

	return reqs
}

func (r *MongoDBUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("MongoDBUser", req.NamespacedName)
	logger.Info("reconciling MongoDBUser")

	// get database resource by namespaced name
	var user infrav1beta1.MongoDBUser
	if err := r.Get(ctx, req.NamespacedName, &user); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// examine DeletionTimestamp to determine if object is under deletion
	if user.ObjectMeta.DeletionTimestamp.IsZero() {
		if !stringutils.ContainsString(user.GetFinalizers(), v1beta1.Finalizer) {
			controllerutil.AddFinalizer(&user, v1beta1.Finalizer)
			if err := r.Update(ctx, &user); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	user, err := r.reconcile(ctx, user)
	res := ctrl.Result{}
	user.Status.ObservedGeneration = user.GetGeneration()

	if err != nil {
		r.Recorder.Event(&user, "Normal", "error", err.Error())
		res = ctrl.Result{Requeue: true}
	} else {
		msg := "User successfully provisioned"
		r.Recorder.Event(&user, "Normal", "info", msg)
		infrav1beta1.UserReadyCondition(&user, infrav1beta1.UserProvisioningSuccessfulReason, msg)
	}

	// Update status after reconciliation.
	if err := r.patchStatus(ctx, &user); err != nil {
		logger.Error(err, "unable to update status after reconciliation")
		return ctrl.Result{Requeue: true}, err
	}

	return res, nil
}

func (r *MongoDBUserReconciler) reconcile(ctx context.Context, user infrav1beta1.MongoDBUser) (infrav1beta1.MongoDBUser, error) {
	// Fetch referencing database
	var db infrav1beta1.MongoDBDatabase
	databaseName := types.NamespacedName{
		Namespace: user.GetNamespace(),
		Name:      user.GetDatabase(),
	}

	err := r.Client.Get(ctx, databaseName, &db)
	if err != nil {
		err = fmt.Errorf("Referencing database was not found: %w", err)
		infrav1beta1.UserNotReadyCondition(&user, infrav1beta1.DatabaseNotFoundReason, err.Error())
		return user, err
	}

	if db.Spec.AtlasGroupId != "" {
		return r.reconcileAtlasUser(ctx, user, db)
	}

	return r.reconcileGenericUser(ctx, user, db)
}

func (r *MongoDBUserReconciler) reconcileGenericUser(ctx context.Context, user infrav1beta1.MongoDBUser, db infrav1beta1.MongoDBDatabase) (infrav1beta1.MongoDBUser, error) {
	// Fetch referencing root secret
	usr, pw, err := getSecret(ctx, r.Client, db.GetRootSecret())

	if err != nil {
		infrav1beta1.UserNotReadyCondition(&user, infrav1beta1.CredentialsNotFoundReason, err.Error())
		return user, err
	}

	dbHandler, err := setupMongoDB(ctx, db, usr, pw)

	if err != nil {
		infrav1beta1.UserNotReadyCondition(&user, infrav1beta1.ConnectionFailedReason, err.Error())
		return user, err
	}

	defer dbHandler.Close(ctx)

	if !user.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.finalizeUser(ctx, user, db, dbHandler)
	}

	// Fetch referencing secret
	usr, pw, err = getSecret(ctx, r.Client, user.GetCredentials())

	if err != nil {
		infrav1beta1.UserNotReadyCondition(&user, infrav1beta1.CredentialsNotFoundReason, err.Error())
		return user, err
	}

	err = dbHandler.SetupUser(ctx, db.GetDatabaseName(), usr, pw, extractMongoDBUserRoles(user.GetRoles()))
	if err != nil {
		err = fmt.Errorf("Failed to provision user account: %w", err)
		infrav1beta1.UserNotReadyCondition(&user, infrav1beta1.ConnectionFailedReason, err.Error())
		return user, err
	}

	user.Status.Username = usr

	return user, nil
}

func (r *MongoDBUserReconciler) reconcileAtlasUser(ctx context.Context, user infrav1beta1.MongoDBUser, db infrav1beta1.MongoDBDatabase) (infrav1beta1.MongoDBUser, error) {
	// Fetch referencing root secret
	pubKey, privKey, err := getSecret(ctx, r.Client, db.GetRootSecret())

	if err != nil {
		infrav1beta1.UserNotReadyCondition(&user, infrav1beta1.CredentialsNotFoundReason, err.Error())
		return user, err
	}

	dbHandler, err := setupAtlas(ctx, db, pubKey, privKey)

	if err != nil {
		infrav1beta1.UserNotReadyCondition(&user, infrav1beta1.ConnectionFailedReason, err.Error())
		return user, err
	}

	defer dbHandler.Close(ctx)

	if !user.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.finalizeUser(ctx, user, db, dbHandler)
	}

	// Fetch referencing secret
	usr, pw, err := getSecret(ctx, r.Client, user.GetCredentials())

	if err != nil {
		infrav1beta1.UserNotReadyCondition(&user, infrav1beta1.CredentialsNotFoundReason, err.Error())
		return user, err
	}

	err = dbHandler.SetupUser(ctx, db.GetDatabaseName(), usr, pw, extractMongoDBUserRoles(user.GetRoles()))
	if err != nil {
		err = fmt.Errorf("Failed to provison user account: %w", err)
		infrav1beta1.UserNotReadyCondition(&user, infrav1beta1.ConnectionFailedReason, err.Error())
		return user, err
	}

	user.Status.Username = usr

	return user, nil
}

func (r *MongoDBUserReconciler) finalizeUser(ctx context.Context, user infrav1beta1.MongoDBUser, db infrav1beta1.MongoDBDatabase, userDropper userDropper) (infrav1beta1.MongoDBUser, error) {
	err := userDropper.DropUser(ctx, db.GetDatabaseName(), user.Status.Username)
	if err != nil {
		err = fmt.Errorf("Failed to remove user account: %w", err)
		infrav1beta1.UserNotReadyCondition(&user, infrav1beta1.ConnectionFailedReason, err.Error())
		return user, err
	}

	if stringutils.ContainsString(user.ObjectMeta.Finalizers, v1beta1.Finalizer) {
		user.ObjectMeta.Finalizers = stringutils.RemoveString(user.ObjectMeta.Finalizers, v1beta1.Finalizer)
		if err := r.Update(ctx, &user); err != nil {
			return user, err
		}
	}

	return user, nil
}

func (r *MongoDBUserReconciler) patchStatus(ctx context.Context, database *infrav1beta1.MongoDBUser) error {
	key := client.ObjectKeyFromObject(database)
	latest := &infrav1beta1.MongoDBUser{}
	if err := r.Client.Get(ctx, key, latest); err != nil {
		return err
	}

	return r.Client.Status().Patch(ctx, database, client.MergeFrom(latest))
}
