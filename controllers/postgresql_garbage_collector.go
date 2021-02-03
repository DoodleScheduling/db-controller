package controllers

import (
	infrav1beta1 "github.com/doodlescheduling/kubedb/api/v1beta1"
	postgresqlAPI "github.com/doodlescheduling/kubedb/common/db/postgresql"
	"github.com/go-logr/logr"
)

type PostgreSQLGarbageCollector struct {
	r   *PostgreSQLReconciler
	cw  *ControllerWrapper
	log *logr.Logger
}

func NewPostgreSQLGarbageCollector(r *PostgreSQLReconciler, cw *ControllerWrapper, log *logr.Logger) *PostgreSQLGarbageCollector {
	return &PostgreSQLGarbageCollector{
		r:   r,
		cw:  cw,
		log: log,
	}
}

/*	For now, Garbage Collector does not drop databases, because we haven't decided if we want to delete all data.
	Possible avenue to proceed is to have a separate option flag/struct (to be set in Spec), that will force deletion of all garbage, including data.
*/
func (g *PostgreSQLGarbageCollector) Clean(database *infrav1beta1.PostgreSQLDatabase) error {
	rootPassword, err := g.cw.GetRootPassword(database.Status.DatabaseStatus.RootSecretLookup.Name, database.Status.DatabaseStatus.RootSecretLookup.Namespace,
		database.Status.DatabaseStatus.RootSecretLookup.Field)
	if err != nil {
		// no point in proceeding. In future could also try with Spec credential
		return err
	}
	postgreSQLServer, err := g.r.ServerCache.Get(database.Status.DatabaseStatus.Host, database.Status.DatabaseStatus.RootUsername, rootPassword,
		database.Status.DatabaseStatus.RootAuthenticationDatabase)
	if err != nil {
		// no point in proceeding. In future could also try with Spec credential
		return err
	}
	// if an error happens, just collect it and try to clean as much as possible. Return it at the end. This is a pattern in for most of this Garbage Collector.
	var errToReturn error
	errToReturn = g.handleHostOrDatabaseChange(postgreSQLServer, database)
	errToReturn = g.handleUnneededCredentials(postgreSQLServer, database)
	return errToReturn
}

// if host or database changed in spec, try to clean on old host/database
func (g *PostgreSQLGarbageCollector) handleHostOrDatabaseChange(postgreSQLServer *postgresqlAPI.PostgreSQLServer, database *infrav1beta1.PostgreSQLDatabase) error {
	var errToReturn error
	if (database.Status.DatabaseStatus.Host != "" && database.Spec.HostName != database.Status.DatabaseStatus.Host) ||
		(database.Status.DatabaseStatus.Name != "" && database.Spec.DatabaseName != database.Status.DatabaseStatus.Name) {
		database.Status.CredentialsStatus.ForEach(func(status *infrav1beta1.CredentialStatus) {
			err := postgreSQLServer.DropUser(database.Status.DatabaseStatus.Name, status.Username)
			if err != nil {
				errToReturn = err
			} else {
				(*g.log).Info("Deleted user on database", "user", status.Username, "database", database.Status.DatabaseStatus.Name)
			}
		})

	}
	return errToReturn
}

func (g *PostgreSQLGarbageCollector) handleUnneededCredentials(postgreSQLServer *postgresqlAPI.PostgreSQLServer, database *infrav1beta1.PostgreSQLDatabase) error {
	var errToReturn error
	// - garbage collection
	// remove all statuses for credentials that are no longer required by spec, and delete users in database
	database.RemoveUnneededCredentialsStatus().ForEach(func(status *infrav1beta1.CredentialStatus) {
		errToReturn = postgreSQLServer.DropUser(database.Status.DatabaseStatus.Name, status.Username)
	})
	return errToReturn
}
