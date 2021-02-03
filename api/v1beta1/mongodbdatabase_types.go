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
	DEFAULT_MONGODB_ROOT_USER                    = "root"
	DEFAULT_MONGODB_ROOT_AUTHENTICATION_DATABASE = "admin"
)

type MongoDBDatabaseRootSecretLookup struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Field     string `json:"field"`
}

type MongoDBDatabaseCredentials []MongoDBDatabaseCredential
type MongoDBDatabaseCredential struct {
	UserName string `json:"username"`
	Vault    Vault  `json:"vault"`
}

// MongoDBDatabaseSpec defines the desired state of MongoDBDatabase
// IMPORTANT: Run "make" to regenerate code after modifying this file
type MongoDBDatabaseSpec struct {
	DatabaseName string `json:"databaseName"`
	HostName     string `json:"hostName"`
	// +optional
	RootUsername string `json:"rootUsername"`
	// +optional
	RootAuthenticationDatabase string                          `json:"rootAuthDatabase"`
	RootSecretLookup           MongoDBDatabaseRootSecretLookup `json:"rootSecretLookup"`
	Credentials                MongoDBDatabaseCredentials      `json:"credentials"`
}

// MongoDBDatabaseStatus defines the observed state of MongoDBDatabase
// IMPORTANT: Run "make" to regenerate code after modifying this file
type MongoDBDatabaseStatus struct {
	DatabaseStatus    DatabaseStatus    `json:"database"`
	CredentialsStatus CredentialsStatus `json:"credentials"`
	LastUpdateTime    metav1.Time       `json:"lastUpdateTime"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// MongoDBDatabase is the Schema for the mongodbs API
type MongoDBDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MongoDBDatabaseSpec   `json:"spec,omitempty"`
	Status MongoDBDatabaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MongoDBDatabaseList contains a list of MongoDBDatabase
type MongoDBDatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MongoDBDatabase `json:"items"`
}

/*
	Alignes credentials status with spec by removing unneeded statuses. Mutates the original.
	Returns removed statuses.
*/
func (d *MongoDBDatabase) RemoveUnneededCredentialsStatus() *CredentialsStatus {
	removedStatuses := make(CredentialsStatus, 0)
	statuses := &d.Status.CredentialsStatus
	for i := 0; i < len(*statuses); i++ {
		status := (*statuses)[i]
		found := false
		if status != nil {
			for _, credential := range d.Spec.Credentials {
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
	d.Status.CredentialsStatus = *statuses
	return &removedStatuses
}

func (d *MongoDBDatabase) SetDefaults() error {
	if d.Spec.RootUsername == "" {
		d.Spec.RootUsername = DEFAULT_MONGODB_ROOT_USER
	}
	if d.Spec.RootAuthenticationDatabase == "" {
		d.Spec.RootAuthenticationDatabase = DEFAULT_MONGODB_ROOT_AUTHENTICATION_DATABASE
	}
	if d.Spec.RootSecretLookup.Name == "" {
		return errors.New("must specify root secret")
	}
	if d.Spec.RootSecretLookup.Field == "" {
		return errors.New("must specify root secret field")
	}
	if d.Spec.RootSecretLookup.Namespace == "" {
		d.Spec.RootSecretLookup.Namespace = d.ObjectMeta.Namespace
	}
	if d.Status.CredentialsStatus == nil || len(d.Status.CredentialsStatus) == 0 {
		d.Status.CredentialsStatus = make([]*CredentialStatus, 0)
	}
	return nil
}

func init() {
	SchemeBuilder.Register(&MongoDBDatabase{}, &MongoDBDatabaseList{})
}
