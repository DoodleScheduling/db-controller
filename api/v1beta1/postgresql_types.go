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

type PostgreSQLDBName string
type PostgreSQLHostName string

type PostgreSQLRootCredential struct {
	UserName string `json:"username"`
	Password string `json:"password"` // TODO just for testing, this MUST be replaced with secretRef lookup
}

type PostgreSQLCredentials []PostgreSQLCredential
type PostgreSQLCredential struct {
	UserName string `json:"username"`
	Vault    Vault  `json:"vault"`
}

// PostgreSQLSpec defines the desired state of PostgreSQL
// IMPORTANT: Run "make" to regenerate code after modifying this file
type PostgreSQLSpec struct {
	// Database name
	DatabaseName PostgreSQLDBName `json:"databaseName"`
	// Database Server host name
	Host           PostgreSQLHostName       `json:"host"`
	Port           int64                    `json:"port"`
	RootCredential PostgreSQLRootCredential `json:"root"`
	// Database credentials
	Credentials PostgreSQLCredentials `json:"credentials"`
}

type PostgreSQLStatusCode string

const (
	PostgreSQLDatabaseAvailable   PostgreSQLStatusCode = "Available"
	PostgreSQLDatabaseUnavailable                      = "Unavailable"
	PostgreSQLDatabasePending                          = "Pending"
)

type PostgreSQLDatabaseStatus struct {
	Status  PostgreSQLStatusCode `json:"status"`
	Message string               `json:"message"`
	Name    PostgreSQLDBName     `json:"name"`
}

type PostgreSQLCredentialsStatus []PostgreSQLCredentialStatus
type PostgreSQLCredentialStatus struct {
	Status   PostgreSQLStatusCode `json:"status"`
	Username string               `json:"username"`
}

// PostgreSQLStatus defines the observed state of PostgreSQL
// IMPORTANT: Run "make" to regenerate code after modifying this file
type PostgreSQLStatus struct {
	DatabaseStatus    PostgreSQLDatabaseStatus   `json:"database"`
	CredentialsStatus PostgreSQLCredentialStatus `json:"credentials"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PostgreSQL is the Schema for the postgresqls API
type PostgreSQL struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgreSQLSpec   `json:"spec,omitempty"`
	Status PostgreSQLStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PostgreSQLList contains a list of PostgreSQL
type PostgreSQLList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgreSQL `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PostgreSQL{}, &PostgreSQLList{})
}
