package common

import (
	"context"
	"errors"
	"github.com/doodlescheduling/kubedb/api/v1beta1"
	"github.com/doodlescheduling/kubedb/common/vault"
	"github.com/doodlescheduling/kubedb/common/vault/kubernetes"
	"github.com/go-logr/logr"
	"github.com/rs/xid"
	"os"

	vaultapi "github.com/hashicorp/vault/api"
)

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

func (v *Vault) Get(cred *DatabaseCredential, databaseName string, logger logr.Logger) (VaultResponse, error) {

	// goran vault
	//h, err := FromCredential(binding, logger)
	h, err := FromCredential(cred, logger)

	r := VaultRequest{
		Path:        cred.Vault.Path,
		UserField:   cred.Vault.UserField,
		SecretField: cred.Vault.SecretField,
	}

	data, response, err := processRequest(h, r, databaseName)

	if err != nil {
		return response, err
	}

	return VaultResponse{
		User:   data[r.UserField].(string),
		Secret: data[r.SecretField].(string),
	}, err
}

func processRequest(h *VaultHandler, r VaultRequest, databaseName string) (map[string]interface{}, VaultResponse, error) {
	data, err := h.Read(r.Path)
	if err != nil {
		return nil, VaultResponse{}, err
	}
	var rewrite = false
	_, existingField := data[r.UserField]
	if !existingField {
		data[r.UserField] = databaseName
		rewrite = true
	}
	_, existingField = data[r.SecretField]
	if !existingField {
		data[r.SecretField] = xid.New().String()
		rewrite = true
	}
	if rewrite {
		_, err = h.c.Logical().Write(r.Path, data)
		if err != nil {
			return nil, VaultResponse{}, err
		}
	}
	return data, VaultResponse{}, nil
}

// part from k8svault controller

// Common errors
var (
	ErrVaultAddrNotFound          = errors.New("Neither vault address nor a default vault address found")
	ErrK8sSecretFieldNotAvailable = errors.New("K8s secret field to be mapped does not exist")
	ErrUnsupportedAuthType        = errors.New("Unsupported vault authentication")
	ErrVaultConfig                = errors.New("Failed to setup default vault configuration")
)

func ConvertPostgreSQLDatabaseCredential(cred v1beta1.PostgreSQLDatabaseCredential) *DatabaseCredential {
	return &DatabaseCredential{
		UserName: cred.UserName,
		Vault:    cred.Vault,
	}
}

func ConvertMongoDBDatabaseCredential(cred v1beta1.MongoDBDatabaseCredential) *DatabaseCredential {
	return &DatabaseCredential{
		UserName: cred.UserName,
		Vault:    cred.Vault,
	}
}

// Setup vault client & authentication from binding
func setupAuth(h *VaultHandler) error {
	auth := vault.NewAuthHandler(&vault.AuthHandlerConfig{
		Logger: h.logger,
		Client: h.c,
	})

	var method vault.AuthMethod

	m, err := authKubernetes(h)
	if err != nil {
		return err
	}

	method = m

	if err := auth.Authenticate(context.TODO(), method); err != nil {
		return err
	}

	return nil
}

// Wrapper around vault kubernetes auth (taken from vault agent)
// Injects env variables if not set on the binding
func authKubernetes(h *VaultHandler) (vault.AuthMethod, error) {
	role := os.Getenv("VAULT_ROLE")
	tokenPath := os.Getenv("VAULT_TOKEN_PATH")

	return kubernetes.NewKubernetesAuthMethod(&vault.AuthConfig{
		Logger:    h.logger,
		MountPath: "/auth/kubernetes",
		Config: map[string]interface{}{
			"role":       role,
			"token_path": tokenPath,
		},
	})
}

// FromCredential creates a vault client handler
// If the binding holds no vault address it will fallback to the env VAULT_ADDRESS
func FromCredential(credential *DatabaseCredential, logger logr.Logger) (*VaultHandler, error) {
	cfg := vaultapi.DefaultConfig()

	if cfg == nil {
		return nil, ErrVaultConfig
	}

	if credential.Vault.Host != "" {
		cfg.Address = credential.Vault.Host
	}

	client, err := vaultapi.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	h := &VaultHandler{
		cfg:    cfg,
		c:      client,
		logger: logger,
	}

	logger.Info("setup vault client", "vault", cfg.Address)

	if err = setupAuth(h); err != nil {
		return nil, err
	}

	return h, nil
}

// VaultHandler
type VaultHandler struct {
	c      *vaultapi.Client
	cfg    *vaultapi.Config
	auth   *vault.AuthHandler
	logger logr.Logger
}

// Read vault path and return data map
// Return empty map if no data exists
func (h *VaultHandler) Read(path string) (map[string]interface{}, error) {
	s, err := h.c.Logical().Read(path)
	if err != nil {
		return nil, err
	}

	// Return empty map if no data exists
	if s == nil || s.Data == nil {
		return make(map[string]interface{}), nil
	}

	return s.Data, nil
}
