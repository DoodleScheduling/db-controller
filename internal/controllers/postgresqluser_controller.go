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
	"crypto/rand"
	"encoding/hex"
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

	infrav1beta1 "github.com/doodlescheduling/db-controller/api/v1beta1"
	"github.com/doodlescheduling/db-controller/internal/database"
	"github.com/doodlescheduling/db-controller/internal/stringutils"
)

// +kubebuilder:rbac:groups=dbprovisioning.infra.doodle.com,resources=postgresqlusers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dbprovisioning.infra.doodle.com,resources=postgresqlusers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// PostgreSQLUserReconciler reconciles a PostgreSQLUser object
type PostgreSQLUserReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func (r *PostgreSQLUserReconciler) SetupWithManager(mgr ctrl.Manager, maxConcurrentReconciles int) error {
	// Index the PostgreSQLUser by the Credentials references they point at
	if err := mgr.GetFieldIndexer().IndexField(context.TODO(), &infrav1beta1.PostgreSQLUser{}, credentialsIndexKey,
		func(o client.Object) []string {
			usr := o.(*infrav1beta1.PostgreSQLUser)
			return []string{
				fmt.Sprintf("%s/%s", usr.GetNamespace(), usr.Spec.Credentials.Name),
			}
		},
	); err != nil {
		return err
	}

	// Index the PostgreSQLUser by the Database references they point at
	if err := mgr.GetFieldIndexer().IndexField(context.TODO(), &infrav1beta1.PostgreSQLUser{}, dbIndexKey,
		func(o client.Object) []string {
			usr := o.(*infrav1beta1.PostgreSQLUser)
			return []string{
				fmt.Sprintf("%s/%s", usr.GetNamespace(), usr.Spec.Database.Name),
			}
		},
	); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1beta1.PostgreSQLUser{}).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.requestsForSecretChange),
		).
		Watches(
			&infrav1beta1.PostgreSQLDatabase{},
			handler.EnqueueRequestsFromMapFunc(r.requestsForDatabaseChange),
		).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxConcurrentReconciles}).
		Complete(r)
}

func (r *PostgreSQLUserReconciler) requestsForSecretChange(ctx context.Context, o client.Object) []reconcile.Request {
	s, ok := o.(*corev1.Secret)
	if !ok {
		panic(fmt.Sprintf("expected a Secret, got %T", o))
	}

	var list infrav1beta1.PostgreSQLUserList
	if err := r.List(ctx, &list, client.MatchingFields{
		credentialsIndexKey: objectKey(s).String(),
	}); err != nil {
		return nil
	}

	var reqs []reconcile.Request
	for _, i := range list.Items {
		r.Log.Info("referenced secret from a postgresqluser change detected", "namespace", i.GetNamespace(), "name", i.GetName())
		reqs = append(reqs, reconcile.Request{NamespacedName: objectKey(&i)})
	}

	return reqs
}

func (r *PostgreSQLUserReconciler) requestsForDatabaseChange(ctx context.Context, o client.Object) []reconcile.Request {
	s, ok := o.(*infrav1beta1.PostgreSQLDatabase)
	if !ok {
		panic(fmt.Sprintf("expected a PostgreSQLDatabase, got %T", o))
	}

	var list infrav1beta1.PostgreSQLUserList
	if err := r.List(ctx, &list, client.MatchingFields{
		dbIndexKey: objectKey(s).String(),
	}); err != nil {
		return nil
	}

	var reqs []reconcile.Request
	for _, i := range list.Items {
		r.Log.Info("referenced database from a postgresqluser change detected, reconcile", "namespace", i.GetNamespace(), "name", i.GetName())
		reqs = append(reqs, reconcile.Request{NamespacedName: objectKey(&i)})
	}

	return reqs
}

