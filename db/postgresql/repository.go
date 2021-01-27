package postgresql

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	"os"
)

type PostgreSQLServer struct {
	Host         string
	Port         string
	RootUser     string
	RootPassword string
}

func NewPostgreSQLServer(host string, port string, rootUser string, rootPassword string) *PostgreSQLServer {
	return &PostgreSQLServer{
		Host:         host,
		Port:         port,
		RootUser:     rootUser,
		RootPassword: rootPassword,
	}
}

func (s *PostgreSQLServer) Connect() (*string, error) {
	conn, err := pgx.Connect(context.Background(), fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres", s.Host, s.Port, s.RootUser, s.RootPassword))
	if err != nil {
		// TODO logging must be handled same as in controller, so pass logger to this struct
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	var greeting string
	err = conn.QueryRow(context.Background(), "SELECT 1 FROM pg_database WHERE datname='postgres'").Scan(&greeting)
	if err != nil {
		return nil, err
	}
	return &greeting, nil
}
