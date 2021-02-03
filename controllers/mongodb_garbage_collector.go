package controllers

import (
	infrav1beta1 "github.com/doodlescheduling/kubedb/api/v1beta1"
	"github.com/doodlescheduling/kubedb/common/db/mongodb"
	"github.com/go-logr/logr"
)

type MongoDBGarbageCollector struct {
	r   *MongoDBReconciler
	cw  *ControllerWrapper
	log *logr.Logger
}

func NewMongoDBGarbageCollector(r *MongoDBReconciler, cw *ControllerWrapper, log *logr.Logger) *MongoDBGarbageCollector {
	return &MongoDBGarbageCollector{
		r:   r,
		cw:  cw,
		log: log,
	}
}

/*	For now, Garbage Collector does not drop databases, because we haven't decided if we want to delete all data.
	Possible avenue to proceed is to have a separate option flag/struct (to be set in Spec), that will force deletion of all garbage, including data.
*/
func (g *MongoDBGarbageCollector) Clean(mongodb *infrav1beta1.MongoDB) error {
	rootPassword, err := g.cw.GetRootPassword(mongodb.Status.DatabaseStatus.RootSecretLookup.Name, mongodb.Status.DatabaseStatus.RootSecretLookup.Namespace,
		mongodb.Status.DatabaseStatus.RootSecretLookup.Field)
	if err != nil {
		// no point in proceeding. In future could also try with Spec credential
		return err
	}
	mongoDBServer, err := g.r.ServerCache.Get(mongodb.Status.DatabaseStatus.Host, mongodb.Status.DatabaseStatus.RootUsername, rootPassword,
		mongodb.Status.DatabaseStatus.RootAuthenticationDatabase)
	if err != nil {
		// no point in proceeding. In future could also try with Spec credential
		return err
	}
	// if an error happens, just collect it and try to clean as much as possible. Return it at the end. This is a pattern in for most of this Garbage Collector.
	var errToReturn error
	errToReturn = g.HandleHostOrDatabaseChange(mongoDBServer, mongodb)
	errToReturn = g.HandleUnneededCredentials(mongoDBServer, mongodb)
	return errToReturn
}

// if host or database changed in spec, try to clean on old host/database
func (g *MongoDBGarbageCollector) HandleHostOrDatabaseChange(mongoDBServer *mongodb.MongoDBServer, mongodb *infrav1beta1.MongoDB) error {
	var errToReturn error
	if (mongodb.Status.DatabaseStatus.Host != "" && mongodb.Spec.HostName != mongodb.Status.DatabaseStatus.Host) ||
		(mongodb.Status.DatabaseStatus.Name != "" && mongodb.Spec.DatabaseName != mongodb.Status.DatabaseStatus.Name) {
		mongodb.Status.CredentialsStatus.ForEach(func(status *infrav1beta1.CredentialStatus) {
			err := mongoDBServer.DropUser(mongodb.Status.DatabaseStatus.Name, status.Username)
			if err != nil {
				errToReturn = err
			} else {
				(*g.log).Info("Deleted user on database", "user", status.Username, "database", mongodb.Status.DatabaseStatus.Name)
			}
		})
	}
	return errToReturn
}

func (g *MongoDBGarbageCollector) HandleUnneededCredentials(mongoDBServer *mongodb.MongoDBServer, mongodb *infrav1beta1.MongoDB) error {
	var errToReturn error
	// - garbage collection
	// remove all statuses for credentials that are no longer required by spec, and delete users in database
	mongodb.RemoveUnneededCredentialsStatus().ForEach(func(status *infrav1beta1.CredentialStatus) {
		errToReturn = mongoDBServer.DropUser(mongodb.Status.DatabaseStatus.Name, status.Username)
	})
	return errToReturn
}
