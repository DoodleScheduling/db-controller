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
	infrav1beta1 "github.com/doodlescheduling/kubedb/api/v1beta1"
	postgresqlAPI "github.com/doodlescheduling/kubedb/common/db/postgresql"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	v12 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO MOVE TO RECONCILER VARIABLE !!!!
var serversCache = make(map[string]*postgresqlAPI.PostgreSQLServer)

// PostgreSQLReconciler reconciles a PostgreSQL object
type PostgreSQLReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infra.doodle.com,resources=postgresqls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infra.doodle.com,resources=postgresqls/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

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
	if err := postgresql.SetDefaults(); err != nil {
		postgresql.Status.SetDatabaseStatus(infrav1beta1.PostgreSQLUnavailable, err.Error(), nil)
		return r.updateAndReturn(&ctx, &postgresql, &log)
	}

	rootPassword, err := r.getRootPassword(&ctx, postgresql.Spec.RootSecretLookup.Name, postgresql.Spec.RootSecretLookup.Namespace, postgresql.Spec.RootSecretLookup.Field)
	if err != nil {
		postgresql.Status.SetDatabaseStatus(infrav1beta1.PostgreSQLUnavailable, err.Error(), nil)
		return r.updateAndReturn(&ctx, &postgresql, &log)
	}

	postgreSQLServer, err := r.getAndStore(string(postgresql.Spec.Host), fmt.Sprintf("%d", postgresql.Spec.Port), postgresql.Spec.RootUsername, rootPassword, &log)
	if err != nil {
		postgresql.Status.SetDatabaseStatus(infrav1beta1.PostgreSQLUnavailable, err.Error(), nil)
		return r.updateAndReturn(&ctx, &postgresql, &log)
	}

	if postgresql.Spec.DatabaseName != postgresql.Status.DatabaseStatus.Name && postgresql.Status.DatabaseStatus.Name != "" {
		// TODO in future, implement FORCE flag. For now, mark status as failed to change db name
		postgresql.Status.SetDatabaseStatus(infrav1beta1.PostgreSQLUnavailable, "Cannot change the name of the database.", nil)
		delete(serversCache, string(postgresql.Spec.Host))
		return r.updateAndReturn(&ctx, &postgresql, &log)
	} else {
		if err := postgreSQLServer.CreateDatabaseIfNotExists(string(postgresql.Spec.DatabaseName)); err != nil {
			postgresql.Status.SetDatabaseStatus(infrav1beta1.PostgreSQLUnavailable, err.Error(), nil)
			delete(serversCache, string(postgresql.Spec.Host))
			return r.updateAndReturn(&ctx, &postgresql, &log)
		} else {
			postgresql.Status.SetDatabaseStatus(infrav1beta1.PostgreSQLAvailable, "Database up.", &postgresql.Spec.DatabaseName)
		}
	}

	if postgresql.Spec.Credentials == nil {
		// TODO drop users in status
		postgresql.Status.CredentialsStatus = make([]*infrav1beta1.PostgreSQLCredentialStatus, 0)
		return r.updateAndReturn(&ctx, &postgresql, &log)
	}

	for _, credential := range postgresql.Spec.Credentials {
		// TODO get secret from vault
		username := credential.UserName
		password := "password"
		postgreSQLCredentialStatus := postgresql.Status.CredentialsStatus.FindOrCreate(username, func(status *infrav1beta1.PostgreSQLCredentialStatus) bool {
			return status != nil && status.Username == username
		})
		if err := postgreSQLServer.SetupUser(username, password, string(postgresql.Spec.DatabaseName)); err != nil {
			postgreSQLCredentialStatus.SetCredentialsStatus(infrav1beta1.PostgreSQLUnavailable, err.Error())
		} else {
			postgreSQLCredentialStatus.SetCredentialsStatus(infrav1beta1.PostgreSQLAvailable, "Credentials up.")
		}
	}

	// TODO delete all users from Status that are not present in Spec

	return r.updateAndReturn(&ctx, &postgresql, &log)
}

func (r *PostgreSQLReconciler) getRootPassword(ctx *context.Context, name string, namespace string, field string) (string, error) {
	var rootSecret v12.Secret
	if err := (*r).Get(*ctx, types.NamespacedName{Name: name, Namespace: namespace}, &rootSecret); err != nil {
		return "", err
	}
	if len(rootSecret.Data[field]) == 0 {
		return "", errors.New("there is no root secret field entry under specified rootSecret field")
	}
	return string(rootSecret.Data[field][:]), nil
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
	postgresql.Status.LastUpdateTime = metav1.Now()
	if err := r.Status().Update(*ctx, postgresql); err != nil {
		(*log).Error(err, "unable to update PostgreSQL status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
