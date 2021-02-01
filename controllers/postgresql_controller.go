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
	vaultAPI "github.com/doodlescheduling/kubedb/common/vault"
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

// errors
var (
	ErrNoRootSecret = errors.New("there is no root secret field entry under specified rootSecret field")
)

// PostgreSQLReconciler reconciles a PostgreSQL object
type PostgreSQLReconciler struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	ServerCache *postgresqlAPI.Cache
	VaultCache  *vaultAPI.Cache
}

// +kubebuilder:rbac:groups=infra.doodle.com,resources=postgresqls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infra.doodle.com,resources=postgresqls/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *PostgreSQLReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("postgresql", req.NamespacedName)

	// get postgresql resource by namespaced name
	var postgresql infrav1beta1.PostgreSQL
	if err := r.Get(ctx, req.NamespacedName, &postgresql); err != nil {
		if apierrors.IsNotFound(err) {
			// resource no longer present. Consider dropping a database? What about data, it will be lost.. Probably acceptable for devboxes
			// How to do it, though? Resource doesn't exist anymore, so we need to list all databases and all manifests and compare?
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// set spec defaults. Does not mutate the spec, since we are not updating resource
	if err := postgresql.SetDefaults(); err != nil {
		postgresql.Status.DatabaseStatus.SetDatabaseStatus(infrav1beta1.Unavailable, err.Error(), "")
		return r.updateAndReturn(&ctx, &postgresql, &log)
	}

	// get root database password from k8s secret
	rootPassword, err := r.getRootPassword(&ctx, postgresql.Spec.RootSecretLookup.Name, postgresql.Spec.RootSecretLookup.Namespace, postgresql.Spec.RootSecretLookup.Field)
	if err != nil {
		postgresql.Status.DatabaseStatus.SetDatabaseStatus(infrav1beta1.Unavailable, err.Error(), "")
		return r.updateAndReturn(&ctx, &postgresql, &log)
	}

	// postgresql connection, cached
	postgreSQLServer, err := r.ServerCache.Get(postgresql.Spec.Host, fmt.Sprintf("%d", postgresql.Spec.Port), postgresql.Spec.RootUsername, rootPassword)
	if err != nil {
		postgresql.Status.DatabaseStatus.SetDatabaseStatus(infrav1beta1.Unavailable, err.Error(), "")
		return r.updateAndReturn(&ctx, &postgresql, &log)
	}
	// vault connection, cached
	vault, err := r.VaultCache.Get(postgresql.Spec.RootSecretLookup.Name)
	if err != nil {
		postgresql.Status.DatabaseStatus.SetDatabaseStatus(infrav1beta1.Unavailable, err.Error(), "")
		return r.updateAndReturn(&ctx, &postgresql, &log)
	}

	// for now, disallow database renaming
	if postgresql.Spec.DatabaseName != postgresql.Status.DatabaseStatus.Name && postgresql.Status.DatabaseStatus.Name != "" {
		postgresql.Status.DatabaseStatus.SetDatabaseStatus(infrav1beta1.Unavailable, "Cannot change the name of the database.", "")
		r.ServerCache.Remove(postgresql.Spec.Host)
		return r.updateAndReturn(&ctx, &postgresql, &log)
	}

	// Setup database
	if err := postgreSQLServer.CreateDatabaseIfNotExists(postgresql.Spec.DatabaseName); err != nil {
		postgresql.Status.DatabaseStatus.SetDatabaseStatus(infrav1beta1.Unavailable, err.Error(), "")
		r.ServerCache.Remove(postgresql.Spec.Host)
		return r.updateAndReturn(&ctx, &postgresql, &log)
	}
	postgresql.Status.DatabaseStatus.SetDatabaseStatus(infrav1beta1.Available, "Database up.", postgresql.Spec.DatabaseName)

	// Delete all credentials if they exist, and are no longer required by spec
	if postgresql.Spec.Credentials == nil {
		postgresql.Status.CredentialsStatus.ForEach(func(status *infrav1beta1.CredentialStatus) {
			_ = postgreSQLServer.DropUser(status.Username, postgresql.Status.DatabaseStatus.Name)
		})
		postgresql.Status.CredentialsStatus = make([]*infrav1beta1.CredentialStatus, 0)
		return r.updateAndReturn(&ctx, &postgresql, &log)
	}

	// setup credentials as per spec
	for _, credential := range postgresql.Spec.Credentials {
		username := credential.UserName
		postgreSQLCredentialStatus := postgresql.Status.CredentialsStatus.FindOrCreate(username, func(status *infrav1beta1.CredentialStatus) bool {
			return status != nil && status.Username == username
		})
		// get user credentials from vault
		vaultResponse, err := vault.Get(vaultAPI.VaultRequest{})
		if err != nil {
			postgreSQLCredentialStatus.SetCredentialsStatus(infrav1beta1.Unavailable, err.Error())
			continue
		}
		password := vaultResponse.Secret

		// setup user credentials and privileges
		if err := postgreSQLServer.SetupUser(username, password, postgresql.Spec.DatabaseName); err != nil {
			postgreSQLCredentialStatus.SetCredentialsStatus(infrav1beta1.Unavailable, err.Error())
		} else {
			postgreSQLCredentialStatus.SetCredentialsStatus(infrav1beta1.Available, "Credentials up.")
		}
	}

	// remove all statuses for credentials that are no longer required by spec, and delete users in database
	postgresql.RemoveUnneededCredentialsStatus().ForEach(func(status *infrav1beta1.CredentialStatus) {
		_ = postgreSQLServer.DropUser(status.Username, postgresql.Status.DatabaseStatus.Name)
	})

	return r.updateAndReturn(&ctx, &postgresql, &log)
}

func (r *PostgreSQLReconciler) getRootPassword(ctx *context.Context, name string, namespace string, field string) (string, error) {
	var rootSecret v12.Secret
	if err := (*r).Get(*ctx, types.NamespacedName{Name: name, Namespace: namespace}, &rootSecret); err != nil {
		return "", err
	}
	if len(rootSecret.Data[field]) == 0 {
		return "", ErrNoRootSecret
	}
	return string(rootSecret.Data[field][:]), nil
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
