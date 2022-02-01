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

// Extension is a resource representing database extension
type Extension struct {
	Name string `json:"name"`
}

// Extensions is a collection of Extension types
type Extensions []Extension

// PostgreSQLDatabaseSpec defines the desired state of PostgreSQLDatabase
type PostgreSQLDatabaseSpec struct {
	*DatabaseSpec `json:",inline"`

	// Database extensions
	// +optional
	Extensions Extensions `json:"extensions,omitempty"`
}

// GetStatusConditions returns a pointer to the Status.Conditions slice
func (in *PostgreSQLDatabase) GetStatusConditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

// PostgreSQLDatabaseStatus defines the observed state of PostgreSQLDatabase
// IMPORTANT: Run "make" to regenerate code after modifying this file
type PostgreSQLDatabaseStatus struct {
	// Conditions holds the conditions for the PostgreSQLDatabase.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration is the last generation reconciled by the controller
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +genclient
// +genclient:Namespaced
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=pgd
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"DatabaseReady\")].status",description=""
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type==\"DatabaseReady\")].message",description=""
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description=""

// PostgreSQLDatabase is the Schema for the postgresqls API
type PostgreSQLDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgreSQLDatabaseSpec   `json:"spec,omitempty"`
	Status PostgreSQLDatabaseStatus `json:"status,omitempty"`
}

func (in *PostgreSQLDatabase) GetRootSecret() *SecretReference {
	if in.Spec.RootSecret.Namespace == "" {
		in.Spec.RootSecret.Namespace = in.GetNamespace()
	}

	return in.Spec.RootSecret
}

func (in *PostgreSQLDatabase) GetDatabaseName() string {
	if in.Spec.DatabaseName != "" {
		return in.Spec.DatabaseName
	}

	return in.GetName()
}

func (in *PostgreSQLDatabase) GetRootDatabaseName() string {
	return ""
}

// +kubebuilder:object:root=true

// PostgreSQLDatabaseList contains a list of PostgreSQLDatabase
type PostgreSQLDatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgreSQLDatabase `json:"items"`
}

func ExtensionNotReadyCondition(in conditionalResource, reason, message string) {
	setResourceCondition(in, ExtensionReadyConditionType, metav1.ConditionFalse, reason, message)
}

func (d *PostgreSQLDatabase) SetDefaults() error {
	if d.Spec.DatabaseName == "" {
		d.Spec.DatabaseName = d.GetName()
	}

	return nil
}

func init() {
	SchemeBuilder.Register(&PostgreSQLDatabase{}, &PostgreSQLDatabaseList{})
}
