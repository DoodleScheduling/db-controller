package v1beta1

type Credential struct {
	UserName string `json:"username"`
	Vault    Vault  `json:"vault"`
}
