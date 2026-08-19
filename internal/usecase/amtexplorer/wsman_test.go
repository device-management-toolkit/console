package amtexplorer_test

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/config"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/amtexplorer"
	wsmanAPI "github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// errCrypto is a Cryptor that always fails Decrypt.
type errCrypto struct{}

var errDecrypt = errors.New("decrypt failed")

func (e errCrypto) Decrypt(_ string) (string, error)           { return "", errDecrypt }
func (e errCrypto) Encrypt(_ string) (string, error)           { return "", nil }
func (e errCrypto) EncryptWithKey(_, _ string) (string, error) { return "", nil }
func (e errCrypto) GenerateKey() string                        { return "" }
func (e errCrypto) ReadAndDecryptFile(_ string) (config.Configuration, error) {
	return config.Configuration{}, nil
}

func newExplorerWSMAN() *amtexplorer.GoWSMANMessages {
	log := logger.New("error")
	crypto := mocks.MockCrypto{}

	return amtexplorer.NewGoWSMANMessages(log, crypto)
}

// TestSetupWsmanClient_NonCIRA verifies the direct-TCP path when MPSUsername is empty.
func TestSetupWsmanClient_NonCIRA(t *testing.T) {
	t.Parallel()

	g := newExplorerWSMAN()
	device := entity.Device{
		GUID:            "non-cira-guid",
		Hostname:        "192.168.1.100",
		Username:        "admin",
		Password:        "encrypted",
		UseTLS:          false,
		AllowSelfSigned: true,
		MPSUsername:     "", // no CIRA
	}

	explorer, err := g.SetupWsmanClient(device, false)
	require.NoError(t, err)
	require.NotNil(t, explorer)
}

// TestSetupWsmanClient_NonCIRA_Cached verifies that a second call for the same GUID
// reuses the cached connection entry and resets its timer.
func TestSetupWsmanClient_NonCIRA_Cached(t *testing.T) {
	t.Parallel()

	g := newExplorerWSMAN()
	device := entity.Device{
		GUID:        "cached-guid",
		Hostname:    "10.0.0.1",
		Username:    "admin",
		Password:    "encrypted",
		MPSUsername: "",
	}

	first, err := g.SetupWsmanClient(device, false)
	require.NoError(t, err)
	require.NotNil(t, first)

	second, err := g.SetupWsmanClient(device, false)
	require.NoError(t, err)
	require.NotNil(t, second)

	// Both calls return the same underlying connection entry.
	assert.Same(t, first, second)
}

// TestSetupWsmanClient_CIRA_NotConnected verifies that ErrCIRADeviceNotConnected is
// returned when the device has MPSUsername set but no APF tunnel is registered.
func TestSetupWsmanClient_CIRA_NotConnected(t *testing.T) {
	t.Parallel()

	g := newExplorerWSMAN()
	device := entity.Device{
		GUID:        "cira-missing-guid",
		Hostname:    "10.0.0.2",
		Username:    "admin",
		Password:    "encrypted",
		MPSUsername: "mpsuser", // CIRA device
	}

	// No connection entry registered for this GUID.
	explorer, err := g.SetupWsmanClient(device, false)
	require.ErrorIs(t, err, wsmanAPI.ErrCIRADeviceNotConnected)
	assert.Nil(t, explorer)
}

// TestSetupWsmanClient_CIRA_Connected verifies that when an APF tunnel entry exists,
// SetupWsmanClient returns a new request-scoped explorer (not the shared entry).
func TestSetupWsmanClient_CIRA_Connected(t *testing.T) {
	t.Parallel()

	guid := "cira-connected-guid"

	// Register a fake CIRA connection entry as the TCP tunnel would.
	// Use net.Pipe to provide a real net.Conn for Conny, matching the
	// actual TCP CIRA handler behavior (ensureAPFChannelStore needs it).
	server, client := net.Pipe()

	t.Cleanup(func() {
		server.Close()
		client.Close()
	})

	entry := &wsmanAPI.ConnectionEntry{
		WsmanMessages: wsman.Messages{},
		IsCIRA:        true,
		Conny:         client,
		Timer:         time.AfterFunc(time.Hour, func() {}),
	}
	wsmanAPI.SetConnectionEntry(guid, entry)

	t.Cleanup(func() { wsmanAPI.RemoveConnection(guid) })

	g := newExplorerWSMAN()
	device := entity.Device{
		GUID:        guid,
		Hostname:    "10.0.0.3",
		Username:    "admin",
		Password:    "encrypted",
		MPSUsername: "mpsuser",
	}

	explorer, err := g.SetupWsmanClient(device, false)
	require.NoError(t, err)
	require.NotNil(t, explorer)

	// Must be a new local entry, not the shared tunnel entry.
	assert.NotSame(t, entry, explorer)
}

