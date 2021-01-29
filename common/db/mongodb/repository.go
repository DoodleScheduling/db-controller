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
	client *mongo.Client
	Host   string
	Port   string
}

func NewMongoDBServer(host string, port string, rootUser string, rootPassword string, authenticationDatabase string) (*MongoDBServer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	o := options.Client().ApplyURI(fmt.Sprintf("mongodb://%s:%s", host, port))
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
		client: client,
		Host:   host,
		Port:   port,
	}, nil
}

func (m *MongoDBServer) SetupUser() {

}

func (m *MongoDBServer) DoesUserExist(database string, username string) (*Users, error) {
	command := &bson.D{primitive.E{Key: "usersInfo", Value: username}}
	r := m.runCommand(database, command)
	if err := r.Err(); err != nil {
		return nil, err
	}
	var user Users
	if err := r.Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
	//if br, err := r.DecodeBytes(); err != nil {
	//	return "", err
	//} else {
	//	return br.String(), nil
	//}
}

func (m *MongoDBServer) runCommand(database string, command *bson.D) *mongo.SingleResult {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return m.client.Database(database).RunCommand(ctx, *command)
}
