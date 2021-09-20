package controllers

import (
	"context"
	"errors"
	"fmt"

	"github.com/doodlescheduling/k8sdb-controller/api/v1beta1"
	infrav1beta1 "github.com/doodlescheduling/k8sdb-controller/api/v1beta1"
	"github.com/doodlescheduling/k8sdb-controller/common/db"
	appsv1 "k8s.io/api/apps/v1"
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
	GetAtlasGroupId() string
	GetDatabaseName() string
	GetRootDatabaseName() string
	GetExtensions() infrav1beta1.Extensions
	MigrationRequired() bool
	GetMigrationWorkloads() []infrav1beta1.WorkloadReference
	GetMigrationRootSecret() *infrav1beta1.SecretReference
	GetMigrationAddress() string
	GetMigrationDatabaseName() string
}

type user interface {
	metav1.Object
	runtime.Object
	GetStatusConditions() *[]metav1.Condition
	GetCredentials() *infrav1beta1.SecretReference
	GetRoles() []infrav1beta1.Role
	GetDatabase() string
}

// objectKey returns c.ObjectKey for the object.
func objectKey(object metav1.Object) client.ObjectKey {
	return client.ObjectKey{
		Namespace: object.GetNamespace(),
		Name:      object.GetName(),
	}
}

func extractRoles(roles []infrav1beta1.Role) db.Roles {
	list := make(db.Roles, 0)
	for _, r := range roles {
		list = append(list, db.Role{
			Name: r.Name,
			DB:   r.DB,
		})
	}

	return list
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

func reconcileDatabase(c client.Client, invoke db.Invoke, database database, recorder record.EventRecorder) (database, ctrl.Result) {
	if database.GetAtlasGroupId() != "" {
		return reconcileAtlasDatabase(c, database, recorder)
	}

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

	if database.MigrationRequired() {
		rootDBHandler, err := invoke(context.TODO(), db.ConnectionOptions{
			URI:              database.GetAddress(),
			AuthDatabaseName: database.GetRootDatabaseName(),
			DatabaseName:     database.GetDatabaseName(),
			Username:         usr,
			Password:         pw,
		})

		if err != nil {
			msg := fmt.Sprintf("Failed to setup connection to database server: %s", err.Error())
			recorder.Event(database, "Normal", "error", msg)
			infrav1beta1.DatabaseNotReadyCondition(database, infrav1beta1.ConnectionFailedReason, msg)
			return database, ctrl.Result{Requeue: true}
		}

		srcUsername := usr
		srcPassword := pw

		if database.GetMigrationRootSecret() != nil {
			// Fetch referencing root secret
			secret := &corev1.Secret{}
			secretName := types.NamespacedName{
				Namespace: database.GetNamespace(),
				Name:      database.GetMigrationRootSecret().Name,
			}
			err := c.Get(context.TODO(), secretName, secret)

			// Failed to fetch referenced secret, requeue immediately
			if err != nil {
				msg := fmt.Sprintf("Referencing migration root secret was not found: %s", err.Error())
				recorder.Event(database, "Normal", "error", msg)
				infrav1beta1.DatabaseNotReadyCondition(database, v1beta1.SecretNotFoundReason, msg)
				return database, ctrl.Result{Requeue: true}
			}

			srcUsername, srcPassword, err = extractCredentials(database.GetMigrationRootSecret(), secret)
		}

		srcOpts := db.ConnectionOptions{
			URI:          database.GetMigrationAddress(),
			DatabaseName: database.GetMigrationDatabaseName(),
			Username:     srcUsername,
			Password:     srcPassword,
		}

		err = downscaleWorkloads(ctx, c, database)
		if err != nil {
			msg := fmt.Sprintf("Failed to scale down referenced workloads: %s", err.Error())
			recorder.Event(database, "Normal", "error", msg)
			infrav1beta1.DatabaseNotReadyCondition(database, infrav1beta1.ConnectionFailedReason, msg)
			return database, ctrl.Result{Requeue: true}
		}

		err = rootDBHandler.RestoreDatabaseFrom(ctx, srcOpts)

		if err != nil {
			msg := fmt.Sprintf("Failed to migrate database: %s", err.Error())
			recorder.Event(database, "Normal", "error", msg)
			infrav1beta1.DatabaseNotReadyCondition(database, v1beta1.MigrationFailedReason, msg)
			return database, ctrl.Result{Requeue: true}
		}

		err = upscaleWorkloads(ctx, c, database)
		if err != nil {
			msg := fmt.Sprintf("Failed to scale up referenced workloads: %s", err.Error())
			recorder.Event(database, "Normal", "error", msg)
			infrav1beta1.DatabaseNotReadyCondition(database, infrav1beta1.ConnectionFailedReason, msg)
			return database, ctrl.Result{Requeue: true}
		}
	} else {
		rootDBHandler, err := invoke(context.TODO(), db.ConnectionOptions{
			URI:              database.GetAddress(),
			AuthDatabaseName: database.GetRootDatabaseName(),
			Username:         usr,
			Password:         pw,
		})

		if err != nil {
			msg := fmt.Sprintf("Failed to setup connection to database server: %s", err.Error())
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
	}

	targetDBHandler, err := invoke(context.TODO(), db.ConnectionOptions{
		URI:              database.GetAddress(),
		AuthDatabaseName: database.GetRootDatabaseName(),
		DatabaseName:     database.GetDatabaseName(),
		Username:         usr,
		Password:         pw,
	})

	if err != nil {
		msg := fmt.Sprintf("Failed to setup connection to database server: %s", err.Error())
		recorder.Event(database, "Normal", "error", msg)
		infrav1beta1.DatabaseNotReadyCondition(database, infrav1beta1.ConnectionFailedReason, msg)
		return database, ctrl.Result{Requeue: true}
	}
	for _, extension := range database.GetExtensions() {
		if err := targetDBHandler.EnableExtension(ctx, extension.Name); err != nil {
			msg := fmt.Sprintf("Failed to create extension %s in database: %s", extension.Name, err.Error())
			recorder.Event(database, "Normal", "error", msg)
			infrav1beta1.ExtensionNotReadyCondition(database, infrav1beta1.CreateExtensionFailedReason, msg)
			return database, ctrl.Result{Requeue: true}
		}
	}

	msg := "Database successfully provisioned"
	recorder.Event(database, "Normal", "info", msg)
	v1beta1.DatabaseReadyCondition(database, v1beta1.DatabaseProvisiningSuccessfulReason, msg)
	return database, ctrl.Result{}
}

func downscaleWorkloads(ctx context.Context, c client.Client, database database) error {
	workloads := database.GetMigrationWorkloads()
	for _, wl := range workloads {
		var r client.Object
		switch wl.Kind {
		case "Deployment":
			r = &appsv1.Deployment{}
		case "StatefulSet":
			r = &appsv1.StatefulSet{}
		case "ReplicaSet":
			r = &appsv1.ReplicaSet{}
		default:
			return fmt.Errorf("unknown workload kind %s given", wl.Kind)
		}

		err := c.Get(ctx, client.ObjectKey{
			Namespace: database.GetNamespace(),
			Name:      wl.Name,
		}, r)

		if err != nil {
			return err
		}

		var zero int32
		switch wl.Kind {
		case "Deployment":
			if r.(*appsv1.Deployment).Spec.Replicas != &zero {
				wl.Replicas = r.(*appsv1.Deployment).Spec.Replicas
			}

			r.(*appsv1.Deployment).Spec.Replicas = &zero
		case "StatefulSet":
			if r.(*appsv1.StatefulSet).Spec.Replicas != &zero {
				wl.Replicas = r.(*appsv1.StatefulSet).Spec.Replicas
			}

			r.(*appsv1.StatefulSet).Spec.Replicas = &zero
		case "ReplicaSet":
			if r.(*appsv1.ReplicaSet).Spec.Replicas != &zero {
				wl.Replicas = r.(*appsv1.ReplicaSet).Spec.Replicas
			}

			r.(*appsv1.ReplicaSet).Spec.Replicas = &zero
		default:
			return fmt.Errorf("unknown workload kind %s given", wl.Kind)
		}

		err = c.Update(ctx, r)
		if err != nil {
			return err
		}
	}

	return nil
}

func upscaleWorkloads(ctx context.Context, c client.Client, database database) error {
	workloads := database.GetMigrationWorkloads()
	for _, wl := range workloads {
		var r client.Object
		switch wl.Kind {
		case "Deployment":
			r = &appsv1.Deployment{}
		case "StatefulSet":
			r = &appsv1.StatefulSet{}
		case "ReplicaSet":
			r = &appsv1.ReplicaSet{}
		default:
			return fmt.Errorf("unknown workload kind %s given", wl.Kind)
		}

		err := c.Get(ctx, client.ObjectKey{
			Namespace: database.GetNamespace(),
			Name:      wl.Name,
		}, r)

		if err != nil {
			return err
		}

		switch wl.Kind {
		case "Deployment":
			r.(*appsv1.Deployment).Spec.Replicas = wl.Replicas
		case "StatefulSet":
			r.(*appsv1.StatefulSet).Spec.Replicas = wl.Replicas
		case "ReplicaSet":
			r.(*appsv1.ReplicaSet).Spec.Replicas = wl.Replicas
		default:
			return fmt.Errorf("unknown workload kind %s given", wl.Kind)
		}

		err = c.Update(ctx, r)
		if err != nil {
			return err
		}
	}

	return nil
}

func reconcileUser(database database, c client.Client, invoke db.Invoke, user user, recorder record.EventRecorder) (user, ctrl.Result) {
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

	if database.GetAtlasGroupId() != "" {
		return reconcileAtlasUser(database, c, user, recorder)
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

	dbHandler, err := invoke(context.TODO(), db.ConnectionOptions{
		URI:              database.GetAddress(),
		AuthDatabaseName: database.GetRootDatabaseName(),
		DatabaseName:     database.GetDatabaseName(),
		Username:         usr,
		Password:         pw,
	})

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
