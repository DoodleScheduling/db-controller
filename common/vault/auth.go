// This package implements authentication for vault
// Most of this code within this package has been borrowed and shrinked from the vault client agent.
// See https://github.com/hashicorp/vault/blob/master/command/agent/auth/auth.go

// Note this package uses API which is in the vault stable release but it was not released in the api package,
// see https://github.com/hashicorp/vault/issues/10490

package vault

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-logr/logr"
	vaultapi "github.com/hashicorp/vault/api"
)

// AuthMethod is the interface that auto-auth methods implement for the agent
// to use.
type AuthMethod interface {
	// Authenticate returns a mount path, header, request body, and error.
	// The header may be nil if no special header is needed.
	Authenticate(context.Context, *vaultapi.Client) (string, http.Header, map[string]interface{}, error)
	NewCreds() chan struct{}
	CredSuccess()
	Shutdown()
}

// AuthMethodWithClient is an extended interface that can return an API client
// for use during the authentication call.
type AuthMethodWithClient interface {
	AuthMethod
	AuthClient(client *vaultapi.Client) (*vaultapi.Client, error)
}

type AuthConfig struct {
	Logger    logr.Logger
	MountPath string
	Config    map[string]interface{}
}

// AuthHandler is responsible for keeping a token alive and renewed and passing
// new tokens to the sink server
type AuthHandler struct {
	logger logr.Logger
	client *vaultapi.Client
}

type AuthHandlerConfig struct {
	Logger logr.Logger
	Client *vaultapi.Client
}

func NewAuthHandler(conf *AuthHandlerConfig) *AuthHandler {
	ah := &AuthHandler{
		logger: conf.Logger,
		client: conf.Client,
	}

	return ah
}

func (ah *AuthHandler) Authenticate(ctx context.Context, am AuthMethod) error {
	if am == nil {
		return errors.New("auth handler: nil auth method")
	}

	path, _, data, err := am.Authenticate(ctx, ah.client)

	if err != nil {
		ah.logger.Error(err, "error getting path or data from method")
		return err
	}

	var clientToUse *vaultapi.Client

	switch am.(type) {
	case AuthMethodWithClient:
		clientToUse, err = am.(AuthMethodWithClient).AuthClient(ah.client)
		if err != nil {
			ah.logger.Error(err, "error creating client for authentication call")
			return err
		}
	default:
		clientToUse = ah.client
	}

	/*for key, values := range header {
		for _, value := range values {
			clientToUse.AddHeader(key, value)
		}
	}*/

	secret, err := clientToUse.Logical().Write(path, data)

	// Check errors/sanity
	if err != nil {
		ah.logger.Error(err, "error authenticating")
		return err
	}

	if secret == nil || secret.Auth == nil {
		ah.logger.Error(err, "authentication returned nil auth info")
		return err
	}

	if secret.Auth.ClientToken == "" {
		ah.logger.Error(err, "authentication returned empty client token")
		return err
	}

	ah.logger.Info("authentication successful")
	ah.client.SetToken(secret.Auth.ClientToken)
	am.CredSuccess()

	return nil
}
