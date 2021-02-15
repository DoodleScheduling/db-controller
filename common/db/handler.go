package db

import (
	"context"
)

// Invoke a database handler
type Invoke (func(ctx context.Context, uri, username, password string) (Handler, error))

// Handler is a wrapper arround a certain database client
type Handler interface {
	Close() error
	SetupUser(database string, username string, password string) error
	DropUser(database string, username string) error
	CreateDatabaseIfNotExists(database string) error
}
