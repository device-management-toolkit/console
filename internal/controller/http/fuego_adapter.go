package http

import (
	"encoding/json"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/go-fuego/fuego"
)

// FuegoAdapter integrates Fuego with existing Gin router for automatic OpenAPI 3.0+ generation
type FuegoAdapter struct {
	server   *fuego.Server
	usecases usecase.Usecases
	logger   logger.Interface
}

// NewFuegoAdapter creates a new Fuego adapter instance
func NewFuegoAdapter(usecases usecase.Usecases, logger logger.Interface) *FuegoAdapter {
	server := fuego.NewServer(
		fuego.WithoutStartupMessages(), // Disable startup messages since we're integrating
	)

	return &FuegoAdapter{
		server:   server,
		usecases: usecases,
		logger:   logger,
	}
}

// WirelessConfigResponse represents the paginated response for wireless configs
type WirelessConfigResponse struct {
	Count int                  `json:"totalCount"`
	Data  []dto.WirelessConfig `json:"data"`
}

// RegisterRoutes registers API routes with Fuego for automatic OpenAPI generation
func (f *FuegoAdapter) RegisterRoutes() {
	// Wireless Configurations
	f.registerWirelessConfigRoutes()

	// CIRA Configurations
	f.registerCIRAConfigRoutes()

	// IEEE 802.1x Configurations
	f.registerIEEE8021xConfigRoutes()

	// Profiles
	f.registerProfileRoutes()

	// AMT Device Management
	f.registerAMTDeviceRoutes()
}

// registerWirelessConfigRoutes registers wireless configuration endpoints
func (f *FuegoAdapter) registerWirelessConfigRoutes() {
	fuego.Get(f.server, "/api/v1/admin/wirelessconfigs", f.getWirelessConfigs,
		fuego.OptionTags("Wireless Configurations"),
		fuego.OptionSummary("List Wireless Configurations"),
		fuego.OptionDescription("Retrieve all wireless configurations with optional pagination"),
		fuego.OptionQueryInt("$top", "Number of records to return"),
		fuego.OptionQueryInt("$skip", "Number of records to skip"),
		fuego.OptionQueryBool("$count", "Include total count"),
	)

	fuego.Post(f.server, "/api/v1/admin/wirelessconfigs", f.createWirelessConfig,
		fuego.OptionTags("Wireless Configurations"),
		fuego.OptionSummary("Create Wireless Configuration"),
		fuego.OptionDescription("Create a new wireless configuration"),
	)

	fuego.Get(f.server, "/api/v1/admin/wirelessconfigs/{name}", f.getWirelessConfigByName,
		fuego.OptionTags("Wireless Configurations"),
		fuego.OptionSummary("Get Wireless Configuration by Name"),
		fuego.OptionDescription("Retrieve a specific wireless configuration by profile name"),
		fuego.OptionPath("name", "Profile name"),
	)

	fuego.Patch(f.server, "/api/v1/admin/wirelessconfigs", f.updateWirelessConfig,
		fuego.OptionTags("Wireless Configurations"),
		fuego.OptionSummary("Update Wireless Configuration"),
		fuego.OptionDescription("Update an existing wireless configuration"),
	)

	fuego.Delete(f.server, "/api/v1/admin/wirelessconfigs/{name}", f.deleteWirelessConfig,
		fuego.OptionTags("Wireless Configurations"),
		fuego.OptionSummary("Delete Wireless Configuration"),
		fuego.OptionDescription("Delete a wireless configuration by profile name"),
		fuego.OptionPath("name", "Profile name"),
	)
}

// getWirelessConfigs handles GET /api/v1/admin/wirelessconfigs
func (f *FuegoAdapter) getWirelessConfigs(c fuego.ContextNoBody) (WirelessConfigResponse, error) {
	// This is a mock implementation - you would integrate with your actual usecase
	configs := []dto.WirelessConfig{
		{
			ProfileName:          "example-wifi",
			SSID:                 "ExampleSSID",
			AuthenticationMethod: 6, // WPA2-Personal
			EncryptionMethod:     4,
			TenantID:             "default",
			Version:              "1.0",
		},
	}

	return WirelessConfigResponse{
		Count: len(configs),
		Data:  configs,
	}, nil
}

