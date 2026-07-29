package amtexplorer

import (
	"sync"
	"time"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/client"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	wsmanAPI "github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/pkg/logger"
)

var (
	connections   = make(map[string]*wsmanAPI.ConnectionEntry)
	connectionsMu sync.Mutex
	expireAfter   = 5 * time.Minute // Set the expiration duration as needed
)

type GoWSMANMessages struct {
	log              logger.Interface
	safeRequirements security.Cryptor
}

func NewGoWSMANMessages(log logger.Interface, safeRequirements security.Cryptor) *GoWSMANMessages {
	return &GoWSMANMessages{
		log:              log,
		safeRequirements: safeRequirements,
	}
}

func (g GoWSMANMessages) DestroyWsmanClient(device dto.Device) {
	connectionsMu.Lock()
	defer connectionsMu.Unlock()

	if entry, ok := connections[device.GUID]; ok {
		entry.Timer.Stop()
		delete(connections, device.GUID)
	}
}

func (g GoWSMANMessages) SetupWsmanClient(device entity.Device, logAMTMessages bool) (AMTExplorer, error) {
	decryptedPassword, err := g.safeRequirements.Decrypt(device.Password)
	if err != nil {
		return nil, err
	}

	// CIRA device: route through the APF tunnel registered by the TCP CIRA handler.
	if device.MPSUsername != "" {
		connection := wsmanAPI.GetConnectionEntry(device.GUID)
		if connection == nil || !connection.IsCIRA {
			return nil, wsmanAPI.ErrCIRADeviceNotConnected
		}

		cp := client.Parameters{
			Target:            device.GUID,
			IsRedirection:     false,
			Username:          device.Username,
			Password:          decryptedPassword,
			SelfSignedAllowed: true,
			UseDigest:         true,
			LogAMTMessages:    logAMTMessages,
			IsCIRA:            true,
			CIRAManager:       connection,
		}

		// Create a local, request-scoped entry so we never mutate the shared
		// ConnectionEntry that the TCP CIRA handler owns. This avoids a data race
		// between concurrent explorer calls and concurrent Get* method calls on
		// the same entry.
		return &wsmanAPI.ConnectionEntry{
			WsmanMessages: wsman.NewMessages(cp),
			IsCIRA:        true,
		}, nil
	}

	clientParams := client.Parameters{
		Target:            device.Hostname,
		Username:          device.Username,
		Password:          decryptedPassword,
		UseDigest:         true,
		UseTLS:            device.UseTLS,
		SelfSignedAllowed: device.AllowSelfSigned,
		LogAMTMessages:    logAMTMessages,
		IsRedirection:     false,
	}

	if device.CertHash != nil {
		clientParams.PinnedCert = *device.CertHash
	}

	connectionsMu.Lock()
	defer connectionsMu.Unlock()

	if entry, ok := connections[device.GUID]; ok {
		entry.Timer.Stop() // Stop the previous timer
		entry.Timer = time.AfterFunc(expireAfter, func() {
			removeConnection(device.GUID)
		})
	} else {
		wsmanMsgs := wsman.NewMessages(clientParams)
		timer := time.AfterFunc(expireAfter, func() {
			removeConnection(device.GUID)
		})
		connections[device.GUID] = &wsmanAPI.ConnectionEntry{
			WsmanMessages: wsmanMsgs,
			Timer:         timer,
		}
	}

	return connections[device.GUID], nil
}

func removeConnection(guid string) {
	connectionsMu.Lock()
	defer connectionsMu.Unlock()

	delete(connections, guid)
}
