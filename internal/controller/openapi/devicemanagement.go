package openapi

import (
	"github.com/go-fuego/fuego"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

func (f *FuegoAdapter) RegisterDeviceManagementRoutes() {
	f.registerKVMAndCertificateRoutes()
	f.registerExplorerRoutes()
	f.registerNetworkAndFeatureRoutes()
	f.registerUserConsentRoutes()
	f.registerPowerRoutes()
	f.registerLogsAndAlarmRoutes()
	f.registerVersionAndHardwareRoutes()
}

func (f *FuegoAdapter) registerKVMAndCertificateRoutes() {
	// kvm displays
	fuego.Get(f.server, "/api/v1/admin/kvm/displays/{guid}", f.getKVMDisplays,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Get KVM displays"),
		fuego.OptionDescription("Retrieve current KVM display settings for a device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	fuego.Put(f.server, "/api/v1/admin/kvm/displays/{guid}", f.setKVMDisplays,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Set KVM displays"),
		fuego.OptionDescription("Update KVM display settings for a device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	// Certificates
	fuego.Get(f.server, "/api/v1/admin/amt/certificates/{guid}", f.getCertificates,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Get Certificates"),
		fuego.OptionDescription("Retrieve certificate and key information for a device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	fuego.Post(f.server, "/api/v1/admin/amt/certificates/{guid}", f.addCertificate,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Add Certificate"),
		fuego.OptionDescription("Add a certificate to the device"),
		fuego.OptionPath("guid", "Device GUID"),
	)
}

func (f *FuegoAdapter) registerExplorerRoutes() {
	// Explorer endpoints
	fuego.Get(f.server, "/api/v1/admin/amt/explorer", f.getCallList,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Get Explorer Calls"),
		fuego.OptionDescription("Retrieve supported AMT explorer calls"),
	)

	fuego.Get(f.server, "/api/v1/admin/amt/explorer/{guid}/{call}", f.executeCall,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Execute Explorer Call"),
		fuego.OptionDescription("Execute an AMT explorer call on a device"),
		fuego.OptionPath("guid", "Device GUID"),
		fuego.OptionPath("call", "Explorer call name"),
	)
}

func (f *FuegoAdapter) registerNetworkAndFeatureRoutes() {
	// TLS settings
	fuego.Get(f.server, "/api/v1/admin/amt/tls/{guid}", f.getTLSSettingData,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Get TLS Setting Data"),
		fuego.OptionDescription("Retrieve TLS setting data for a device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	// Network settings
	fuego.Get(f.server, "/api/v1/admin/amt/networkSettings/{guid}", f.getNetworkSettings,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Get Network Settings"),
		fuego.OptionDescription("Retrieve network settings for a device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	// Features
	fuego.Get(f.server, "/api/v1/admin/amt/features/{guid}", f.getFeatures,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Get Features"),
		fuego.OptionDescription("Retrieve feature flags for a device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	fuego.Post(f.server, "/api/v1/admin/amt/features/{guid}", f.setFeatures,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Set Features"),
		fuego.OptionDescription("Update feature flags for a device"),
		fuego.OptionPath("guid", "Device GUID"),
	)
}

func (f *FuegoAdapter) registerUserConsentRoutes() {
	// User consent code
	fuego.Get(f.server, "/api/v1/admin/amt/userConsentCode/cancel/{guid}", f.cancelUserConsentCode,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Cancel User Consent Code"),
		fuego.OptionDescription("Cancel a previously issued user consent code for a device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	fuego.Get(f.server, "/api/v1/admin/amt/userConsentCode/{guid}", f.getUserConsentCode,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Get User Consent Code"),
		fuego.OptionDescription("Retrieve the current user consent code for a device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	fuego.Post(f.server, "/api/v1/admin/amt/userConsentCode/{guid}", f.sendConsentCode,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Send User Consent Code"),
		fuego.OptionDescription("Send a user consent code to the device"),
		fuego.OptionPath("guid", "Device GUID"),
	)
}

func (f *FuegoAdapter) registerPowerRoutes() {
	// Power endpoints
	fuego.Get(f.server, "/api/v1/admin/amt/power/state/{guid}", f.getPowerState,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Get Power State"),
		fuego.OptionDescription("Retrieve the current power state of a device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	fuego.Post(f.server, "/api/v1/admin/amt/power/action/{guid}", f.powerAction,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Perform Power Action"),
		fuego.OptionDescription("Perform a power action on a device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	fuego.Post(f.server, "/api/v1/admin/amt/power/bootOptions/{guid}", f.setBootOptions,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Set Boot Options"),
		fuego.OptionDescription("Set boot options on a device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	fuego.Post(f.server, "/api/v1/admin/amt/power/bootoptions/{guid}", f.setBootOptions,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Set Boot Options (alt path)"),
		fuego.OptionDescription("Set boot options on a device (alternate path)"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	fuego.Get(f.server, "/api/v1/admin/amt/power/bootSources/{guid}", f.getBootSources,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Get Boot Sources"),
		fuego.OptionDescription("Retrieve available boot sources for a device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	fuego.Get(f.server, "/api/v1/admin/amt/power/capabilities/{guid}", f.getPowerCapabilities,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Get Power Capabilities"),
		fuego.OptionDescription("Retrieve power capabilities for a device"),
		fuego.OptionPath("guid", "Device GUID"),
	)
}

func (f *FuegoAdapter) registerLogsAndAlarmRoutes() {
	// Audit and Event logs
	fuego.Get(f.server, "/api/v1/admin/amt/log/audit/{guid}", f.getAuditLog,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Get Audit Log"),
		fuego.OptionDescription("Retrieve audit log entries for a device"),
		fuego.OptionPath("guid", "Device GUID"),
		fuego.OptionQuery("startIndex", "Start index for pagination"),
	)

	fuego.Get(f.server, "/api/v1/admin/amt/log/audit/{guid}/download", f.downloadAuditLog,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Download Audit Log"),
		fuego.OptionDescription("Download audit logs as CSV for a device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	fuego.Get(f.server, "/api/v1/admin/amt/log/event/{guid}", f.getEventLog,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Get Event Log"),
		fuego.OptionDescription("Retrieve event log entries for a device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	fuego.Get(f.server, "/api/v1/admin/amt/log/event/{guid}/download", f.downloadEventLog,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Download Event Log"),
		fuego.OptionDescription("Download event logs as CSV for a device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	// Alarm occurrences
	fuego.Get(f.server, "/api/v1/admin/amt/alarmOccurrences/{guid}", f.getAlarmOccurrences,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Get Alarm Occurrences"),
		fuego.OptionDescription("Retrieve alarm occurrences for a device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	fuego.Post(f.server, "/api/v1/admin/amt/alarmOccurrences/{guid}", f.createAlarmOccurrences,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Create Alarm Occurrence"),
		fuego.OptionDescription("Create an alarm occurrence on a device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	fuego.Delete(f.server, "/api/v1/admin/amt/alarmOccurrences/{guid}", f.deleteAlarmOccurrences,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Delete Alarm Occurrence"),
		fuego.OptionDescription("Delete an alarm occurrence from a device"),
		fuego.OptionPath("guid", "Device GUID"),
	)
}

func (f *FuegoAdapter) registerVersionAndHardwareRoutes() {
	// Version
	fuego.Get(f.server, "/api/v1/admin/amt/version/{guid}", f.getVersion,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Get Version"),
		fuego.OptionDescription("Retrieve AMT/software version information for a device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	// Hardware
	fuego.Get(f.server, "/api/v1/admin/amt/hardwareInfo/{guid}", f.getHardwareInfo,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Get Hardware Info"),
		fuego.OptionDescription("Retrieve hardware information for a device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	// Disk Info
	fuego.Get(f.server, "/api/v1/admin/amt/diskInfo/{guid}", f.getDiskInfo,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Get Disk Info"),
		fuego.OptionDescription("Retrieve disk information for a device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	// General Settings
	fuego.Get(f.server, "/api/v1/admin/amt/generalSettings/{guid}", f.getGeneralSettings,
		fuego.OptionTags("Device Management"),
		fuego.OptionSummary("Get General Settings"),
		fuego.OptionDescription("Retrieve general settings for a device"),
		fuego.OptionPath("guid", "Device GUID"),
	)
}

func (f *FuegoAdapter) getKVMDisplays(_ fuego.ContextNoBody) (dto.KVMScreenSettings, error) {
	return dto.KVMScreenSettings{
		Displays: []dto.KVMScreenDisplay{
			{
				DisplayIndex: 0,
				IsActive:     true,
				ResolutionX:  1920,
				ResolutionY:  1080,
				UpperLeftX:   0,
				UpperLeftY:   0,
				Role:         "primary",
				IsDefault:    true,
			},
		},
	}, nil
}

func (f *FuegoAdapter) setKVMDisplays(c fuego.ContextWithBody[dto.KVMScreenSettingsRequest]) (dto.KVMScreenSettings, error) {
	req, err := c.Body()
	if err != nil {
		return dto.KVMScreenSettings{}, err
	}

	display := dto.KVMScreenDisplay{
		DisplayIndex: req.DisplayIndex,
		IsActive:     true,
		ResolutionX:  1920,
		ResolutionY:  1080,
		UpperLeftX:   0,
		UpperLeftY:   0,
		Role:         "primary",
		IsDefault:    true,
	}

	return dto.KVMScreenSettings{Displays: []dto.KVMScreenDisplay{display}}, nil
}

func (f *FuegoAdapter) getCertificates(_ fuego.ContextNoBody) (dto.SecuritySettings, error) {
	return dto.SecuritySettings{}, nil
}

func (f *FuegoAdapter) addCertificate(_ fuego.ContextWithBody[dto.CertInfo]) (string, error) {
	return "example-handle-123", nil
}

func (f *FuegoAdapter) getCallList(_ fuego.ContextNoBody) ([]string, error) {
	return []string{"GetInventory", "Reboot", "CollectLogs"}, nil
}

func (f *FuegoAdapter) executeCall(_ fuego.ContextNoBody) (any, error) {
	return map[string]any{"status": "executed", "result": "ok"}, nil
}

func (f *FuegoAdapter) getTLSSettingData(_ fuego.ContextNoBody) ([]dto.SettingDataResponse, error) {
	return []dto.SettingDataResponse{}, nil
}

func (f *FuegoAdapter) getNetworkSettings(_ fuego.ContextNoBody) (dto.NetworkSettings, error) {
	return dto.NetworkSettings{}, nil
}

func (f *FuegoAdapter) cancelUserConsentCode(_ fuego.ContextNoBody) (dto.UserConsentMessage, error) {
	return dto.UserConsentMessage{}, nil
}

func (f *FuegoAdapter) getUserConsentCode(_ fuego.ContextNoBody) (dto.UserConsentMessage, error) {
	return dto.UserConsentMessage{}, nil
}

func (f *FuegoAdapter) sendConsentCode(_ fuego.ContextWithBody[dto.UserConsentCode]) (dto.UserConsentMessage, error) {
	return dto.UserConsentMessage{}, nil
}

func (f *FuegoAdapter) getPowerState(_ fuego.ContextNoBody) (dto.PowerState, error) {
	return dto.PowerState{}, nil
}

func (f *FuegoAdapter) powerAction(_ fuego.ContextWithBody[dto.PowerAction]) (dto.PowerActionResponse, error) {
	return dto.PowerActionResponse{}, nil
}

func (f *FuegoAdapter) setBootOptions(_ fuego.ContextWithBody[dto.BootSetting]) (dto.BootSetting, error) {
	return dto.BootSetting{}, nil
}

func (f *FuegoAdapter) getBootSources(_ fuego.ContextNoBody) ([]dto.BootSources, error) {
	return []dto.BootSources{}, nil
}

func (f *FuegoAdapter) getPowerCapabilities(_ fuego.ContextNoBody) (dto.PowerCapabilities, error) {
	return dto.PowerCapabilities{}, nil
}

func (f *FuegoAdapter) getAlarmOccurrences(_ fuego.ContextNoBody) ([]dto.AlarmClockOccurrence, error) {
	return []dto.AlarmClockOccurrence{}, nil
}

func (f *FuegoAdapter) createAlarmOccurrences(_ fuego.ContextWithBody[dto.AlarmClockOccurrenceInput]) (dto.AddAlarmOutput, error) {
	return dto.AddAlarmOutput{}, nil
}

func (f *FuegoAdapter) deleteAlarmOccurrences(_ fuego.ContextWithBody[dto.DeleteAlarmOccurrenceRequest]) (any, error) {
	return nil, nil
}

func (f *FuegoAdapter) getFeatures(_ fuego.ContextNoBody) (dto.Features, error) {
	return dto.Features{}, nil
}

func (f *FuegoAdapter) setFeatures(_ fuego.ContextWithBody[dto.Features]) (dto.Features, error) {
	return dto.Features{}, nil
}

func (f *FuegoAdapter) getAuditLog(_ fuego.ContextNoBody) (dto.AuditLog, error) {
	return dto.AuditLog{}, nil
}

func (f *FuegoAdapter) downloadAuditLog(_ fuego.ContextNoBody) (string, error) {
	return "timestamp,level,message\n", nil
}

func (f *FuegoAdapter) getEventLog(_ fuego.ContextNoBody) (dto.EventLogs, error) {
	return dto.EventLogs{}, nil
}

func (f *FuegoAdapter) downloadEventLog(_ fuego.ContextNoBody) (string, error) {
	return "timestamp,source,desc\n", nil
}

func (f *FuegoAdapter) getVersion(_ fuego.ContextNoBody) (dto.Version, error) {
	return dto.Version{}, nil
}

func (f *FuegoAdapter) getHardwareInfo(_ fuego.ContextNoBody) (dto.HardwareInfo, error) {
	return dto.HardwareInfo{}, nil
}

func (f *FuegoAdapter) getDiskInfo(_ fuego.ContextNoBody) (dto.DiskInfo, error) {
	return dto.DiskInfo{}, nil
}

func (f *FuegoAdapter) getGeneralSettings(_ fuego.ContextNoBody) (dto.GeneralSettings, error) {
	return dto.GeneralSettings{}, nil
}
