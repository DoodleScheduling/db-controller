package postgresql

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
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

// TODO Prepared Statements
func (s *PostgreSQLServer) CreateDatabaseIfNotExists(database string) error {
	conn, err := s.connect()
	if err != nil {
		return err
	}
	defer conn.Close(context.Background())

	if databaseExists, err := s.doesDatabaseExist(conn, database); err != nil {
		return err
	} else {
		if databaseExists {
			return nil
		}
		if _, err := conn.Exec(context.Background(), fmt.Sprintf("CREATE DATABASE \"%s\";", database)); err != nil {
			return err
		} else {
			if databaseExistsNow, err := s.doesDatabaseExist(conn, database); err != nil {
				return err
			} else {
				if databaseExistsNow {
					return nil
				} else {
					return errors.New("database doesn't exist after create")
				}
			}
		}
	}
}

func (s *PostgreSQLServer) connect() (*pgx.Conn, error) {
	conn, err := pgx.Connect(context.Background(), fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres", s.Host, s.Port, s.RootUser, s.RootPassword))
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (s *PostgreSQLServer) doesDatabaseExist(conn *pgx.Conn, database string) (bool, error) {
	var result int64
	err := conn.QueryRow(context.Background(), fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname='%s';", database)).Scan(&result)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return result == 1, nil
}