// createWirelessConfig handles POST /api/v1/admin/wirelessconfigs
func (f *FuegoAdapter) createWirelessConfig(c fuego.ContextWithBody[dto.WirelessConfig]) (dto.WirelessConfig, error) {
	body, err := c.Body()
	if err != nil {
		return dto.WirelessConfig{}, err
	}

	// Mock creation - integrate with your actual usecase
	return body, nil
}

// getWirelessConfigByName handles GET /api/v1/admin/wirelessconfigs/{name}
func (f *FuegoAdapter) getWirelessConfigByName(c fuego.ContextNoBody) (dto.WirelessConfig, error) {
	// Mock implementation - integrate with your actual usecase
	return dto.WirelessConfig{
		ProfileName:          "example-wifi",
		SSID:                 "ExampleSSID",
		AuthenticationMethod: 6,
		EncryptionMethod:     4,
		TenantID:             "default",
		Version:              "1.0",
	}, nil
}

// updateWirelessConfig handles PATCH /api/v1/admin/wirelessconfigs
func (f *FuegoAdapter) updateWirelessConfig(c fuego.ContextWithBody[dto.WirelessConfig]) (dto.WirelessConfig, error) {
	body, err := c.Body()
	if err != nil {
		return dto.WirelessConfig{}, err
	}

	// Mock update - integrate with your actual usecase
	return body, nil
}

// deleteWirelessConfig handles DELETE /api/v1/admin/wirelessconfigs/{name}
func (f *FuegoAdapter) deleteWirelessConfig(c fuego.ContextNoBody) (DeleteResponse, error) {
	// Mock deletion - integrate with your actual usecase
	return DeleteResponse{Status: "deleted"}, nil
}

// CIRA Configuration methods
func (f *FuegoAdapter) registerCIRAConfigRoutes() {
	fuego.Get(f.server, "/api/v1/admin/ciraconfigs", f.getCIRAConfigs,
		fuego.OptionTags("CIRA Configurations"),
		fuego.OptionSummary("List CIRA Configurations"),
		fuego.OptionDescription("Retrieve all CIRA configurations with optional pagination"),
		fuego.OptionQueryInt("$top", "Number of records to return"),
		fuego.OptionQueryInt("$skip", "Number of records to skip"),
		fuego.OptionQueryBool("$count", "Include total count"),
	)

	fuego.Post(f.server, "/api/v1/admin/ciraconfigs", f.createCIRAConfig,
		fuego.OptionTags("CIRA Configurations"),
		fuego.OptionSummary("Create CIRA Configuration"),
		fuego.OptionDescription("Create a new CIRA configuration"),
	)

	fuego.Get(f.server, "/api/v1/admin/ciraconfigs/{name}", f.getCIRAConfigByName,
		fuego.OptionTags("CIRA Configurations"),
		fuego.OptionSummary("Get CIRA Configuration by Name"),
		fuego.OptionDescription("Retrieve a specific CIRA configuration by name"),
		fuego.OptionPath("name", "Configuration name"),
	)

	fuego.Patch(f.server, "/api/v1/admin/ciraconfigs", f.updateCIRAConfig,
		fuego.OptionTags("CIRA Configurations"),
		fuego.OptionSummary("Update CIRA Configuration"),
		fuego.OptionDescription("Update an existing CIRA configuration"),
	)

	fuego.Delete(f.server, "/api/v1/admin/ciraconfigs/{name}", f.deleteCIRAConfig,
		fuego.OptionTags("CIRA Configurations"),
		fuego.OptionSummary("Delete CIRA Configuration"),
		fuego.OptionDescription("Delete a CIRA configuration by name"),
		fuego.OptionPath("name", "Configuration name"),
	)
}

type CIRAConfigResponse struct {
	Count int              `json:"totalCount"`
	Data  []dto.CIRAConfig `json:"data"`
}

