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
	vaultAPI "github.com/doodlescheduling/kubedb/common/vault"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1beta1 "github.com/doodlescheduling/kubedb/api/v1beta1"
	mongodbAPI "github.com/doodlescheduling/kubedb/common/db/mongodb"
)

// MongoDBReconciler reconciles a MongoDB object
type MongoDBReconciler struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	ServerCache *mongodbAPI.Cache
	VaultCache  *vaultAPI.Cache
}

// +kubebuilder:rbac:groups=infra.doodle.com,resources=mongodbs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infra.doodle.com,resources=mongodbs/status,verbs=get;update;patch

func (r *MongoDBReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("mongodb", req.NamespacedName)

	// common controller functions
	cw := NewControllerWrapper(*r, &ctx)

	var mongodb infrav1beta1.MongoDB
	if err := r.Get(ctx, req.NamespacedName, &mongodb); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, errors.Wrap(err, "unable to fetch Mongodb")
	}

	s := make(infrav1beta1.CredentialsStatus, 0)
	mongodb.Status.CredentialsStatus = s

	// root password
	rootPassword, err := cw.GetRootPassword(mongodb.Spec.RootSecretLookup.Name, mongodb.Spec.RootSecretLookup.Namespace, mongodb.Spec.RootSecretLookup.Field)
	if err != nil {
		mongodb.Status.DatabaseStatus.SetDatabaseStatus(infrav1beta1.Unavailable, err.Error(), "", "")
		mongodb.Status.CredentialsStatus = make(infrav1beta1.CredentialsStatus, 0)
		return r.updateAndReturn(&ctx, &mongodb, &log)
	}

	// mongoDB server
	mongoDBServer, err := r.ServerCache.Get(mongodb.Spec.HostName, mongodb.Spec.RootUsername, rootPassword, mongodb.Spec.RootAuthenticationDatabase)
	if err != nil {
		log.Error(err, "Error while connecting to mongodb")
		mongodb.Status.DatabaseStatus.Status = infrav1beta1.Unavailable
		mongodb.Status.DatabaseStatus.Message = err.Error()
		return r.updateAndReturn(&ctx, &mongodb, &log)
	}

	// vault connection, cached
	vault, err := r.VaultCache.Get(mongodb.Spec.RootSecretLookup.Name)
	if err != nil {
		mongodb.Status.DatabaseStatus.SetDatabaseStatus(infrav1beta1.Unavailable, err.Error(), "", mongodb.Spec.HostName)
		mongodb.Status.CredentialsStatus = make(infrav1beta1.CredentialsStatus, 0)
		return r.updateAndReturn(&ctx, &mongodb, &log)
	}

	for _, credential := range mongodb.Spec.Credentials {
		username := credential.UserName
		mongodbCredentialStatus := mongodb.Status.CredentialsStatus.FindOrCreate(username, func(status *infrav1beta1.CredentialStatus) bool {
			return status != nil && status.Username == username
		})
		// get user credentials from vault
		vaultResponse, err := vault.Get(vaultAPI.VaultRequest{})
		if err != nil {
			mongodbCredentialStatus.SetCredentialsStatus(infrav1beta1.Unavailable, err.Error())
			continue
		}
		password := vaultResponse.Secret
		if err := mongoDBServer.SetupUser(mongodb.Spec.DatabaseName, username, password); err != nil {
			mongodbCredentialStatus.SetCredentialsStatus(infrav1beta1.Unavailable, err.Error())
		} else {
			mongodbCredentialStatus.SetCredentialsStatus(infrav1beta1.Available, "Credentials up.")
		}
	}
	mongodb.Status.DatabaseStatus.Status = infrav1beta1.Available
	mongodb.Status.DatabaseStatus.Message = "Database up."

	return r.updateAndReturn(&ctx, &mongodb, &log)
}

func (r *MongoDBReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1beta1.MongoDB{}).
		Complete(r)
}

func (r *MongoDBReconciler) updateAndReturn(ctx *context.Context, mongodb *infrav1beta1.MongoDB, log *logr.Logger) (ctrl.Result, error) {
	mongodb.Status.LastUpdateTime = metav1.Now()
	if err := r.Status().Update(*ctx, mongodb); err != nil {
		(*log).Error(err, "unable to update MongoDB status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
