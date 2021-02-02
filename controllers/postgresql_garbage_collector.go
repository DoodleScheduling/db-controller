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
func (g *PostgreSQLGarbageCollector) Clean(postgresql *infrav1beta1.PostgreSQL) error {
	rootPassword, err := g.cw.GetRootPassword(postgresql.Status.DatabaseStatus.RootSecretLookup.Name, postgresql.Status.DatabaseStatus.RootSecretLookup.Namespace, postgresql.Status.DatabaseStatus.RootSecretLookup.Field)
	if err != nil {
		// no point in proceeding. In future could also try with Spec credential
		return err
	}
	postgreSQLServer, err := g.r.ServerCache.Get(postgresql.Status.DatabaseStatus.Host, postgresql.Status.DatabaseStatus.RootUsername, rootPassword, postgresql.Status.DatabaseStatus.RootAuthenticationDatabase)
	if err != nil {
		// no point in proceeding. In future could also try with Spec credential
		return err
	}
	// if an error happens, just collect it and try to clean as much as possible. Return it at the end. This is a pattern in for most of this Garbage Collector.
	var errToReturn error
	errToReturn = g.HandleHostOrDatabaseChange(postgreSQLServer, postgresql, rootPassword)
	errToReturn = g.HandleUnneededCredentials(postgresql, postgreSQLServer)
	return errToReturn
}

// if host or database changed in spec, try to clean on old host/database
func (g *PostgreSQLGarbageCollector) HandleHostOrDatabaseChange(postgreSQLServer *postgresqlAPI.PostgreSQLServer, postgresql *infrav1beta1.PostgreSQL, rootPassword string) error {
	var errToReturn error
	if (postgresql.Status.DatabaseStatus.Host != "" && postgresql.Spec.HostName != postgresql.Status.DatabaseStatus.Host) ||
		(postgresql.Status.DatabaseStatus.Name != "" && postgresql.Spec.DatabaseName != postgresql.Status.DatabaseStatus.Name) {
		postgresql.Status.CredentialsStatus.ForEach(func(status *infrav1beta1.CredentialStatus) {
			err := postgreSQLServer.DropUser(status.Username, postgresql.Status.DatabaseStatus.Name)
			if err != nil {
				errToReturn = err
			} else {
				(*g.log).Info("Deleted user on database", "user", status.Username, "database", postgresql.Status.DatabaseStatus.Name)
			}
		})

	}
	return errToReturn
}

func (g *PostgreSQLGarbageCollector) HandleUnneededCredentials(postgresql *infrav1beta1.PostgreSQL, postgreSQLServer *postgresqlAPI.PostgreSQLServer) error {
	var errToReturn error
	// - garbage collection
	// remove all statuses for credentials that are no longer required by spec, and delete users in database
	postgresql.RemoveUnneededCredentialsStatus().ForEach(func(status *infrav1beta1.CredentialStatus) {
		errToReturn = postgreSQLServer.DropUser(status.Username, postgresql.Status.DatabaseStatus.Name)
	})
	return errToReturn
}