func (f *FuegoAdapter) getCIRAConfigs(c fuego.ContextNoBody) (CIRAConfigResponse, error) {
	configs := []dto.CIRAConfig{
		{
			ConfigName:          "example-cira",
			MPSAddress:          "https://mps.example.com",
			MPSPort:             4433,
			Username:            "admin",
			CommonName:          "mps.example.com",
			ServerAddressFormat: 201,
			AuthMethod:          2,
			MPSRootCertificate:  "-----BEGIN CERTIFICATE-----\nexample\n-----END CERTIFICATE-----",
			TenantID:            "default",
		},
	}

	return CIRAConfigResponse{
		Count: len(configs),
		Data:  configs,
	}, nil
}

func (f *FuegoAdapter) createCIRAConfig(c fuego.ContextWithBody[dto.CIRAConfig]) (dto.CIRAConfig, error) {
	body, err := c.Body()
	if err != nil {
		return dto.CIRAConfig{}, err
	}
	return body, nil
}

func (f *FuegoAdapter) getCIRAConfigByName(c fuego.ContextNoBody) (dto.CIRAConfig, error) {
	return dto.CIRAConfig{
		ConfigName:          "example-cira",
		MPSAddress:          "https://mps.example.com",
		MPSPort:             4433,
		Username:            "admin",
		CommonName:          "mps.example.com",
		ServerAddressFormat: 201,
		AuthMethod:          2,
		MPSRootCertificate:  "-----BEGIN CERTIFICATE-----\nexample\n-----END CERTIFICATE-----",
		TenantID:            "default",
	}, nil
}

func (f *FuegoAdapter) updateCIRAConfig(c fuego.ContextWithBody[dto.CIRAConfig]) (dto.CIRAConfig, error) {
	body, err := c.Body()
	if err != nil {
		return dto.CIRAConfig{}, err
	}
	return body, nil
}

func (f *FuegoAdapter) deleteCIRAConfig(c fuego.ContextNoBody) (DeleteResponse, error) {
	return DeleteResponse{Status: "deleted"}, nil
}

// IEEE 802.1x Configuration methods
func (f *FuegoAdapter) registerIEEE8021xConfigRoutes() {
	fuego.Get(f.server, "/api/v1/admin/ieee8021xconfigs", f.getIEEE8021xConfigs,
		fuego.OptionTags("IEEE 802.1x Configurations"),
		fuego.OptionSummary("List IEEE 802.1x Configurations"),
		fuego.OptionDescription("Retrieve all IEEE 802.1x configurations with optional pagination"),
		fuego.OptionQueryInt("$top", "Number of records to return"),
		fuego.OptionQueryInt("$skip", "Number of records to skip"),
		fuego.OptionQueryBool("$count", "Include total count"),
	)

	fuego.Post(f.server, "/api/v1/admin/ieee8021xconfigs", f.createIEEE8021xConfig,
		fuego.OptionTags("IEEE 802.1x Configurations"),
		fuego.OptionSummary("Create IEEE 802.1x Configuration"),
		fuego.OptionDescription("Create a new IEEE 802.1x configuration"),
	)

	fuego.Get(f.server, "/api/v1/admin/ieee8021xconfigs/{name}", f.getIEEE8021xConfigByName,
		fuego.OptionTags("IEEE 802.1x Configurations"),
		fuego.OptionSummary("Get IEEE 802.1x Configuration by Name"),
		fuego.OptionDescription("Retrieve a specific IEEE 802.1x configuration by name"),
		fuego.OptionPath("name", "Configuration name"),
	)

	fuego.Patch(f.server, "/api/v1/admin/ieee8021xconfigs", f.updateIEEE8021xConfig,
		fuego.OptionTags("IEEE 802.1x Configurations"),
		fuego.OptionSummary("Update IEEE 802.1x Configuration"),
		fuego.OptionDescription("Update an existing IEEE 802.1x configuration"),
	)

	fuego.Delete(f.server, "/api/v1/admin/ieee8021xconfigs/{name}", f.deleteIEEE8021xConfig,
		fuego.OptionTags("IEEE 802.1x Configurations"),
		fuego.OptionSummary("Delete IEEE 802.1x Configuration"),
		fuego.OptionDescription("Delete an IEEE 802.1x configuration by name"),
		fuego.OptionPath("name", "Configuration name"),
	)
}

