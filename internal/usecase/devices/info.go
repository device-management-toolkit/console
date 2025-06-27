package devices

import (
	"context"
	"strconv"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/setupandconfiguration"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/software"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	dtov2 "github.com/device-management-toolkit/console/internal/entity/dto/v2"
)

func (uc *UseCase) GetVersion(c context.Context, guid string) (v1 dto.Version, v2 dtov2.Version, err error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return v1, v2, err
	}

	if item == nil || item.GUID == "" {
		return v1, v2, ErrNotFound
	}

	device := uc.device.SetupWsmanClient(*item, false, true)

	softwareIdentity, err := device.GetAMTVersion()
	if err != nil {
		return v1, v2, err
	}

	data, err := device.GetSetupAndConfiguration()
	if err != nil {
		return v1, v2, err
	}

	// iterate over the data and convert each entity to dto
	d1 := make([]dto.SoftwareIdentity, len(softwareIdentity))

	for i := range softwareIdentity {
		tmpEntity := softwareIdentity[i] // create a new variable to avoid memory aliasing
		d1[i] = *uc.softwareIdentityEntityToDTOv1(&tmpEntity)
	}

	// iterate over the data and convert each entity to dto
	d3 := make([]dto.SetupAndConfigurationServiceResponse, len(data))

	for i := range data {
		tmpEntity := data[i] // create a new variable to avoid memory aliasing
		d3[i] = *uc.setupAndConfigurationServiceResponseEntityToDTO(&tmpEntity)
	}

	v1 = dto.Version{
		CIMSoftwareIdentity:             dto.SoftwareIdentityResponses{Responses: d1},
		AMTSetupAndConfigurationService: dto.SetupAndConfigurationServiceResponses{Response: d3[0]},
	}

	v2 = *uc.softwareIdentityEntityToDTOv2(softwareIdentity)

	return v1, v2, nil
}

func (uc *UseCase) GetHardwareInfo(c context.Context, guid string) (interface{}, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return nil, err
	}

	if item == nil || item.GUID == "" {
		return nil, ErrNotFound
	}

	device := uc.device.SetupWsmanClient(*item, false, true)

	hwInfo, err := device.GetHardwareInfo()
	if err != nil {
		return nil, err
	}

	return hwInfo, nil
}

func (uc *UseCase) GetDiskInfo(c context.Context, guid string) (interface{}, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return nil, err
	}

	if item == nil || item.GUID == "" {
		return nil, ErrNotFound
	}

	device := uc.device.SetupWsmanClient(*item, false, true)

	diskInfo, err := device.GetDiskInfo()
	if err != nil {
		return nil, err
	}

	return diskInfo, nil
}

func (uc *UseCase) GetAuditLog(c context.Context, startIndex int, guid string) (dto.AuditLog, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return dto.AuditLog{}, err
	}

	if item == nil || item.GUID == "" {
		return dto.AuditLog{}, ErrNotFound
	}

	device := uc.device.SetupWsmanClient(*item, false, true)

	response, err := device.GetAuditLog(startIndex)
	if err != nil {
		return dto.AuditLog{}, err
	}

	auditLogResponse := dto.AuditLog{}
	auditLogResponse.TotalCount = response.Body.ReadRecordsResponse.TotalRecordCount
	auditLogResponse.Records = response.Body.DecodedRecordsResponse

	return auditLogResponse, nil
}

