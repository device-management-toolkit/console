// Package httpserver implements HTTP server.
package httpserver

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	appLogger "github.com/device-management-toolkit/console/pkg/logger"
)

const (
	_defaultReadTimeout     = 15 * time.Second
	_defaultWriteTimeout    = 15 * time.Second
	_defaultAddr            = ":80"
	_defaultShutdownTimeout = 3 * time.Second

	_filePerm   = 0o600
	_rsaKeyBits = 2048
)

// Server -.
type Server struct {
	server          *http.Server
	notify          chan error
	shutdownTimeout time.Duration
	useTLS          bool
	certFile        string
	keyFile         string
	listener        net.Listener
	log             appLogger.Interface
}

// New -.
func New(handler http.Handler, opts ...Option) *Server {
	httpServer := &http.Server{
		Handler:      handler,
		ReadTimeout:  _defaultReadTimeout,
		WriteTimeout: _defaultWriteTimeout,
		Addr:         _defaultAddr,
	}

	s := &Server{
		server:          httpServer,
		notify:          make(chan error, 1),
		shutdownTimeout: _defaultShutdownTimeout,
		useTLS:          false,
		certFile:        "",
		keyFile:         "",
		log:             appLogger.New("info"),
	}

	// Custom options
	for _, opt := range opts {
		opt(s)
	}

	s.start()

	return s
}

func (s *Server) start() {
	go func() {
		s.notify <- s.serve()

		close(s.notify)
	}()
}

func (s *Server) serve() error {
	if s.useTLS {
		return s.serveTLS()
	}

	return s.servePlain()
}

func (s *Server) servePlain() error {
	if s.listener != nil {
		return s.server.Serve(s.listener)
	}

	return s.server.ListenAndServe()
}

func (s *Server) serveTLS() error {
	// If cert and key files are provided, ensure they exist
	if s.certFile != "" || s.keyFile != "" {
		if s.certFile == "" || s.keyFile == "" {
			return ErrTLSCertKeyMismatch
		}

		if _, err := os.Stat(s.certFile); err != nil {
			return err
		}

		if _, err := os.Stat(s.keyFile); err != nil {
			return err
		}

		if s.listener != nil {
			return s.server.ServeTLS(s.listener, s.certFile, s.keyFile)
		}

		return s.server.ListenAndServeTLS(s.certFile, s.keyFile)
	}

	return s.generateAndServeSelfSignedTLS()
}

func (s *Server) generateAndServeSelfSignedTLS() error {
	// Temp file paths
	certPath := filepath.Join(os.TempDir(), "console_selfsigned.crt")
	keyPath := filepath.Join(os.TempDir(), "console_selfsigned.key")

	// Reuse if previously generated files exist
	if _, err := os.Stat(certPath); err == nil {
		if _, err2 := os.Stat(keyPath); err2 == nil {
			s.log.Info(fmt.Sprintf("TLS: using existing self-signed certificate cert=%s key=%s", certPath, keyPath))

			if s.listener != nil {
				return s.server.ServeTLS(s.listener, certPath, keyPath)
			}

			return s.server.ListenAndServeTLS(certPath, keyPath)
		}
	}

	// Otherwise, generate self-signed certificate on the fly
	cert, key, err := generateSelfSignedCert()
	if err != nil {
		return err
	}

	if err := os.WriteFile(certPath, cert, _filePerm); err != nil {
		return err
	}

	if err := os.WriteFile(keyPath, key, _filePerm); err != nil {
		return err
	}

	s.log.Info(fmt.Sprintf("TLS: generated self-signed certificate cert=%s key=%s", certPath, keyPath))

	if s.listener != nil {
		return s.server.ServeTLS(s.listener, certPath, keyPath)
	}

	return s.server.ListenAndServeTLS(certPath, keyPath)
}

// Notify -.
func (s *Server) Notify() <-chan error {
	return s.notify
}

// Shutdown -.
func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	return s.server.Shutdown(ctx)
}

// generateSelfSignedCert creates a minimal self-signed cert for localhost usage.
func generateSelfSignedCert() (certPEM, keyPEM []byte, err error) {
	priv, err := rsa.GenerateKey(rand.Reader, _rsaKeyBits)
	if err != nil {
		return nil, nil, err
	}

	tmpl := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "localhost"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:              []string{"localhost"},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, err
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	return certPEM, keyPEM, nil
}

// Errors.
var (
	ErrTLSCertKeyMismatch = errors.New("tls cert/key mismatch: both certFile and keyFile must be set when TLS is enabled")
)
