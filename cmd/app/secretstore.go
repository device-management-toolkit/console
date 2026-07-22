package main

import (
	"errors"
	"log"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/config"
)

// secretStore resolves named secrets from a remote store (Vault) with an OS keyring fallback.
type secretStore struct {
	remote security.Storager
	local  security.Storager
}

// newSecretStore builds a store from the optional Vault client and the local keyring.
func newSecretStore(cfg *config.Config) *secretStore {
	remote, err := handleSecretsConfig(cfg)
	if err != nil {
		remote = nil
	}

	return &secretStore{remote: remote, local: security.NewKeyRingStorage(keyringServiceName)}
}

// get returns the secret for name, preferring the remote store and falling back to the
// keyring (syncing a keyring-only value back to remote). Returns "" when absent; a non-nil
// error is a real storage failure the caller must decide how to handle.
func (s *secretStore) get(name string) (string, error) {
	if s.remote != nil {
		if v, err := s.remote.GetKeyValue(name); err == nil && v != "" {
			return v, nil
		}
	}

	v, err := s.local.GetKeyValue(name)
	if err != nil {
		if errors.Is(err, security.ErrKeyNotFound) {
			return "", nil
		}

		return "", err
	}

	if v != "" && s.remote != nil {
		if syncErr := s.remote.SetKeyValue(name, v); syncErr != nil {
			log.Printf("Warning: failed to sync %s to secret store: %v", name, syncErr)
		}
	}

	return v, nil
}

// set stores value under name in the remote store when available, else the keyring.
func (s *secretStore) set(name, value string) error {
	if s.remote != nil {
		return s.remote.SetKeyValue(name, value)
	}

	return s.local.SetKeyValue(name, value)
}
