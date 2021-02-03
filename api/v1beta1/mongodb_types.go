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

type MongoDBRootSecretLookup struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Field     string `json:"field"`
}

type MongoDBCredentials []MongoDBCredential
type MongoDBCredential struct {
	UserName string `json:"username"`
	Vault    Vault  `json:"vault"`
}

// MongoDBSpec defines the desired state of MongoDB
// IMPORTANT: Run "make" to regenerate code after modifying this file
type MongoDBSpec struct {
	DatabaseName string `json:"databaseName"`
	HostName     string `json:"hostName"`
	// +optional
	RootUsername string `json:"rootUsername"`
	// +optional
	RootAuthenticationDatabase string                  `json:"rootAuthDatabase"`
	RootSecretLookup           MongoDBRootSecretLookup `json:"rootSecretLookup"`
	Credentials                MongoDBCredentials      `json:"credentials"`
}

// MongoDBStatus defines the observed state of MongoDB
// IMPORTANT: Run "make" to regenerate code after modifying this file
type MongoDBStatus struct {
	DatabaseStatus    DatabaseStatus    `json:"database"`
	CredentialsStatus CredentialsStatus `json:"credentials"`
	LastUpdateTime    metav1.Time       `json:"lastUpdateTime"`
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

/*
	Alignes credentials status with spec by removing unneeded statuses. Mutates the original.
	Returns removed statuses.
*/
func (mongodb *MongoDB) RemoveUnneededCredentialsStatus() *CredentialsStatus {
	removedStatuses := make(CredentialsStatus, 0)
	statuses := &mongodb.Status.CredentialsStatus
	for i := 0; i < len(*statuses); i++ {
		status := (*statuses)[i]
		found := false
		if status != nil {
			for _, credential := range mongodb.Spec.Credentials {
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
	mongodb.Status.CredentialsStatus = *statuses
	return &removedStatuses
}

func (this *MongoDB) SetDefaults() error {
	if this.Spec.RootUsername == "" {
		this.Spec.RootUsername = DEFAULT_MONGODB_ROOT_USER
	}
	if this.Spec.RootAuthenticationDatabase == "" {
		this.Spec.RootAuthenticationDatabase = DEFAULT_MONGODB_ROOT_AUTHENTICATION_DATABASE
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
	SchemeBuilder.Register(&MongoDB{}, &MongoDBList{})
}
