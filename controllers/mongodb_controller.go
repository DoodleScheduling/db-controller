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
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infra.doodle.com,resources=mongodbs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infra.doodle.com,resources=mongodbs/status,verbs=get;update;patch

func (r *MongoDBReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("mongodb", req.NamespacedName)

	var mongodb infrav1beta1.MongoDB
	if err := r.Get(ctx, req.NamespacedName, &mongodb); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, errors.Wrap(err, "unable to fetch Mongodb")
	}

	log.Info("updating Mongodb status...")

	s := make(infrav1beta1.MongoDBCredentialsStatus, 0)
	mongodb.Status.CredentialsStatus = s

	mongodbserver, err := mongodbAPI.NewMongoDBServer("mongodb.devops", "27017", "root", "admin", "admin")
	if err != nil {
		log.Error(err, "Error while connecting to mongodb")
		mongodb.Status.MongoDBAvailabilityStatus.Status = infrav1beta1.MongoDBDatabaseUnavailable
		mongodb.Status.MongoDBAvailabilityStatus.Message = err.Error()
		return r.updateAndReturn(&ctx, &mongodb, &log)
	}
	if u, err := mongodbserver.DoesUserExist("admin", "admin"); err != nil {
		log.Error(err, "Error while getting user")
		mongodb.Status.MongoDBAvailabilityStatus.Status = infrav1beta1.MongoDBDatabaseUnavailable
		mongodb.Status.MongoDBAvailabilityStatus.Message = err.Error()
		return r.updateAndReturn(&ctx, &mongodb, &log)
	} else {
		if u == nil {
			log.Info("user is nil")
		} else {
			log.Info("user returned", "whole struct", fmt.Sprintf("%+v", u))
		}
		//log.Info("user returned", "user", u)
		mongodb.Status.MongoDBAvailabilityStatus.Status = infrav1beta1.MongoDBDatabaseAvailable
	}

	//if err := r.Status().Update(ctx, &mongodb); err != nil {
	//	log.Error(err, "unable to update Mongodb status")
	//	return ctrl.Result{}, err
	//}
	log.Info("Mongodb status updated")
	return ctrl.Result{}, nil
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
