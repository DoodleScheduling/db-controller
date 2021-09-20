package db

import (
	"context"
	"errors"
	"fmt"
	"net/url"

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

func NewPostgreSQLRepository(ctx context.Context, opts ConnectionOptions) (Handler, error) {
	popt, err := url.Parse(opts.URI)
	if err != nil {
		return nil, err
	}

	popt.User = url.UserPassword(opts.Username, opts.Password)

	q, _ := url.ParseQuery(popt.RawQuery)
	hasConnectTimeout := false
	for k, _ := range q {
		if k == "connect_timeout" {
			hasConnectTimeout = true
			break
		}
	}

	if hasConnectTimeout == false {
		q.Add("connect_timeout", "2")
	}

	popt.RawQuery = q.Encode()

	if opts.DatabaseName != "" {
		popt.Path = opts.DatabaseName
	}

	dbpool, err := pgxpool.Connect(ctx, popt.String())
	if err != nil {
		return nil, err
	}

	return &PostgreSQLRepository{
		dbpool: dbpool,
	}, nil
}

func (s *PostgreSQLRepository) Close(ctx context.Context) error {
	s.dbpool.Close()
	return nil
}

func (s *PostgreSQLRepository) RestoreDatabaseFrom(ctx context.Context, source ConnectionOptions) error {
	return errors.New("not yet implemented")
}

// TODO Prepared Statements
func (s *PostgreSQLRepository) CreateDatabaseIfNotExists(ctx context.Context, database string) error {
	if databaseExists, err := s.doesDatabaseExist(ctx, database); err != nil {
		return err
	} else {
		if databaseExists {
			return nil
		}
		if _, err := s.dbpool.Exec(ctx, fmt.Sprintf("CREATE DATABASE \"%s\";", database)); err != nil {
			return err
		} else {
			if databaseExistsNow, err := s.doesDatabaseExist(ctx, database); err != nil {
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

func (s *PostgreSQLRepository) SetupUser(ctx context.Context, database string, user string, password string, roles Roles) error {
	if err := s.createUserIfNotExists(ctx, user); err != nil {
		return err
	}
	if err := s.setPasswordForUser(ctx, user, password); err != nil {
		return err
	}
	if err := s.grantAllPrivileges(ctx, database, user); err != nil {
		return err
	}
	return nil
}

func (s *PostgreSQLRepository) DropUser(ctx context.Context, database string, user string) error {
	if err := s.revokeAllPrivileges(ctx, database, user); err != nil {
		return err
	}
	if err := s.dropUserIfNotExist(ctx, user); err != nil {
		return err
	}
	return nil
}

func (s *PostgreSQLRepository) EnableExtension(ctx context.Context, name string) error {
	if extensionExists, err := s.doesExtensionExist(ctx, name); err != nil {
		return err
	} else if !extensionExists {
		return s.createExtension(ctx, name)
	}
	return nil
}

func (s *PostgreSQLRepository) createUserIfNotExists(ctx context.Context, user string) error {
	if userExists, err := s.doesUserExist(ctx, user); err != nil {
		return err
	} else {
		if userExists {
			return nil
		}
		if _, err := s.dbpool.Exec(ctx, fmt.Sprintf("CREATE USER \"%s\";", user)); err != nil {
			return err
		} else {
			if userExistsNow, err := s.doesUserExist(ctx, user); err != nil {
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

func (s *PostgreSQLRepository) createExtension(ctx context.Context, name string) error {
	_, err := s.dbpool.Exec(ctx, fmt.Sprintf("CREATE EXTENSION %s;", name))
	return err
}

func (s *PostgreSQLRepository) dropUserIfNotExist(ctx context.Context, user string) error {
	if userExists, err := s.doesUserExist(ctx, user); err != nil {
		return err
	} else {
		if !userExists {
			return nil
		}
		if _, err := s.dbpool.Exec(ctx, fmt.Sprintf("DROP USER \"%s\";", user)); err != nil {
			return err
		} else {
			if userExistsNow, err := s.doesUserExist(ctx, user); err != nil {
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

func (s *PostgreSQLRepository) setPasswordForUser(ctx context.Context, user string, password string) error {
	if _, err := s.dbpool.Exec(ctx, fmt.Sprintf("ALTER USER \"%s\" WITH ENCRYPTED PASSWORD '%s';", user, password)); err != nil {
		return err
	}
	return nil
}

func (s *PostgreSQLRepository) grantAllPrivileges(ctx context.Context, database string, user string) error {
	if _, err := s.dbpool.Exec(ctx, fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE \"%s\" TO \"%s\";", database, user)); err != nil {
		return err
	}
	return nil
}

func (s *PostgreSQLRepository) revokeAllPrivileges(ctx context.Context, database string, user string) error {
	if _, err := s.dbpool.Exec(ctx, fmt.Sprintf("REVOKE ALL PRIVILEGES ON DATABASE \"%s\" FROM \"%s\";", database, user)); err != nil {
		return err
	}
	return nil
}

func (s *PostgreSQLRepository) doesDatabaseExist(ctx context.Context, database string) (bool, error) {
	var result int64
	err := s.dbpool.QueryRow(ctx, fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname='%s';", database)).Scan(&result)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return result == 1, nil
}

func (s *PostgreSQLRepository) doesUserExist(ctx context.Context, user string) (bool, error) {
	var result int64
	err := s.dbpool.QueryRow(ctx, fmt.Sprintf("SELECT 1 FROM pg_roles WHERE rolname='%s';", user)).Scan(&result)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return result == 1, nil
}

func (s *PostgreSQLRepository) doesExtensionExist(ctx context.Context, name string) (bool, error) {
	var result int64
	err := s.dbpool.QueryRow(ctx, fmt.Sprintf("SELECT 1 from pg_extension where extname='%s';", name)).Scan(&result)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return result == 1, nil
}
