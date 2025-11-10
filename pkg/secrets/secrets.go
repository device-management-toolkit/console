package secrets

import (
	"context"
	"fmt"
)

type SecretData map[string]interface{}

type Secret struct {
	Path     string
	Data     SecretData
	Version  int
	Metadata map[string]interface{}
}

// writes a secret to Vault (KV v2)
func (c *Client) WriteSecret(ctx context.Context, secretPath string, data SecretData) error {
	if secretPath == "" {
		return fmt.Errorf("secret path cannot be empty")
	}

	if len(data) == 0 {
		return fmt.Errorf("secret data cannot be empty")
	}

	secretData := map[string]interface{}{
		"data": data,
	}

	_, err := c.client.Logical().WriteWithContext(ctx, secretPath, secretData)
	if err != nil {
		return err
	}

	return nil
}

// reads a secret from Vault (KV v2)
func (c *Client) ReadSecret(ctx context.Context, secretPath string) (*Secret, error) {
	if secretPath == "" {
		return nil, fmt.Errorf("secret path cannot be empty")
	}

	vaultSecret, err := c.client.Logical().ReadWithContext(ctx, secretPath)
	if err != nil {
		return nil, err
	}

	if vaultSecret == nil {
		return nil, fmt.Errorf("secret not found at path: %s", secretPath)
	}

	// Extract data from KV v2 response
	data, ok := vaultSecret.Data["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected secret data format at %s", secretPath)
	}

	// Extract metadata
	metadata, _ := vaultSecret.Data["metadata"].(map[string]interface{})

	// Extract version
	version := 0
	if meta, ok := metadata["version"]; ok {
		if v, ok := meta.(int); ok {
			version = v
		}
	}

	return &Secret{
		Path:     secretPath,
		Data:     data,
		Version:  version,
		Metadata: metadata,
	}, nil
}

// Simple key-value operations implementing security.Storager interface.
// For Vault, these use a default path "secret/data/console" to mimic simple key-value behavior.

// GetKeyValue reads a single key from the default Vault secret path.
func (c *Client) GetKeyValue(key string) (string, error) {
	return c.GetSecretValue(context.Background(), "secret/data/console", key)
}

// SetKeyValue writes a single key-value pair to the default Vault secret path.
func (c *Client) SetKeyValue(key, value string) error {
	return c.SetSecretValue(context.Background(), "secret/data/console", key, value)
}

// DeleteKeyValue deletes a specific key from the default Vault secret path.
func (c *Client) DeleteKeyValue(key string) error {
	// For Vault KV v2, we need to preserve other keys, so we read first
	secret, err := c.ReadSecret(context.Background(), "secret/data/console")
	if err != nil {
		return err
	}

	delete(secret.Data, key)
	return c.WriteSecret(context.Background(), "secret/data/console", secret.Data)
}

// Remote secret operations implementing security.Storager interface.
// These are the native Vault operations with full path control.

// GetSecret reads a secret from Vault and returns the data map.
func (c *Client) GetSecret(ctx context.Context, path string) (map[string]interface{}, error) {
	secret, err := c.ReadSecret(ctx, path)
	if err != nil {
		return nil, err
	}
	return secret.Data, nil
}

// SetSecret writes a secret to Vault from a data map.
func (c *Client) SetSecret(ctx context.Context, path string, data map[string]interface{}) error {
	return c.WriteSecret(ctx, path, data)
}

// DeleteSecret deletes a secret at the given Vault path.
func (c *Client) DeleteSecret(ctx context.Context, path string) error {
	if path == "" {
		return fmt.Errorf("secret path cannot be empty")
	}

	_, err := c.client.Logical().DeleteWithContext(ctx, path)
	return err
}

// GetSecretValue reads a specific key from a Vault secret.
func (c *Client) GetSecretValue(ctx context.Context, path, key string) (string, error) {
	secret, err := c.ReadSecret(ctx, path)
	if err != nil {
		return "", err
	}

	value, ok := secret.Data[key]
	if !ok {
		return "", fmt.Errorf("key %s not found in secret at path %s", key, path)
	}

	strValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("value for key %s is not a string", key)
	}

	return strValue, nil
}

// SetSecretValue writes a specific key-value pair to a Vault secret.
func (c *Client) SetSecretValue(ctx context.Context, path, key, value string) error {
	// First try to read existing secret to preserve other keys
	secret, err := c.ReadSecret(ctx, path)
	data := make(map[string]interface{})

	if err == nil {
		// Secret exists, preserve existing data
		data = secret.Data
	}

	// Set or update the specific key
	data[key] = value

	return c.WriteSecret(ctx, path, data)
}
