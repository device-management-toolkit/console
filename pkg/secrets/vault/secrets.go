package secrets

import (
	"context"
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"
)

// Vault KV v2 field name constants.
const (
	vaultDataField  = "data"
	vaultValueField = "value"
)

var pathKeyPattern = regexp.MustCompile(`^[a-zA-Z0-9._/-]+$`)

// Sentinel errors for secret operations.
var (
	ErrSecretNotFound       = errors.New("secret not found")
	ErrUnexpectedDataFormat = errors.New("unexpected secret data format")
	ErrKeyNotFound          = errors.New("key not found in secret")
	ErrValueNotString       = errors.New("value is not a string")
	ErrInvalidPathKey       = errors.New("invalid path-based secret key")
)

// validatePathKey ensures path-based secret keys stay inside the configured namespace.
func validatePathKey(key string) error {
	// Keep this guard as a defensive invariant for direct/helper usage and
	// future call sites, even though current public methods usually reject
	// empty keys before calling validatePathKey.
	if key == "" {
		return fmt.Errorf("%w: empty key", ErrInvalidPathKey)
	}

	// Vault paths are URL-style logical paths and must always use '/'
	// separators, even on Windows hosts.
	if strings.Contains(key, "\\") {
		return fmt.Errorf("%w: backslash is not allowed: %q", ErrInvalidPathKey, key)
	}

	if !pathKeyPattern.MatchString(key) {
		return fmt.Errorf("%w: key contains unsupported characters: %q", ErrInvalidPathKey, key)
	}

	if strings.HasPrefix(key, "/") {
		return fmt.Errorf("%w: absolute path not allowed: %q", ErrInvalidPathKey, key)
	}

	cleanKey := path.Clean(key)
	if cleanKey == "." || cleanKey == ".." || strings.HasPrefix(cleanKey, "../") {
		return fmt.Errorf("%w: traversal not allowed: %q", ErrInvalidPathKey, key)
	}

	if cleanKey != key {
		return fmt.Errorf("%w: non-normalized path not allowed: %q", ErrInvalidPathKey, key)
	}

	segments := strings.Split(key, "/")
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf("%w: invalid path segment in key: %q", ErrInvalidPathKey, key)
		}
	}

	return nil
}

// buildScopedSecretPath joins a validated key under the configured base path and
// enforces that the normalized result stays inside the same namespace.
func buildScopedSecretPath(basePath, key string) (string, error) {
	if err := validatePathKey(key); err != nil {
		return "", err
	}

	base := path.Clean("/" + basePath)
	joined := path.Clean(path.Join(base, key))
	prefix := base + "/"

	if !strings.HasPrefix(joined, prefix) {
		return "", fmt.Errorf("%w: path escapes base namespace: %q", ErrInvalidPathKey, key)
	}

	return strings.TrimPrefix(joined, "/"), nil
}

// GetKeyValue reads a value from Vault.
// If the key contains "/", it's treated as a separate path: {basePath}/{key} with data stored under "value".
// Otherwise, it's stored in {basePath}/keys with the key as a field name.
func (c *Client) GetKeyValue(key string) (string, error) {
	ctx := context.Background()

	var (
		secretPath string
		dataKey    string
		err        error
	)

	if strings.Contains(key, "/") {
		secretPath, err = buildScopedSecretPath(c.path, key)
		if err != nil {
			return "", err
		}

		// Path-based storage: {basePath}/{key} with "value" field
		dataKey = vaultValueField
	} else {
		// Key-based storage: {basePath}/keys with key as field name
		secretPath, err = buildScopedSecretPath(c.path, "keys")
		if err != nil {
			return "", err
		}

		dataKey = key
	}

	secret, err := c.client.Logical().ReadWithContext(ctx, secretPath)
	if err != nil {
		return "", err
	}

	if secret == nil {
		return "", fmt.Errorf("%w at path: %s", ErrSecretNotFound, secretPath)
	}

	// Extract data from KV v2 response
	data, ok := secret.Data[vaultDataField].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("%w at %s", ErrUnexpectedDataFormat, secretPath)
	}

	value, ok := data[dataKey]
	if !ok {
		return "", fmt.Errorf("%w: %s at path %s", ErrKeyNotFound, dataKey, secretPath)
	}

	strValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%w: key %s", ErrValueNotString, dataKey)
	}

	return strValue, nil
}

// SetKeyValue writes a value to Vault.
// If the key contains "/", it's treated as a separate path: {basePath}/{key} with data stored under "value".
// Otherwise, it's stored in {basePath}/keys with the key as a field name.
func (c *Client) SetKeyValue(key, value string) error {
	ctx := context.Background()
	if strings.Contains(key, "/") {
		return c.setPathBasedKeyValue(ctx, key, value)
	}

	return c.setFieldKeyValue(ctx, key, value)
}

