package main

import (
	"crypto/rsa"
	"crypto/x509"
	"os"
	"testing"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/certificates"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func TestShouldAutoLaunchBrowser(t *testing.T) {
	tests := []struct {
		name    string
		ginMode string
		setEnv  bool
		want    bool
	}{
		{name: "unset", setEnv: false, want: false},
		{name: "empty", ginMode: "", setEnv: true, want: false},
		{name: "release", ginMode: "release", setEnv: true, want: false},
		{name: "test", ginMode: "test", setEnv: true, want: false},
		{name: "debug", ginMode: ginModeDebug, setEnv: true, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setEnv {
				t.Setenv("GIN_MODE", tc.ginMode)
			} else {
				_ = os.Unsetenv("GIN_MODE")
			}

			if got := shouldAutoLaunchBrowser(); got != tc.want {
				t.Errorf("shouldAutoLaunchBrowser() with GIN_MODE=%q: got %v, want %v", tc.ginMode, got, tc.want)
			}
		})
	}
}

func TestMainFunction(_ *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying env variables.
	os.Setenv("GIN_MODE", "debug")

	// Mock functions
	initializeConfigFunc = func() (*config.Config, error) {
		return &config.Config{HTTP: config.HTTP{Port: "8080"}, App: config.App{EncryptionKey: "test"}, Log: config.Log{Level: "info"}}, nil
	}

	initializeAppFunc = func(_ *config.Config) error {
		return nil
	}

	runAppFunc = func(_ *config.Config, _ logger.Interface) {}

	// Mock certificate functions
	loadOrGenerateRootCertFunc = func(_ security.Storager, _ bool, _, _, _ string, _ bool) (*x509.Certificate, *rsa.PrivateKey, error) {
		return &x509.Certificate{}, &rsa.PrivateKey{}, nil
	}

	loadOrGenerateWebServerCertFunc = func(_ security.Storager, _ certificates.CertAndKeyType, _ bool, _, _, _ string, _ bool) (*x509.Certificate, *rsa.PrivateKey, error) {
		return &x509.Certificate{}, &rsa.PrivateKey{}, nil
	}

	// Call the main function
	main()
}
