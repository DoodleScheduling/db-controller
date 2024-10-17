package database

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type MongoDBOptions struct {
	URI              string
	DatabaseName     string
	AuthDatabaseName string
	Username         string
	Password         string
}

type MongoDBRoles []MongoDBRole
type MongoDBRole struct {
	Name string `json:"role" bson:"role"`
	DB   string `json:"db" bson:"db"`
}

type MongoDBUsers []MongoDBUser
type MongoDBUser struct {
	User  string       `json:"user" bson:"user"`
	DB    string       `json:"db" bson:"db"`
	Roles MongoDBRoles `json:"roles" bson:"roles"`
}

const (
	adminDatabase   = "admin"
	usersCollection = "system.users"
)

type MongoDBRepository struct {
	client *mongo.Client
	opts   MongoDBOptions
}

func NewMongoDBRepository(ctx context.Context, opts MongoDBOptions) (*MongoDBRepository, error) {
	o := options.Client()
	o.ApplyURI(opts.URI)

	o.SetAuth(options.Credential{
		Username: opts.Username,
		Password: opts.Password,
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
		opts:   opts,
	}, nil
}

func (m *MongoDBRepository) Close(ctx context.Context) error {
	if m.client != nil {
		return m.client.Disconnect(ctx)
	}

	return nil
}

func (m *MongoDBRepository) SetupUser(ctx context.Context, database string, username string, password string, roles MongoDBRoles) error {
	doesUserExist, err := m.doesUserExist(ctx, database, username)
	if err != nil {
		return err
	}

	if !doesUserExist {
		if err := m.createUser(ctx, database, username, password, roles); err != nil {
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

func (m *MongoDBRepository) DropUser(ctx context.Context, database string, username string) error {
	command := &bson.D{primitive.E{Key: "dropUser", Value: username}}
	r := m.runCommand(ctx, database, command)
	if _, err := r.Raw(); err != nil {
		return err
	}
	return nil
}

func (m *MongoDBRepository) doesUserExist(ctx context.Context, database string, username string) (bool, error) {
	users, err := m.getAllUsers(ctx, database, username)
	if err != nil {
		return false, err
	}

	return len(users) > 0, nil
}

func (m *MongoDBRepository) getAllUsers(ctx context.Context, database string, username string) (MongoDBUsers, error) {
	users := make(MongoDBUsers, 0)

	collection := m.client.Database(adminDatabase).Collection(usersCollection)
	cursor, err := collection.Find(ctx, bson.D{primitive.E{Key: "user", Value: username}, primitive.E{Key: "db", Value: database}})
	if err != nil {
		return users, err
	}

	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var user MongoDBUser
		if err := cursor.Decode(&user); err != nil {
			return users, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (m *MongoDBRepository) getRoles(database string, roles MongoDBRoles) []bson.M {
	// by default, assign readWrite role (backward compatibility)
	if len(roles) == 0 {
		return []bson.M{{
			"role": "readWrite",
			"db":   database,
		}}
	}
	rs := make([]bson.M, 0)
	for _, r := range roles {
		db := r.DB
		if db == "" {
			db = database
		}

		rs = append(rs, bson.M{
			"role": r.Name,
			"db":   db,
		})
	}
	return rs
}

func (m *MongoDBRepository) createUser(ctx context.Context, database string, username string, password string, roles MongoDBRoles) error {
	command := &bson.D{primitive.E{Key: "createUser", Value: username}, primitive.E{Key: "pwd", Value: password},
		primitive.E{Key: "roles", Value: m.getRoles(database, roles)}}
	r := m.runCommand(ctx, database, command)
	if _, err := r.Raw(); err != nil {
		return err
	}
	return nil
}

func (m *MongoDBRepository) updateUserPasswordAndRoles(ctx context.Context, database string, username string, password string, roles MongoDBRoles) error {
	command := &bson.D{primitive.E{Key: "updateUser", Value: username}, primitive.E{Key: "pwd", Value: password},
		primitive.E{Key: "roles", Value: m.getRoles(database, roles)}}
	r := m.runCommand(ctx, database, command)
	if _, err := r.Raw(); err != nil {
		return err
	}
	return nil
}

func (m *MongoDBRepository) runCommand(ctx context.Context, database string, command *bson.D) *mongo.SingleResult {
	return m.client.Database(database).RunCommand(ctx, *command)
}
