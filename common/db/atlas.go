package db

import (
	"context"
	"errors"

	"github.com/mongodb-forks/digest"
	"go.mongodb.org/atlas/mongodbatlas"

	infrav1beta1 "github.com/doodlescheduling/k8sdb-controller/api/v1beta1"
)

type AtlasRepository struct {
	atlas   *mongodbatlas.Client
	groupId string
}

func NewAtlasRepository(ctx context.Context, groupId, publicKey, privateKey string) (Handler, error) {
	t := digest.NewTransport(publicKey, privateKey)
	tc, err := t.Client()
	if err != nil {
		return nil, err
	}

	return &AtlasRepository{
		groupId: groupId,
		atlas:   mongodbatlas.NewClient(tc),
	}, nil
}

func (m *AtlasRepository) Close(ctx context.Context) error {
	return nil
}

// CreateDatabaseIfNotExists is a dummy to apply to fulfill the contract,
// we don't need to create the database on Atlas
func (m *AtlasRepository) CreateDatabaseIfNotExists(ctx context.Context, database string) error {
	return nil
}

func (m *AtlasRepository) SetupUser(ctx context.Context, database string, username string, password string, roles []infrav1beta1.Role) error {
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
	//not implemented
	return errors.New("not yet supported")
}

func (m *AtlasRepository) EnableExtension(ctx context.Context, name string) error {
	// NOOP
	return nil
}

func (m *AtlasRepository) doesUserExist(ctx context.Context, database string, username string) (bool, error) {
	user, _, err := m.atlas.DatabaseUsers.Get(ctx, database, m.groupId, username)
	if user == nil {
		return false, err
	}

	return true, err
}

func (m *AtlasRepository) getRoles(database string, roles []infrav1beta1.Role) []mongodbatlas.Role {
	// by default, assign readWrite role (backward compatibility)
	if len(roles) == 0 {
		return []mongodbatlas.Role{{
			RoleName:     "readWrite",
			DatabaseName: database,
		}}
	}

	rs := make([]mongodbatlas.Role, 0)
	for _, r := range roles {
		db := r.Db
		if db == nil {
			db = &database
		}

		rs = append(rs, mongodbatlas.Role{
			RoleName:     r.Name,
			DatabaseName: *db,
		})
	}

	return rs
}

func (m *AtlasRepository) createUser(ctx context.Context, database string, username string, password string, roles []infrav1beta1.Role) error {
	user := &mongodbatlas.DatabaseUser{
		Username: username,
		Password: password,
		Roles:    m.getRoles(database, roles),
	}

	_, _, err := m.atlas.DatabaseUsers.Create(ctx, m.groupId, user)
	if err != nil {
		return err
	}

	return nil
}

func (m *AtlasRepository) updateUserPasswordAndRoles(ctx context.Context, database string, username string, password string, roles []infrav1beta1.Role) error {
	user := &mongodbatlas.DatabaseUser{
		Username: username,
		Password: password,
		Roles:    m.getRoles(database, roles),
	}

	_, _, err := m.atlas.DatabaseUsers.Update(ctx, m.groupId, username, user)
	if err != nil {
		return err
	}

	return nil
}