func (uc *UseCase) GetEventLog(c context.Context, startIndex, maxReadRecords int, guid string) (dto.EventLogs, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return dto.EventLogs{}, err
	}

	if item == nil || item.GUID == "" {
		return dto.EventLogs{}, ErrNotFound
	}

	device := uc.device.SetupWsmanClient(*item, false, true)

	eventLogs, err := device.GetEventLog(startIndex, maxReadRecords)
	if err != nil {
		return dto.EventLogs{}, err
	}

	// Initialize with nil if no records
	var events []dto.EventLog
	if len(eventLogs.RefinedEventData) > 0 {
		events = make([]dto.EventLog, len(eventLogs.RefinedEventData))

		for idx := range eventLogs.RefinedEventData {
			event := &eventLogs.RefinedEventData[idx]
			dtoEvent := dto.EventLog{
				// DeviceAddress:   event.DeviceAddress,
				// EventSensorType: event.EventSensorType,
				// EventType:       event.EventType,
				// EventOffset:     event.EventOffset,
				// EventSourceType: event.EventSourceType,
				EventSeverity: event.EventSeverity,
				// SensorNumber:    event.SensorNumber,
				Entity: event.Entity,
				// EntityInstance:  event.EntityInstance,
				// EventData:       event.EventData,
				Time: event.TimeStamp.String(),
				// EntityStr:       event.EntityStr,
				Description: event.Description,
				// EventTypeDesc:   event.EventTypeDesc,
			}

			events[idx] = dtoEvent
		}
	}

	return dto.EventLogs{
		Records:        events,
		HasMoreRecords: !eventLogs.NoMoreRecords,
	}, nil
}

func (uc *UseCase) GetGeneralSettings(c context.Context, guid string) (interface{}, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return nil, err
	}

	if item == nil || item.GUID == "" {
		return nil, ErrNotFound
	}

	device := uc.device.SetupWsmanClient(*item, false, true)

	generalSettings, err := device.GetGeneralSettings()
	if err != nil {
		return nil, err
	}

	response := map[string]interface{}{
		"Body": generalSettings,
	}

	return response, nil
}

func (uc *UseCase) softwareIdentityEntityToDTOv1(d *software.SoftwareIdentity) *dto.SoftwareIdentity {
	d1 := &dto.SoftwareIdentity{
		InstanceID:    d.InstanceID,
		VersionString: d.VersionString,
		IsEntity:      d.IsEntity,
	}

	return d1
}

func (uc *UseCase) softwareIdentityEntityToDTOv2(d []software.SoftwareIdentity) *dtov2.Version {
	data := make(map[string]string)
	for i := range d {
		data[d[i].InstanceID] = d[i].VersionString
	}

	var legacyModePointer *bool

	legacyMode, err := strconv.ParseBool(data["Legacy Mode"])
	if err == nil {
		legacyModePointer = &legacyMode
	}

	return &dtov2.Version{
		Flash:               data["Flash"],
		Netstack:            data["Netstack"],
		AMTApps:             data["AMTApps"],
		AMT:                 data["AMT"],
		SKU:                 data["Sku"],
		VendorID:            data["VendorID"],
		BuildNumber:         data["Build Number"],
		RecoveryVersion:     data["Recovery Version"],
		RecoveryBuildNumber: data["Recovery Build Num"],
		LegacyMode:          legacyModePointer,
		AMTFWCoreVersion:    data["AMT FW Core Version"],
	}
}

func (uc *UseCase) setupAndConfigurationServiceResponseEntityToDTO(d *setupandconfiguration.SetupAndConfigurationServiceResponse) *dto.SetupAndConfigurationServiceResponse {
	d1 := &dto.SetupAndConfigurationServiceResponse{
		RequestedState:                d.RequestedState,
		EnabledState:                  d.EnabledState,
		ElementName:                   d.ElementName,
		SystemCreationClassName:       d.SystemCreationClassName,
		SystemName:                    d.SystemName,
		CreationClassName:             d.CreationClassName,
		Name:                          d.Name,
		ProvisioningMode:              d.ProvisioningMode,
		ProvisioningState:             d.ProvisioningState,
		ZeroTouchConfigurationEnabled: d.ZeroTouchConfigurationEnabled,
		ProvisioningServerOTP:         d.ProvisioningServerOTP,
		ConfigurationServerFQDN:       d.ConfigurationServerFQDN,
		PasswordModel:                 d.PasswordModel,
		DhcpDNSSuffix:                 d.DhcpDNSSuffix,
		TrustedDNSSuffix:              d.TrustedDNSSuffix,
	}

	return d1
}
