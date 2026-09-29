package httpserver

import (
	"bytes"
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

	appLogger "github.com/device-management-toolkit/console/pkg/logger"
)

func TestGenerateSelfSignedCert_UsesRSA3072AndSHA384(t *testing.T) {
	t.Parallel()

	certPEM, _, err := generateSelfSignedCert()
	if err != nil {
		t.Fatalf("generateSelfSignedCert failed: %v", err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		t.Fatalf("failed to decode generated certificate PEM")

		return
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse certificate failed: %v", err)
	}

	if cert.SignatureAlgorithm != x509.SHA384WithRSA {
		t.Fatalf("expected signature algorithm SHA384WithRSA, got %v", cert.SignatureAlgorithm)
	}

	rsaPub, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		t.Fatalf("expected RSA public key")
	}

	if rsaPub.N.BitLen() != 3072 {
		t.Fatalf("expected RSA key size 3072, got %d", rsaPub.N.BitLen())
	}
}

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

// pinTempDir points os.TempDir() at a per-test directory and returns it.
// generateAndServeSelfSignedTLS caches console_selfsigned.crt/.key in the
// system temp dir, so without pinning it, whether the generate branch or the
// reuse branch runs depends on files left behind by an earlier run -- which
// makes the coverage of this package differ between machines and CI runs.
func pinTempDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	t.Setenv("TMPDIR", dir) // POSIX
	t.Setenv("TMP", dir)    // Windows
	t.Setenv("TEMP", dir)   // Windows

	return dir
}

// getOK polls url until the server goroutine is listening, then asserts 200.
func getOK(t *testing.T, client *http.Client, url string) string {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)

	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	var (
		resp *http.Response
		err  error
	)

	for time.Now().Before(deadline) {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
		if reqErr != nil {
			t.Fatalf("create request: %v", reqErr)
		}

		resp, err = client.Do(req)
		if err == nil {
			break
		}

		time.Sleep(50 * time.Millisecond)
	}

	if err != nil {
		t.Fatalf("GET %s failed: %v", url, err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	return string(body)
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
	pinTempDir(t)

	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })

	l := newTestListener(t)

	s := New(handler, Listener(l), TLS(true, "", ""), Logger(appLogger.New("info")))

	defer func() { _ = s.Shutdown() }() // ensure server is shutdown; ignore error for cleanup

	// build url from listener
	addr := l.Addr().String()
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}

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

func TestTLS_TLS12OnlyClient_IsRejected(t *testing.T) { //nolint:paralleltest // binds a port
	cert, key := writeTempCertPair(t)
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })

	l := newTestListener(t)
	s := New(handler, Listener(l), TLS(true, cert, key))

	defer func() { _ = s.Shutdown() }()

	addr := l.Addr().String()

	certPEM, rerr := os.ReadFile(cert)
	if rerr != nil {
		t.Fatalf("read cert: %v", rerr)
	}

	roots := x509.NewCertPool()
	if ok := roots.AppendCertsFromPEM(certPEM); !ok {
		t.Fatalf("failed to append cert to pool")
	}

	// First confirm the server is reachable with a TLS 1.3-capable client.
	tls13Client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS13}}}

	deadline := time.Now().Add(2 * time.Second)

	var (
		resp *http.Response
		err  error
	)

	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	for time.Now().Before(deadline) {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+addr+"/", http.NoBody)
		if reqErr != nil {
			t.Fatalf("create request: %v", reqErr)
		}

		resp, err = tls13Client.Do(req)
		if err == nil {
			break
		}

		time.Sleep(50 * time.Millisecond)
	}

	if err != nil {
		t.Fatalf("TLS 1.3 readiness check failed: %v", err)
	}

	_ = resp.Body.Close()

	tls12OnlyClient := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12, MaxVersion: tls.VersionTLS12}}}

	req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+addr+"/", http.NoBody)
	if reqErr != nil {
		t.Fatalf("create request: %v", reqErr)
	}

	resp, err = tls12OnlyClient.Do(req)
	if err == nil {
		defer resp.Body.Close()

		t.Fatalf("expected TLS 1.2-only client handshake to fail, got status %d", resp.StatusCode)
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

// The generate branch: an empty temp dir has no cached pair, so one is written.
func TestTLS_SelfSigned_WritesPairWhenAbsent(t *testing.T) { //nolint:paralleltest // binds a port
	dir := pinTempDir(t)

	certPath := filepath.Join(dir, "console_selfsigned.crt")
	keyPath := filepath.Join(dir, "console_selfsigned.key")

	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("generated")) })

	l := newTestListener(t)
	s := New(handler, Listener(l), TLS(true, "", ""), Logger(appLogger.New("info")))

	defer func() { _ = s.Shutdown() }()

	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}

	if body := getOK(t, client, "https://"+l.Addr().String()+"/"); body != "generated" {
		t.Fatalf("unexpected body: %q", body)
	}

	for _, path := range []string{certPath, keyPath} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("expected %s to be generated: %v", path, err)
		}

		if info.Size() == 0 {
			t.Fatalf("expected %s to be non-empty", path)
		}
	}
}

// The reuse branch: a cached pair in the temp dir is served as-is, not rewritten.
func TestTLS_SelfSigned_ReusesCachedPair(t *testing.T) { //nolint:paralleltest // binds a port
	dir := pinTempDir(t)

	cert, key := writeTempCertPair(t)

	certPath := filepath.Join(dir, "console_selfsigned.crt")
	keyPath := filepath.Join(dir, "console_selfsigned.key")

	certPEM, err := os.ReadFile(cert)
	if err != nil {
		t.Fatalf("read cert: %v", err)
	}

	keyPEM, err := os.ReadFile(key)
	if err != nil {
		t.Fatalf("read key: %v", err)
	}

	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		t.Fatalf("seed cert: %v", err)
	}

	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		t.Fatalf("seed key: %v", err)
	}

	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("reused")) })

	l := newTestListener(t)
	s := New(handler, Listener(l), TLS(true, "", ""), Logger(appLogger.New("info")))

	defer func() { _ = s.Shutdown() }()

	// Trusting only the seeded certificate proves the cached pair was served
	// rather than a freshly generated one.
	roots := x509.NewCertPool()
	if ok := roots.AppendCertsFromPEM(certPEM); !ok {
		t.Fatalf("failed to append cert to pool")
	}

	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS13}}}

	if body := getOK(t, client, "https://"+l.Addr().String()+"/"); body != "reused" {
		t.Fatalf("unexpected body: %q", body)
	}

	after, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("read cached cert: %v", err)
	}

	if !bytes.Equal(after, certPEM) {
		t.Error("expected the cached certificate to be left untouched")
	}
}

// An unwritable temp dir must surface as an error on Notify, not a panic.
func TestTLS_SelfSigned_UnwritableTempDir_ReturnsError(t *testing.T) {
	// A path that does not exist: os.WriteFile fails regardless of the user the
	// tests run as, so this stays deterministic on CI and in containers.
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "absent"))
	t.Setenv("TMP", filepath.Join(t.TempDir(), "absent"))
	t.Setenv("TEMP", filepath.Join(t.TempDir(), "absent"))

	handler := http.NewServeMux()
	l := newTestListener(t)
	s := New(handler, Listener(l), TLS(true, "", ""), Logger(appLogger.New("info")))

	select {
	case err := <-s.Notify():
		if err == nil {
			t.Fatal("expected an error when the certificate cannot be written, got nil")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for server error")
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
