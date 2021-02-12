package v1beta1

import (
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Status conditions
const (
	ProvisionedCondition = "Provisioned"
)

// Status reasons
const (
	SecretNotFoundReason                = "SecretNotFoundFailed"
	ConnectionFailedReason              = "ConnectionFailed"
	DatabaseProvisioningFailedReason    = "DatabaseProvisioningFailed"
	DatabaseProvisiningSuccessfulReason = "DatabaseProvisiningSuccessful"
	DatabaseNotFoundReason              = "DatabaseNotFoundReason"
	UserNotProvisionedReason            = "UserNotProvisioned"
	UserProvisioningSuccessfulReason    = "UserProvisioningSuccessful"
	CredentialsNotFoundReason           = "CredentialsNotFound"
)

// DatabaseSpec defines the desired state of MongoDBDatabase
type DatabaseSpec struct {
	// The name of the database, if not set the name is taken from metadata.name
	// +optional
	DatabaseName string `json:"databaseName"`

	// The MongoDB URI
	// +required
	Address string `json:"address"`

	// +required
	RootSecret *SecretReference `json:"rootSecret"`
}

type DatabaseReference struct {
	Name string `json:"name"`
}

type SecretReference struct {
	// +required
	Name string `json:"name"`

	// +optional
	UserField string `json:"userField"`

	// +required
	PasswordField string `json:"passwordField"`
}

// ConditionalResource is a resource with conditions
type conditionalResource interface {
	GetStatusConditions() *[]metav1.Condition
}

func NotProvisioned(in conditionalResource, reason, message string) {
	setResourceCondition(in, ProvisionedCondition, metav1.ConditionFalse, reason, message)
}

func Provisioned(in conditionalResource, reason, message string) {
	setResourceCondition(in, ProvisionedCondition, metav1.ConditionTrue, reason, message)
}

// setResourceCondition sets the given condition with the given status,
// reason and message on a resource.
func setResourceCondition(resource conditionalResource, condition string, status metav1.ConditionStatus, reason, message string) {
	conditions := resource.GetStatusConditions()

	newCondition := metav1.Condition{
		Type:    condition,
		Status:  status,
		Reason:  reason,
		Message: message,
	}

	apimeta.SetStatusCondition(conditions, newCondition)
}
