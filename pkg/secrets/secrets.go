package secrets

import (
	"context"
	"fmt"
)

// GetKeyValue reads a single key from the default secret path.
func (c *Client) GetKeyValue(key string) (string, error) {
	ctx := context.Background()
	secretPath := "secret/data/console"

	secret, err := c.client.Logical().ReadWithContext(ctx, secretPath)
	if err != nil {
		return "", err
	}

	if secret == nil {
		return "", fmt.Errorf("secret not found at path: %s", secretPath)
	}

	// Extract data from KV v2 response
	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected secret data format at %s", secretPath)
	}

	value, ok := data[key]
	if !ok {
		return "", fmt.Errorf("key %s not found in secret at path %s", key, secretPath)
	}

	strValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("value for key %s is not a string", key)
	}

	return strValue, nil
}

// SetKeyValue writes a single key-value pair to the default secret path.
func (c *Client) SetKeyValue(key, value string) error {
	ctx := context.Background()
	secretPath := "secret/data/console"

	// Read existing secret to preserve other keys
	secret, err := c.client.Logical().ReadWithContext(ctx, secretPath)
	data := make(map[string]interface{})

	if err == nil && secret != nil {
		// Secret exists, preserve existing data
		if d, ok := secret.Data["data"].(map[string]interface{}); ok {
			data = d
		}
	}

	// Set or update the specific key
	data[key] = value

	secretData := map[string]interface{}{
		"data": data,
	}

	_, err = c.client.Logical().WriteWithContext(ctx, secretPath, secretData)
	if err != nil {
		return err
	}

	return nil
}

// DeleteKeyValue deletes a specific key from the default secret path.
func (c *Client) DeleteKeyValue(key string) error {
	ctx := context.Background()
	secretPath := "secret/data/console"

	// Read existing secret
	secret, err := c.client.Logical().ReadWithContext(ctx, secretPath)
	if err != nil {
		return err
	}

	if secret == nil {
		return fmt.Errorf("secret not found at path: %s", secretPath)
	}

	// Extract data from KV v2 response
	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected secret data format at %s", secretPath)
	}

	delete(data, key)

	secretData := map[string]interface{}{
		"data": data,
	}

	_, err = c.client.Logical().WriteWithContext(ctx, secretPath, secretData)
	return err
}
