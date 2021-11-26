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

type MongoDBUserRole struct {
	Name string `json:"name"`

	// +optional
	DB string `json:"db,omitempty"`
}

type MongoDBUserSpec struct {
	// +required
	Database *DatabaseReference `json:"database"`

	// +required
	Credentials *SecretReference `json:"credentials"`

	// +optional
	// +kubebuilder:default:={{name: readWrite}}
	Roles *[]MongoDBUserRole `json:"roles"`
}

// GetStatusConditions returns a pointer to the Status.Conditions slice
func (in *MongoDBUser) GetStatusConditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

// MongoDBUserStatus defines the observed state of MongoDBUser
// IMPORTANT: Run "make" to regenerate code after modifying this file
type MongoDBUserStatus struct {
	// Conditions holds the conditions for the MongoDBUser.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Username of the created user.
	// +optional
	Username string `json:"username,omitempty"`
}

// +genclient
// +genclient:Namespaced
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=mdu
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"UserReady\")].status",description=""
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type==\"UserReady\")].message",description=""
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description=""

// MongoDBUser is the Schema for the mongodbs API
type MongoDBUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MongoDBUserSpec   `json:"spec,omitempty"`
	Status MongoDBUserStatus `json:"status,omitempty"`
}

func (in *MongoDBUser) GetDatabase() string {
	return in.Spec.Database.Name
}

func (in *MongoDBUser) GetCredentials() *SecretReference {
	sec := in.Spec.Credentials
	if sec.Namespace == "" {
		sec.Namespace = in.GetNamespace()
	}

	return sec
}

func (in *MongoDBUser) GetRoles() []MongoDBUserRole {
	if in.Spec.Roles == nil {
		return []MongoDBUserRole{}
	}

	return *in.Spec.Roles
}

// +kubebuilder:object:root=true

// MongoDBUserList contains a list of MongoDBUser
type MongoDBUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MongoDBUser `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MongoDBUser{}, &MongoDBUserList{})
}
