package v1beta1

type Vault struct {
	UseAvailable bool   `json:"useAvailable"`
	Host         string `json:"host"`
	Path         string `json:"path"`
	UserField    string `json:"userField"`
	SecretField  string `json:"secretField"`
}
