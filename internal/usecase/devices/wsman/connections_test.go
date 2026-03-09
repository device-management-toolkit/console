package wsman

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetConnectionEntry(t *testing.T) {
	t.Parallel()

	key := "test-set-entry"

	t.Cleanup(func() { RemoveConnection(key) })

	entry := &ConnectionEntry{
		IsCIRA: true,
		Timer:  time.AfterFunc(time.Hour, func() {}),
	}

	SetConnectionEntry(key, entry)

	connectionsMu.RLock()

	got := connections[key]

	connectionsMu.RUnlock()

	assert.Equal(t, entry, got)
}

func TestGetConnectionEntry(t *testing.T) {
	t.Parallel()

	key := "test-get-entry"

	t.Cleanup(func() { RemoveConnection(key) })

	entry := &ConnectionEntry{IsCIRA: true}
	SetConnectionEntry(key, entry)

	t.Run("existing entry", func(t *testing.T) {
		t.Parallel()

		got := GetConnectionEntry(key)
		require.NotNil(t, got)
		assert.True(t, got.IsCIRA)
	})

	t.Run("missing entry returns nil", func(t *testing.T) {
		t.Parallel()

		got := GetConnectionEntry("nonexistent-get-entry")
		assert.Nil(t, got)
	})
}

func TestRemoveConnection(t *testing.T) {
	t.Parallel()

	key := "test-remove-entry"

	SetConnectionEntry(key, &ConnectionEntry{})

	RemoveConnection(key)

	got := GetConnectionEntry(key)
	assert.Nil(t, got)
}

func TestRemoveConnectionNonexistent(t *testing.T) {
	t.Parallel()

	// Should not panic when removing a key that doesn't exist.
	RemoveConnection("does-not-exist-remove")
}

func TestHasConnections(t *testing.T) {
	t.Parallel()

	key := "test-has-connections"

	t.Cleanup(func() { RemoveConnection(key) })

	SetConnectionEntry(key, &ConnectionEntry{})

	assert.True(t, HasConnections(), "should be true after adding an entry")
}

func TestEnsureAPFChannelStore(t *testing.T) {
	t.Parallel()

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	entry := &ConnectionEntry{
		Conny: client,
	}

	// APFChannelStore should be nil initially
	assert.Nil(t, entry.APFChannelStore)

	// First call should initialize it
	entry.ensureAPFChannelStore()
	require.NotNil(t, entry.APFChannelStore)

	// Save reference to verify idempotency
	first := entry.APFChannelStore

	// Second call should return the same instance (sync.Once)
	entry.ensureAPFChannelStore()
	assert.Same(t, first, entry.APFChannelStore)
}

func TestRegisterAPFChannel(t *testing.T) {
	t.Parallel()

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	entry := &ConnectionEntry{
		Conny: client,
	}

	ch := entry.RegisterAPFChannel()
	assert.NotNil(t, ch)
	// APFChannelStore should have been lazily created
	require.NotNil(t, entry.APFChannelStore)
}

func TestGetConnection(t *testing.T) {
	t.Parallel()

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	entry := &ConnectionEntry{
		Conny: client,
	}

	assert.Equal(t, client, entry.GetConnection())
}

func TestGetAPFChannelNilStore(t *testing.T) {
	t.Parallel()

	entry := &ConnectionEntry{}
	ch := entry.GetAPFChannel(1)
	assert.Nil(t, ch)
}

func TestUnregisterAPFChannelNilStore(t *testing.T) {
	t.Parallel()

	entry := &ConnectionEntry{}
	// Should not panic when store is nil
	entry.UnregisterAPFChannel(1)
}

func TestUnregisterAPFChannel(t *testing.T) {
	t.Parallel()

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	entry := &ConnectionEntry{
		Conny: client,
	}

	ch := entry.RegisterAPFChannel()
	require.NotNil(t, ch)

	entry.UnregisterAPFChannel(ch.GetSenderChannel())

	assert.Nil(t, entry.GetAPFChannel(ch.GetSenderChannel()))
}

func TestWriteToConnection(t *testing.T) {
	t.Parallel()

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	entry := &ConnectionEntry{
		Conny: client,
	}

	// Write in a goroutine since net.Pipe is synchronous
	done := make(chan error, 1)

	go func() {
		done <- entry.WriteToConnection([]byte("hello"))
	}()

	buf := make([]byte, 5)
	_, err := server.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, []byte("hello"), buf)

	require.NoError(t, <-done)
	// Should have lazily initialized the store
	require.NotNil(t, entry.APFChannelStore)
}