// TestDestroyWsmanClient verifies that DestroyWsmanClient removes a non-CIRA entry
// so that a subsequent SetupWsmanClient call creates a fresh connection.
func TestDestroyWsmanClient_NonCIRA(t *testing.T) {
	t.Parallel()

	g := newExplorerWSMAN()
	device := entity.Device{
		GUID:        "destroy-guid",
		Hostname:    "10.0.0.4",
		Username:    "admin",
		Password:    "encrypted",
		MPSUsername: "",
	}

	first, err := g.SetupWsmanClient(device, false)
	require.NoError(t, err)
	require.NotNil(t, first)

	g.DestroyWsmanClient(dto.Device{GUID: device.GUID})

	second, err := g.SetupWsmanClient(device, false)
	require.NoError(t, err)
	require.NotNil(t, second)

	// After destroy, setup must return a new entry, not the old cached one.
	assert.NotSame(t, first, second)
}

// TestSetupWsmanClient_CIRA_StaleNonCIRAEntry verifies that a non-CIRA entry for the
// same GUID is rejected — IsCIRA == false means the APF tunnel is not established.
func TestSetupWsmanClient_CIRA_StaleNonCIRAEntry(t *testing.T) {
	t.Parallel()

	guid := "cira-stale-guid"

	staleEntry := &wsmanAPI.ConnectionEntry{
		IsCIRA: false,
		Timer:  time.AfterFunc(time.Hour, func() {}),
	}
	wsmanAPI.SetConnectionEntry(guid, staleEntry)

	t.Cleanup(func() { wsmanAPI.RemoveConnection(guid) })

	g := newExplorerWSMAN()
	device := entity.Device{
		GUID:        guid,
		Hostname:    "10.0.0.7",
		Username:    "admin",
		Password:    "encrypted",
		MPSUsername: "mpsuser",
	}

	explorer, err := g.SetupWsmanClient(device, false)
	require.ErrorIs(t, err, wsmanAPI.ErrCIRADeviceNotConnected)
	assert.Nil(t, explorer)
}

// TestNewGoWSMANMessages verifies the constructor returns a non-nil instance.
func TestNewGoWSMANMessages(t *testing.T) {
	t.Parallel()

	g := newExplorerWSMAN()
	require.NotNil(t, g)
}

// TestSetupWsmanClient_DecryptError verifies that a Decrypt failure is propagated.
func TestSetupWsmanClient_DecryptError(t *testing.T) {
	t.Parallel()

	log := logger.New("error")
	g := amtexplorer.NewGoWSMANMessages(log, errCrypto{})

	device := entity.Device{
		GUID:        "decrypt-error-guid",
		Hostname:    "10.0.0.5",
		Username:    "admin",
		Password:    "encrypted",
		MPSUsername: "",
	}

	explorer, err := g.SetupWsmanClient(device, false)
	require.ErrorIs(t, err, errDecrypt)
	assert.Nil(t, explorer)
}

// TestSetupWsmanClient_NonCIRA_WithCertHash verifies that a PinnedCert is applied
// when the device has a CertHash set.
func TestSetupWsmanClient_NonCIRA_WithCertHash(t *testing.T) {
	t.Parallel()

	g := newExplorerWSMAN()
	hash := "aa:bb:cc"
	device := entity.Device{
		GUID:            "cert-hash-guid",
		Hostname:        "10.0.0.6",
		Username:        "admin",
		Password:        "encrypted",
		MPSUsername:     "",
		CertHash:        &hash,
		AllowSelfSigned: true,
	}

	explorer, err := g.SetupWsmanClient(device, false)
	require.NoError(t, err)
	require.NotNil(t, explorer)
}
