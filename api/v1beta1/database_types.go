package v1beta1

import (
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Finalizer
const (
	Finalizer = "infra.finalizers.doodle.com"
)

// Status conditions
const (
	DatabaseReadyConditionType  = "DatabaseReady"
	UserReadyConditionType      = "UserReady"
	ExtensionReadyConditionType = "ExtensionReady"
	SchemaReadyConditionType    = "SchemaReady"
)

// Status reasons
const (
	SecretNotFoundReason                 = "SecretNotFoundFailed"
	ConnectionFailedReason               = "ConnectionFailed"
	DatabaseProvisioningFailedReason     = "DatabaseProvisioningFailed"
	DatabaseProvisioningSuccessfulReason = "DatabaseProvisiningSuccessful"
	DatabaseNotFoundReason               = "DatabaseNotFoundReason"
	UserNotProvisionedReason             = "UserNotProvisioned"
	UserProvisioningSuccessfulReason     = "UserProvisioningSuccessful"
	CredentialsNotFoundReason            = "CredentialsNotFound"
	CreateDatabaseFailedReason           = "CreateDatabaseFailed"
	CreateExtensionsFailedReason         = "CreateExtensionsFailed"
	CreateSchemasFailedReason            = "CreateSchemasFailed"
	ProgressingReason                    = "ProgressingReason"
	CreateExtensionsSuccessfulReason     = "CreateExtensionsSuccessful"
	CreateSchemasSuccessfulReason        = "CreateSchemasSuccessful"
)

// DatabaseSpec defines the desired state of a *Database
type DatabaseSpec struct {
	// DatabaseName is by default the same as metata.name
	// +optional
	DatabaseName string `json:"databaseName"`

	// The connect URI
	// +optional
	Address string `json:"address,omitempty"`

	// Contains a credentials set of a user with enough permission to manage databases and user accounts
	// +required
	RootSecret *SecretReference `json:"rootSecret"`
}

// DatabaseReference is a named reference to a database kind
type DatabaseReference struct {
	// Name referrs to the name of the database kind, mist be located within the same namespace
	// +required
	Name string `json:"name"`
}

// SecretReference is a named reference to a secret which contains user credentials
type SecretReference struct {
	// Name referrs to the name of the secret, must be located whithin the same namespace
	// +required
	Name string `json:"name"`

	// Namespace, by default the same namespace is used.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// +optional
	// +kubebuilder:default:=username
	UserField string `json:"userField"`

	// +optional
	// +kubebuilder:default:=password
	PasswordField string `json:"passwordField"`
}

// conditionalResource is a resource with conditions
type conditionalResource interface {
	GetStatusConditions() *[]metav1.Condition
}

func DatabaseNotReadyCondition(in conditionalResource, reason, message string) {
	setResourceCondition(in, DatabaseReadyConditionType, metav1.ConditionFalse, reason, message)
}

func DatabaseReadyCondition(in conditionalResource, reason, message string) {
	setResourceCondition(in, DatabaseReadyConditionType, metav1.ConditionTrue, reason, message)
}

func UserNotReadyCondition(in conditionalResource, reason, message string) {
	setResourceCondition(in, UserReadyConditionType, metav1.ConditionFalse, reason, message)
}

func UserReadyCondition(in conditionalResource, reason, message string) {
	setResourceCondition(in, UserReadyConditionType, metav1.ConditionTrue, reason, message)
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
