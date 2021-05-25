package db

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const (
	adminDatabase   = "admin"
	usersCollection = "system.users"
)

type Roles []Role
type Role struct {
	Role string `json:"role" bson:"role"`
	DB   string `json:"db" bson:"db"`
}

type Users []User
type User struct {
	User  string `json:"user" bson:"user"`
	DB    string `json:"db" bson:"db"`
	Roles Roles  `json:"roles" bson:"roles"`
}

type MongoDBRepository struct {
	client *mongo.Client
}

func NewMongoDBRepository(ctx context.Context, uri, database, username, password string) (Handler, error) {
	o := options.Client().ApplyURI(uri)
	o.SetAuth(options.Credential{
		Username: username,
		Password: password,
	})

	client, err := mongo.Connect(ctx, o)
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, err
	}

	return &MongoDBRepository{
		client: client,
	}, nil
}

func (m *MongoDBRepository) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return m.client.Disconnect(ctx)
}

// CreateDatabaseIfNotExists is a dummy to apply to fulfill the contract,
// we don't need to create the database on MongoDB
func (m *MongoDBRepository) CreateDatabaseIfNotExists(database string) error {
	return nil
}

func (m *MongoDBRepository) SetupUser(database string, username string, password string, roles []string) error {
	doesUserExist, err := m.doesUserExist(database, username)
	if err != nil {
		return err
	}

	if !doesUserExist {
		if err := m.createUser(database, username, password, roles); err != nil {
			return err
		}
		if doesUserExistNow, err := m.doesUserExist(database, username); err != nil {
			return err
		} else if !doesUserExistNow {
			return errors.New("user doesn't exist after create")
		}
	} else {
		if err := m.updateUserPasswordAndRoles(database, username, password, roles); err != nil {
			return err
		}
	}

	return nil
}

func (m *MongoDBRepository) DropUser(database string, username string) error {
	command := &bson.D{primitive.E{Key: "dropUser", Value: username}}
	r := m.runCommand(database, command)
	if _, err := r.DecodeBytes(); err != nil {
		return err
	}
	return nil
}

func (m *MongoDBRepository) EnableExtension(name string) error {
	// NOOP
	return nil
}

func (m *MongoDBRepository) doesUserExist(database string, username string) (bool, error) {
	users, err := m.getAllUsers(database, username)
	if err != nil {
		return false, err
	}

	return users != nil && len(users) > 0, nil
}

func (m *MongoDBRepository) getAllUsers(database string, username string) (Users, error) {
	users := make(Users, 0)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	collection := m.client.Database(adminDatabase).Collection(usersCollection)
	cursor, err := collection.Find(ctx, bson.D{primitive.E{Key: "user", Value: username}, primitive.E{Key: "db", Value: database}})
	if err != nil {
		return users, err
	}

	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var user User
		if err := cursor.Decode(&user); err != nil {
			return users, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (m *MongoDBRepository) getRoles(database string, roles []string) []bson.M {
	// by default, assign readWrite role (backward compatibility)
	if roles == nil || len(roles) == 0 {
		return []bson.M{{
			"role": "readWrite",
			"db":   database,
		}}
	}
	rs := make([]bson.M, 0)
	for _, r := range roles {
		rs = append(rs, bson.M{
			"role": r,
			"db":   database,
		})
	}
	return rs
}

func (m *MongoDBRepository) createUser(database string, username string, password string, roles []string) error {
	command := &bson.D{primitive.E{Key: "createUser", Value: username}, primitive.E{Key: "pwd", Value: password},
		primitive.E{Key: "roles", Value: m.getRoles(database, roles)}}
	r := m.runCommand(database, command)
	if _, err := r.DecodeBytes(); err != nil {
		return err
	}
	return nil
}

func (m *MongoDBRepository) updateUserPasswordAndRoles(database string, username string, password string, roles []string) error {
	command := &bson.D{primitive.E{Key: "updateUser", Value: username}, primitive.E{Key: "pwd", Value: password},
		primitive.E{Key: "roles", Value: m.getRoles(database, roles)}}
	r := m.runCommand(database, command)
	if _, err := r.DecodeBytes(); err != nil {
		return err
	}
	return nil
}

func (m *MongoDBRepository) runCommand(database string, command *bson.D) *mongo.SingleResult {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return m.client.Database(database).RunCommand(ctx, *command)
}
