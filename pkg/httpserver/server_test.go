package httpserver

import (
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) { //nolint:paralleltest // httpserver can't be bind to multiple ports at the same time for tests
	handler := http.NewServeMux()
	s := New(handler, Port("localhost", "9090")) // Use a different port

	defer s.server.Close()

	assert.Equal(t, handler, s.server.Handler, "expected handler to be set")
	assert.Equal(t, _defaultReadTimeout, s.server.ReadTimeout, "expected read timeout to be set correctly")
	assert.Equal(t, _defaultWriteTimeout, s.server.WriteTimeout, "expected write timeout to be set correctly")
	assert.Equal(t, net.JoinHostPort("localhost", "9090"), s.server.Addr, "expected addr to be set correctly")
	assert.Equal(t, _defaultShutdownTimeout, s.shutdownTimeout, "expected shutdown timeout to be set correctly")
}

// The read/write budget must outlast the wsman client's own 30s per-request
// timeout, so a slow AMT device fails there first and the handler can turn it
// into a clean response instead of the server cutting the connection.
func TestDefaultTimeoutsOutlastWsmanClient(t *testing.T) {
	t.Parallel()

	const wsmanClientTimeout = 30 * time.Second

	assert.Greater(t, _defaultReadTimeout, wsmanClientTimeout, "read timeout must outlast the wsman client timeout")
	assert.Greater(t, _defaultWriteTimeout, wsmanClientTimeout, "write timeout must outlast the wsman client timeout")
}

func TestServePlainWithInjectedListener(t *testing.T) { //nolint:paralleltest // server lifecycle
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("plain")) })

	l := newTestListener(t)
	s := New(handler, Listener(l))

	defer func() { _ = s.Shutdown() }()

	if body := getOK(t, http.DefaultClient, "http://"+l.Addr().String()+"/"); body != "plain" {
		t.Fatalf("unexpected body: %q", body)
	}
}
