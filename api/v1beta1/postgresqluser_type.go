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

type PostgreSQLUserSpec struct {
	Database    *DatabaseReference `json:"database"`
	Credentials *SecretReference   `json:"credentials"`
}

// GetStatusConditions returns a pointer to the Status.Conditions slice
func (in *PostgreSQLUser) GetStatusConditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

// PostgreSQLUserStatus defines the observed state of PostgreSQLUser
// IMPORTANT: Run "make" to regenerate code after modifying this file
type PostgreSQLUserStatus struct {
	// Conditions holds the conditions for the VaultBinding.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +genclient
// +genclient:Namespaced
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=pgu
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"UserReady\")].status",description=""
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type==\"UserReady\")].message",description=""
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description=""

// PostgreSQLUser is the Schema for the mongodbs API
type PostgreSQLUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgreSQLUserSpec   `json:"spec,omitempty"`
	Status PostgreSQLUserStatus `json:"status,omitempty"`
}

func (in *PostgreSQLUser) GetDatabase() string {
	return in.Spec.Database.Name
}

func (in *PostgreSQLUser) GetCredentials() *SecretReference {
	return in.Spec.Credentials
}

// +kubebuilder:object:root=true

// PostgreSQLUserList contains a list of PostgreSQLUser
type PostgreSQLUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgreSQLUser `json:"items"`
}

// If object doesn't contain finalizer, set it and call update function 'updateF'.
// Only do this if object is not being deleted (judged by DeletionTimestamp being zero)
func (d *PostgreSQLUser) SetFinalizer(updateF func() error) error {
	if !d.ObjectMeta.DeletionTimestamp.IsZero() {
		return nil
	}
	if !stringutils.ContainsString(d.ObjectMeta.Finalizers, Finalizer) {
		d.ObjectMeta.Finalizers = append(d.ObjectMeta.Finalizers, Finalizer)
		return updateF()
	}
	return nil
}

// Finalize object if deletion timestamp is not zero (i.e. object is being deleted).
// Call finalize function 'finalizeF', which should handle finalization logic.
// Remove finalizer from the object (so that object can be deleted), and update by calling update function 'updateF'.
func (d *PostgreSQLUser) Finalize(updateF func() error, finalizeF func() error) (bool, error) {
	if d.ObjectMeta.DeletionTimestamp.IsZero() {
		return false, nil
	}
	if stringutils.ContainsString(d.ObjectMeta.Finalizers, Finalizer) {
		_ = finalizeF()
		d.ObjectMeta.Finalizers = stringutils.RemoveString(d.ObjectMeta.Finalizers, Finalizer)
		return true, updateF()
	}
	return true, nil
}

func init() {
	SchemeBuilder.Register(&PostgreSQLUser{}, &PostgreSQLUserList{})
}
