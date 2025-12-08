package secrets

import (
	"testing"

	"github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"

	"github.com/device-management-toolkit/console/config"
)

func TestNewVaultClient(t *testing.T) {
	mockVaultClient := &api.Client{}

	client := NewVaultClient(mockVaultClient)

	assert.NotNil(t, client)
	assert.Equal(t, mockVaultClient, client.client)
}

func TestNewClient_ValidConfig(t *testing.T) {
	cfg := &config.Secrets{
		Address: "http://localhost:8200",
		Token:   "test-token",
	}

	client, err := NewClient(cfg)

	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, client.client)
}

func TestNewClient_EmptyToken(t *testing.T) {
	cfg := &config.Secrets{
		Address: "http://localhost:8200",
		Token:   "",
	}

	client, err := NewClient(cfg)

	// Vault client creation succeeds even with empty token
	// Token is set after client creation
	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, client.client)
}

func TestNewClient_EmptyAddress(t *testing.T) {
	cfg := &config.Secrets{
		Address: "",
		Token:   "test-token",
	}

	client, err := NewClient(cfg)

	assert.NoError(t, err)
	assert.NotNil(t, client)
}
