package devices

import (
	"context"

	"github.com/gorilla/websocket"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/power"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	dtov2 "github.com/device-management-toolkit/console/internal/entity/dto/v2"
	wsmanAPI "github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
)

type (
	WSMAN interface {
		SetupWsmanClient(device entity.Device, isRedirection, logMessages bool) wsmanAPI.Management
		DestroyWsmanClient(device dto.Device)
		Worker()
	}

	WebSocketConn interface {
		ReadMessage() (int, []byte, error)
		WriteMessage(messageType int, data []byte) error
		Close() error
	}

	Redirection interface {
		SetupWsmanClient(device entity.Device, isRedirection, logMessages bool) wsman.Messages
		RedirectConnect(ctx context.Context, deviceConnection *DeviceConnection) error
		RedirectClose(ctx context.Context, deviceConnection *DeviceConnection) error
		RedirectListen(ctx context.Context, deviceConnection *DeviceConnection) ([]byte, error)
		RedirectSend(ctx context.Context, deviceConnection *DeviceConnection, message []byte) error
	}
	Repository interface {
		GetCount(context.Context, string) (int, error)
		Get(ctx context.Context, top, skip int, tenantID string) ([]entity.Device, error)
		GetByID(ctx context.Context, guid, tenantID string) (*entity.Device, error)
		GetDistinctTags(ctx context.Context, tenantID string) ([]string, error)
		GetByTags(ctx context.Context, tags []string, method string, limit, offset int, tenantID string) ([]entity.Device, error)
		Delete(ctx context.Context, guid, tenantID string) (bool, error)
		Update(ctx context.Context, d *entity.Device) (bool, error)
		Insert(ctx context.Context, d *entity.Device) (string, error)
		GetByColumn(ctx context.Context, columnName, queryValue, tenantID string) ([]entity.Device, error)
	}
	Feature interface {
		// Repository/Database Calls
		GetCount(context.Context, string) (int, error)
		Get(ctx context.Context, top, skip int, tenantID string) ([]dto.Device, error)
		GetByID(ctx context.Context, guid, tenantID string, includeSecrets bool) (*dto.Device, error)
		GetDistinctTags(ctx context.Context, tenantID string) ([]string, error)
		GetByTags(ctx context.Context, tags, method string, limit, offset int, tenantID string) ([]dto.Device, error)
		Delete(ctx context.Context, guid, tenantID string) error
		Update(ctx context.Context, d *dto.Device) (*dto.Device, error)
		Insert(ctx context.Context, d *dto.Device) (*dto.Device, error)
		GetByColumn(ctx context.Context, columnName, queryValue, tenantID string) ([]dto.Device, error)
		// Management Calls
		GetVersion(ctx context.Context, guid string) (dto.Version, dtov2.Version, error)
		GetFeatures(ctx context.Context, guid string) (dto.Features, dtov2.Features, error)
		SetFeatures(ctx context.Context, guid string, features dto.Features) (dto.Features, dtov2.Features, error)
		GetAlarmOccurrences(ctx context.Context, guid string) ([]dto.AlarmClockOccurrence, error)
		CreateAlarmOccurrences(ctx context.Context, guid string, alarm dto.AlarmClockOccurrenceInput) (dto.AddAlarmOutput, error)
		DeleteAlarmOccurrences(ctx context.Context, guid, instanceID string) error
		GetHardwareInfo(ctx context.Context, guid string) (interface{}, error)
		GetPowerState(ctx context.Context, guid string) (dto.PowerState, error)
		GetPowerCapabilities(ctx context.Context, guid string) (dto.PowerCapabilities, error)
		GetGeneralSettings(ctx context.Context, guid string) (interface{}, error)
		CancelUserConsent(ctx context.Context, guid string) (dto.UserConsentMessage, error)
		GetUserConsentCode(ctx context.Context, guid string) (dto.GetUserConsentMessage, error)
		SendConsentCode(ctx context.Context, code dto.UserConsentCode, guid string) (dto.UserConsentMessage, error)
		SendPowerAction(ctx context.Context, guid string, action int) (power.PowerActionResponse, error)
		SetBootOptions(ctx context.Context, guid string, bootSetting dto.BootSetting) (power.PowerActionResponse, error)
		GetAuditLog(ctx context.Context, startIndex int, guid string) (dto.AuditLog, error)
		GetEventLog(ctx context.Context, startIndex, maxReadRecords int, guid string) (dto.EventLogs, error)
		Redirect(ctx context.Context, conn *websocket.Conn, guid, mode string) error
		GetNetworkSettings(c context.Context, guid string) (dto.NetworkSettings, error)
		GetCertificates(c context.Context, guid string) (dto.SecuritySettings, error)
		GetTLSSettingData(c context.Context, guid string) ([]dto.SettingDataResponse, error)
		GetDiskInfo(c context.Context, guid string) (interface{}, error)
		GetDeviceCertificate(c context.Context, guid string) (dto.Certificate, error)
		AddCertificate(c context.Context, guid string, certInfo dto.CertInfo) (string, error)
		GetBootSourceSetting(c context.Context, guid string) ([]dto.BootSources, error)
		// KVM Screen Settings (IPS_ScreenSettingData)
		GetKVMScreenSettings(c context.Context, guid string) (dto.KVMScreenSettings, error)
		SetKVMScreenSettings(c context.Context, guid string, req dto.KVMScreenSettingsRequest) (dto.KVMScreenSettings, error)
	}
)
