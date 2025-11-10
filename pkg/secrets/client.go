package secrets

import (
	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"
	"github.com/hashicorp/vault/api"
)

// Client implements the security.Storager interface for HashiCorp Vault.
type Client struct {
	client *api.Client
}

// Ensure Client implements security.Storager interface.
var _ security.Storager = (*Client)(nil)

// NewVaultClient creates a new Vault Client instance with an existing Vault API client (for testing).
func NewVaultClient(vaultClient *api.Client) *Client {
	return &Client{client: vaultClient}
}

// NewClient creates a new Vault Client instance from configuration (production use).
func NewClient(cfg *config.Secrets) (*Client, error) {
	vaultConfig := api.DefaultConfig()
	vaultConfig.Address = cfg.Address
	client, err := api.NewClient(vaultConfig)
	if err != nil {
		return nil, err
	}
	client.SetToken(cfg.Token)
	return &Client{client: client}, nil
}
