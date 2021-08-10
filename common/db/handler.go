package db

import (
	"context"

	infrav1beta1 "github.com/doodlescheduling/k8sdb-controller/api/v1beta1"
)

// Invoke a database handler
type Invoke func(ctx context.Context, uri, database, username, password string) (Handler, error)

// Handler is a wrapper arround a certain database client
type Handler interface {
	Close(ctx context.Context) error
	SetupUser(ctx context.Context, database string, username string, password string, roles []infrav1beta1.Role) error
	DropUser(ctx context.Context, database string, username string) error
	CreateDatabaseIfNotExists(ctx context.Context, database string) error
	EnableExtension(ctx context.Context, name string) error
}
