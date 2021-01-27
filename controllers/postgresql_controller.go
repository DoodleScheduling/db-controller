/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	postgresqlAPI "github.com/doodlescheduling/kubedb/common/db/postgresql"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1beta1 "github.com/doodlescheduling/kubedb/api/v1beta1"
)

// TODO MOVE TO RECONCILER VARIABLE !!!!
var serversCache = make(map[string]*postgresqlAPI.PostgreSQLServer)

// is it worth to decrease security for more performance?
var credentialsCache = make(map[string]string)

// PostgreSQLReconciler reconciles a PostgreSQL object
type PostgreSQLReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infra.doodle.com,resources=postgresqls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infra.doodle.com,resources=postgresqls/status,verbs=get;update;patch

func (r *PostgreSQLReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("postgresql", req.NamespacedName)

	var postgresql infrav1beta1.PostgreSQL
	if err := r.Get(ctx, req.NamespacedName, &postgresql); err != nil {
		if apierrors.IsNotFound(err) {
			// TODO Resource no longer present. Consider dropping a database? What about data, it will be lost.. Probably acceptable for devboxes
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, errors.Wrap(err, "unable to fetch PostgreSQL")
	}

	postgreSQLServer, err := r.getAndStore(string(postgresql.Spec.Host), fmt.Sprintf("%d", postgresql.Spec.Port), postgresql.Spec.RootCredential.UserName, postgresql.Spec.RootCredential.Password, &log)
	if err != nil {
		postgresql.Status.SetDatabaseStatus(infrav1beta1.PostgreSQLDatabaseUnavailable, err.Error(), nil)
		return r.updateAndReturn(&ctx, &postgresql, &log)
	}
	log.V(1).Info("reusing database pool", "host", postgresql.Spec.Host)

	if postgresql.Spec.DatabaseName != postgresql.Status.DatabaseStatus.Name && postgresql.Status.DatabaseStatus.Name != "" {
		// TODO in future, implement FORCE flag. For now, mark status as failed to change db name
		postgresql.Status.SetDatabaseStatus(infrav1beta1.PostgreSQLDatabaseUnavailable, "Cannot change the name of the database.", nil)
		delete(serversCache, string(postgresql.Spec.Host))
		return r.updateAndReturn(&ctx, &postgresql, &log)
	} else {
		if err := postgreSQLServer.CreateDatabaseIfNotExists(string(postgresql.Spec.DatabaseName)); err != nil {
			postgresql.Status.SetDatabaseStatus(infrav1beta1.PostgreSQLDatabaseUnavailable, err.Error(), nil)
			delete(serversCache, string(postgresql.Spec.Host))
			return r.updateAndReturn(&ctx, &postgresql, &log)
		} else {
			postgresql.Status.SetDatabaseStatus(infrav1beta1.PostgreSQLDatabaseAvailable, "Database up.", &postgresql.Spec.DatabaseName)
		}
	}

	if postgresql.Spec.Credentials == nil {
		// TODO drop users in status?
		return r.updateAndReturn(&ctx, &postgresql, &log)
	}
	if postgresql.Status.CredentialsStatus == nil || len(postgresql.Status.CredentialsStatus) == 0 {
		// TODO create all users from SPEC
		return r.updateAndReturn(&ctx, &postgresql, &log)
	}

	for _, credential := range postgresql.Spec.Credentials {
		found := false
		for _, statusCredential := range postgresql.Status.CredentialsStatus {
			if credential.UserName == statusCredential.Username {
				found = true
			}
		}
		if !found {
			// TODO create new user per spec; read credentials from vault
		}
	}

	return r.updateAndReturn(&ctx, &postgresql, &log)
}

// TODO there should be a separate struct representing cache; move the method there
func (r *PostgreSQLReconciler) getAndStore(host string, port string, rootUsername string, rootPassword string, log *logr.Logger) (*postgresqlAPI.PostgreSQLServer, error) {
	// Try to get the connection to server from cache
	if _, ok := serversCache[host]; !ok {
		(*log).V(1).Info("connecting to new database host", "host", host)
		if server, err := postgresqlAPI.NewPostgreSQLServer(host, port, rootUsername, rootPassword); err != nil {
			return nil, err
		} else {
			serversCache[host] = server
			(*log).V(1).Info("successfully connected to database host", "host", host)
		}
	}

	return serversCache[host], nil
}

func (r *PostgreSQLReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1beta1.PostgreSQL{}).
		Complete(r)
}

func (r *PostgreSQLReconciler) updateAndReturn(ctx *context.Context, postgresql *infrav1beta1.PostgreSQL, log *logr.Logger) (ctrl.Result, error) {
	if err := r.Status().Update(*ctx, postgresql); err != nil {
		(*log).Error(err, "unable to update PostgreSQL status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
