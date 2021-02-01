package v1beta1

type StatusCode string

const (
	Available   StatusCode = "Available"
	Unavailable            = "Unavailable"
	Pending                = "Pending"
)

type CredentialsStatus []*CredentialStatus

func (cs *CredentialsStatus) ForEach(consumer func(*CredentialStatus)) {
	for _, status := range *cs {
		consumer(status)
	}
}

func (cs *CredentialsStatus) Filter(predicate func(*CredentialStatus) bool) *CredentialStatus {
	for _, status := range *cs {
		if predicate(status) {
			return status
		}
	}
	return nil
}

func (cs *CredentialsStatus) FindOrCreate(name string, predicate func(status *CredentialStatus) bool) *CredentialStatus {
	postgresqlCredentialStatus := cs.Filter(predicate)
	if postgresqlCredentialStatus == nil {
		postgresqlCredentialStatus = &CredentialStatus{
			Username: name,
		}
		*cs = append(*cs, postgresqlCredentialStatus)
	}
	return postgresqlCredentialStatus
}

type CredentialStatus struct {
	Status   StatusCode `json:"status"`
	Message  string     `json:"message"`
	Username string     `json:"username"`
}

func (s *CredentialStatus) SetCredentialsStatus(code StatusCode, message string) {
	s.Status = code
	s.Message = message
}

type DatabaseStatus struct {
	Status  StatusCode `json:"status"`
	Message string     `json:"message"`
	Name    string     `json:"name"`
}

func (s *DatabaseStatus) SetDatabaseStatus(code StatusCode, message string, name string) {
	s.Status = code
	s.Message = message
	if name != "" {
		s.Name = name
	}
}