type IEEE8021xConfigResponse struct {
	Count int                   `json:"totalCount"`
	Data  []dto.IEEE8021xConfig `json:"data"`
}

func (f *FuegoAdapter) getIEEE8021xConfigs(c fuego.ContextNoBody) (IEEE8021xConfigResponse, error) {
	timeout := 60
	configs := []dto.IEEE8021xConfig{
		{
			ProfileName:            "example-8021x",
			AuthenticationProtocol: 2,
			PXETimeout:             &timeout,
			WiredInterface:         true,
			TenantID:               "default",
			Version:                "1.0.0",
		},
	}

	return IEEE8021xConfigResponse{
		Count: len(configs),
		Data:  configs,
	}, nil
}

func (f *FuegoAdapter) createIEEE8021xConfig(c fuego.ContextWithBody[dto.IEEE8021xConfig]) (dto.IEEE8021xConfig, error) {
	body, err := c.Body()
	if err != nil {
		return dto.IEEE8021xConfig{}, err
	}
	return body, nil
}

func (f *FuegoAdapter) getIEEE8021xConfigByName(c fuego.ContextNoBody) (dto.IEEE8021xConfig, error) {
	timeout := 60
	return dto.IEEE8021xConfig{
		ProfileName:            "example-8021x",
		AuthenticationProtocol: 2,
		PXETimeout:             &timeout,
		WiredInterface:         true,
		TenantID:               "default",
		Version:                "1.0.0",
	}, nil
}

func (f *FuegoAdapter) updateIEEE8021xConfig(c fuego.ContextWithBody[dto.IEEE8021xConfig]) (dto.IEEE8021xConfig, error) {
	body, err := c.Body()
	if err != nil {
		return dto.IEEE8021xConfig{}, err
	}
	return body, nil
}

func (f *FuegoAdapter) deleteIEEE8021xConfig(c fuego.ContextNoBody) (DeleteResponse, error) {
	return DeleteResponse{Status: "deleted"}, nil
}

// Profile Configuration methods
func (f *FuegoAdapter) registerProfileRoutes() {
	fuego.Get(f.server, "/api/v1/admin/profiles", f.getProfiles,
		fuego.OptionTags("Profiles"),
		fuego.OptionSummary("List Profiles"),
		fuego.OptionDescription("Retrieve all profiles with optional pagination"),
		fuego.OptionQueryInt("$top", "Number of records to return"),
		fuego.OptionQueryInt("$skip", "Number of records to skip"),
		fuego.OptionQueryBool("$count", "Include total count"),
	)

	fuego.Post(f.server, "/api/v1/admin/profiles", f.createProfile,
		fuego.OptionTags("Profiles"),
		fuego.OptionSummary("Create Profile"),
		fuego.OptionDescription("Create a new profile"),
	)

	fuego.Get(f.server, "/api/v1/admin/profiles/{name}", f.getProfileByName,
		fuego.OptionTags("Profiles"),
		fuego.OptionSummary("Get Profile by Name"),
		fuego.OptionDescription("Retrieve a specific profile by name"),
		fuego.OptionPath("name", "Profile name"),
	)

	fuego.Patch(f.server, "/api/v1/admin/profiles", f.updateProfile,
		fuego.OptionTags("Profiles"),
		fuego.OptionSummary("Update Profile"),
		fuego.OptionDescription("Update an existing profile"),
	)

	fuego.Delete(f.server, "/api/v1/admin/profiles/{name}", f.deleteProfile,
		fuego.OptionTags("Profiles"),
		fuego.OptionSummary("Delete Profile"),
		fuego.OptionDescription("Delete a profile by name"),
		fuego.OptionPath("name", "Profile name"),
	)

	fuego.Get(f.server, "/api/v1/admin/profiles/export/{name}", f.exportProfile,
		fuego.OptionTags("Profiles"),
		fuego.OptionSummary("Export Profile"),
		fuego.OptionDescription("Export a profile configuration"),
		fuego.OptionPath("name", "Profile name"),
		fuego.OptionQuery("domainName", "Domain name for export"),
	)
}

