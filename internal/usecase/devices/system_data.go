package devices

import (
	"context"
	"time"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/boot"

	"github.com/device-management-toolkit/console/internal/entity"
	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	dtov2 "github.com/device-management-toolkit/console/internal/entity/dto/v2"
	wsmanAPI "github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
)

// systemStaticDataCacheTTL is how long the cached static per-device data
// (hardware inventory and AMT control-mode/version) is reused before it is
// re-fetched from the device. Hardware inventory is effectively immutable and
// the AMT control mode only changes on (de)activation, so a short-to-moderate
// TTL trades a small window of control-mode staleness for skipping ~3.5s of
// WS-Man round-trips on repeat ComputerSystem requests.
const systemStaticDataCacheTTL = 5 * time.Minute

// systemStaticCacheEntry holds the cached static data for a single device.
type systemStaticCacheEntry struct {
	hardwareInfo dto.HardwareInfo
	version      dto.Version
	expiresAt    time.Time
}

// SystemData aggregates the per-device information required to render a Redfish
// ComputerSystem resource. It is populated using a single WS-Man client setup so
// that the per-call request-queue overhead (and redundant authentication/DB
// lookups) are paid only once instead of once per data point.
//
// Only the data actually consumed by the Redfish ComputerSystem response is
// fetched: the more expensive WS-Man round-trips that the response never reads
// (OS power-saving state, AMT software identity and One-Click-Recovery boot
// data) are intentionally skipped to keep the request fast.
type SystemData struct {
	Device       entity.Device
	PowerState   dto.PowerState
	HardwareInfo dto.HardwareInfo
	FeaturesV2   dtov2.Features
	Version      dto.Version
	BootData     boot.BootSettingDataResponse

	// FeaturesErr, VersionErr and BootErr capture best-effort failures so the
	// caller can preserve the previous behaviour of returning a partial system
	// when these optional sections cannot be retrieved.
	FeaturesErr error
	VersionErr  error
	BootErr     error
}

// GetSystemData retrieves the power state, hardware info, features, version and
// boot data for a device using a single WS-Man client setup. Power state and
// hardware info are mandatory (an error is returned if they fail), while
// features, version and boot data are best-effort and their failures are
// reported via the corresponding error fields on the returned SystemData.
//
// The effectively static data (hardware info and AMT control-mode/version) is
// served from a short-lived per-device cache (see systemStaticDataCacheTTL) so
// that repeat requests skip those round-trips; power state, features and boot
// data are always fetched fresh.
func (uc *UseCase) GetSystemData(c context.Context, guid string) (SystemData, error) {
	item, err := uc.repo.GetByID(c, guid, "")
	if err != nil {
		return SystemData{}, err
	}

	if item == nil || item.GUID == "" {
		return SystemData{}, ErrNotFound
	}

	device, err := uc.device.SetupWsmanClient(c, *item, false, true)
	if err != nil {
		return SystemData{}, err
	}

	result := SystemData{Device: *item}

	// Power state is mandatory and dynamic, so it is always fetched fresh. Only
	// the CIM power state is needed by Redfish; the OS power-saving state (an
	// extra round-trip) is intentionally skipped.
	powerState, err := uc.powerStateForRedfish(device)
	if err != nil {
		return SystemData{}, err
	}

	result.PowerState = powerState

	// Hardware info and version (AMT control mode) are effectively static, so
	// they are served from a short-lived per-device cache to avoid their ~3.5s
	// of WS-Man round-trips on repeat requests. On a cache miss they are fetched
	// and, when both succeed, stored for subsequent requests.
	if cached, ok := uc.getCachedStaticData(guid); ok {
		result.HardwareInfo = cached.hardwareInfo
		result.Version = cached.version
	} else {
		// Hardware info is mandatory.
		hwResults, hwErr := device.GetHardwareInfo()
		if hwErr != nil {
			return SystemData{}, hwErr
		}

		hwInfo := uc.hardwareInfoToDTO(hwResults)
		result.HardwareInfo = hwInfo

		// Version is best-effort and only used to derive the AMT control mode,
		// which comes from the setup-and-configuration service; the AMT software
		// identity (an extra round-trip) is intentionally skipped.
		result.Version, result.VersionErr = uc.controlModeVersionFromDevice(device)

		if result.VersionErr == nil {
			uc.setCachedStaticData(guid, hwInfo, result.Version)
		}
	}

	// Features are best-effort and dynamic (user-consent/opt-in state can change
	// mid-session), so they are always fetched fresh. Only redirection,
	// user-consent and KVM state are needed for the Graphical/Serial console;
	// the One-Click-Recovery data (4 extra round-trips) is intentionally skipped.
	result.FeaturesV2, result.FeaturesErr = redfishFeaturesFromDevice(device)

	// Boot data is best-effort and dynamic.
	result.BootData, result.BootErr = device.GetBootData()

	return result, nil
}

