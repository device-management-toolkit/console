package secrets

import (
	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"
	"github.com/hashicorp/vault/api"
)

// Default path for secrets if not configured.
const DefaultSecretPath = "secret/data/console"

// ObjectStorager extends security.Storager with object storage capabilities.
// This allows storing structured data (like certificates) as proper JSON objects in Vault.
type ObjectStorager interface {
	security.Storager
	GetObject(key string) (map[string]string, error)
	SetObject(key string, data map[string]string) error
}

// Client implements the security.Storager interface for HashiCorp Vault.
type Client struct {
	client *api.Client
	path   string // Base path for all secrets (e.g., "secret/data/console")
}

// Ensure Client implements security.Storager and ObjectStorager interfaces.
var (
	_ security.Storager = (*Client)(nil)
	_ ObjectStorager    = (*Client)(nil)
)

// Option is a functional option for configuring the Client.
type Option func(*Client)

// WithPath sets a custom path for secrets storage.
func WithPath(path string) Option {
	return func(c *Client) {
		if path != "" {
			c.path = path
		}
	}
}

// WithClient sets a pre-configured Vault API client (useful for testing).
func WithClient(client *api.Client) Option {
	return func(c *Client) {
		c.client = client
	}
}

// NewClient creates a new Vault Client instance.
// For production: pass config to create a new API client.
// For testing: use WithClient option to inject a mock client.
func NewClient(cfg *config.Secrets, opts ...Option) (*Client, error) {
	c := &Client{
		path: DefaultSecretPath,
	}

	// Apply options first (allows WithClient to skip API client creation)
	for _, opt := range opts {
		opt(c)
	}

	// If no client was injected via options, create one from config
	if c.client == nil {
		vaultConfig := api.DefaultConfig()
		vaultConfig.Address = cfg.Address

		client, err := api.NewClient(vaultConfig)
		if err != nil {
			return nil, err
		}

		client.SetToken(cfg.Token)
		c.client = client
	}

	// Apply path from config if not set by options
	if cfg != nil && cfg.Path != "" && c.path == DefaultSecretPath {
		c.path = cfg.Path
	}

	return c, nil
}