type ProfileResponse struct {
	Count int           `json:"totalCount"`
	Data  []dto.Profile `json:"data"`
}

func (f *FuegoAdapter) getProfiles(c fuego.ContextNoBody) (ProfileResponse, error) {
	profiles := []dto.Profile{
		{
			ProfileName:                "example-profile",
			AMTPassword:                "Password123!",
			GenerateRandomPassword:     false,
			Activation:                 "ccmactivate",
			MEBXPassword:               "MEBXPass123!",
			GenerateRandomMEBxPassword: false,
			DHCPEnabled:                true,
			IPSyncEnabled:              true,
			LocalWiFiSyncEnabled:       true,
			TenantID:                   "default",
			TLSMode:                    1,
			TLSSigningAuthority:        "SelfSigned",
			UserConsent:                "All",
			IDEREnabled:                true,
		},
	}

	return ProfileResponse{
		Count: len(profiles),
		Data:  profiles,
	}, nil
}

func (f *FuegoAdapter) createProfile(c fuego.ContextWithBody[dto.Profile]) (dto.Profile, error) {
	body, err := c.Body()
	if err != nil {
		return dto.Profile{}, err
	}
	return body, nil
}

func (f *FuegoAdapter) getProfileByName(c fuego.ContextNoBody) (dto.Profile, error) {
	return dto.Profile{
		ProfileName:                "example-profile",
		AMTPassword:                "Password123!",
		GenerateRandomPassword:     false,
		Activation:                 "ccmactivate",
		MEBXPassword:               "MEBXPass123!",
		GenerateRandomMEBxPassword: false,
		DHCPEnabled:                true,
		IPSyncEnabled:              true,
		LocalWiFiSyncEnabled:       true,
		TenantID:                   "default",
		TLSMode:                    1,
		TLSSigningAuthority:        "SelfSigned",
		UserConsent:                "All",
		IDEREnabled:                true,
	}, nil
}

func (f *FuegoAdapter) updateProfile(c fuego.ContextWithBody[dto.Profile]) (dto.Profile, error) {
	body, err := c.Body()
	if err != nil {
		return dto.Profile{}, err
	}
	return body, nil
}

func (f *FuegoAdapter) deleteProfile(c fuego.ContextNoBody) (DeleteResponse, error) {
	return DeleteResponse{Status: "deleted"}, nil
}

type ExportResponse struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
	Key      string `json:"key"`
}

func (f *FuegoAdapter) exportProfile(c fuego.ContextNoBody) (ExportResponse, error) {
	return ExportResponse{
		Filename: "example-profile.yaml",
		Content:  "# Example profile YAML content",
		Key:      "example-key",
	}, nil
}

