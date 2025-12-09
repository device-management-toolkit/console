package secrets

import (
	"context"
	"fmt"
	"strings"
)

// GetKeyValue reads a value from Vault.
// If the key contains "/", it's treated as a separate path: {basePath}/{key} with data stored under "value".
// Otherwise, it's stored in {basePath}/keys with the key as a field name.
func (c *Client) GetKeyValue(key string) (string, error) {
	ctx := context.Background()

	var secretPath string
	var dataKey string

	if strings.Contains(key, "/") {
		// Path-based storage: {basePath}/{key} with "value" field
		secretPath = c.path + "/" + key
		dataKey = "value"
	} else {
		// Key-based storage: {basePath}/keys with key as field name
		secretPath = c.path + "/keys"
		dataKey = key
	}

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

	value, ok := data[dataKey]
	if !ok {
		return "", fmt.Errorf("key %s not found in secret at path %s", dataKey, secretPath)
	}

	strValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("value for key %s is not a string", dataKey)
	}

	return strValue, nil
}

// SetKeyValue writes a value to Vault.
// If the key contains "/", it's treated as a separate path: {basePath}/{key} with data stored under "value".
// Otherwise, it's stored in {basePath}/keys with the key as a field name.
func (c *Client) SetKeyValue(key, value string) error {
	ctx := context.Background()

	var secretPath string
	var secretData map[string]interface{}

	if strings.Contains(key, "/") {
		// Path-based storage: {basePath}/{key} with "value" field
		secretPath = c.path + "/" + key
		secretData = map[string]interface{}{
			"data": map[string]interface{}{
				"value": value,
			},
		}
	} else {
		// Key-based storage: {basePath}/keys with key as field name
		secretPath = c.path + "/keys"

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

		secretData = map[string]interface{}{
			"data": data,
		}
	}

	_, err := c.client.Logical().WriteWithContext(ctx, secretPath, secretData)
	return err
}

// DeleteKeyValue deletes a value from Vault.
// If the key contains "/", it deletes the entire secret at {basePath}/{key}.
// Otherwise, it removes the key from {basePath}/keys.
func (c *Client) DeleteKeyValue(key string) error {
	ctx := context.Background()

	if strings.Contains(key, "/") {
		// Path-based storage: permanently delete the secret at {basePath}/{key}
		// For KV v2, we need to delete metadata to permanently remove (not just soft delete)
		// Convert secret/data/console/... to secret/metadata/console/...
		metadataPath := strings.Replace(c.path, "/data/", "/metadata/", 1) + "/" + key
		_, err := c.client.Logical().DeleteWithContext(ctx, metadataPath)
		return err
	}

	// Key-based storage: remove from {basePath}/keys
	secretPath := c.path + "/keys"

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

// GetObject retrieves a map of string values from a path-based secret.
// The key must contain "/" to specify the path: {basePath}/{key}.
func (c *Client) GetObject(key string) (map[string]string, error) {
	ctx := context.Background()
	secretPath := c.path + "/" + key

	secret, err := c.client.Logical().ReadWithContext(ctx, secretPath)
	if err != nil {
		return nil, err
	}

	if secret == nil {
		return nil, fmt.Errorf("secret not found at path: %s", secretPath)
	}

	// Extract data from KV v2 response
	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected secret data format at %s", secretPath)
	}

	result := make(map[string]string)
	for k, v := range data {
		if strVal, ok := v.(string); ok {
			result[k] = strVal
		}
	}

	return result, nil
}

// SetObject stores a map of string values at a path-based secret.
// The key must contain "/" to specify the path: {basePath}/{key}.
func (c *Client) SetObject(key string, data map[string]string) error {
	ctx := context.Background()
	secretPath := c.path + "/" + key

	// Convert map[string]string to map[string]interface{}
	dataInterface := make(map[string]interface{})
	for k, v := range data {
		dataInterface[k] = v
	}

	secretData := map[string]interface{}{
		"data": dataInterface,
	}

	_, err := c.client.Logical().WriteWithContext(ctx, secretPath, secretData)
	return err
}
