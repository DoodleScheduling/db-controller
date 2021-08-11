package db

import (
	"context"
)

type Roles []Role
type Role struct {
	Name string `json:"role" bson:"role"`
	DB   string `json:"db" bson:"db"`
}

type Users []User
type User struct {
	User  string `json:"user" bson:"user"`
	DB    string `json:"db" bson:"db"`
	Roles Roles  `json:"roles" bson:"roles"`
}

// Invoke a database handler
type Invoke func(ctx context.Context, uri, database, username, password string) (Handler, error)

// Handler is a wrapper arround a certain database client
type Handler interface {
	Close(ctx context.Context) error
	SetupUser(ctx context.Context, database string, username string, password string, roles Roles) error
	DropUser(ctx context.Context, database string, username string) error
	CreateDatabaseIfNotExists(ctx context.Context, database string) error
	EnableExtension(ctx context.Context, name string) error
}
