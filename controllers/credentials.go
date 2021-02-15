package controllers

import (
	"context"
	"errors"
	"fmt"

	"github.com/doodlescheduling/kubedb/api/v1beta1"
	infrav1beta1 "github.com/doodlescheduling/kubedb/api/v1beta1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// objectKey returns client.ObjectKey for the object.
func objectKey(object metav1.Object) client.ObjectKey {
	return client.ObjectKey{
		Namespace: object.GetNamespace(),
		Name:      object.GetName(),
	}
}

func extractCredentials(credentials *infrav1beta1.SecretReference, secret *corev1.Secret) (string, string, error) {
	var (
		user string
		pw   string
	)

	if val, ok := secret.Data[credentials.UserField]; !ok {
		return "", "", errors.New("Defined username field not found in secret")
	} else {
		user = string(val)
	}

	if val, ok := secret.Data[credentials.PasswordField]; !ok {
		return "", "", errors.New("Defined password field not found in secret")
	} else {
		pw = string(val)
	}

	return user, pw, nil
}

func () reconcileDatabase(ctx context.Context, r *MongoDBDatabaseReconciler, database infrav1beta1.MongoDBDatabase, logger logr.Logger) (infrav1beta1.MongoDBDatabase, ctrl.Result, error) {
	// Fetch referencing root secret
	secret := &corev1.Secret{}
	secretName := types.NamespacedName{
		Namespace: database.GetNamespace(),
		Name:      database.Spec.RootSecret.Name,
	}
	err := r.Client.Get(context.TODO(), secretName, secret)

	// Failed to fetch referenced secret, requeue immediately
	if err != nil {
		msg := fmt.Sprintf("Referencing root secret was not found: %s", err.Error())
		r.Recorder.Event(&database, "Normal", "error", msg)
		infrav1beta1.DatabaseNotReadyCondition(&database, v1beta1.SecretNotFoundReason, msg)
		return database, ctrl.Result{Requeue: true}, err
	}

	usr, pw, err := extractCredentials(database.Spec.RootSecret, secret)

	if err != nil {
		msg := fmt.Sprintf("Credentials field not found in referenced rootSecret: %s", err.Error())
		r.Recorder.Event(&database, "Normal", "error", msg)
		infrav1beta1.DatabaseNotReadyCondition(&database, infrav1beta1.CredentialsNotFoundReason, msg)
		return database, ctrl.Result{Requeue: true}, err
	}

	// mongoDB connection to spec host, cached
	_, err = r.ClientPool.FromURI(context.TODO(), database.Spec.Address, usr, pw)
	if err != nil {
		msg := fmt.Sprintf("Failed to setup connection to database server: %s", err.Error())
		r.Recorder.Event(&database, "Normal", "error", msg)
		infrav1beta1.DatabaseNotReadyCondition(&database, infrav1beta1.ConnectionFailedReason, msg)
		return database, ctrl.Result{Requeue: true}, err
	}

	//There is nothing todo for MongoDB at this point, we can only verify the connection

	msg := "Database successfully provisioned"
	r.Recorder.Event(&database, "Normal", "info", msg)
	v1beta1.DatabaseReadyCondition(&database, v1beta1.DatabaseProvisiningSuccessfulReason, msg)
	return database, ctrl.Result{}, err
}

func reconcileUser(ctx context.Context, client, user infrav1beta1.MongoDBUser, logger logr.Logger) (infrav1beta1.MongoDBUser, ctrl.Result, error) {
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
