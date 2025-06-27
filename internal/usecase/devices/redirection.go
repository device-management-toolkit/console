package devices

import (
	"context"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/client"

	"github.com/device-management-toolkit/console/internal/entity"
)

type Redirector struct {
	SafeRequirements security.Cryptor
}

func (g *Redirector) SetupWsmanClient(device entity.Device, isRedirection, logAMTMessages bool) wsman.Messages {
	clientParams := client.Parameters{
		Target:            device.Hostname,
		Username:          device.Username,
		UseDigest:         true,
		UseTLS:            device.UseTLS,
		SelfSignedAllowed: device.AllowSelfSigned,
		LogAMTMessages:    logAMTMessages,
		IsRedirection:     isRedirection,
	}

	if device.CertHash != nil {
		clientParams.PinnedCert = *device.CertHash
	}

	clientParams.Password, _ = g.SafeRequirements.Decrypt(device.Password)

	return wsman.NewMessages(clientParams)
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