func (c *Client) setPathBasedKeyValue(ctx context.Context, key, value string) error {
	secretPath, err := buildScopedSecretPath(c.path, key)
	if err != nil {
		return err
	}

	// Path-based storage: {basePath}/{key} with "value" field
	secretData := map[string]interface{}{
		vaultDataField: map[string]interface{}{
			vaultValueField: value,
		},
	}

	_, err = c.client.Logical().WriteWithContext(ctx, secretPath, secretData)

	return err
}

func (c *Client) setFieldKeyValue(ctx context.Context, key, value string) error {
	// Key-based storage: {basePath}/keys with key as field name
	secretPath, err := buildScopedSecretPath(c.path, "keys")
	if err != nil {
		return err
	}

	// Read existing secret to preserve other keys
	secret, err := c.client.Logical().ReadWithContext(ctx, secretPath)
	data := make(map[string]interface{})

	if err == nil && secret != nil {
		// Secret exists, preserve existing data
		if d, ok := secret.Data[vaultDataField].(map[string]interface{}); ok {
			data = d
		}
	}

	// Set or update the specific key
	data[key] = value

	secretData := map[string]interface{}{
		vaultDataField: data,
	}

	_, err = c.client.Logical().WriteWithContext(ctx, secretPath, secretData)

	return err
}

// DeleteKeyValue deletes a value from Vault.
// If the key contains "/", it deletes the entire secret at {basePath}/{key}.
// Otherwise, it removes the key from {basePath}/keys.
func (c *Client) DeleteKeyValue(key string) error {
	ctx := context.Background()

	if strings.Contains(key, "/") {
		metadataBasePath := strings.Replace(c.path, "/data/", "/metadata/", 1)

		metadataPath, err := buildScopedSecretPath(metadataBasePath, key)
		if err != nil {
			return err
		}

		// Path-based storage: permanently delete the secret at {basePath}/{key}
		// For KV v2, we need to delete metadata to permanently remove (not just soft delete)
		// Convert secret/data/console/... to secret/metadata/console/...
		_, err = c.client.Logical().DeleteWithContext(ctx, metadataPath)

		return err
	}

	// Key-based storage: remove from {basePath}/keys
	secretPath, err := buildScopedSecretPath(c.path, "keys")
	if err != nil {
		return err
	}

	// Read existing secret
	secret, err := c.client.Logical().ReadWithContext(ctx, secretPath)
	if err != nil {
		return err
	}

	if secret == nil {
		return fmt.Errorf("%w at path: %s", ErrSecretNotFound, secretPath)
	}

	// Extract data from KV v2 response
	data, ok := secret.Data[vaultDataField].(map[string]interface{})
	if !ok {
		return fmt.Errorf("%w at %s", ErrUnexpectedDataFormat, secretPath)
	}

	delete(data, key)

	secretData := map[string]interface{}{
		vaultDataField: data,
	}

	_, err = c.client.Logical().WriteWithContext(ctx, secretPath, secretData)

	return err
}

// GetObject retrieves a map of string values from a path-based secret.
// The key must contain "/" to specify the path: {basePath}/{key}.
func (c *Client) GetObject(key string) (map[string]string, error) {
	if !strings.Contains(key, "/") {
		return nil, fmt.Errorf("%w: object key must contain '/': %q", ErrInvalidPathKey, key)
	}

	secretPath, err := buildScopedSecretPath(c.path, key)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	secret, err := c.client.Logical().ReadWithContext(ctx, secretPath)
	if err != nil {
		return nil, err
	}

	if secret == nil {
		return nil, fmt.Errorf("%w at path: %s", ErrSecretNotFound, secretPath)
	}

	// Extract data from KV v2 response
	data, ok := secret.Data[vaultDataField].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%w at %s", ErrUnexpectedDataFormat, secretPath)
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
	if !strings.Contains(key, "/") {
		return fmt.Errorf("%w: object key must contain '/': %q", ErrInvalidPathKey, key)
	}

	secretPath, err := buildScopedSecretPath(c.path, key)
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Convert map[string]string to map[string]interface{}
	dataInterface := make(map[string]interface{})
	for k, v := range data {
		dataInterface[k] = v
	}

	secretData := map[string]interface{}{
		vaultDataField: dataInterface,
	}

	_, err = c.client.Logical().WriteWithContext(ctx, secretPath, secretData)

	return err
}
