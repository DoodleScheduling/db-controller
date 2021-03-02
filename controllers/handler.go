package controllers

import (
	"context"
	"errors"
	"fmt"

	"github.com/doodlescheduling/k8sdb-controller/api/v1beta1"
	infrav1beta1 "github.com/doodlescheduling/k8sdb-controller/api/v1beta1"
	"github.com/doodlescheduling/k8sdb-controller/common/db"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Index keys
const (
	secretIndexKey      string = ".metadata.secret"
	credentialsIndexKey string = ".metadata.credentials"
	dbIndexKey          string = ".metadata.database"
)

type database interface {
	metav1.Object
	runtime.Object
	GetStatusConditions() *[]metav1.Condition
	GetRootSecret() *infrav1beta1.SecretReference
	GetAddress() string
	GetDatabaseName() string
	GetRootDatabaseName() string
	GetExtensions() infrav1beta1.Extensions
}

type user interface {
	metav1.Object
	runtime.Object
	GetStatusConditions() *[]metav1.Condition
	GetCredentials() *infrav1beta1.SecretReference
	GetDatabase() string
}

// objectKey returns c.ObjectKey for the object.
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

func reconcileDatabase(c client.Client, pool *db.ClientPool, invoke db.Invoke, database database, recorder record.EventRecorder) (database, ctrl.Result, error) {
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
		return database, ctrl.Result{Requeue: true}, nil
	}

	usr, pw, err := extractCredentials(database.GetRootSecret(), secret)

	if err != nil {
		msg := fmt.Sprintf("Credentials field not found in referenced rootSecret: %s", err.Error())
		recorder.Event(database, "Normal", "error", msg)
		infrav1beta1.DatabaseNotReadyCondition(database, infrav1beta1.CredentialsNotFoundReason, msg)
		return database, ctrl.Result{Requeue: true}, nil
	}

	rootDBHandler, err := pool.FromURI(context.TODO(), invoke, database.GetAddress(), database.GetRootDatabaseName(), usr, pw)
	if err != nil {
		msg := fmt.Sprintf("Failed to setup connection to database server: %s", err.Error())
		recorder.Event(database, "Normal", "error", msg)
		infrav1beta1.DatabaseNotReadyCondition(database, infrav1beta1.ConnectionFailedReason, msg)
		return database, ctrl.Result{Requeue: true}, nil
	}

	err = rootDBHandler.CreateDatabaseIfNotExists(database.GetDatabaseName())
	if err != nil {
		msg := fmt.Sprintf("Failed to provision database: %s", err.Error())
		recorder.Event(database, "Normal", "error", msg)
		infrav1beta1.DatabaseNotReadyCondition(database, infrav1beta1.CreateDatabaseFailedReason, msg)
		return database, ctrl.Result{Requeue: true}, nil
	}

	targetDBHandler, err := pool.FromURI(context.TODO(), invoke, database.GetAddress(), database.GetDatabaseName(), usr, pw)
	if err != nil {
		msg := fmt.Sprintf("Failed to setup connection to database server: %s", err.Error())
		recorder.Event(database, "Normal", "error", msg)
		infrav1beta1.DatabaseNotReadyCondition(database, infrav1beta1.ConnectionFailedReason, msg)
		return database, ctrl.Result{Requeue: true}, nil
	}
	for _, extension := range database.GetExtensions() {
		if err := targetDBHandler.EnableExtension(extension.Name); err != nil {
			msg := fmt.Sprintf("Failed to create extension %s in database: %s", extension.Name, err.Error())
			recorder.Event(database, "Normal", "error", msg)
			infrav1beta1.DatabaseNotReadyCondition(database, infrav1beta1.CreateDatabaseFailedReason, msg)
			return database, ctrl.Result{Requeue: true}, nil
		}
	}

	msg := "Database successfully provisioned"
	recorder.Event(database, "Normal", "info", msg)
	v1beta1.DatabaseReadyCondition(database, v1beta1.DatabaseProvisiningSuccessfulReason, msg)
	return database, ctrl.Result{}, nil
}

func reconcileUser(database database, c client.Client, pool *db.ClientPool, invoke db.Invoke, user user, recorder record.EventRecorder) (user, ctrl.Result, error) {
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
		infrav1beta1.DatabaseNotReadyCondition(user, v1beta1.DatabaseNotFoundReason, msg)
		return user, ctrl.Result{Requeue: true}, nil
	}

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
		infrav1beta1.DatabaseNotReadyCondition(user, v1beta1.SecretNotFoundReason, msg)
		return user, ctrl.Result{Requeue: true}, nil
	}

	usr, pw, err := extractCredentials(database.GetRootSecret(), secret)

	if err != nil {
		msg := fmt.Sprintf("Credentials field not found in referenced rootSecret: %s", err.Error())
		recorder.Event(user, "Normal", "error", msg)
		infrav1beta1.DatabaseNotReadyCondition(user, infrav1beta1.CredentialsNotFoundReason, msg)
		return user, ctrl.Result{Requeue: true}, nil
	}

	dbHandler, err := pool.FromURI(context.TODO(), invoke, database.GetAddress(), database.GetRootDatabaseName(), usr, pw)

	if err != nil {
		msg := fmt.Sprintf("Failed to setup connection to database server: %s", err.Error())
		recorder.Event(user, "Normal", "error", msg)
		infrav1beta1.DatabaseNotReadyCondition(user, infrav1beta1.ConnectionFailedReason, msg)
		return user, ctrl.Result{Requeue: true}, nil
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
		return user, ctrl.Result{Requeue: true}, nil
	}

	err = dbHandler.SetupUser(database.GetDatabaseName(), usr, pw)
	if err != nil {
		msg := fmt.Sprintf("Failed to provison user account: %s", err.Error())
		recorder.Event(user, "Normal", "error", msg)
		infrav1beta1.UserNotReadyCondition(user, infrav1beta1.ConnectionFailedReason, msg)
		return user, ctrl.Result{Requeue: true}, nil
	}

	msg := "User successfully provisioned"
	recorder.Event(user, "Normal", "info", msg)
	v1beta1.UserReadyCondition(user, v1beta1.UserProvisioningSuccessfulReason, msg)
	return user, ctrl.Result{}, nil
}
