package postgresql

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type PostgreSQLServer struct {
	dbpool       *pgxpool.Pool
	Host         string
	Port         string
	RootUser     string
	RootPassword string
}

func NewPostgreSQLServer(host string, port string, rootUser string, rootPassword string) (*PostgreSQLServer, error) {
	dbpool, err := pgxpool.Connect(context.Background(), fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres", host, port, rootUser, rootPassword))
	if err != nil {
		return nil, err
	}
	return &PostgreSQLServer{
		dbpool:       dbpool,
		Host:         host,
		Port:         port,
		RootUser:     rootUser,
		RootPassword: rootPassword,
	}, nil
}

// TODO Prepared Statements
func (s *PostgreSQLServer) CreateDatabaseIfNotExists(database string) error {
	if s == nil || s.dbpool == nil {
		return errors.New("dbpool is empty")
	}
	if databaseExists, err := s.doesDatabaseExist(database); err != nil {
		return err
	} else {
		if databaseExists {
			return nil
		}
		if _, err := s.dbpool.Exec(context.Background(), fmt.Sprintf("CREATE DATABASE \"%s\";", database)); err != nil {
			return err
		} else {
			if databaseExistsNow, err := s.doesDatabaseExist(database); err != nil {
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

func (s *PostgreSQLServer) connect() (*pgxpool.Pool, error) {
	dbpool, err := pgxpool.Connect(context.Background(), fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres", s.Host, s.Port, s.RootUser, s.RootPassword))
	if err != nil {
		return nil, err
	}
	return dbpool, nil
}

func (s *PostgreSQLServer) doesDatabaseExist(database string) (bool, error) {
	if s == nil || s.dbpool == nil {
		return false, errors.New("dbpool is empty")
	}
	var result int64
	err := s.dbpool.QueryRow(context.Background(), fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname='%s';", database)).Scan(&result)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return result == 1, nil
}
