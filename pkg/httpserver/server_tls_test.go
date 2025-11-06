package httpserver

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// helper to create a basic cert/key pair on disk.
func writeTempCertPair(t *testing.T) (certPath, keyPath string) { // named results for clarity
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	tmpl := x509.Certificate{
		SerialNumber:          big.NewInt(2),
		Subject:               pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(24 * time.Hour),
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	dir := t.TempDir()
	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")

	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		t.Fatalf("write cert: %v", err)
	}

	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}

	return certPath, keyPath
}

func newTestListener(t *testing.T) net.Listener {
	t.Helper()

	// Use ListenConfig with context to satisfy noctx and allow cancellation if needed
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	l, err := (&net.ListenConfig{}).Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	return l
}

func TestTLS_SelfSigned_GeneratesAndServes(t *testing.T) { //nolint:paralleltest // binds a port
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })

	l := newTestListener(t)

	s := New(handler, Listener(l), TLS(true, "", ""))

	defer func() { _ = s.Shutdown() }() // ensure server is shutdown; ignore error for cleanup

	// build url from listener
	addr := l.Addr().String()
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}} //nolint:gosec // test connects to self-signed server generated at runtime

	// try for a short while to allow server goroutine to start
	deadline := time.Now().Add(2 * time.Second)

	var (
		resp *http.Response
		err  error
	)

	// Use a single context with deadline for all attempts
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	for time.Now().Before(deadline) {
		req, rerr := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+addr+"/", http.NoBody)
		if rerr != nil {
			t.Fatalf("create request: %v", rerr)
		}

		resp, err = client.Do(req)
		if err == nil {
			break
		}

		time.Sleep(50 * time.Millisecond)
	}

	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	b, _ := io.ReadAll(resp.Body)
	if string(b) != "ok" {
		t.Fatalf("unexpected body: %q", string(b))
	}
}

func TestTLS_WithProvidedCerts_Serves(t *testing.T) { //nolint:paralleltest // binds a port
	cert, key := writeTempCertPair(t)
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })

	l := newTestListener(t)

	s := New(handler, Listener(l), TLS(true, cert, key))

	defer func() { _ = s.Shutdown() }()

	addr := l.Addr().String()
	// Trust the generated certificate instead of skipping verification
	certPEM, rerr := os.ReadFile(cert)
	if rerr != nil {
		t.Fatalf("read cert: %v", rerr)
	}

	roots := x509.NewCertPool()
	if ok := roots.AppendCertsFromPEM(certPEM); !ok {
		t.Fatalf("failed to append cert to pool")
	}

	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}}

	deadline := time.Now().Add(2 * time.Second)

	var (
		resp *http.Response
		err  error
	)

	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	for time.Now().Before(deadline) {
		req, rerr := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+addr+"/", http.NoBody)
		if rerr != nil {
			t.Fatalf("create request: %v", rerr)
		}

		resp, err = client.Do(req)
		if err == nil {
			break
		}

		time.Sleep(50 * time.Millisecond)
	}

	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestTLS_MissingFiles_ReturnsError(t *testing.T) { //nolint:paralleltest // server lifecycle
	handler := http.NewServeMux()
	l := newTestListener(t)
	s := New(handler, Listener(l), TLS(true, filepath.Join(t.TempDir(), "missing.crt"), filepath.Join(t.TempDir(), "missing.key")))

	// Expect an error on notify
	select {
	case err := <-s.Notify():
		if err == nil {
			t.Fatalf("expected error for missing certs, got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for server error")
	}

	_ = s.Shutdown()
}

func TestTLS_Mismatch_ReturnsError(t *testing.T) { //nolint:paralleltest // server lifecycle
	handler := http.NewServeMux()
	l := newTestListener(t)
	s := New(handler, Listener(l), TLS(true, "onlycert.pem", ""))

	select {
	case err := <-s.Notify():
		if err == nil {
			t.Fatalf("expected mismatch error, got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for server error")
	}

	_ = s.Shutdown()
}
