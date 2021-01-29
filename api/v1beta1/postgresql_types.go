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

const DEFAULT_POSTGRESQL_ROOT_USER = "postgres"

type PostgreSQLDBName string
type PostgreSQLHostName string

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
	// Database name
	DatabaseName PostgreSQLDBName `json:"databaseName"`
	// Database Server host name
	Host PostgreSQLHostName `json:"host"`
	Port int64              `json:"port"`
	// +optional
	RootUsername     string                     `json:"rootUsername"`
	RootSecretLookup PostgreSQLRootSecretLookup `json:"rootSecretLookup"`
	// Database credentials
	Credentials PostgreSQLCredentials `json:"credentials"`
}

type PostgreSQLStatusCode string

const (
	PostgreSQLAvailable   PostgreSQLStatusCode = "Available"
	PostgreSQLUnavailable                      = "Unavailable"
	PostgreSQLPending                          = "Pending"
)

type PostgreSQLDatabaseStatus struct {
	Status  PostgreSQLStatusCode `json:"status"`
	Message string               `json:"message"`
	Name    PostgreSQLDBName     `json:"name"`
}

type PostgreSQLCredentialsStatus []*PostgreSQLCredentialStatus
type PostgreSQLCredentialStatus struct {
	Status   PostgreSQLStatusCode `json:"status"`
	Message  string               `json:"message"`
	Username string               `json:"username"`
}

// PostgreSQLStatus defines the observed state of PostgreSQL
// IMPORTANT: Run "make" to regenerate code after modifying this file
type PostgreSQLStatus struct {
	DatabaseStatus    PostgreSQLDatabaseStatus    `json:"database"`
	CredentialsStatus PostgreSQLCredentialsStatus `json:"credentials"`
	LastUpdateTime    metav1.Time                 `json:"lastUpdateTime"`
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

func (statuses *PostgreSQLCredentialsStatus) ForEach(consumer func(*PostgreSQLCredentialStatus)) {
	for _, status := range *statuses {
		consumer(status)
	}
}

func (statuses *PostgreSQLCredentialsStatus) Filter(predicate func(*PostgreSQLCredentialStatus) bool) *PostgreSQLCredentialStatus {
	for _, status := range *statuses {
		if predicate(status) {
			return status
		}
	}
	return nil
}

/*
	Alignes credentials status with spec by removing unneeded statuses. Mutates the original.
	Returns removed statuses.
*/
func (postgresql *PostgreSQL) RemoveUnneededCredentialsStatus() *PostgreSQLCredentialsStatus {
	removedStatuses := make(PostgreSQLCredentialsStatus, 0)
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

func (statuses *PostgreSQLCredentialsStatus) FindOrCreate(name string, predicate func(status *PostgreSQLCredentialStatus) bool) *PostgreSQLCredentialStatus {
	postgresqlCredentialStatus := statuses.Filter(predicate)
	if postgresqlCredentialStatus == nil {
		postgresqlCredentialStatus = &PostgreSQLCredentialStatus{
			Username: name,
		}
		*statuses = append(*statuses, postgresqlCredentialStatus)
	}
	return postgresqlCredentialStatus
}

func (this *PostgreSQLCredentialStatus) SetCredentialsStatus(code PostgreSQLStatusCode, message string) {
	this.Status = code
	this.Message = message
}

func (this *PostgreSQLStatus) SetDatabaseStatus(code PostgreSQLStatusCode, message string, name *PostgreSQLDBName) {
	this.DatabaseStatus.Status = code
	this.DatabaseStatus.Message = message
	if name != nil {
		this.DatabaseStatus.Name = *name
	}
}

func (this *PostgreSQL) SetDefaults() error {
	if this.Spec.RootUsername == "" {
		this.Spec.RootUsername = DEFAULT_POSTGRESQL_ROOT_USER
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
		this.Status.CredentialsStatus = make([]*PostgreSQLCredentialStatus, 0)
	}
	return nil
}

func init() {
	SchemeBuilder.Register(&PostgreSQL{}, &PostgreSQLList{})
}
