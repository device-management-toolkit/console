package app

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/pkg/logger"
)

type noopCloser struct{}

func (noopCloser) Close() error { return nil }

func writeSelfSignedCert(t *testing.T, dir, commonName string) {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	require.NoError(t, err)

	keyBytes, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)

	certOut := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyOut := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})

	require.NoError(t, os.WriteFile(filepath.Join(dir, fmt.Sprintf("%s_cert.pem", commonName)), certOut, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, fmt.Sprintf("%s_key.pem", commonName)), keyOut, 0o600))
}

func TestSetupCIRAServer(t *testing.T) {
	//nolint:paralleltest // uses t.Chdir, which conflicts with parallel subtests
	t.Run("disabled returns nil without touching certs", func(t *testing.T) {
		cfg := &config.Config{App: config.App{DisableCIRA: true}}

		server := setupCIRAServer(cfg, logger.New("error"), noopCloser{}, &usecase.Usecases{})

		assert.Nil(t, server)
	})

	t.Run("enabled starts the CIRA listener", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(dir, "config"), 0o750))
		writeSelfSignedCert(t, filepath.Join(dir, "config"), "console-test")

		t.Chdir(dir)

		cfg := &config.Config{App: config.App{
			DisableCIRA: false,
			CommonName:  "console-test",
		}}

		ctrl := gomock.NewController(t)
		mockDevices := mocks.NewMockDeviceManagementFeature(ctrl)

		server := setupCIRAServer(cfg, logger.New("error"), noopCloser{}, &usecase.Usecases{Devices: mockDevices})

		require.NotNil(t, server)

		require.Eventually(t, func() bool {
			conn, err := net.Dial("tcp", "127.0.0.1:4433")
			if err != nil {
				return false
			}

			_ = conn.Close()

			return true
		}, time.Second, 5*time.Millisecond, "CIRA server never started listening on port 4433")

		require.NoError(t, server.Shutdown())
	})
}
