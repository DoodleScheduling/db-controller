package controllers

import (
	"context"
	"fmt"

	"github.com/doodlescheduling/k8sdb-controller/api/v1beta1"
	infrav1beta1 "github.com/doodlescheduling/k8sdb-controller/api/v1beta1"
	"github.com/doodlescheduling/k8sdb-controller/common/db"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func reconcileAtlasDatabase(c client.Client, database database, recorder record.EventRecorder) (database, ctrl.Result) {
	// Fetch referencing root secret
	secret := &corev1.Secret{}
	secretName := types.NamespacedName{
		Namespace: database.GetNamespace(),
		Name:      database.GetRootSecret().Name,
	}
	err := c.Get(context.TODO(), secretName, secret)

	// Failed to fetch referenced secret, requeue immediately
	if err != nil {
		msg := fmt.Sprintf("Referencing root secret was not found: %s", err.Error())
		recorder.Event(database, "Normal", "error", msg)
		infrav1beta1.DatabaseNotReadyCondition(database, v1beta1.SecretNotFoundReason, msg)
		return database, ctrl.Result{Requeue: true}
	}

	ctx := context.TODO()

	usr, pw, err := extractCredentials(database.GetRootSecret(), secret)

	if err != nil {
		msg := fmt.Sprintf("Credentials field not found in referenced rootSecret: %s", err.Error())
		recorder.Event(database, "Normal", "error", msg)
		infrav1beta1.DatabaseNotReadyCondition(database, infrav1beta1.CredentialsNotFoundReason, msg)
		return database, ctrl.Result{Requeue: true}
	}

	rootDBHandler, err := db.NewAtlasRepository(context.TODO(), database.GetAtlasGroupId(), usr, pw)
	if err != nil {
		msg := fmt.Sprintf("Failed to setup connection to atlas: %s", err.Error())
		recorder.Event(database, "Normal", "error", msg)
		infrav1beta1.DatabaseNotReadyCondition(database, infrav1beta1.ConnectionFailedReason, msg)
		return database, ctrl.Result{Requeue: true}
	}

	err = rootDBHandler.CreateDatabaseIfNotExists(ctx, database.GetDatabaseName())
	if err != nil {
		msg := fmt.Sprintf("Failed to provision database: %s", err.Error())
		recorder.Event(database, "Normal", "error", msg)
		infrav1beta1.DatabaseNotReadyCondition(database, infrav1beta1.CreateDatabaseFailedReason, msg)
		return database, ctrl.Result{Requeue: true}
	}

	msg := "Database successfully provisioned"
	recorder.Event(database, "Normal", "info", msg)
	v1beta1.DatabaseReadyCondition(database, v1beta1.DatabaseProvisiningSuccessfulReason, msg)
	return database, ctrl.Result{}
}

func reconcileAtlasUser(database database, c client.Client, user user, recorder record.EventRecorder) (user, ctrl.Result) {
	// Fetch referencing database
	databaseName := types.NamespacedName{
		Namespace: user.GetNamespace(),
		Name:      user.GetDatabase(),
	}
	err := c.Get(context.TODO(), databaseName, database)

	// Failed to fetch referenced database, requeue immediately
	if err != nil {
		msg := fmt.Sprintf("Referencing database was not found: %s", err.Error())
		recorder.Event(user, "Normal", "error", msg)
		infrav1beta1.UserNotReadyCondition(user, v1beta1.DatabaseNotFoundReason, msg)
		return user, ctrl.Result{Requeue: true}
	}

	ctx := context.TODO()

	// Fetch referencing root secret
	secret := &corev1.Secret{}
	secretName := types.NamespacedName{
		Namespace: database.GetNamespace(),
		Name:      database.GetRootSecret().Name,
	}
	err = c.Get(context.TODO(), secretName, secret)

	// Failed to fetch referenced secret, requeue immediately
	if err != nil {
		msg := fmt.Sprintf("Referencing root secret was not found: %s", err.Error())
		recorder.Event(user, "Normal", "error", msg)
		infrav1beta1.UserNotReadyCondition(user, v1beta1.SecretNotFoundReason, msg)
		return user, ctrl.Result{Requeue: true}
	}

	usr, pw, err := extractCredentials(database.GetRootSecret(), secret)

	if err != nil {
		msg := fmt.Sprintf("Credentials field not found in referenced rootSecret: %s", err.Error())
		recorder.Event(user, "Normal", "error", msg)
		infrav1beta1.UserNotReadyCondition(user, infrav1beta1.CredentialsNotFoundReason, msg)
		return user, ctrl.Result{Requeue: true}
	}

	dbHandler, err := db.NewAtlasRepository(context.TODO(), database.GetAtlasGroupId(), usr, pw)

	if err != nil {
		msg := fmt.Sprintf("Failed to setup connection to database server: %s", err.Error())
		recorder.Event(user, "Normal", "error", msg)
		infrav1beta1.UserNotReadyCondition(user, infrav1beta1.ConnectionFailedReason, msg)
		return user, ctrl.Result{Requeue: true}
	}

	// Fetch referencing credentials secret
	credentials := &corev1.Secret{}
	credentialsName := types.NamespacedName{
		Namespace: user.GetNamespace(),
		Name:      user.GetCredentials().Name,
	}

	err = c.Get(context.TODO(), credentialsName, credentials)
	usr, pw, err = extractCredentials(user.GetCredentials(), credentials)

	if err != nil {
		msg := fmt.Sprintf("No credentials found to provision user account: %s", err.Error())
		recorder.Event(user, "Normal", "error", msg)
		infrav1beta1.UserNotReadyCondition(user, infrav1beta1.CredentialsNotFoundReason, msg)
		return user, ctrl.Result{Requeue: true}
	}

	err = dbHandler.SetupUser(ctx, database.GetDatabaseName(), usr, pw, extractRoles(user.GetRoles()))
	if err != nil {
		msg := fmt.Sprintf("Failed to provison user account: %s", err.Error())
		recorder.Event(user, "Normal", "error", msg)
		infrav1beta1.UserNotReadyCondition(user, infrav1beta1.ConnectionFailedReason, msg)
		return user, ctrl.Result{Requeue: true}
	}

	msg := "User successfully provisioned"
	recorder.Event(user, "Normal", "info", msg)
	v1beta1.UserReadyCondition(user, v1beta1.UserProvisioningSuccessfulReason, msg)
	return user, ctrl.Result{}
}
