package controllers

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1beta1 "github.com/doodlescheduling/db-controller/api/v1beta1"
	"github.com/doodlescheduling/db-controller/internal/database"
)

// Index keys
const (
	secretIndexKey      string = ".metadata.secret"
	credentialsIndexKey string = ".metadata.credentials"
	dbIndexKey          string = ".metadata.database"
)

type userDropper interface {
	DropUser(ctx context.Context, db, username string) error
}

// objectKey returns c.ObjectKey for the object.
func objectKey(object metav1.Object) client.ObjectKey {
	return client.ObjectKey{
		Namespace: object.GetNamespace(),
		Name:      object.GetName(),
	}
}

func extractMongoDBUserRoles(roles []infrav1beta1.MongoDBUserRole) database.MongoDBRoles {
	list := make(database.MongoDBRoles, 0)
	for _, r := range roles {
		list = append(list, database.MongoDBRole{
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

	userField := credentials.UserField
	if userField == "" {
		userField = "username"
	}

	pwField := credentials.PasswordField
	if pwField == "" {
		pwField = "password"
	}

	if val, ok := secret.Data[userField]; !ok {
		return "", "", errors.New("defined username field not found in secret")
	} else {
		user = string(val)
	}

	if val, ok := secret.Data[pwField]; !ok {
		return "", "", errors.New("defined password field not found in secret")
	} else {
		pw = string(val)
	}

	return user, pw, nil
}

func setupAtlas(ctx context.Context, db infrav1beta1.MongoDBDatabase, pubKey, privKey string) (*database.AtlasRepository, error) {
	handler, err := database.NewAtlasRepository(context.TODO(), database.AtlasOptions{
		GroupID:    db.Spec.AtlasGroupId,
		PrivateKey: privKey,
		PublicKey:  pubKey,
	})

	if err != nil {
		return handler, fmt.Errorf("failed to setup connection to mongodb atlas: %w", err)
	}

	return handler, nil
}

func setupPostgreSQL(ctx context.Context, db infrav1beta1.PostgreSQLDatabase, usr, pw string, switchDB bool) (*database.PostgreSQLRepository, error) {
	opts := database.PostgreSQLOptions{
		URI:      db.Spec.Address,
		Username: usr,
		Password: pw,
	}

	if switchDB {
		opts.DatabaseName = db.GetDatabaseName()
	}

	handler, err := database.NewPostgreSQLRepository(context.TODO(), opts)

	if err != nil {
		return handler, fmt.Errorf("failed to setup connection to postgres server: %w", err)
	}

	return handler, nil
}

func setupMongoDB(ctx context.Context, db infrav1beta1.MongoDBDatabase, usr, pw string) (*database.MongoDBRepository, error) {
	handler, err := database.NewMongoDBRepository(context.TODO(), database.MongoDBOptions{
		URI:              db.Spec.Address,
		AuthDatabaseName: db.GetRootDatabaseName(),
		DatabaseName:     db.GetDatabaseName(),
		Username:         usr,
		Password:         pw,
	})

	if err != nil {
		return handler, fmt.Errorf("failed to setup connection to mongodb: %w", err)
	}

	return handler, nil
}

func getSecret(ctx context.Context, c client.Client, sec *infrav1beta1.SecretReference) (string, string, error) {
	// Fetch referencing root secret
	secret := &corev1.Secret{}
	secretName := types.NamespacedName{
		Namespace: sec.Namespace,
		Name:      sec.Name,
	}
	err := c.Get(ctx, secretName, secret)

	// Failed to fetch referenced secret, requeue immediately
	if err != nil {
		return "", "", fmt.Errorf("referencing secret was not found: %w", err)
	}

	usr, pw, err := extractCredentials(sec, secret)
	if err != nil {
		return usr, pw, fmt.Errorf("credentials field not found in referenced secret: %w", err)
	}

	return usr, pw, err
}
