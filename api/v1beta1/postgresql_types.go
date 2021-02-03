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
	"errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// defaults
const (
	DEFAULT_POSTGRESQL_ROOT_USER                    = "postgres"
	DEFAULT_POSTGRESQL_ROOT_AUTHENTICATION_DATABASE = "postgres"
)

type PostgreSQLRootSecretLookup struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Field     string `json:"field"`
}

type PostgreSQLCredentials []PostgreSQLCredential
type PostgreSQLCredential struct {
	UserName string `json:"username"`
	Vault    Vault  `json:"vault"`
}

// PostgreSQLSpec defines the desired state of PostgreSQL
// IMPORTANT: Run "make" to regenerate code after modifying this file
type PostgreSQLSpec struct {
	DatabaseName string `json:"databaseName"`
	HostName     string `json:"hostName"`
	// +optional
	RootUsername string `json:"rootUsername"`
	// +optional
	RootAuthenticationDatabase string                     `json:"rootAuthDatabase"`
	RootSecretLookup           PostgreSQLRootSecretLookup `json:"rootSecretLookup"`
	Credentials                PostgreSQLCredentials      `json:"credentials"`
}

// PostgreSQLStatus defines the observed state of PostgreSQL
// IMPORTANT: Run "make" to regenerate code after modifying this file
type PostgreSQLStatus struct {
	DatabaseStatus    DatabaseStatus    `json:"database"`
	CredentialsStatus CredentialsStatus `json:"credentials"`
	LastUpdateTime    metav1.Time       `json:"lastUpdateTime"`
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

/*
	Alignes credentials status with spec by removing unneeded statuses. Mutates the original.
	Returns removed statuses.
*/
func (postgresql *PostgreSQL) RemoveUnneededCredentialsStatus() *CredentialsStatus {
	removedStatuses := make(CredentialsStatus, 0)
	statuses := &postgresql.Status.CredentialsStatus
	for i := 0; i < len(*statuses); i++ {
		status := (*statuses)[i]
		found := false
		if status != nil {
			for _, credential := range postgresql.Spec.Credentials {
				if credential.UserName == status.Username {
					found = true
				}
			}
		}
		if !found {
			removedStatuses = append(removedStatuses, status)
			s := append((*statuses)[:i], (*statuses)[i+1:]...)
			statuses = &s
			i--
		}
	}
	postgresql.Status.CredentialsStatus = *statuses
	return &removedStatuses
}

func (this *PostgreSQL) SetDefaults() error {
	if this.Spec.RootUsername == "" {
		this.Spec.RootUsername = DEFAULT_POSTGRESQL_ROOT_USER
	}
	if this.Spec.RootAuthenticationDatabase == "" {
		this.Spec.RootAuthenticationDatabase = DEFAULT_POSTGRESQL_ROOT_AUTHENTICATION_DATABASE
	}
	if this.Spec.RootSecretLookup.Name == "" {
		return errors.New("must specify root secret")
	}
	if this.Spec.RootSecretLookup.Field == "" {
		return errors.New("must specify root secret field")
	}
	if this.Spec.RootSecretLookup.Namespace == "" {
		this.Spec.RootSecretLookup.Namespace = this.ObjectMeta.Namespace
	}
	if this.Status.CredentialsStatus == nil || len(this.Status.CredentialsStatus) == 0 {
		this.Status.CredentialsStatus = make([]*CredentialStatus, 0)
	}
	return nil
}

func init() {
	SchemeBuilder.Register(&PostgreSQL{}, &PostgreSQLList{})
}