func (r *PostgreSQLUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("PostgreSQLUser", req.NamespacedName)
	logger.Info("reconciling PostgreSQLUser")

	// common controller functions
	//cw := NewControllerWrapper(*r, &ctx)

	// garbage collector
	//gc := NewPostgreSQLGarbageCollector(r, cw, &logger)

	// get database resource by namespaced name
	var user infrav1beta1.PostgreSQLUser
	if err := r.Get(ctx, req.NamespacedName, &user); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// examine DeletionTimestamp to determine if object is under deletion
	if user.ObjectMeta.DeletionTimestamp.IsZero() {
		if !stringutils.ContainsString(user.GetFinalizers(), infrav1beta1.Finalizer) {
			controllerutil.AddFinalizer(&user, infrav1beta1.Finalizer)
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

func (r *PostgreSQLUserReconciler) reconcile(ctx context.Context, user infrav1beta1.PostgreSQLUser) (infrav1beta1.PostgreSQLUser, error) {
	// Fetch referencing database
	var db infrav1beta1.PostgreSQLDatabase
	databaseName := types.NamespacedName{
		Namespace: user.GetNamespace(),
		Name:      user.GetDatabase(),
	}

	err := r.Client.Get(ctx, databaseName, &db)
	if err != nil {
		err = fmt.Errorf("referencing database was not found: %w", err)
		infrav1beta1.UserNotReadyCondition(&user, infrav1beta1.DatabaseNotFoundReason, err.Error())
		return user, err
	}

	// Fetch referencing root secret
	usr, pw, err := getSecret(ctx, r.Client, db.GetRootSecret())

	if err != nil {
		infrav1beta1.UserNotReadyCondition(&user, infrav1beta1.CredentialsNotFoundReason, err.Error())
		return user, err
	}

	dbHandler, err := setupPostgreSQL(ctx, db, usr, pw, true)

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

	var grants []database.Grant
	for _, grant := range user.Spec.Grants {
		var privs []database.Privilege
		for _, p := range grant.Privileges {
			privs = append(privs, database.Privilege(p))
		}

		grants = append(grants, database.Grant{
			Object:     grant.Object,
			ObjectName: grant.ObjectName,
			User:       grant.User,
			Privileges: privs,
		})
	}

	userSpec := database.PostgresqlUser{
		Database: db.GetDatabaseName(),
		Username: usr,
		Password: pw,
		Roles:    user.Spec.Roles,
		Grants:   grants,
	}

	err = dbHandler.SetupUser(ctx, userSpec)
	if err != nil {
		err = fmt.Errorf("failed to provison user account: %w", err)
		infrav1beta1.UserNotReadyCondition(&user, infrav1beta1.ConnectionFailedReason, err.Error())
		return user, err
	}

	user.Status.Username = usr

	return user, nil
}

func generateToken(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

func (r *PostgreSQLUserReconciler) finalizeUser(ctx context.Context, user infrav1beta1.PostgreSQLUser, db infrav1beta1.PostgreSQLDatabase, dbHandler *database.PostgreSQLRepository) (infrav1beta1.PostgreSQLUser, error) {
	if user.Status.Username == "" {
		return user, nil
	}

	//We can't easily drop a user from postgres since it ownes objects
	//err := userDropper.DropUser(ctx, db.GetDatabaseName(), user.Status.Username)

	userSpec := database.PostgresqlUser{
		Database: db.GetDatabaseName(),
		Username: user.Status.Username,
		Password: generateToken(32),
	}

	//Instead privileges are revoked and the password gets randomized
	err := dbHandler.SetupUser(ctx, userSpec)
	if err != nil {
		err = fmt.Errorf("failed to update user account: %w", err)
		infrav1beta1.UserNotReadyCondition(&user, infrav1beta1.ConnectionFailedReason, err.Error())
		return user, err
	}

	err = dbHandler.RevokeAllPrivileges(ctx, userSpec)

	if err != nil {
		err = fmt.Errorf("failed to revoke privileges from user account: %w", err)
		infrav1beta1.UserNotReadyCondition(&user, infrav1beta1.ConnectionFailedReason, err.Error())
		return user, err
	}

	if stringutils.ContainsString(user.ObjectMeta.Finalizers, infrav1beta1.Finalizer) {
		user.ObjectMeta.Finalizers = stringutils.RemoveString(user.ObjectMeta.Finalizers, infrav1beta1.Finalizer)
		if err := r.Update(ctx, &user); err != nil {
			return user, err
		}
	}

	return user, nil
}

func (r *PostgreSQLUserReconciler) patchStatus(ctx context.Context, database *infrav1beta1.PostgreSQLUser) error {
	key := client.ObjectKeyFromObject(database)
	latest := &infrav1beta1.PostgreSQLUser{}
	if err := r.Client.Get(ctx, key, latest); err != nil {
		return err
	}

	return r.Client.Status().Patch(ctx, database, client.MergeFrom(latest))
}
