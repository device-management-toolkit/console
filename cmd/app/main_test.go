package main

import (
	"crypto/rsa"
	"crypto/x509"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/certificates"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func TestConfigureWSMANLogLevel_UsesConfigLevel(t *testing.T) { //nolint:paralleltest // uses global logrus level.
	oldLevel := logrus.GetLevel()
	t.Cleanup(func() {
		logrus.SetLevel(oldLevel)
	})

	cfg := &config.Config{Log: config.Log{Level: "debug"}}

	configureWSMANLogLevel(cfg, logger.New("debug"))

	assert.Equal(t, logrus.DebugLevel, logrus.GetLevel())
}

func TestConfigureWSMANLogLevel_UsesTraceConfigLevel(t *testing.T) { //nolint:paralleltest // uses global logrus level.
	oldLevel := logrus.GetLevel()
	t.Cleanup(func() {
		logrus.SetLevel(oldLevel)
	})

	cfg := &config.Config{Log: config.Log{Level: "trace"}}

	configureWSMANLogLevel(cfg, logger.New("debug"))

	assert.Equal(t, logrus.TraceLevel, logrus.GetLevel())
}

func TestConfigureWSMANLogLevel_InvalidValueFallsBackToInfo(t *testing.T) { //nolint:paralleltest // uses global logrus level.
	oldLevel := logrus.GetLevel()
	t.Cleanup(func() {
		logrus.SetLevel(oldLevel)
	})

	cfg := &config.Config{Log: config.Log{Level: "not-a-level"}}

	configureWSMANLogLevel(cfg, logger.New("debug"))

	assert.Equal(t, logrus.InfoLevel, logrus.GetLevel())
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
