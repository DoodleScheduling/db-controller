package database

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/jackc/pgx/v4"
)

type PostgreSQLOptions struct {
	URI          string
	DatabaseName string
	Username     string
	Password     string
}

type PostgreSQLRepository struct {
	conn *pgx.Conn
}

const (
	DefaultPostgreSQLReadRole      = "read"
	DefaultPostgreSQLReadWriteRole = "readWrite"
)

func NewPostgreSQLRepository(ctx context.Context, opts PostgreSQLOptions) (*PostgreSQLRepository, error) {
	popt, err := url.Parse(opts.URI)
	if err != nil {
		return nil, err
	}

	popt.User = url.UserPassword(opts.Username, opts.Password)

	q, _ := url.ParseQuery(popt.RawQuery)
	hasConnectTimeout := false
	for k := range q {
		if k == "connect_timeout" {
			hasConnectTimeout = true
			break
		}
	}

	if !hasConnectTimeout {
		q.Add("connect_timeout", "2")
	}

	popt.RawQuery = q.Encode()

	if opts.DatabaseName != "" {
		popt.Path = opts.DatabaseName
	}

	conn, err := pgx.Connect(ctx, popt.String())
	if err != nil {
		return nil, err
	}

	return &PostgreSQLRepository{
		conn: conn,
	}, nil
}

func (s *PostgreSQLRepository) Close(ctx context.Context) error {
	if s.conn != nil {
		s.conn.Close(ctx)
	}

	return nil
}

