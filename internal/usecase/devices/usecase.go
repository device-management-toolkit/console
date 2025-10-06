package devices

import (
	"strings"
	"sync"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
	"github.com/device-management-toolkit/console/pkg/logger"
)

const (
	RedirectionCommandsStartRedirectionSession      = 16
	RedirectionCommandsStartRedirectionSessionReply = 17
	RedirectionCommandsEndRedirectionSession        = 18
	RedirectionCommandsAuthenticateSession          = 19
	RedirectionCommandsAuthenticateSessionReply     = 20
	StartRedirectionSessionReplyStatusSuccess       = 0
	StartRedirectionSessionReplyStatusUnknown       = 1
	StartRedirectionSessionReplyStatusBusy          = 2
	StartRedirectionSessionReplyStatusUnsupported   = 3
	StartRedirectionSessionReplyStatusError         = 0xFF
	AuthenticationTypeQuery                         = 0
	AuthenticationTypeUserPass                      = 1
	AuthenticationTypeKerberos                      = 2
	AuthenticationTypeBadDigest                     = 3
	AuthenticationTypeDigest                        = 4
	AuthenticationStatusSuccess                     = 0
	AuthenticationStatusFail                        = 1
	AuthenticationStatusNotSupported                = 2

	// MinAMTVersion - minimum AMT version required for certain features in power capabilities.
	MinAMTVersion = 9
)

// UseCase -.
type UseCase struct {
	repo             Repository
	device           WSMAN
	redirection      Redirection
	redirConnections map[string]*DeviceConnection
	redirMutex       sync.RWMutex // Protects redirConnections map
	log              logger.Interface
	safeRequirements security.Cryptor
}

var ErrAMT = AMTError{Console: consoleerrors.CreateConsoleError("DevicesUseCase")}

// New -.
func New(r Repository, d WSMAN, redirection Redirection, log logger.Interface, safeRequirements security.Cryptor) *UseCase {
	uc := &UseCase{
		repo:             r,
		device:           d,
		redirection:      redirection,
		redirConnections: make(map[string]*DeviceConnection),
		log:              log,
		safeRequirements: safeRequirements,
	}
	// start up the worker
	go d.Worker()

	return uc
}

// convert dto.Device to entity.Device.
func (uc *UseCase) dtoToEntity(d *dto.Device) *entity.Device {
	// convert []string to comma separated string
	if d.Tags == nil {
		d.Tags = []string{}
	}

	tags := strings.Join(d.Tags, ",")

	d1 := &entity.Device{
		ConnectionStatus: d.ConnectionStatus,
		MPSInstance:      d.MPSInstance,
		Hostname:         d.Hostname,
		GUID:             d.GUID,
		MPSUsername:      d.MPSUsername,
		Tags:             tags,
		TenantID:         d.TenantID,
		FriendlyName:     d.FriendlyName,
		DNSSuffix:        d.DNSSuffix,
		LastConnected:    d.LastConnected,
		LastSeen:         d.LastSeen,
		LastDisconnected: d.LastDisconnected,
		// DeviceInfo:       d.DeviceInfo,
		Username:        d.Username,
		Password:        d.Password,
		UseTLS:          d.UseTLS,
		AllowSelfSigned: d.AllowSelfSigned,
	}

	var err error

	d1.Password, err = uc.safeRequirements.Encrypt(d1.Password)
	if err != nil {
		uc.log.Error("Error encrypting password")
	}

	if d.CertHash == "" {
		d1.CertHash = nil
	} else {
		d1.CertHash = &d.CertHash
	}

	return d1
}

// convert entity.Device to dto.Device.
func (uc *UseCase) entityToDTO(d *entity.Device) *dto.Device {
	// convert comma separated string to []string
	var tags []string
	if d.Tags != "" {
		tags = strings.Split(d.Tags, ",")
	}

	d1 := &dto.Device{
		ConnectionStatus: d.ConnectionStatus,
		MPSInstance:      d.MPSInstance,
		Hostname:         d.Hostname,
		GUID:             d.GUID,
		MPSUsername:      d.MPSUsername,
		Tags:             tags,
		TenantID:         d.TenantID,
		FriendlyName:     d.FriendlyName,
		DNSSuffix:        d.DNSSuffix,
		LastConnected:    d.LastConnected,
		LastSeen:         d.LastSeen,
		LastDisconnected: d.LastDisconnected,
		// DeviceInfo:       d.DeviceInfo,
		Username: d.Username,
		// Password:        d.Password,
		UseTLS:          d.UseTLS,
		AllowSelfSigned: d.AllowSelfSigned,
	}

	if d.CertHash != nil {
		d1.CertHash = *d.CertHash
	}

	return d1
}
