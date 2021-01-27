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

type MongoDBDBName string
type MongoDBHostName string
type MongoDBCredentials []MongoDBCredential
type MongoDBCredential struct {
	UserName string `json:"username"`
	Vault    Vault  `json:"vault"`
}

// MongoDBSpec defines the desired state of MongoDB
// IMPORTANT: Run "make" to regenerate code after modifying this file
type MongoDBSpec struct {
	// Database name
	DBName MongoDBDBName `json:"dbName"`
	// Database Server host name
	HostName MongoDBHostName `json:"hostName"`
	// Database credentials
	Credentials MongoDBCredentials `json:"credentials"`
}

type MongoDBStatusCode string

const (
	MongoDBDatabaseAvailable   MongoDBStatusCode = "Available"
	MongoDBDatabaseUnavailable                   = "Unavailable"
	MongoDBDatabasePending                       = "Pending"
)

type MongoDBDatabaseStatus struct {
	Status MongoDBStatusCode `json:"status"`
	DBName MongoDBDBName     `json:"dbName"`
}

type MongoDBCredentialsStatus []MongoDBCredentialStatus
type MongoDBCredentialStatus struct {
	Status   MongoDBStatusCode `json:"status"`
	Username string            `json:"username"`
}

// MongoDBStatus defines the observed state of MongoDB
// IMPORTANT: Run "make" to regenerate code after modifying this file
type MongoDBStatus struct {
	MongoDBAvailabilityStatus MongoDBDatabaseStatus    `json:"database"`
	CredentialsStatus         MongoDBCredentialsStatus `json:"credentials"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// MongoDB is the Schema for the mongodbs API
type MongoDB struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MongoDBSpec   `json:"spec,omitempty"`
	Status MongoDBStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MongoDBList contains a list of MongoDB
type MongoDBList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MongoDB `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MongoDB{}, &MongoDBList{})
}