// getCachedStaticData returns the cached static data for a device if present and
// not expired. Reading a nil cache map is safe and reports a miss.
func (uc *UseCase) getCachedStaticData(guid string) (systemStaticCacheEntry, bool) {
	uc.systemStaticMutex.RLock()
	entry, ok := uc.systemStaticCache[guid]
	uc.systemStaticMutex.RUnlock()

	if !ok || time.Now().After(entry.expiresAt) {
		return systemStaticCacheEntry{}, false
	}

	return entry, true
}

// setCachedStaticData stores the static data for a device with a fresh TTL. The
// cache map is lazily initialised so the method is safe even when the UseCase
// was constructed without New (e.g. in tests).
func (uc *UseCase) setCachedStaticData(guid string, hardwareInfo dto.HardwareInfo, version dto.Version) {
	uc.systemStaticMutex.Lock()
	defer uc.systemStaticMutex.Unlock()

	if uc.systemStaticCache == nil {
		uc.systemStaticCache = make(map[string]systemStaticCacheEntry)
	}

	uc.systemStaticCache[guid] = systemStaticCacheEntry{
		hardwareInfo: hardwareInfo,
		version:      version,
		expiresAt:    time.Now().Add(systemStaticDataCacheTTL),
	}
}

// powerStateForRedfish fetches only the CIM power state (skipping the OS
// power-saving state round-trip) since the Redfish response uses only the
// former.
func (uc *UseCase) powerStateForRedfish(device wsmanAPI.Management) (dto.PowerState, error) {
	state, err := device.GetPowerState()
	if err != nil {
		return dto.PowerState{}, err
	}

	if len(state) == 0 {
		return dto.PowerState{}, ErrDeviceUseCase.Wrap("GetPowerState", "device.GetPowerState returned empty state", nil)
	}

	return dto.PowerState{PowerState: int(state[0].PowerState)}, nil
}

// redfishFeaturesFromDevice retrieves only the feature settings consumed by the
// Redfish Graphical/Serial console (redirection, user consent and KVM),
// skipping the One-Click-Recovery boot queries that the response never reads.
func redfishFeaturesFromDevice(device wsmanAPI.Management) (dtov2.Features, error) {
	var features dtov2.Features

	if err := getRedirectionService(&features, device); err != nil {
		return features, err
	}

	if err := getUserConsent(&features, device); err != nil {
		return features, err
	}

	if err := getKVM(&features, device); err != nil {
		return features, err
	}

	return features, nil
}

// controlModeVersionFromDevice fetches only the setup-and-configuration service
// response needed to derive the AMT control mode, skipping the AMT software
// identity round-trip.
func (uc *UseCase) controlModeVersionFromDevice(device wsmanAPI.Management) (dto.Version, error) {
	data, err := device.GetSetupAndConfiguration()
	if err != nil {
		return dto.Version{}, err
	}

	version := dto.Version{}
	if len(data) > 0 {
		resp := uc.setupAndConfigurationServiceResponseEntityToDTO(&data[0])
		version.AMTSetupAndConfigurationService = dto.SetupAndConfigurationServiceResponses{Response: *resp}
	}

	return version, nil
}
