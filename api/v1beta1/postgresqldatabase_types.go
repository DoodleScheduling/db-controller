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
	DEFAULT_POSTGRESQL_ROOT_USER                    = "postgres"
	DEFAULT_POSTGRESQL_ROOT_AUTHENTICATION_DATABASE = "postgres"
)

// Finalizer
const (
	PostgreSQLDatabaseControllerFinalizer = "infra.finalizers.doodle.com"
)

// PostgreSQLDatabaseSpec defines the desired state of PostgreSQLDatabase
type PostgreSQLDatabaseSpec struct {
	*DatabaseSpec `json:",inline"`
}

// GetStatusConditions returns a pointer to the Status.Conditions slice
func (in *PostgreSQLDatabase) GetStatusConditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

// PostgreSQLDatabaseStatus defines the observed state of PostgreSQLDatabase
// IMPORTANT: Run "make" to regenerate code after modifying this file
type PostgreSQLDatabaseStatus struct {
	// Conditions holds the conditions for the VaultBinding.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PostgreSQLDatabase is the Schema for the postgresqls API
type PostgreSQLDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgreSQLDatabaseSpec   `json:"spec,omitempty"`
	Status PostgreSQLDatabaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PostgreSQLDatabaseList contains a list of PostgreSQLDatabase
type PostgreSQLDatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgreSQLDatabase `json:"items"`
}

/*
	If object doesn't contain finalizer, set it and call update function 'updateF'.
	Only do this if object is not being deleted (judged by DeletionTimestamp being zero)
*/
func (d *PostgreSQLDatabase) SetFinalizer(updateF func() error) error {
	if !d.ObjectMeta.DeletionTimestamp.IsZero() {
		return nil
	}
	if !stringutils.ContainsString(d.ObjectMeta.Finalizers, PostgreSQLDatabaseControllerFinalizer) {
		d.ObjectMeta.Finalizers = append(d.ObjectMeta.Finalizers, PostgreSQLDatabaseControllerFinalizer)
		return updateF()
	}
	return nil
}

/*
	Finalize object if deletion timestamp is not zero (i.e. object is being deleted).
	Call finalize function 'finalizeF', which should handle finalization logic.
	Remove finalizer from the object (so that object can be deleted), and update by calling update function 'updateF'.
*/
func (d *PostgreSQLDatabase) Finalize(updateF func() error, finalizeF func() error) (bool, error) {
	if d.ObjectMeta.DeletionTimestamp.IsZero() {
		return false, nil
	}
	if stringutils.ContainsString(d.ObjectMeta.Finalizers, PostgreSQLDatabaseControllerFinalizer) {
		_ = finalizeF()
		d.ObjectMeta.Finalizers = stringutils.RemoveString(d.ObjectMeta.Finalizers, PostgreSQLDatabaseControllerFinalizer)
		return true, updateF()
	}
	return true, nil
}

/*func (d *PostgreSQLDatabase) SetDefaults() error {
	if d.Spec.RootUsername == "" {
		d.Spec.RootUsername = DEFAULT_POSTGRESQL_ROOT_USER
	}
	if d.Spec.RootAuthenticationDatabase == "" {
		d.Spec.RootAuthenticationDatabase = DEFAULT_POSTGRESQL_ROOT_AUTHENTICATION_DATABASE
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
	SchemeBuilder.Register(&PostgreSQLDatabase{}, &PostgreSQLDatabaseList{})
}