// TODO Prepared Statements
func (s *PostgreSQLRepository) CreateDatabaseIfNotExists(ctx context.Context, database string) error {
	if databaseExists, err := s.doesDatabaseExist(ctx, database); err != nil {
		return err
	} else {
		if databaseExists {
			return nil
		}
		if _, err := s.conn.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s;", (pgx.Identifier{database}).Sanitize())); err != nil {
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

type PostgresqlUser struct {
	Database string
	Username string
	Password string
	Roles    []string
	Grants   []Grant
}

type Grant struct {
	Object     string
	ObjectName string
	User       string
	Privileges []Privilege
}

type Privilege string

var SelectPrivilege Privilege = "SELECT"
var AlPrivilege Privilege = "ALL"

func (s *PostgreSQLRepository) SetupUser(ctx context.Context, user PostgresqlUser) error {
	if err := s.createUserIfNotExists(ctx, user); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	if err := s.setPasswordForUser(ctx, user); err != nil {
		return fmt.Errorf("failed to set password: %w", err)
	}
	if err := s.grantAllPrivileges(ctx, user); err != nil {
		return fmt.Errorf("failed to grant all privileges: %w", err)
	}
	if err := s.grantRoles(ctx, user); err != nil {
		return fmt.Errorf("failed to grant roles: %w", err)
	}
	if err := s.grantRules(ctx, user); err != nil {
		return fmt.Errorf("failed to apply grant rules: %w", err)
	}
	return nil
}

func (s *PostgreSQLRepository) DropUser(ctx context.Context, user PostgresqlUser) error {
	if err := s.RevokeAllPrivileges(ctx, user); err != nil {
		return err
	}
	if err := s.dropUserIfNotExist(ctx, user); err != nil {
		return err
	}
	return nil
}

func (s *PostgreSQLRepository) EnableExtension(ctx context.Context, db, name string) error {
	if extensionExists, err := s.doesExtensionExist(ctx, db, name); err != nil {
		return err
	} else if !extensionExists {
		return s.createExtension(ctx, db, name)
	}
	return nil
}

func (s *PostgreSQLRepository) createUserIfNotExists(ctx context.Context, user PostgresqlUser) error {
	if userExists, err := s.doesUserExist(ctx, user); err != nil {
		return err
	} else {
		if userExists {
			return nil
		}
		if _, err := s.conn.Exec(ctx, fmt.Sprintf("CREATE USER %s;", (pgx.Identifier{user.Username}).Sanitize())); err != nil {
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

func (s *PostgreSQLRepository) createExtension(ctx context.Context, db, name string) error {
	_, err := s.conn.Exec(ctx, fmt.Sprintf("CREATE EXTENSION %s;", (pgx.Identifier{name}).Sanitize()))
	return err
}

func (s *PostgreSQLRepository) dropUserIfNotExist(ctx context.Context, user PostgresqlUser) error {
	if userExists, err := s.doesUserExist(ctx, user); err != nil {
		return err
	} else {
		if !userExists {
			return nil
		}
		if _, err := s.conn.Exec(ctx, fmt.Sprintf("DROP USER %s;", (pgx.Identifier{user.Username}).Sanitize())); err != nil {
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

func (s *PostgreSQLRepository) setPasswordForUser(ctx context.Context, user PostgresqlUser) error {
	password, err := s.conn.PgConn().EscapeString(user.Password)
	if err != nil {
		return err
	}

	_, err = s.conn.Exec(ctx, fmt.Sprintf("ALTER USER %s WITH ENCRYPTED PASSWORD '%s';", (pgx.Identifier{user.Username}).Sanitize(), password))
	return err
}

func (s *PostgreSQLRepository) grantAllPrivileges(ctx context.Context, user PostgresqlUser) error {
	_, err := s.conn.Exec(ctx, fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s;", (pgx.Identifier{user.Database}).Sanitize(), (pgx.Identifier{user.Username}).Sanitize()))
	return err
}

func (s *PostgreSQLRepository) grantRoles(ctx context.Context, user PostgresqlUser) error {
	for _, role := range user.Roles {
		_, err := s.conn.Exec(ctx, fmt.Sprintf("GRANT %s TO %s;", (pgx.Identifier{role}).Sanitize(), (pgx.Identifier{user.Username}).Sanitize()))
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *PostgreSQLRepository) grantRules(ctx context.Context, user PostgresqlUser) error {
	for _, grant := range user.Grants {
		for _, p := range grant.Privileges {
			_, err := s.conn.Exec(ctx, fmt.Sprintf("GRANT %s ON %s %s TO %s;", string(p), grant.Object, (pgx.Identifier{grant.ObjectName}).Sanitize(), (pgx.Identifier{user.Username}).Sanitize()))
			if err != nil {
				return err
			}
		}

	}

	return nil
}

func (s *PostgreSQLRepository) RevokeAllPrivileges(ctx context.Context, user PostgresqlUser) error {
	_, err := s.conn.Exec(ctx, fmt.Sprintf("REVOKE ALL PRIVILEGES ON DATABASE %s FROM %s;", (pgx.Identifier{user.Database}).Sanitize(), (pgx.Identifier{user.Username}).Sanitize()))
	return err
}

func (s *PostgreSQLRepository) doesDatabaseExist(ctx context.Context, database string) (bool, error) {
	database, err := s.conn.PgConn().EscapeString(database)
	if err != nil {
		return false, err
	}

	var result int64
	err = s.conn.QueryRow(ctx, fmt.Sprintf("SELECT 1 FROM pg_database WHERE datname='%s';", database)).Scan(&result)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return result == 1, nil
}

func (s *PostgreSQLRepository) doesUserExist(ctx context.Context, user PostgresqlUser) (bool, error) {
	username, err := s.conn.PgConn().EscapeString(user.Username)
	if err != nil {
		return false, err
	}

	var result int64
	err = s.conn.QueryRow(ctx, fmt.Sprintf("SELECT 1 FROM pg_roles WHERE rolname='%s';", username)).Scan(&result)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return result == 1, nil
}

func (s *PostgreSQLRepository) doesExtensionExist(ctx context.Context, db, name string) (bool, error) {
	name, err := s.conn.PgConn().EscapeString(name)
	if err != nil {
		return false, err
	}

	var result int64
	err = s.conn.QueryRow(ctx, fmt.Sprintf("SELECT 1 from pg_extension where extname='%s';", name)).Scan(&result)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return result == 1, nil
}
