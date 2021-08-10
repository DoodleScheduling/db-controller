package db

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	infrav1beta1 "github.com/doodlescheduling/k8sdb-controller/api/v1beta1"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type PostgreSQLRepository struct {
	dbpool *pgxpool.Pool
}

const (
	DefaultPostgreSQLReadRole      = "read"
	DefaultPostgreSQLReadWriteRole = "readWrite"
)

func NewPostgreSQLRepository(ctx context.Context, uri, database, username, password string) (Handler, error) {
	opt, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	opt.User = url.UserPassword(username, password)

	if database != "" {
		opt.Path = database
	}
	dbpool, err := pgxpool.Connect(context.Background(), opt.String())
	if err != nil {
		return nil, err
	}

	return &PostgreSQLRepository{
		dbpool: dbpool,
	}, nil
}

func (s *PostgreSQLRepository) Close() error {
	s.dbpool.Close()
	return nil
}

// TODO Prepared Statements
func (s *PostgreSQLRepository) CreateDatabaseIfNotExists(database string) error {
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

func (s *PostgreSQLRepository) SetupUser(database string, user string, password string, roles []infrav1beta1.Role) error {
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

func (s *PostgreSQLRepository) DropUser(database string, user string) error {
	if err := s.revokeAllPrivileges(database, user); err != nil {
		return err
	}
	if err := s.dropUserIfNotExist(user); err != nil {
		return err
	}
	return nil
}

func (s *PostgreSQLRepository) EnableExtension(name string) error {
	if extensionExists, err := s.doesExtensionExist(name); err != nil {
		return err
	} else if !extensionExists {
		return s.createExtension(name)
	}
	return nil
}

func (s *PostgreSQLRepository) createUserIfNotExists(user string) error {
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

func (s *PostgreSQLRepository) createExtension(name string) error {
	_, err := s.dbpool.Exec(context.Background(), fmt.Sprintf("CREATE EXTENSION %s;", name))
	return err
}

func (s *PostgreSQLRepository) dropUserIfNotExist(user string) error {
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

func (s *PostgreSQLRepository) setPasswordForUser(user string, password string) error {
	if _, err := s.dbpool.Exec(context.Background(), fmt.Sprintf("ALTER USER \"%s\" WITH ENCRYPTED PASSWORD '%s';", user, password)); err != nil {
		return err
	}
	return nil
}

func (s *PostgreSQLRepository) grantAllPrivileges(database string, user string) error {
	if _, err := s.dbpool.Exec(context.Background(), fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE \"%s\" TO \"%s\";", database, user)); err != nil {
		return err
	}
	return nil
}

func (s *PostgreSQLRepository) revokeAllPrivileges(database string, user string) error {
	if _, err := s.dbpool.Exec(context.Background(), fmt.Sprintf("REVOKE ALL PRIVILEGES ON DATABASE \"%s\" FROM \"%s\";", database, user)); err != nil {
		return err
	}
	return nil
}

func (s *PostgreSQLRepository) doesDatabaseExist(database string) (bool, error) {
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

func (s *PostgreSQLRepository) doesUserExist(user string) (bool, error) {
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

func (s *PostgreSQLRepository) doesExtensionExist(name string) (bool, error) {
	var result int64
	err := s.dbpool.QueryRow(context.Background(), fmt.Sprintf("SELECT 1 from pg_extension where extname='%s';", name)).Scan(&result)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return result == 1, nil
}
