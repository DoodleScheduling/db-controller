package database

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/mongodb/mongo-tools/common/db"
	toolsoptions "github.com/mongodb/mongo-tools/common/options"
	"github.com/mongodb/mongo-tools/mongodump"
	"github.com/mongodb/mongo-tools/mongorestore"
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
	o.SetConnectTimeout(time.Duration(2) * time.Second)
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

type mdump struct {
	opts         *toolsoptions.ToolOptions
	conns        int
	databaseName string
}

func toMongoToolsOpts(opts *MongoDBOptions) *toolsoptions.ToolOptions {
	return &toolsoptions.ToolOptions{
		URI: &toolsoptions.URI{ConnectionString: opts.URI},
		Auth: &toolsoptions.Auth{
			Username: opts.Username,
			Password: opts.Password,
			Source:   "admin",
		},
		Direct:     true,
		Connection: &toolsoptions.Connection{},
		Namespace:  &toolsoptions.Namespace{},
	}
}

func (m *MongoDBRepository) DatabaseExists(ctx context.Context, name string) (bool, error) {
	dbs, err := m.client.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		return false, err
	}

	for _, v := range dbs {
		if v == name {
			return true, nil
		}
	}

	return false, nil
}

// Restore database from another database
func (m *MongoDBRepository) RestoreDatabaseFrom(ctx context.Context, src MongoDBOptions) error {
	srcOpts := toMongoToolsOpts(&src)
	srcOpts.Namespace.DB = src.DatabaseName

	dumper := &mdump{
		databaseName: src.DatabaseName,
		opts:         srcOpts,
	}

	//Namespace options for mongorestore to actually rename the database on the fly (regex to match all collecitions within the database)
	nsopts := &mongorestore.NSOptions{
		NSInclude: []string{fmt.Sprintf("%s.*", src.DatabaseName)},
		NSFrom:    []string{fmt.Sprintf("%s.*", src.DatabaseName)},
		NSTo:      []string{fmt.Sprintf("%s.*", m.opts.DatabaseName)},
	}

	dstOpts := toMongoToolsOpts(&m.opts)
	return pipeMongoDBDump(ctx, dumper, dstOpts, nsopts)
}

// Write always return 0 as written bytes. Needed to satisfy io.WriteTo
func (d *mdump) WriteTo(w io.Writer) (int64, error) {
	//pm := progress.NewBarWriter(&progressWriter{}, time.Second*60, 24, false)
	mdump := mongodump.MongoDump{
		ToolOptions: d.opts,
		OutputOptions: &mongodump.OutputOptions{
			// Archive = "-" means, for mongodump, use the provided Writer
			// instead of creating a file. This is not clear at plain sight,
			// you nee to look the code to discover it.
			Archive:                "-",
			NumParallelCollections: 1,
		},
		InputOptions:    &mongodump.InputOptions{},
		SessionProvider: &db.SessionProvider{},
		OutputWriter:    w,
		//ProgressManager: pm,
	}

	err := mdump.Init()
	if err != nil {
		return 0, fmt.Errorf("failed initialize monogdump: %w", err)
	}

	/*pm.Start()
	defer pm.Stop()*/

	err = mdump.Dump()

	if err != nil {
		return 0, fmt.Errorf("failed to dump database: %w", err)
	}

	return 0, nil
}

// Upload writes data to dst from given src and returns an amount of written bytes
func pipeMongoDBDump(ctx context.Context, src io.WriterTo, opts *toolsoptions.ToolOptions, nsopts *mongorestore.NSOptions) error {
	rsession, err := db.NewSessionProvider(*opts)
	if err != nil {
		return fmt.Errorf("failed create session for mongorestore: %w", err)
	}

	r, w := io.Pipe()

	restoreOpts := mongorestore.Options{
		ToolOptions: opts,
		InputOptions: &mongorestore.InputOptions{
			Archive: "-",
		},
		OutputOptions: &mongorestore.OutputOptions{
			BypassDocumentValidation: true,
			//Drop:                     true,
			NumParallelCollections: 1,
			//There will be 0 documents processed if not at least set to 1
			NumInsertionWorkers: 1,
			StopOnError:         true,
			BulkBufferSize:      1000,
			WriteConcern:        "majority",
		},
		NSOptions: nsopts,
	}

	mr, err := mongorestore.New(restoreOpts)

	if err != nil {
		return fmt.Errorf("failed to initialize mongorestore: %w", err)
	}

	mr.InputReader = r
	mr.SessionProvider = rsession

	dumpDone := make(chan error)
	go func() {
		_, err := src.WriteTo(w)
		w.Close()

		dumpDone <- err
	}()

	restoreDone := make(chan error)
	go func() {
		rdumpResult := mr.Restore()
		mr.Close()

		if rdumpResult.Err != nil {
			restoreDone <- fmt.Errorf("failed to restore mongo dump (successes: %d / fails: %d): %w", rdumpResult.Successes, rdumpResult.Failures, rdumpResult.Err)
		}

		restoreDone <- nil
	}()

	select {
	case <-ctx.Done():
		// return if passed context was closed
		return closePipe(r, w, errors.New("context closed"))
	case err = <-dumpDone:
		// return error if mongodump fails
		if err != nil {
			return closePipe(r, w, err)
		}
	case err = <-restoreDone:
		// immediately return result of mongorestore, we don't need to wait for mongodump since it either finished or failed before
		return closePipe(r, w, err)
	}

	return nil
}

func closePipe(r, w io.Closer, err error) error {
	r.Close()
	w.Close()
	return err
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
	if _, err := r.DecodeBytes(); err != nil {
		return err
	}
	return nil
}

func (m *MongoDBRepository) doesUserExist(ctx context.Context, database string, username string) (bool, error) {
	users, err := m.getAllUsers(ctx, database, username)
	if err != nil {
		return false, err
	}

	return users != nil && len(users) > 0, nil
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
	if _, err := r.DecodeBytes(); err != nil {
		return err
	}
	return nil
}

func (m *MongoDBRepository) updateUserPasswordAndRoles(ctx context.Context, database string, username string, password string, roles MongoDBRoles) error {
	command := &bson.D{primitive.E{Key: "updateUser", Value: username}, primitive.E{Key: "pwd", Value: password},
		primitive.E{Key: "roles", Value: m.getRoles(database, roles)}}
	r := m.runCommand(ctx, database, command)
	if _, err := r.DecodeBytes(); err != nil {
		return err
	}
	return nil
}

func (m *MongoDBRepository) runCommand(ctx context.Context, database string, command *bson.D) *mongo.SingleResult {
	return m.client.Database(database).RunCommand(ctx, *command)
}