// AMT Device Management methods
func (f *FuegoAdapter) registerAMTDeviceRoutes() {
	// Device version information
	fuego.Get(f.server, "/api/v1/amt/version/{guid}", f.getAMTVersion,
		fuego.OptionTags("AMT Device Management"),
		fuego.OptionSummary("Get AMT Version"),
		fuego.OptionDescription("Get AMT firmware version information for a device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	// Device features
	fuego.Get(f.server, "/api/v1/amt/features/{guid}", f.getAMTFeatures,
		fuego.OptionTags("AMT Device Management"),
		fuego.OptionSummary("Get AMT Features"),
		fuego.OptionDescription("Get AMT features enabled on device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	fuego.Post(f.server, "/api/v1/amt/features/{guid}", f.setAMTFeatures,
		fuego.OptionTags("AMT Device Management"),
		fuego.OptionSummary("Set AMT Features"),
		fuego.OptionDescription("Configure AMT features on device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	// Hardware information
	fuego.Get(f.server, "/api/v1/amt/hardwareInfo/{guid}", f.getHardwareInfo,
		fuego.OptionTags("AMT Device Management"),
		fuego.OptionSummary("Get Hardware Information"),
		fuego.OptionDescription("Get hardware information from AMT device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	// Power management
	fuego.Get(f.server, "/api/v1/amt/power/state/{guid}", f.getPowerState,
		fuego.OptionTags("AMT Power Management"),
		fuego.OptionSummary("Get Power State"),
		fuego.OptionDescription("Get current power state of AMT device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	fuego.Post(f.server, "/api/v1/amt/power/action/{guid}", f.powerAction,
		fuego.OptionTags("AMT Power Management"),
		fuego.OptionSummary("Power Action"),
		fuego.OptionDescription("Perform power action on AMT device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	fuego.Get(f.server, "/api/v1/amt/power/capabilities/{guid}", f.getPowerCapabilities,
		fuego.OptionTags("AMT Power Management"),
		fuego.OptionSummary("Get Power Capabilities"),
		fuego.OptionDescription("Get power capabilities of AMT device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	fuego.Get(f.server, "/api/v1/amt/power/bootSources/{guid}", f.getBootSources,
		fuego.OptionTags("AMT Power Management"),
		fuego.OptionSummary("Get Boot Sources"),
		fuego.OptionDescription("Get available boot sources for AMT device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	// Network settings
	fuego.Get(f.server, "/api/v1/amt/networkSettings/{guid}", f.getNetworkSettings,
		fuego.OptionTags("AMT Network Management"),
		fuego.OptionSummary("Get Network Settings"),
		fuego.OptionDescription("Get network settings from AMT device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	// User consent
	fuego.Get(f.server, "/api/v1/amt/userConsentCode/{guid}", f.getUserConsentCode,
		fuego.OptionTags("AMT User Consent"),
		fuego.OptionSummary("Get User Consent Code"),
		fuego.OptionDescription("Get user consent code for AMT device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	fuego.Post(f.server, "/api/v1/amt/userConsentCode/{guid}", f.sendConsentCode,
		fuego.OptionTags("AMT User Consent"),
		fuego.OptionSummary("Send Consent Code"),
		fuego.OptionDescription("Send user consent code to AMT device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	// Audit and Event Logs
	fuego.Get(f.server, "/api/v1/amt/log/audit/{guid}", f.getAuditLog,
		fuego.OptionTags("AMT Logging"),
		fuego.OptionSummary("Get Audit Log"),
		fuego.OptionDescription("Get audit log from AMT device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	fuego.Get(f.server, "/api/v1/amt/log/event/{guid}", f.getEventLog,
		fuego.OptionTags("AMT Logging"),
		fuego.OptionSummary("Get Event Log"),
		fuego.OptionDescription("Get event log from AMT device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	// KVM display settings
	fuego.Get(f.server, "/api/v1/amt/kvm/displays/{guid}", f.getKVMDisplays,
		fuego.OptionTags("AMT KVM"),
		fuego.OptionSummary("Get KVM Display Settings"),
		fuego.OptionDescription("Get KVM display settings for AMT device"),
		fuego.OptionPath("guid", "Device GUID"),
	)

	fuego.Put(f.server, "/api/v1/amt/kvm/displays/{guid}", f.setKVMDisplays,
		fuego.OptionTags("AMT KVM"),
		fuego.OptionSummary("Set KVM Display Settings"),
		fuego.OptionDescription("Configure KVM display settings for AMT device"),
		fuego.OptionPath("guid", "Device GUID"),
	)
}

// AMT Device Management implementation methods
func (f *FuegoAdapter) getAMTVersion(c fuego.ContextNoBody) (map[string]interface{}, error) {
	return map[string]interface{}{
		"AMT":       "16.1.25.1830",
		"Build":     "3425",
		"Sku":       "16392",
		"VendorID":  "8086",
		"BuildDate": "Mar 30 2021",
	}, nil
}

func (f *FuegoAdapter) getAMTFeatures(c fuego.ContextNoBody) (dto.Features, error) {
	return dto.Features{
		UserConsent:           "None",
		EnableSOL:             true,
		EnableIDER:            true,
		EnableKVM:             true,
		Redirection:           true,
		OptInState:            0,
		KVMAvailable:          true,
		OCR:                   true,
		HTTPSBootSupported:    true,
		WinREBootSupported:    false,
		LocalPBABootSupported: false,
		RemoteErase:           true,
	}, nil
}

func (f *FuegoAdapter) setAMTFeatures(c fuego.ContextWithBody[dto.Features]) (dto.Features, error) {
	body, err := c.Body()
	if err != nil {
		return dto.Features{}, err
	}
	return body, nil
}

func (f *FuegoAdapter) getHardwareInfo(c fuego.ContextNoBody) (map[string]interface{}, error) {
	return map[string]interface{}{
		"systemUUID":            "4c4c4544-0050-3710-8048-b6c04f503732",
		"totalPhysicalMemory":   8589934592,
		"manufacturer":          "Dell Inc.",
		"model":                 "OptiPlex 7090",
		"baseBoardManufacturer": "Dell Inc.",
		"baseBoardProduct":      "0K240Y",
		"processorInfo":         "Intel(R) Core(TM) i7-10700 CPU @ 2.90GHz",
	}, nil
}

func (f *FuegoAdapter) getPowerState(c fuego.ContextNoBody) (dto.PowerState, error) {
	return dto.PowerState{
		PowerState:         2, // On
		OSPowerSavingState: 0,
	}, nil
}

func (f *FuegoAdapter) powerAction(c fuego.ContextWithBody[dto.PowerAction]) (map[string]string, error) {
	body, err := c.Body()
	if err != nil {
		return nil, err
	}

	actionNames := map[int]string{
		2:   "Power Up",
		5:   "Power Cycle",
		8:   "Power Down",
		10:  "Reset",
		11:  "Power Off - Soft",
		12:  "Power Off - Hard",
		100: "Master Bus Reset",
		101: "Diagnostic Interrupt (NMI)",
	}

	actionName := actionNames[body.Action]
	if actionName == "" {
		actionName = "Unknown Action"
	}

	return map[string]string{
		"status": "success",
		"action": actionName,
	}, nil
}

func (f *FuegoAdapter) getPowerCapabilities(c fuego.ContextNoBody) (dto.PowerCapabilities, error) {
	return dto.PowerCapabilities{
		PowerUp:    1,
		PowerCycle: 1,
		PowerDown:  1,
		Reset:      1,
		SoftOff:    1,
		SoftReset:  1,
		Sleep:      1,
		Hibernate:  1,
	}, nil
}

func (f *FuegoAdapter) getBootSources(c fuego.ContextNoBody) ([]dto.BootSources, error) {
	return []dto.BootSources{
		{
			BIOSBootString:       "UEFI: PXE IPv4 Intel(R) Ethernet Controller I225-LM",
			BootString:           "Intel® AMT: Force PXE Boot",
			ElementName:          "Intel® AMT: Boot Source",
			FailThroughSupported: 2,
			InstanceID:           "Intel® AMT: Force PXE Boot",
			StructuredBootString: "CIM:Network:1",
		},
		{
			BIOSBootString:       "Hard Disk",
			BootString:           "Intel® AMT: Force Hard-drive Boot",
			ElementName:          "Intel® AMT: Boot Source",
			FailThroughSupported: 2,
			InstanceID:           "Intel® AMT: Force Hard-drive Boot",
			StructuredBootString: "CIM:Hard-Disk:1",
		},
	}, nil
}

func (f *FuegoAdapter) getNetworkSettings(c fuego.ContextNoBody) (map[string]interface{}, error) {
	return map[string]interface{}{
		"sharedFQDN":  true,
		"macAddress":  "a4:bb:6d:12:34:56",
		"isWireless":  false,
		"dhcpEnabled": true,
		"dhcpMode":    1,
	}, nil
}

func (f *FuegoAdapter) getUserConsentCode(c fuego.ContextNoBody) (map[string]interface{}, error) {
	return map[string]interface{}{
		"name":  "User Consent",
		"value": 4294967295,
	}, nil
}

type ConsentCodeRequest struct {
	ConsentCode int `json:"consentCode" binding:"required"`
}

// DeleteResponse represents a successful deletion response
type DeleteResponse struct {
	Status string `json:"status" example:"deleted"`
}

func (f *FuegoAdapter) sendConsentCode(c fuego.ContextWithBody[ConsentCodeRequest]) (map[string]interface{}, error) {
	body, err := c.Body()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status":      "success",
		"message":     "Consent code accepted",
		"consentCode": body.ConsentCode,
	}, nil
}

func (f *FuegoAdapter) getAuditLog(c fuego.ContextNoBody) (map[string]interface{}, error) {
	return map[string]interface{}{
		"totalCount": 1,
		"records": []map[string]interface{}{
			{
				"auditAppID":     16,
				"eventID":        1600,
				"initiatorType":  1,
				"auditApp":       "Security Admin",
				"event":          "Provisioning Started",
				"initiator":      "||HWAsset|CPU|GenuineIntel",
				"mcLocationType": 0,
				"networkAccess":  1,
			},
		},
	}, nil
}

func (f *FuegoAdapter) getEventLog(c fuego.ContextNoBody) (map[string]interface{}, error) {
	return map[string]interface{}{
		"records": []map[string]interface{}{
			{
				"deviceAddress":   32,
				"eventSensorType": 15,
				"eventType":       111,
				"eventOffset":     0,
				"eventSourceType": 32,
				"eventSeverity":   "16",
				"sensorNumber":    35,
				"entity":          "7",
				"entityInstance":  0,
				"eventData":       []int{170, 48, 0, 170, 48, 0},
				"timeStamp":       1234567890,
			},
		},
	}, nil
}

func (f *FuegoAdapter) getKVMDisplays(c fuego.ContextNoBody) (map[string]interface{}, error) {
	return map[string]interface{}{
		"isActive":    true,
		"numDisplays": 1,
		"displays": []map[string]interface{}{
			{
				"isActive": true,
				"height":   1080,
				"width":    1920,
				"upper":    0,
				"lower":    1079,
				"left":     0,
				"right":    1919,
				"pipe":     0,
			},
		},
	}, nil
}

func (f *FuegoAdapter) setKVMDisplays(c fuego.ContextWithBody[map[string]interface{}]) (map[string]string, error) {
	_, err := c.Body()
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"status":  "success",
		"message": "KVM display settings updated",
	}, nil
}

// GetOpenAPISpec returns the generated OpenAPI 3.0+ specification as JSON bytes
func (f *FuegoAdapter) GetOpenAPISpec() ([]byte, error) {
	spec := f.server.OutputOpenAPISpec()
	return json.MarshalIndent(spec, "", "  ")
}

// AddToGinRouter adds Fuego-generated OpenAPI endpoints to existing Gin router
func (f *FuegoAdapter) AddToGinRouter(router *gin.Engine) {
	// Add OpenAPI JSON endpoint to Gin
	router.GET("/api/openapi.json", func(c *gin.Context) {
		spec, err := f.GetOpenAPISpec()
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to generate OpenAPI spec"})
			return
		}
		c.Header("Content-Type", "application/json")
		c.Data(200, "application/json", spec)
	})

	// Add OpenAPI UI endpoint
	router.GET("/api/docs", func(c *gin.Context) {
		c.Header("Content-Type", "text/html")
		html := `<!DOCTYPE html>
		<html>
		<head>
			<title>API Documentation - OpenAPI 3.0+</title>
			<link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui.css" />
		</head>
		<body>
			<div id="swagger-ui"></div>
			<script src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-bundle.js"></script>
			<script>
				SwaggerUIBundle({
					url: '/api/openapi.json',
					dom_id: '#swagger-ui',
					presets: [
						SwaggerUIBundle.presets.apis,
						SwaggerUIBundle.presets.standalone
					],
					layout: "StandaloneLayout"
				});
			</script>
		</body>
		</html>`
		c.Data(200, "text/html", []byte(html))
	})
}
