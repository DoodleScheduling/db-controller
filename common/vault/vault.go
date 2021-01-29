package vault

type VaultRequest struct {
	Path        string
	UserField   string
	SecretField string
}

type VaultResponse struct {
	User   string
	Secret string
}

type Vault struct {
	Host string
}

func NewVault(host string) (*Vault, error) {
	// TODO implement Vault integration
	return &Vault{
		Host: host,
	}, nil
}

func (v *Vault) Get(r VaultRequest) (VaultResponse, error) {
	return VaultResponse{
		User:   "",
		Secret: "password",
	}, nil
}
