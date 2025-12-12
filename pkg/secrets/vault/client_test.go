/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

package secrets

import (
	"testing"

	"github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"

	"github.com/device-management-toolkit/console/config"
)

func TestNewClient_WithInjectedClient(t *testing.T) {
	t.Parallel()

	mockVaultClient := &api.Client{}

	client, err := NewClient(nil, WithClient(mockVaultClient))

	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, mockVaultClient, client.client)
	assert.Equal(t, DefaultSecretPath, client.path)
}

func TestNewClient_WithInjectedClientAndPath(t *testing.T) {
	t.Parallel()

	mockVaultClient := &api.Client{}

	client, err := NewClient(nil, WithClient(mockVaultClient), WithPath("secret/data/custom"))

	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, mockVaultClient, client.client)
	assert.Equal(t, "secret/data/custom", client.path)
}

func TestNewClient_ValidConfig(t *testing.T) {
	t.Parallel()

	cfg := &config.Secrets{
		Address: "http://localhost:8200",
		Token:   "test-token",
	}

	client, err := NewClient(cfg)

	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, client.client)
	assert.Equal(t, DefaultSecretPath, client.path)
}

func TestNewClient_ConfigWithPath(t *testing.T) {
	t.Parallel()

	cfg := &config.Secrets{
		Address: "http://localhost:8200",
		Token:   "test-token",
		Path:    "secret/data/myapp",
	}

	client, err := NewClient(cfg)

	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "secret/data/myapp", client.path)
}

func TestNewClient_EmptyToken(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	cfg := &config.Secrets{
		Address: "",
		Token:   "test-token",
	}

	client, err := NewClient(cfg)

	assert.NoError(t, err)
	assert.NotNil(t, client)
}

func TestWithPath_EmptyString(t *testing.T) {
	t.Parallel()

	mockVaultClient := &api.Client{}

	// Empty path should not override default
	client, err := NewClient(nil, WithClient(mockVaultClient), WithPath(""))

	assert.NoError(t, err)
	assert.Equal(t, DefaultSecretPath, client.path)
}
