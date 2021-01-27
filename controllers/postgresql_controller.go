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
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1beta1 "github.com/doodlescheduling/kubedb/api/v1beta1"
)

var servers = make(map[string]*postgresqlAPI.PostgreSQLServer)

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

	// Try to get the connection to server from cache
	if _, ok := servers[string(postgresql.Spec.Host)]; !ok {
		log.V(1).Info("connecting to new database host", "host", postgresql.Spec.Host)
		if server, err := postgresqlAPI.NewPostgreSQLServer(string(postgresql.Spec.Host), fmt.Sprintf("%d", postgresql.Spec.Port), postgresql.Spec.RootCredential.UserName, postgresql.Spec.RootCredential.Password); err != nil {
			postgresql.Status.DatabaseStatus.Status = infrav1beta1.PostgreSQLDatabaseUnavailable
			postgresql.Status.DatabaseStatus.Message = "Cannot change the name of the database."
			return r.updateAndReturn(&ctx, &postgresql, &log)
		} else {
			servers[string(postgresql.Spec.Host)] = server
			log.V(1).Info("successfully connected to database host", "host", postgresql.Spec.Host)
		}
	}

	postgreSQLServer := servers[string(postgresql.Spec.Host)]
	log.V(1).Info("reusing database pool", "host", postgresql.Spec.Host)

	if postgresql.Spec.DatabaseName != postgresql.Status.DatabaseStatus.Name && postgresql.Status.DatabaseStatus.Name != "" {
		// TODO in future, implement FORCE flag. For now, mark status as failed to change db name
		postgresql.Status.DatabaseStatus.Status = infrav1beta1.PostgreSQLDatabaseUnavailable
		postgresql.Status.DatabaseStatus.Message = "Cannot change the name of the database."
		// Remove the entry from cache
		delete(servers, string(postgresql.Spec.Host))
		return r.updateAndReturn(&ctx, &postgresql, &log)
	} else {
		if err := postgreSQLServer.CreateDatabaseIfNotExists(string(postgresql.Spec.DatabaseName)); err != nil {
			postgresql.Status.DatabaseStatus.Status = infrav1beta1.PostgreSQLDatabaseUnavailable
			postgresql.Status.DatabaseStatus.Message = err.Error()
			// Remove the entry from cache
			delete(servers, string(postgresql.Spec.Host))
			return r.updateAndReturn(&ctx, &postgresql, &log)
		} else {
			postgresql.Status.DatabaseStatus.Status = infrav1beta1.PostgreSQLDatabaseAvailable
			postgresql.Status.DatabaseStatus.Name = postgresql.Spec.DatabaseName
			postgresql.Status.DatabaseStatus.Message = "Database up."
		}
	}
	return r.updateAndReturn(&ctx, &postgresql, &log)
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
