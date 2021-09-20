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
	"github.com/doodlescheduling/k8sdb-controller/common/stringutils"
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

func (in *MongoDBDatabase) GetAtlasGroupId() string {
	return in.Spec.AtlasGroupId
}

func (in *MongoDBDatabase) GetAddress() string {
	return in.Spec.Address
}

func (in *MongoDBDatabase) GetRootSecret() *SecretReference {
	return in.Spec.RootSecret
}

func (in *MongoDBDatabase) GetDatabaseName() string {
	if in.Spec.DatabaseName != "" {
		return in.Spec.DatabaseName
	}

	return in.GetName()
}

func (in *MongoDBDatabase) MigrationRequired() bool {
	if in.Spec.Migration == nil {
		return false
	}

	return true
}

func (in *MongoDBDatabase) GetMigrationAddress() string {
	if in.Spec.Migration == nil {
		return ""
	}

	if in.Spec.Migration.Address == "" {
		return in.Spec.Address
	}

	return in.Spec.Migration.Address
}

func (in *MongoDBDatabase) GetMigrationRootSecret() *SecretReference {
	if in.Spec.Migration == nil {
		return nil
	}

	if in.Spec.Migration.RootSecret == nil {
		return in.GetRootSecret()
	}

	return in.Spec.Migration.RootSecret
}

func (in *MongoDBDatabase) GetMigrationDatabaseName() string {
	if in.Spec.Migration == nil {
		return ""
	}

	if in.Spec.Migration.DatabaseName == "" {
		return in.GetDatabaseName()
	}

	return in.Spec.Migration.DatabaseName
}

func (in *MongoDBDatabase) GetMigrationWorkloads() []WorkloadReference {
	if in.Spec.Migration == nil {
		return []WorkloadReference{}
	}

	return in.Spec.Migration.Workloads
}

func (in *MongoDBDatabase) GetRootDatabaseName() string {
	return ""
}

func (in *MongoDBDatabase) GetExtensions() Extensions {
	return in.Spec.Extensions
}

// +kubebuilder:object:root=true

// MongoDBDatabaseList contains a list of MongoDBDatabase
type MongoDBDatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MongoDBDatabase `json:"items"`
}

// If object doesn't contain finalizer, set it and call update function 'updateF'.
// Only do this if object is not being deleted (judged by DeletionTimestamp being zero)
func (d *MongoDBDatabase) SetFinalizer(updateF func() error) error {
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
func (d *MongoDBDatabase) Finalize(updateF func() error, finalizeF func() error) (bool, error) {
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

func (d *MongoDBDatabase) SetDefaults() error {
	if d.Spec.DatabaseName == "" {
		d.Spec.DatabaseName = d.GetName()
	}

	return nil
}

func init() {
	SchemeBuilder.Register(&MongoDBDatabase{}, &MongoDBDatabaseList{})
}
