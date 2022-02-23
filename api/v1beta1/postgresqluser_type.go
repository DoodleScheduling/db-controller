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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PostgreSQLUserSpec struct {
	// +required
	Database *DatabaseReference `json:"database"`

	// +required
	Credentials *SecretReference `json:"credentials"`
}

// GetStatusConditions returns a pointer to the Status.Conditions slice
func (in *PostgreSQLUser) GetStatusConditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

// PostgreSQLUserStatus defines the observed state of PostgreSQLUser
// IMPORTANT: Run "make" to regenerate code after modifying this file
type PostgreSQLUserStatus struct {
	// Conditions holds the conditions for the PostgreSQLUser.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Username of the created user.
	// +optional
	Username string `json:"username,omitempty"`

	// ObservedGeneration is the last generation reconciled by the controller
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
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
	sec := in.Spec.Credentials
	if sec.Namespace == "" {
		sec.Namespace = in.GetNamespace()
	}

	return sec
}

// +kubebuilder:object:root=true

// PostgreSQLUserList contains a list of PostgreSQLUser
type PostgreSQLUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgreSQLUser `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PostgreSQLUser{}, &PostgreSQLUserList{})
}
