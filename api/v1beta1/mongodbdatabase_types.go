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

// MongoDBDatabaseSpec defines the desired state of MongoDBDatabase
type MongoDBDatabaseSpec struct {
	*DatabaseSpec `json:",inline"`
	AtlasGroupId  string `json:"atlasGroupId,omitempty"`
}

// GetStatusConditions returns a pointer to the Status.Conditions slice
func (in *MongoDBDatabase) GetStatusConditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

// MongoDBDatabaseStatus defines the observed state of MongoDBDatabase
// IMPORTANT: Run "make" to regenerate code after modifying this file
type MongoDBDatabaseStatus struct {
	// Conditions holds the conditions for the MongoDBDatabase.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration is the last generation reconciled by the controller
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +genclient
// +genclient:Namespaced
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=mdb
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"DatabaseReady\")].status",description=""
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type==\"DatabaseReady\")].message",description=""
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description=""

// MongoDBDatabase is the Schema for the mongodbs API
type MongoDBDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MongoDBDatabaseSpec   `json:"spec,omitempty"`
	Status MongoDBDatabaseStatus `json:"status,omitempty"`
}

func (in *MongoDBDatabase) GetRootSecret() *SecretReference {
	if in.Spec.RootSecret.Namespace == "" {
		in.Spec.RootSecret.Namespace = in.GetNamespace()
	}

	return in.Spec.RootSecret
}

func (in *MongoDBDatabase) GetDatabaseName() string {
	if in.Spec.DatabaseName != "" {
		return in.Spec.DatabaseName
	}

	return in.GetName()
}

func (in *MongoDBDatabase) GetRootDatabaseName() string {
	return ""
}

// +kubebuilder:object:root=true

// MongoDBDatabaseList contains a list of MongoDBDatabase
type MongoDBDatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MongoDBDatabase `json:"items"`
}

func (d *MongoDBDatabase) SetDefaults() error {
	if d.Spec.DatabaseName == "" {
		d.Spec.DatabaseName = d.GetName()
	}

	return nil
}

func init() {
	SchemeBuilder.Register(&MongoDBDatabase{}, &MongoDBDatabaseList{})
}
