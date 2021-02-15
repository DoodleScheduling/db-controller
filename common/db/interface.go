package db

import (
	"context"
)

type Invoke (func(ctx context.Context, uri, username, password string) (Interface, error))

type Interface interface {
	Close() error
	SetupUser(database string, username string, password string) error
	CreateDatabaseIfNotExists(database string) error
}
