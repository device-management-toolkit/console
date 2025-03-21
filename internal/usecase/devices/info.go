package devices

import (
	"context"
	"strconv"
	"unsafe"

	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/setupandconfiguration"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/card"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/chip"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/mediaaccess"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/physical"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/processor"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/software"

	"github.com/open-amt-cloud-toolkit/console/internal/entity/dto/v1"
	dtov2 "github.com/open-amt-cloud-toolkit/console/internal/entity/dto/v2"
	wsmanAPI "github.com/open-amt-cloud-toolkit/console/internal/usecase/devices/wsman"
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

func (uc *UseCase) GetHardwareInfo(c context.Context, guid string) (v1 dto.HardwareInfoResults, v2 dtov2.HardwareInfoResults, err error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return dto.HardwareInfoResults{}, dtov2.HardwareInfoResults{}, err
	}

	if item == nil || item.GUID == "" {
		return dto.HardwareInfoResults{}, dtov2.HardwareInfoResults{}, ErrNotFound
	}

	device := uc.device.SetupWsmanClient(*item, false, true)

	hwInfo, err := device.GetHardwareInfo()
	if err != nil {
		return dto.HardwareInfoResults{}, dtov2.HardwareInfoResults{}, err
	}

	d1 := *uc.getHardwareInfoEntityToDTO(&hwInfo)

	d2 := *uc.getHardwareInfoEntityToDTOv2(&hwInfo)

	return d1, d2, nil
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

func (uc *UseCase) getHardwareInfoEntityToDTO(d *wsmanAPI.HWResults) *dto.HardwareInfoResults {
	d1 := &dto.HardwareInfoResults{
		CIMComputerSystemPackage: dto.CIMComputerSystemPackage{
			Response:  d.CSPResult.Body.GetResponse.PlatformGUID,
			Responses: d.CSPResult.Body.GetResponse.PlatformGUID,
		},
		CIMSystemPackage: dto.CIMSystemPackage{},
		CIMChassis: dto.CIMChassis{
			Response: dto.CIMChassisResponse{
				Version:            d.ChassisResult.Body.PackageResponse.Version,
				SerialNumber:       d.ChassisResult.Body.PackageResponse.SerialNumber,
				Model:              d.ChassisResult.Body.PackageResponse.Model,
				Manufacturer:       d.ChassisResult.Body.PackageResponse.Manufacturer,
				ElementName:        d.ChassisResult.Body.PackageResponse.ElementName,
				CreationClassName:  d.ChassisResult.Body.PackageResponse.CreationClassName,
				Tag:                d.ChassisResult.Body.PackageResponse.Tag,
				OperationalStatus:  *(*[]int)(unsafe.Pointer(&d.ChassisResult.Body.PackageResponse.OperationalStatus)),
				PackageType:        int(d.ChassisResult.Body.PackageResponse.PackageType),
				ChassisPackageType: int(d.ChassisResult.Body.PackageResponse.ChassisPackageType),
			},
		},
		CIMChip: dto.CIMChips{
			Responses: CimChipArray(d),
		},
		CIMCard: dto.CIMCard{
			Response: dto.CIMCardResponseGet{
				CanBeFRUed:        d.CardResult.Body.PackageResponse.CanBeFRUed,
				CreationClassName: d.CardResult.Body.PackageResponse.CreationClassName,
				ElementName:       d.CardResult.Body.PackageResponse.ElementName,
				Manufacturer:      d.CardResult.Body.PackageResponse.Manufacturer,
				Model:             d.CardResult.Body.PackageResponse.Model,
				OperationalStatus: *(*[]int)(unsafe.Pointer(&d.CardResult.Body.PackageResponse.OperationalStatus)),
				PackageType:       int(d.CardResult.Body.PackageResponse.PackageType),
				SerialNumber:      d.CardResult.Body.PackageResponse.SerialNumber,
				Tag:               d.CardResult.Body.PackageResponse.Tag,
				Version:           d.CardResult.Body.PackageResponse.Version,
			},
		},
		CIMBIOSElement: dto.CIMBIOSElement{
			Response: dto.CIMBIOSElementResponse{
				TargetOperatingSystem: dto.TargetOperatingSystem(d.BiosResult.Body.GetResponse.TargetOperatingSystem),
				SoftwareElementID:     d.BiosResult.Body.GetResponse.SoftwareElementID,
				SoftwareElementState:  dto.SoftwareElementState(d.BiosResult.Body.GetResponse.SoftwareElementState),
				Name:                  d.BiosResult.Body.GetResponse.Name,
				OperationalStatus:     *(*[]int)(unsafe.Pointer(&d.BiosResult.Body.GetResponse.OperationalStatus)),
				ElementName:           d.BiosResult.Body.GetResponse.ElementName,
				Version:               d.BiosResult.Body.GetResponse.Version,
				Manufacturer:          d.BiosResult.Body.GetResponse.Manufacturer,
				PrimaryBIOS:           d.BiosResult.Body.GetResponse.PrimaryBIOS,
				ReleaseDate:           dto.Time(d.BiosResult.Body.GetResponse.ReleaseDate),
			},
		},
		CIMProcessor: dto.CIMProcessor{
			Responses: CimProcessorArray(d),
		},
		CIMPhysicalMemory: dto.CIMPhysicalMemory{
			Responses: CimPhysicalMemoryArray(d),
		},
	}

	return d1
}

//nolint:funlen // struct is large
func (uc *UseCase) getHardwareInfoEntityToDTOv2(d *wsmanAPI.HWResults) *dtov2.HardwareInfoResults {
	d1 := &dtov2.HardwareInfoResults{
		ComputerSystemPackage: dtov2.CIMComputerSystemPackage{
			PlatformGUID: d.CSPResult.Body.GetResponse.PlatformGUID,
		},
		SystemPackage: dtov2.CIMSystemPackage{},
		Chassis: dtov2.CIMChassis{
			Version:            d.ChassisResult.Body.PackageResponse.Version,
			SerialNumber:       d.ChassisResult.Body.PackageResponse.SerialNumber,
			Model:              d.ChassisResult.Body.PackageResponse.Model,
			Manufacturer:       d.ChassisResult.Body.PackageResponse.Manufacturer,
			ElementName:        d.ChassisResult.Body.PackageResponse.ElementName,
			CreationClassName:  d.ChassisResult.Body.PackageResponse.CreationClassName,
			Tag:                d.ChassisResult.Body.PackageResponse.Tag,
			OperationalStatus:  *(*[]int)(unsafe.Pointer(&d.ChassisResult.Body.PackageResponse.OperationalStatus)),
			PackageType:        dtov2.PackageType(d.ChassisResult.Body.PackageResponse.PackageType),
			ChassisPackageType: dtov2.ChassisPackageType(d.ChassisResult.Body.PackageResponse.ChassisPackageType),
		},
		Chip: dtov2.CIMChip{
			Get: dtov2.ChipItems{
				CanBeFRUed:        d.ChipResult.Body.PackageResponse.CanBeFRUed,
				CreationClassName: d.ChipResult.Body.PackageResponse.CreationClassName,
				ElementName:       d.ChipResult.Body.PackageResponse.ElementName,
				Manufacturer:      d.ChipResult.Body.PackageResponse.Manufacturer,
				OperationalStatus: *(*[]int)(unsafe.Pointer(&d.ChipResult.Body.PackageResponse.OperationalStatus)),
				Tag:               d.ChipResult.Body.PackageResponse.Tag,
				Version:           d.ChipResult.Body.PackageResponse.Version,
			},
			Pull: ChipItemsToDTOv2(d.ChipResult.Body.PullResponse.ChipItems),
		},
		Card: dtov2.CIMCard{
			Get: dtov2.CardItems{
				CanBeFRUed:        d.CardResult.Body.PackageResponse.CanBeFRUed,
				CreationClassName: d.CardResult.Body.PackageResponse.CreationClassName,
				ElementName:       d.CardResult.Body.PackageResponse.ElementName,
				Manufacturer:      d.CardResult.Body.PackageResponse.Manufacturer,
				Model:             d.CardResult.Body.PackageResponse.Model,
				OperationalStatus: *(*[]int)(unsafe.Pointer(&d.CardResult.Body.PackageResponse.OperationalStatus)),
				PackageType:       dtov2.PackageType(d.CardResult.Body.PackageResponse.PackageType),
				SerialNumber:      d.CardResult.Body.PackageResponse.SerialNumber,
				Tag:               d.CardResult.Body.PackageResponse.Tag,
				Version:           d.CardResult.Body.PackageResponse.Version,
			},
			Pull: CardItemsToDTOv2(d.CardResult.Body.PullResponse.CardItems),
		},
		BIOSElement: dtov2.CIMBIOSElement{
			TargetOperatingSystem: dtov2.TargetOperatingSystem(d.BiosResult.Body.GetResponse.TargetOperatingSystem),
			SoftwareElementID:     d.BiosResult.Body.GetResponse.SoftwareElementID,
			SoftwareElementState:  dtov2.SoftwareElementState(d.BiosResult.Body.GetResponse.SoftwareElementState),
			Name:                  d.BiosResult.Body.GetResponse.Name,
			OperationalStatus:     *(*[]int)(unsafe.Pointer(&d.BiosResult.Body.GetResponse.OperationalStatus)),
			ElementName:           d.BiosResult.Body.GetResponse.ElementName,
			Version:               d.BiosResult.Body.GetResponse.Version,
			Manufacturer:          d.BiosResult.Body.GetResponse.Manufacturer,
			PrimaryBIOS:           d.BiosResult.Body.GetResponse.PrimaryBIOS,
			ReleaseDate:           dtov2.Time(d.BiosResult.Body.GetResponse.ReleaseDate),
		},
		Processor: dtov2.CIMProcessor{
			Get: dtov2.ProcessorItems{
				DeviceID:                d.ProcessorResult.Body.PackageResponse.DeviceID,
				CreationClassName:       d.ProcessorResult.Body.PackageResponse.CreationClassName,
				SystemName:              d.ProcessorResult.Body.PackageResponse.SystemName,
				SystemCreationClassName: d.ProcessorResult.Body.PackageResponse.SystemCreationClassName,
				ElementName:             d.ProcessorResult.Body.PackageResponse.ElementName,
				OperationalStatus:       *(*[]int)(unsafe.Pointer(&d.ProcessorResult.Body.PackageResponse.OperationalStatus)),
				HealthState:             dtov2.HealthState(d.ProcessorResult.Body.PackageResponse.HealthState),
				EnabledState:            dtov2.EnabledState(d.ProcessorResult.Body.PackageResponse.EnabledState),
				RequestedState:          dtov2.RequestedState(d.ProcessorResult.Body.PackageResponse.RequestedState),
				Role:                    d.ProcessorResult.Body.PackageResponse.Role,
				Family:                  d.ProcessorResult.Body.PackageResponse.Family,
				OtherFamilyDescription:  d.ProcessorResult.Body.PackageResponse.OtherFamilyDescription,
				UpgradeMethod:           dtov2.UpgradeMethod(d.ProcessorResult.Body.PackageResponse.UpgradeMethod),
				MaxClockSpeed:           d.ProcessorResult.Body.PackageResponse.MaxClockSpeed,
				CurrentClockSpeed:       d.ProcessorResult.Body.PackageResponse.CurrentClockSpeed,
				Stepping:                d.ProcessorResult.Body.PackageResponse.Stepping,
				CPUStatus:               dtov2.CPUStatus(d.ProcessorResult.Body.PackageResponse.CPUStatus),
				ExternalBusClockSpeed:   d.ProcessorResult.Body.PackageResponse.ExternalBusClockSpeed,
			},
			Pull: ProcessorItemsToDTOv2(d.ProcessorResult.Body.PullResponse.PackageItems),
		},
		PhysicalMemory: dtov2.CIMPhysicalMemory{
			Get: dtov2.PhysicalMemory{
				PartNumber:                 d.PhysicalMemoryResult.Body.MemoryResponse.PartNumber,
				SerialNumber:               d.PhysicalMemoryResult.Body.MemoryResponse.SerialNumber,
				Manufacturer:               d.PhysicalMemoryResult.Body.MemoryResponse.Manufacturer,
				ElementName:                d.PhysicalMemoryResult.Body.MemoryResponse.ElementName,
				CreationClassName:          d.PhysicalMemoryResult.Body.MemoryResponse.CreationClassName,
				Tag:                        d.PhysicalMemoryResult.Body.MemoryResponse.Tag,
				OperationalStatus:          *(*[]int)(unsafe.Pointer(&d.PhysicalMemoryResult.Body.MemoryResponse.OperationalStatus)),
				FormFactor:                 d.PhysicalMemoryResult.Body.MemoryResponse.FormFactor,
				MemoryType:                 dtov2.MemoryType(d.PhysicalMemoryResult.Body.MemoryResponse.MemoryType),
				Speed:                      d.PhysicalMemoryResult.Body.MemoryResponse.Speed,
				Capacity:                   d.PhysicalMemoryResult.Body.MemoryResponse.Capacity,
				BankLabel:                  d.PhysicalMemoryResult.Body.MemoryResponse.BankLabel,
				ConfiguredMemoryClockSpeed: d.PhysicalMemoryResult.Body.MemoryResponse.ConfiguredMemoryClockSpeed,
				IsSpeedInMhz:               d.PhysicalMemoryResult.Body.MemoryResponse.IsSpeedInMhz,
				MaxMemorySpeed:             d.PhysicalMemoryResult.Body.MemoryResponse.MaxMemorySpeed,
			},
			Pull: PhysicalMemoryToDTOv2(d.PhysicalMemoryResult.Body.PullResponse.MemoryItems),
		},
		MediaAccessDevices: dtov2.CIMMediaAccessDevice{
			Pull: MediaAccessDeviceToDTOv2(d.MediaAccessPullResult.Body.PullResponse.MediaAccessDevices),
		},
		PhysicalPackage: dtov2.CIMPhysicalPackage{
			PullMemoryItems: PpPullResponseMemoryToDTOv2(d.PPPullResult.Body.PullResponse),
			PullCardItems:   PpPullResponseCardToDTOv2(d.PPPullResult.Body.PullResponse),
		},
	}

	return d1
}

func ChipItemsToDTOv2(d []chip.PackageResponse) []dtov2.ChipItems {
	// iterate over the data and convert each entity to dto
	d2 := make([]dtov2.ChipItems, len(d))

	for i := range d {
		d2[i] = dtov2.ChipItems{
			CanBeFRUed:        d[i].CanBeFRUed,
			CreationClassName: d[i].CreationClassName,
			ElementName:       d[i].ElementName,
			Manufacturer:      d[i].Manufacturer,
			OperationalStatus: *(*[]int)(unsafe.Pointer(&d[i].OperationalStatus)),
			Tag:               d[i].Tag,
			Version:           d[i].Version,
		}
	}

	return d2
}

func CardItemsToDTOv2(d []card.PackageResponse) []dtov2.CardItems {
	// iterate over the data and convert each entity to dto
	d2 := make([]dtov2.CardItems, len(d))

	for i := range d {
		d2[i] = dtov2.CardItems{
			CanBeFRUed:        d[i].CanBeFRUed,
			CreationClassName: d[i].CreationClassName,
			ElementName:       d[i].ElementName,
			Manufacturer:      d[i].Manufacturer,
			Model:             d[i].Model,
			OperationalStatus: *(*[]int)(unsafe.Pointer(&d[i].OperationalStatus)),
			PackageType:       dtov2.PackageType(d[i].PackageType),
			SerialNumber:      d[i].SerialNumber,
			Tag:               d[i].Tag,
			Version:           d[i].Version,
		}
	}

	return d2
}

func ProcessorItemsToDTOv2(d []processor.PackageResponse) []dtov2.ProcessorItems {
	// iterate over the data and convert each entity to dto
	d2 := make([]dtov2.ProcessorItems, len(d))

	for i := range d {
		d2[i] = dtov2.ProcessorItems{
			DeviceID:                d[i].DeviceID,
			CreationClassName:       d[i].CreationClassName,
			SystemName:              d[i].SystemName,
			SystemCreationClassName: d[i].SystemCreationClassName,
			ElementName:             d[i].ElementName,
			OperationalStatus:       *(*[]int)(unsafe.Pointer(&d[i].OperationalStatus)),
			HealthState:             dtov2.HealthState(d[i].HealthState),
			EnabledState:            dtov2.EnabledState(d[i].EnabledState),
			RequestedState:          dtov2.RequestedState(d[i].RequestedState),
			Role:                    d[i].Role,
			Family:                  d[i].Family,
			OtherFamilyDescription:  d[i].OtherFamilyDescription,
			UpgradeMethod:           dtov2.UpgradeMethod(d[i].UpgradeMethod),
			MaxClockSpeed:           d[i].MaxClockSpeed,
			CurrentClockSpeed:       d[i].CurrentClockSpeed,
			Stepping:                d[i].Stepping,
			CPUStatus:               dtov2.CPUStatus(d[i].CPUStatus),
			ExternalBusClockSpeed:   d[i].ExternalBusClockSpeed,
		}
	}

	return d2
}

func PhysicalMemoryToDTOv2(d []physical.PhysicalMemory) []dtov2.PhysicalMemory {
	// iterate over the data and convert each entity to dto
	d2 := make([]dtov2.PhysicalMemory, len(d))

	for i := range d {
		d2[i] = dtov2.PhysicalMemory{
			PartNumber:                 d[i].PartNumber,
			SerialNumber:               d[i].SerialNumber,
			Manufacturer:               d[i].Manufacturer,
			ElementName:                d[i].ElementName,
			CreationClassName:          d[i].CreationClassName,
			Tag:                        d[i].Tag,
			OperationalStatus:          *(*[]int)(unsafe.Pointer(&d[i].OperationalStatus)),
			FormFactor:                 d[i].FormFactor,
			MemoryType:                 dtov2.MemoryType(d[i].MemoryType),
			Speed:                      d[i].Speed,
			Capacity:                   d[i].Capacity,
			BankLabel:                  d[i].BankLabel,
			ConfiguredMemoryClockSpeed: d[i].ConfiguredMemoryClockSpeed,
			IsSpeedInMhz:               d[i].IsSpeedInMhz,
			MaxMemorySpeed:             d[i].MaxMemorySpeed,
		}
	}

	return d2
}

func MediaAccessDeviceToDTOv2(d []mediaaccess.MediaAccessDevice) []dtov2.MediaAccessDevice {
	// iterate over the data and convert each entity to dto
	d2 := make([]dtov2.MediaAccessDevice, len(d))

	for i := range d {
		d2[i] = dtov2.MediaAccessDevice{
			CreationClassName:       d[i].CreationClassName,
			DeviceID:                d[i].DeviceID,
			ElementName:             d[i].ElementName,
			EnabledDefault:          dtov2.EnabledDefault(d[i].EnabledDefault),
			EnabledState:            dtov2.EnabledState(d[i].EnabledState),
			MaxMediaSize:            d[i].MaxMediaSize,
			OperationalStatus:       *(*[]int)(unsafe.Pointer(&d[i].OperationalStatus)),
			RequestedState:          dtov2.RequestedState(d[i].RequestedState),
			Security:                dtov2.Security(d[i].Security),
			SystemCreationClassName: d[i].SystemCreationClassName,
			SystemName:              d[i].SystemName,
		}
	}

	return d2
}

func PpPullResponseMemoryToDTOv2(d physical.PullResponse) []dtov2.PhysicalMemory {
	// iterate over the data and convert each entity to dto
	d2 := make([]dtov2.PhysicalMemory, len(d.MemoryItems))

	for i := range d.MemoryItems {
		d2[i] = dtov2.PhysicalMemory{
			PartNumber:                 d.MemoryItems[i].PartNumber,
			SerialNumber:               d.MemoryItems[i].SerialNumber,
			Manufacturer:               d.MemoryItems[i].Manufacturer,
			ElementName:                d.MemoryItems[i].ElementName,
			CreationClassName:          d.MemoryItems[i].CreationClassName,
			Tag:                        d.MemoryItems[i].Tag,
			OperationalStatus:          *(*[]int)(unsafe.Pointer(&d.MemoryItems[i].OperationalStatus)),
			FormFactor:                 d.MemoryItems[i].FormFactor,
			MemoryType:                 dtov2.MemoryType(d.MemoryItems[i].MemoryType),
			Speed:                      d.MemoryItems[i].Speed,
			Capacity:                   d.MemoryItems[i].Capacity,
			BankLabel:                  d.MemoryItems[i].BankLabel,
			ConfiguredMemoryClockSpeed: d.MemoryItems[i].ConfiguredMemoryClockSpeed,
			IsSpeedInMhz:               d.MemoryItems[i].IsSpeedInMhz,
			MaxMemorySpeed:             d.MemoryItems[i].MaxMemorySpeed,
		}
	}

	return d2
}

func PpPullResponseCardToDTOv2(d physical.PullResponse) []dtov2.CardItems {
	// iterate over the data and convert each entity to dto
	d2 := make([]dtov2.CardItems, len(d.PhysicalPackage))

	for i := range d.PhysicalPackage {
		d2[i] = dtov2.CardItems{
			CanBeFRUed:        d.PhysicalPackage[i].CanBeFRUed,
			CreationClassName: d.PhysicalPackage[i].CreationClassName,
			ElementName:       d.PhysicalPackage[i].ElementName,
			Manufacturer:      d.PhysicalPackage[i].Manufacturer,
			Model:             d.PhysicalPackage[i].Model,
			OperationalStatus: *(*[]int)(unsafe.Pointer(&d.PhysicalPackage[i].OperationalStatus)),
			PackageType:       dtov2.PackageType(d.PhysicalPackage[i].PackageType),
			SerialNumber:      d.PhysicalPackage[i].SerialNumber,
			Tag:               d.PhysicalPackage[i].Tag,
		}
	}

	return d2
}

func CimChipArray(d *wsmanAPI.HWResults) []dto.CIMChipGet {
	var y []dto.CIMChipGet

	z := dto.CIMChipGet{
		CanBeFRUed:        d.ChipResult.Body.PackageResponse.CanBeFRUed,
		CreationClassName: d.ChipResult.Body.PackageResponse.CreationClassName,
		ElementName:       d.ChipResult.Body.PackageResponse.ElementName,
		Manufacturer:      d.ChipResult.Body.PackageResponse.Manufacturer,
		OperationalStatus: *(*[]int)(unsafe.Pointer(&d.ChipResult.Body.PackageResponse.OperationalStatus)),
		Tag:               d.ChipResult.Body.PackageResponse.Tag,
		Version:           d.ChipResult.Body.PackageResponse.Version,
	}
	y = append(y, z)

	return y
}

func CimPhysicalMemoryArray(d *wsmanAPI.HWResults) []dto.CIMPhysicalMemoryResponse {
	var y []dto.CIMPhysicalMemoryResponse

	for i := range d.PhysicalMemoryResult.Body.PullResponse.MemoryItems {
		z := dto.CIMPhysicalMemoryResponse{
			PartNumber:                 d.PhysicalMemoryResult.Body.PullResponse.MemoryItems[i].PartNumber,
			SerialNumber:               d.PhysicalMemoryResult.Body.PullResponse.MemoryItems[i].SerialNumber,
			Manufacturer:               d.PhysicalMemoryResult.Body.PullResponse.MemoryItems[i].Manufacturer,
			ElementName:                d.PhysicalMemoryResult.Body.PullResponse.MemoryItems[i].ElementName,
			CreationClassName:          d.PhysicalMemoryResult.Body.PullResponse.MemoryItems[i].CreationClassName,
			Tag:                        d.PhysicalMemoryResult.Body.PullResponse.MemoryItems[i].Tag,
			OperationalStatus:          *(*[]int)(unsafe.Pointer(&d.PhysicalMemoryResult.Body.PullResponse.MemoryItems[0].OperationalStatus)),
			FormFactor:                 d.PhysicalMemoryResult.Body.PullResponse.MemoryItems[i].FormFactor,
			MemoryType:                 int(d.PhysicalMemoryResult.Body.PullResponse.MemoryItems[i].MemoryType),
			Speed:                      d.PhysicalMemoryResult.Body.PullResponse.MemoryItems[i].Speed,
			Capacity:                   d.PhysicalMemoryResult.Body.PullResponse.MemoryItems[i].Capacity,
			BankLabel:                  d.PhysicalMemoryResult.Body.PullResponse.MemoryItems[i].BankLabel,
			ConfiguredMemoryClockSpeed: d.PhysicalMemoryResult.Body.PullResponse.MemoryItems[i].ConfiguredMemoryClockSpeed,
			IsSpeedInMhz:               d.PhysicalMemoryResult.Body.PullResponse.MemoryItems[i].IsSpeedInMhz,
			MaxMemorySpeed:             d.PhysicalMemoryResult.Body.PullResponse.MemoryItems[i].MaxMemorySpeed,
		}
		y = append(y, z)
	}

	return y
}

func CimProcessorArray(d *wsmanAPI.HWResults) []dto.CIMProcessorResponse {
	var y []dto.CIMProcessorResponse

	z := dto.CIMProcessorResponse{
		DeviceID:                d.ProcessorResult.Body.PackageResponse.DeviceID,
		CreationClassName:       d.ProcessorResult.Body.PackageResponse.CreationClassName,
		SystemName:              d.ProcessorResult.Body.PackageResponse.SystemName,
		SystemCreationClassName: d.ProcessorResult.Body.PackageResponse.SystemCreationClassName,
		ElementName:             d.ProcessorResult.Body.PackageResponse.ElementName,
		OperationalStatus:       *(*[]int)(unsafe.Pointer(&d.ProcessorResult.Body.PackageResponse.OperationalStatus)),
		HealthState:             int(d.ProcessorResult.Body.PackageResponse.HealthState),
		EnabledState:            int(d.ProcessorResult.Body.PackageResponse.EnabledState),
		RequestedState:          int(d.ProcessorResult.Body.PackageResponse.RequestedState),
		Role:                    d.ProcessorResult.Body.PackageResponse.Role,
		Family:                  d.ProcessorResult.Body.PackageResponse.Family,
		OtherFamilyDescription:  d.ProcessorResult.Body.PackageResponse.OtherFamilyDescription,
		UpgradeMethod:           int(d.ProcessorResult.Body.PackageResponse.UpgradeMethod),
		MaxClockSpeed:           d.ProcessorResult.Body.PackageResponse.MaxClockSpeed,
		CurrentClockSpeed:       d.ProcessorResult.Body.PackageResponse.CurrentClockSpeed,
		Stepping:                d.ProcessorResult.Body.PackageResponse.Stepping,
		CPUStatus:               int(d.ProcessorResult.Body.PackageResponse.CPUStatus),
		ExternalBusClockSpeed:   d.ProcessorResult.Body.PackageResponse.ExternalBusClockSpeed,
	}
	y = append(y, z)

	return y
}
