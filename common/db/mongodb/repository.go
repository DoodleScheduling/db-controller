package mongodb

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"time"
)

type Roles []Role
type Role struct {
	Role string `json:"role" bson:"role"`
	DB   string `json:"db" bson:"db"`
}

type Users struct {
	Users []User `json:"users" bson:"users"`
}
type User struct {
	User  string `json:"user" bson:"user"`
	DB    string `json:"db" bson:"db"`
	Roles Roles  `json:"roles" bson:"roles"`
}

type UserHolder struct {
	User Users `json:"user" bson:"user"`
}

type MongoDBServer struct {
	client                 *mongo.Client
	uri                    string
	authenticationDatabase string
}

func NewMongoDBServer(uri string, rootUser string, rootPassword string, authenticationDatabase string) (*MongoDBServer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	o := options.Client().ApplyURI(fmt.Sprintf("mongodb://%s", uri))
	o.SetMaxPoolSize(100)
	o.SetAuth(options.Credential{
		AuthMechanism: "SCRAM-SHA-1",
		AuthSource:    authenticationDatabase,
		Username:      rootUser,
		Password:      rootPassword,
	})
	client, err := mongo.Connect(ctx, o)
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, err
	}
	return &MongoDBServer{
		client:                 client,
		uri:                    uri,
		authenticationDatabase: authenticationDatabase,
	}, nil
}

func (m *MongoDBServer) SetupUser(database string, username string, password string) (string, error) {
	doesUserExist, err := m.doesUserExist(database, username)
	if err != nil {
		return "", err
	}
	if !doesUserExist {
		return m.createUser(database, username, password)
	}
	return "user already exists", nil
}

func (m *MongoDBServer) doesUserExist(database string, username string) (bool, error) {
	command := &bson.D{primitive.E{Key: "usersInfo", Value: username}}
	r := m.runCommand(database, command)
	if err := r.Err(); err != nil {
		return false, err
	}
	var user Users
	if err := r.Decode(&user); err != nil {
		return false, err
	}
	if user.Users == nil || len(user.Users) == 0 {
		return false, nil
	}
	return true, nil
	//if br, err := r.DecodeBytes(); err != nil {
	//	return "", err
	//} else {
	//	return br.String(), nil
	//}
}

func (m *MongoDBServer) createUser(database string, username string, password string) (string, error) {
	//command := &bson.D{{"createUser", username}, {"pwd", password}, {"roles", []bson.M{{"role": "readWrite", "db": database}}}}
	//r := m.runCommand(m.authenticationDatabase, command)
	//if br, err := r.DecodeBytes(); err != nil {
	//	return "", err
	//} else {
	//	return br.String(), nil
	//}
	return "", nil
}

func (m *MongoDBServer) runCommand(database string, command *bson.D) *mongo.SingleResult {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return m.client.Database(database).RunCommand(ctx, *command)
}
