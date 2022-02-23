package database

import (
	"context"
	"errors"

	"github.com/mongodb-forks/digest"
	"go.mongodb.org/atlas/mongodbatlas"
)

type AtlasOptions struct {
	GroupID    string
	PublicKey  string
	PrivateKey string
}

type AtlasRepository struct {
	atlas   *mongodbatlas.Client
	groupId string
}

func NewAtlasRepository(ctx context.Context, opts AtlasOptions) (*AtlasRepository, error) {
	t := digest.NewTransport(opts.PublicKey, opts.PrivateKey)
	tc, err := t.Client()
	if err != nil {
		return nil, err
	}

	return &AtlasRepository{
		groupId: opts.GroupID,
		atlas:   mongodbatlas.NewClient(tc),
	}, nil
}

func (m *AtlasRepository) Close(ctx context.Context) error {
	return nil
}

func (m *AtlasRepository) SetupUser(ctx context.Context, database string, username string, password string, roles MongoDBRoles) error {
	doesUserExist, err := m.doesUserExist(ctx, database, username)
	if err != nil {
		return err
	}

	if !doesUserExist {
		if err := m.createUser(context.Background(), database, username, password, roles); err != nil {
			return err
		}
		if doesUserExistNow, err := m.doesUserExist(ctx, database, username); err != nil {
			return err
		} else if !doesUserExistNow {
			return errors.New("user doesn't exist after create")
		}
	} else {
		if err := m.updateUserPasswordAndRoles(ctx, database, username, password, roles); err != nil {
			return err
		}
	}

	return nil
}

func (m *AtlasRepository) DropUser(ctx context.Context, database string, username string) error {
	_, err := m.atlas.DatabaseUsers.Delete(ctx, database, m.groupId, username)
	return err
}

func (m *AtlasRepository) doesUserExist(ctx context.Context, database string, username string) (bool, error) {
	_, _, err := m.atlas.DatabaseUsers.Get(ctx, database, m.groupId, username)
	if err != nil {
		return false, nil
	}

	return true, err
}

func (m *AtlasRepository) getRoles(database string, roles MongoDBRoles) []mongodbatlas.Role {
	// by default, assign readWrite role (backward compatibility)
	if len(roles) == 0 {
		return []mongodbatlas.Role{{
			RoleName:     "readWrite",
			DatabaseName: database,
		}}
	}

	rs := make([]mongodbatlas.Role, 0)
	for _, r := range roles {
		db := r.DB
		if db == "" {
			db = database
		}

		rs = append(rs, mongodbatlas.Role{
			RoleName:     r.Name,
			DatabaseName: db,
		})
	}

	return rs
}

func (m *AtlasRepository) createUser(ctx context.Context, database string, username string, password string, roles MongoDBRoles) error {
	user := &mongodbatlas.DatabaseUser{
		Username:     username,
		Password:     password,
		DatabaseName: database,
		Roles:        m.getRoles(database, roles),
	}

	_, _, err := m.atlas.DatabaseUsers.Create(ctx, m.groupId, user)
	return err
}

func (m *AtlasRepository) updateUserPasswordAndRoles(ctx context.Context, database string, username string, password string, roles MongoDBRoles) error {
	user := &mongodbatlas.DatabaseUser{
		Username: username,
		Password: password,
		Roles:    m.getRoles(database, roles),
	}

	_, _, err := m.atlas.DatabaseUsers.Update(ctx, m.groupId, username, user)
	return err
}
