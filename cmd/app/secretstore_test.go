package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"
)

var errStorage = errors.New("storage boom")

// fakeStorage is an in-memory security.Storager for testing secretStore.
type fakeStorage struct {
	data   map[string]string
	getErr error
	setErr error
}

func newFakeStorage() *fakeStorage { return &fakeStorage{data: map[string]string{}} }

func (f *fakeStorage) GetKeyValue(key string) (string, error) {
	if f.getErr != nil {
		return "", f.getErr
	}

	v, ok := f.data[key]
	if !ok {
		return "", security.ErrKeyNotFound
	}

	return v, nil
}

func (f *fakeStorage) SetKeyValue(key, value string) error {
	if f.setErr != nil {
		return f.setErr
	}

	f.data[key] = value

	return nil
}

func (f *fakeStorage) DeleteKeyValue(key string) error {
	delete(f.data, key)

	return nil
}

func TestSecretStoreGet_RemoteHit(t *testing.T) {
	t.Parallel()

	remote := newFakeStorage()
	remote.data["k"] = "remote-val"
	local := newFakeStorage()
	local.data["k"] = "local-val"

	s := &secretStore{remote: remote, local: local}

	v, err := s.get("k")
	require.NoError(t, err)
	assert.Equal(t, "remote-val", v)
}

func TestSecretStoreGet_LocalFallbackSyncsToRemote(t *testing.T) {
	t.Parallel()

	remote := newFakeStorage() // empty
	local := newFakeStorage()
	local.data["k"] = "local-val"

	s := &secretStore{remote: remote, local: local}

	v, err := s.get("k")
	require.NoError(t, err)
	assert.Equal(t, "local-val", v)
	assert.Equal(t, "local-val", remote.data["k"], "keyring-only value should sync to remote")
}

func TestSecretStoreGet_NotFound(t *testing.T) {
	t.Parallel()

	s := &secretStore{remote: nil, local: newFakeStorage()}

	v, err := s.get("missing")
	require.NoError(t, err)
	assert.Empty(t, v)
}

func TestSecretStoreGet_RealKeyringError(t *testing.T) {
	t.Parallel()

	local := newFakeStorage()
	local.getErr = errStorage

	s := &secretStore{remote: nil, local: local}

	_, err := s.get("k")
	require.ErrorIs(t, err, errStorage)
}

func TestSecretStoreSet_PrefersRemote(t *testing.T) {
	t.Parallel()

	remote := newFakeStorage()
	local := newFakeStorage()

	s := &secretStore{remote: remote, local: local}

	require.NoError(t, s.set("k", "v"))
	assert.Equal(t, "v", remote.data["k"])
	assert.Empty(t, local.data["k"], "should not write keyring when remote is available")
}

func TestSecretStoreSet_FallsBackToLocal(t *testing.T) {
	t.Parallel()

	local := newFakeStorage()

	s := &secretStore{remote: nil, local: local}

	require.NoError(t, s.set("k", "v"))
	assert.Equal(t, "v", local.data["k"])
}
