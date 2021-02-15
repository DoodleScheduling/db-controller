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

package v1beta1

import (
	"github.com/doodlescheduling/kubedb/common/stringutils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// defaults
const (
	DEFAULT_MONGODB_ROOT_USER                    = "root"
	DEFAULT_MONGODB_ROOT_AUTHENTICATION_DATABASE = "admin"
)

// Finalizer
const (
	MongoSQLDatabaseControllerFinalizer = "infra.finalizers.doodle.com"
)

// MongoDBDatabaseSpec defines the desired state of MongoDBDatabase
type MongoDBDatabaseSpec struct {
	*DatabaseSpec `json:",inline"`
}

// GetStatusConditions returns a pointer to the Status.Conditions slice
func (in *MongoDBDatabase) GetStatusConditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

// MongoDBDatabaseStatus defines the observed state of MongoDBDatabase
// IMPORTANT: Run "make" to regenerate code after modifying this file
type MongoDBDatabaseStatus struct {
	// Conditions holds the conditions for the VaultBinding.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +genclient
// +genclient:Namespaced
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=mdb
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"Provisioned\")].status",description=""
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type==\"Provisioned\")].message",description=""
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description=""

// MongoDBDatabase is the Schema for the mongodbs API
type MongoDBDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MongoDBDatabaseSpec   `json:"spec,omitempty"`
	Status MongoDBDatabaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MongoDBDatabaseList contains a list of MongoDBDatabase
type MongoDBDatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MongoDBDatabase `json:"items"`
}

/*
	If object doesn't contain finalizer, set it and call update function 'updateF'.
	Only do this if object is not being deleted (judged by DeletionTimestamp being zero)
*/
func (d *MongoDBDatabase) SetFinalizer(updateF func() error) error {
	if !d.ObjectMeta.DeletionTimestamp.IsZero() {
		return nil
	}
	if !stringutils.ContainsString(d.ObjectMeta.Finalizers, MongoSQLDatabaseControllerFinalizer) {
		d.ObjectMeta.Finalizers = append(d.ObjectMeta.Finalizers, MongoSQLDatabaseControllerFinalizer)
		return updateF()
	}
	return nil
}

/*
	Finalize object if deletion timestamp is not zero (i.e. object is being deleted).
	Call finalize function 'finalizeF', which should handle finalization logic.
	Remove finalizer from the object (so that object can be deleted), and update by calling update function 'updateF'.
*/
func (d *MongoDBDatabase) Finalize(updateF func() error, finalizeF func() error) (bool, error) {
	if d.ObjectMeta.DeletionTimestamp.IsZero() {
		return false, nil
	}
	if stringutils.ContainsString(d.ObjectMeta.Finalizers, MongoSQLDatabaseControllerFinalizer) {
		_ = finalizeF()
		d.ObjectMeta.Finalizers = stringutils.RemoveString(d.ObjectMeta.Finalizers, MongoSQLDatabaseControllerFinalizer)
		return true, updateF()
	}
	return true, nil
}

/*func (d *MongoDBDatabase) SetDefaults() error {
	if d.Spec.RootUsername == "" {
		d.Spec.RootUsername = DEFAULT_MONGODB_ROOT_USER
	}
	if d.Spec.RootAuthenticationDatabase == "" {
		d.Spec.RootAuthenticationDatabase = DEFAULT_MONGODB_ROOT_AUTHENTICATION_DATABASE
	}
	if d.Spec.RootSecretLookup.Name == "" {
		return errors.New("must specify root secret")
	}
	if d.Spec.RootSecretLookup.Field == "" {
		return errors.New("must specify root secret field")
	}
	if d.Spec.RootSecretLookup.Namespace == "" {
		d.Spec.RootSecretLookup.Namespace = d.ObjectMeta.Namespace
	}
	if d.Status.CredentialsStatus == nil || len(d.Status.CredentialsStatus) == 0 {
		d.Status.CredentialsStatus = make([]*CredentialStatus, 0)
	}
	return nil
}*/

func init() {
	SchemeBuilder.Register(&MongoDBDatabase{}, &MongoDBDatabaseList{})
}
