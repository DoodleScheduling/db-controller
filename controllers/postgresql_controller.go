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
	postgresqlAPI "github.com/doodlescheduling/kubedb/common/db/postgresql"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1beta1 "github.com/doodlescheduling/kubedb/api/v1beta1"
)

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
			// TODO consider dropping database? What about data, it will be lost..
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, errors.Wrap(err, "unable to fetch PostgreSQL")
	}

	postgreSQLServer := postgresqlAPI.NewPostgreSQLServer("postgres-postgresql.devops.svc.cluster.local", "5432", "postgres", "postgres")

	if postgresql.Spec.DatabaseName != postgresql.Status.DatabaseStatus.Name {
		if postgresql.Status.DatabaseStatus.Name == "" {
			if err := postgreSQLServer.CreateDatabaseIfNotExists(string(postgresql.Spec.DatabaseName)); err != nil {
				postgresql.Status.DatabaseStatus.Status = infrav1beta1.PostgreSQLDatabaseUnavailable
				postgresql.Status.DatabaseStatus.Message = err.Error()
			} else {
				postgresql.Status.DatabaseStatus.Status = infrav1beta1.PostgreSQLDatabaseAvailable
				postgresql.Status.DatabaseStatus.Name = postgresql.Spec.DatabaseName
				postgresql.Status.DatabaseStatus.Message = "Database up."
			}
		} else {
			// TODO in future, implement FORCE flag. For now, mark status as failed to change db name
			postgresql.Status.DatabaseStatus.Status = infrav1beta1.PostgreSQLDatabaseUnavailable
			postgresql.Status.DatabaseStatus.Message = "Cannot change the name of the database."
		}
	}

	/// check if database status is empty, if yes - need to create new database
	///// - check if database exists, create if not, update status
	/// if database status not empty, then
	///// - check if status == desired
	//////// if yes, check if database exists, create if not, update status
	//////// if no, check if status database exists, drop if yes, create desired database, update status

	//if databaseExists, err := postgresqlAPI.NewPostgreSQLServer("postgres-postgresql.devops.svc.cluster.local", "5432", "postgres", "postgres").DatabaseExists(string(desiredDBName)); err != nil {
	//	log.Error(err, "error connecting to postgres")
	//} else {
	//	if databaseExists {
	//		log.Info("database exists")
	//	} else {
	//		log.Info("database does not exist")
	//	}
	//}

	if err := r.Status().Update(ctx, &postgresql); err != nil {
		log.Error(err, "unable to update PostgreSQL status")
		return ctrl.Result{}, err
	}
	log.Info("PostgreSQL status updated")
	return ctrl.Result{}, nil
}

func (r *PostgreSQLReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1beta1.PostgreSQL{}).
		Complete(r)
}
