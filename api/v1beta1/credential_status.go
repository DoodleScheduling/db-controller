package v1beta1

type CredentialStatus struct {
	Username string             `json:"username"`
	Status   AvailabilityStatus `json:"status"`
}
