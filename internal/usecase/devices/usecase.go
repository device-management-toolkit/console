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
func (uc *UseCase) dtoToEntity(d *dto.Device) (*entity.Device, error) {
	// convert []string to comma separated string
	if d.Tags == nil {
		d.Tags = []string{}
	}

	tags := strings.Join(d.Tags, ",")

	d1 := &entity.Device{
		ConnectionStatus: d.ConnectionStatus,
		MPSInstance:      d.MPSInstance,
		Hostname:         d.Hostname,
		GUID:             strings.ToLower(d.GUID), // Normalize GUID to lowercase for case-insensitive matching
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
		return nil, ErrDeviceUseCase.Wrap("dtoToEntity", "failed to encrypt password", err)
	}

	if d.MPSPassword == "" {
		d1.MPSPassword = nil
	} else {
		encrypted, err := uc.safeRequirements.Encrypt(d.MPSPassword)
		if err != nil {
			return nil, ErrDeviceUseCase.Wrap("dtoToEntity", "failed to encrypt MPS password", err)
		}

		d1.MPSPassword = &encrypted
	}

	if d.MEBXPassword == "" {
		d1.MEBXPassword = nil
	} else {
		encrypted, err := uc.safeRequirements.Encrypt(d.MEBXPassword)
		if err != nil {
			return nil, ErrDeviceUseCase.Wrap("dtoToEntity", "failed to encrypt MEBX password", err)
		}

		d1.MEBXPassword = &encrypted
	}

	if d.CertHash == "" {
		d1.CertHash = nil
	} else {
		d1.CertHash = &d.CertHash
	}

	return d1, nil
}

// Keys are lowercased to match encoding/json's case-insensitive unmarshal.
// guid and tenantId identify the record; deviceInfo doesn't round-trip through
// dtoToEntity/entityToDTO — all three are intentionally omitted.
var deviceFieldSetters = map[string]func(dst, src *dto.Device){
	"connectionstatus": func(dst, src *dto.Device) { dst.ConnectionStatus = src.ConnectionStatus },
	"mpsinstance":      func(dst, src *dto.Device) { dst.MPSInstance = src.MPSInstance },
	"hostname":         func(dst, src *dto.Device) { dst.Hostname = src.Hostname },
	"mpsusername":      func(dst, src *dto.Device) { dst.MPSUsername = src.MPSUsername },
	"tags":             func(dst, src *dto.Device) { dst.Tags = src.Tags },
	"friendlyname":     func(dst, src *dto.Device) { dst.FriendlyName = src.FriendlyName },
	"dnssuffix":        func(dst, src *dto.Device) { dst.DNSSuffix = src.DNSSuffix },
	"lastconnected":    func(dst, src *dto.Device) { dst.LastConnected = src.LastConnected },
	"lastseen":         func(dst, src *dto.Device) { dst.LastSeen = src.LastSeen },
	"lastdisconnected": func(dst, src *dto.Device) { dst.LastDisconnected = src.LastDisconnected },
	"username":         func(dst, src *dto.Device) { dst.Username = src.Username },
	"password":         func(dst, src *dto.Device) { dst.Password = src.Password },
	"mpspassword":      func(dst, src *dto.Device) { dst.MPSPassword = src.MPSPassword },
	"mebxpassword":     func(dst, src *dto.Device) { dst.MEBXPassword = src.MEBXPassword },
	"usetls":           func(dst, src *dto.Device) { dst.UseTLS = src.UseTLS },
	"allowselfsigned":  func(dst, src *dto.Device) { dst.AllowSelfSigned = src.AllowSelfSigned },
	"certhash":         func(dst, src *dto.Device) { dst.CertHash = src.CertHash },
}

func mergeDeviceFields(dst, src *dto.Device, fields map[string]bool) {
	for key := range fields {
		if apply, ok := deviceFieldSetters[key]; ok {
			apply(dst, src)
		}
	}
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

	if d.MPSPassword != nil {
		d1.MPSPassword = *d.MPSPassword
	}

	if d.MEBXPassword != nil {
		d1.MEBXPassword = *d.MEBXPassword
	}

	return d1
}
