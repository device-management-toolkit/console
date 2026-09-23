package devices

import (
	"context"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/client"

	"github.com/device-management-toolkit/console/internal/entity"
	wsmanAPI "github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
)

type Redirector struct {
	SafeRequirements security.Cryptor
	creds            *wsmanAPI.CredentialResolver
}

// SetCredentialResolver wires the Vault resolver shared with RPS.
func (g *Redirector) SetCredentialResolver(creds *wsmanAPI.CredentialResolver) {
	g.creds = creds
}

func (g *Redirector) SetupWsmanClient(_ context.Context, device entity.Device, isRedirection, logAMTMessages bool) (wsman.Messages, entity.Device, error) {
	decryptedPassword, err := wsmanAPI.DecryptStoredPassword(g.SafeRequirements, device.Password)
	if err != nil {
		return wsman.Messages{}, device, err
	}

	device.Password = decryptedPassword
	device = g.creds.ApplyAMT(device)

	// CIRA device: route redirection through the APF tunnel
	if isRedirection && device.MPSUsername != "" {
		connection := wsmanAPI.GetConnectionEntry(device.GUID)
		if connection == nil {
			return wsman.Messages{}, device, wsmanAPI.ErrCIRADeviceNotConnected
		}

		return wsman.NewCIRARedirectionMessages(connection), device, nil
	}

	clientParams := client.Parameters{
		Target:            device.Hostname,
		Username:          device.Username,
		Password:          device.Password,
		UseDigest:         true,
		UseTLS:            device.UseTLS,
		SelfSignedAllowed: device.AllowSelfSigned,
		LogAMTMessages:    logAMTMessages,
		IsRedirection:     isRedirection,
	}

	if device.CertHash != nil {
		clientParams.PinnedCert = *device.CertHash
	}

	return wsman.NewMessages(clientParams), device, nil
}

func NewRedirector(safeRequirements security.Cryptor) *Redirector {
	return &Redirector{
		SafeRequirements: safeRequirements,
	}
}

func (g *Redirector) RedirectConnect(_ context.Context, deviceConnection *DeviceConnection) error {
	err := deviceConnection.wsmanMessages.Client.Connect()
	if err != nil {
		return err
	}

	return nil
}

func (g *Redirector) RedirectSend(_ context.Context, deviceConnection *DeviceConnection, data []byte) error {
	err := deviceConnection.wsmanMessages.Client.Send(data)
	if err != nil {
		return err
	}

	return nil
}

func (g *Redirector) RedirectListen(_ context.Context, deviceConnection *DeviceConnection) ([]byte, error) {
	data, err := deviceConnection.wsmanMessages.Client.Receive()
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (g *Redirector) RedirectClose(_ context.Context, deviceConnection *DeviceConnection) error {
	err := deviceConnection.wsmanMessages.Client.CloseConnection()
	if err != nil {
		return err
	}

	return nil
}
