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

type MongoDBServer struct {
	client *mongo.Client
}

func NewMongoDBServer(ctx context.Context, uri, username, password string) (Handler, error) {
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

	return &MongoDBServer{
		client: client,
	}, nil
}

func (m *MongoDBServer) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return m.client.Disconnect(ctx)
}

// CreateDatabaseIfNotExists is a dummy to apply to fullfill the contract,
// we don't need to create the database on MongoDB
func (m *MongoDBServer) CreateDatabaseIfNotExists(database string) error {
	return nil
}

func (m *MongoDBServer) SetupUser(database string, username string, password string) error {
	doesUserExist, err := m.doesUserExist(database, username)
	if err != nil {
		return err
	}

	if !doesUserExist {
		if err := m.createUser(database, username, password); err != nil {
			return err
		}
		if doesUserExistNow, err := m.doesUserExist(database, username); err != nil {
			return err
		} else if !doesUserExistNow {
			return errors.New("user doesn't exist after create")
		}
	} else {
		if err := m.updateUserPasswordAndRoles(database, username, password); err != nil {
			return err
		}
	}

	return nil
}

func (m *MongoDBServer) DropUser(database string, username string) error {
	command := &bson.D{primitive.E{Key: "dropUser", Value: username}}
	r := m.runCommand(database, command)
	if _, err := r.DecodeBytes(); err != nil {
		return err
	}
	return nil
}

func (m *MongoDBServer) EnableExtension(name string) error {
	// NOOP
	return nil
}

func (m *MongoDBServer) doesUserExist(database string, username string) (bool, error) {
	users, err := m.getAllUsers(database, username)
	if err != nil {
		return false, err
	}

	return users != nil && len(users) > 0, nil
}

func (m *MongoDBServer) getAllUsers(database string, username string) (Users, error) {
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

func (m *MongoDBServer) createUser(database string, username string, password string) error {
	command := &bson.D{primitive.E{Key: "createUser", Value: username}, primitive.E{Key: "pwd", Value: password},
		primitive.E{Key: "roles", Value: []bson.M{{"role": "readWrite", "db": database}}}}
	r := m.runCommand(database, command)
	if _, err := r.DecodeBytes(); err != nil {
		return err
	}
	return nil
}

func (m *MongoDBServer) updateUserPasswordAndRoles(database string, username string, password string) error {
	command := &bson.D{primitive.E{Key: "updateUser", Value: username}, primitive.E{Key: "pwd", Value: password},
		primitive.E{Key: "roles", Value: []bson.M{{"role": "readWrite", "db": database}}}}
	r := m.runCommand(database, command)
	if _, err := r.DecodeBytes(); err != nil {
		return err
	}
	return nil
}

func (m *MongoDBServer) runCommand(database string, command *bson.D) *mongo.SingleResult {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return m.client.Database(database).RunCommand(ctx, *command)
}
