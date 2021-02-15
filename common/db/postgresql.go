package db

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type PostgreSQLServer struct {
	dbpool *pgxpool.Pool
}

func NewPostgreSQLServer(ctx context.Context, uri, username, password string) (Handler, error) {
	opt, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	opt.User = url.UserPassword(username, password)
	dbpool, err := pgxpool.Connect(context.Background(), opt.String())
	if err != nil {
		return nil, err

	}

	return &PostgreSQLServer{
		dbpool: dbpool,
	}, nil
}

func (s *PostgreSQLServer) Close() error {
	s.dbpool.Close()
	return nil
}

// TODO Prepared Statements
func (s *PostgreSQLServer) CreateDatabaseIfNotExists(database string) error {
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

func (s *PostgreSQLServer) SetupUser(database string, user string, password string) error {
	if err := s.createUserIfNotExists(user); err != nil {
		return err
	}
	if err := s.setPasswordForUser(user, password); err != nil {
		return err
	}
	if err := s.grantAllPrivileges(database, user); err != nil {
		return err
	}
	return nil
}

func (s *PostgreSQLServer) DropUser(database string, user string) error {
	if err := s.revokeAllPrivileges(database, user); err != nil {
		return err
	}
	if err := s.dropUserIfNotExist(user); err != nil {
		return err
	}
	return nil
}

func (s *PostgreSQLServer) createUserIfNotExists(user string) error {
	if userExists, err := s.doesUserExist(user); err != nil {
		return err
	} else {
		if userExists {
			return nil
		}
		if _, err := s.dbpool.Exec(context.Background(), fmt.Sprintf("CREATE USER \"%s\";", user)); err != nil {
			return err
		} else {
			if userExistsNow, err := s.doesUserExist(user); err != nil {
				return err
			} else {
				if userExistsNow {
					return nil
				} else {
					return errors.New("user doesn't exist after create")
				}
			}
		}
	}
}

func (s *PostgreSQLServer) dropUserIfNotExist(user string) error {
	if userExists, err := s.doesUserExist(user); err != nil {
		return err
	} else {
		if !userExists {
			return nil
		}
		if _, err := s.dbpool.Exec(context.Background(), fmt.Sprintf("DROP USER \"%s\";", user)); err != nil {
			return err
		} else {
			if userExistsNow, err := s.doesUserExist(user); err != nil {
				return err
			} else {
				if !userExistsNow {
					return nil
				} else {
					return errors.New("user still exists after drop")
				}
			}
		}
	}
}

func (s *PostgreSQLServer) setPasswordForUser(user string, password string) error {
	if _, err := s.dbpool.Exec(context.Background(), fmt.Sprintf("ALTER USER \"%s\" WITH ENCRYPTED PASSWORD '%s';", user, password)); err != nil {
		return err
	}
	return nil
}

func (s *PostgreSQLServer) grantAllPrivileges(database string, user string) error {
	if _, err := s.dbpool.Exec(context.Background(), fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE \"%s\" TO \"%s\";", database, user)); err != nil {
		return err
	}
	return nil
}

func (s *PostgreSQLServer) revokeAllPrivileges(database string, user string) error {
	if _, err := s.dbpool.Exec(context.Background(), fmt.Sprintf("REVOKE ALL PRIVILEGES ON DATABASE \"%s\" FROM \"%s\";", database, user)); err != nil {
		return err
	}
	return nil
}

func (s *PostgreSQLServer) doesDatabaseExist(database string) (bool, error) {
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

func (s *PostgreSQLServer) doesUserExist(user string) (bool, error) {
	var result int64
	err := s.dbpool.QueryRow(context.Background(), fmt.Sprintf("SELECT 1 FROM pg_roles WHERE rolname='%s';", user)).Scan(&result)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return result == 1, nil
}
