package openapi

import (
	"net/http"
	"time"

	"github.com/go-fuego/fuego"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

func (f *FuegoAdapter) RegisterDeviceRoutes() {
	f.registerDeviceAuthRoutes()
	f.registerDeviceQueryRoutes()
	f.registerDeviceCertificateRoutes()
	f.registerDeviceMutationRoutes()
}

func (f *FuegoAdapter) registerDeviceAuthRoutes() {
	fuego.Post(f.server, "/api/v1/authorize", f.login,
		fuego.OptionTags("Devices"),
		fuego.OptionSummary("Authorize"),
		fuego.OptionDescription("Authenticate and return an access token"),
		apiRouteOptions(),
	)

	fuego.Get(f.server, "/api/v1/authorize/redirection/{id}", f.loginRedirection,
		fuego.OptionTags("Devices"),
		fuego.OptionSummary("Authorize Redirection"),
		fuego.OptionDescription("Generate an authorization token for device redirection"),
		fuego.OptionPath("id", "Device ID"),
		protectedRouteOptions(),
	)
}

func (f *FuegoAdapter) registerDeviceQueryRoutes() {
	fuego.Get(f.server, "/api/v1/devices", f.getDevices,
		fuego.OptionTags("Devices"),
		fuego.OptionSummary("List Devices"),
		fuego.OptionDescription("Retrieve all devices with optional pagination and filtering"),
		fuego.OptionQueryInt("$top", "Number of records to return"),
		fuego.OptionQueryInt("$skip", "Number of records to skip"),
		fuego.OptionQueryBool("$count", "Include total count"),
		fuego.OptionQuery("tags", "Comma-separated list of tags to filter devices"),
		fuego.OptionQuery("method", "Method to filter tags (any/all)"),
		fuego.OptionQuery("hostname", "Filter devices by host name"),
		fuego.OptionQuery("friendlyName", "Filter devices by friendly name"),
		protectedRouteOptions(),
	)

	fuego.Get(f.server, "/api/v1/devices/stats", f.getDeviceStats,
		fuego.OptionTags("Devices"),
		fuego.OptionSummary("Get Device Statistics"),
		fuego.OptionDescription("Retrieve statistics for devices"),
		protectedRouteOptions(),
	)

	fuego.Get(f.server, "/api/v1/devices/redirectstatus/{guid}", f.getRedirectStatus,
		fuego.OptionTags("Devices"),
		fuego.OptionSummary("Get Redirect Status"),
		fuego.OptionDescription("Retrieve redirect status for a specific device"),
		fuego.OptionPath("guid", "Device GUID"),
		protectedRouteOptions(),
	)

	fuego.Get(f.server, "/api/v1/devices/{guid}", f.getDeviceByID,
		fuego.OptionTags("Devices"),
		fuego.OptionSummary("Get Device by ID"),
		fuego.OptionDescription("Retrieve a specific device by ID"),
		fuego.OptionPath("guid", "Device GUID"),
		protectedRouteOptions(),
	)

	fuego.Get(f.server, "/api/v1/devices/tags", f.getTags,
		fuego.OptionTags("Devices"),
		fuego.OptionSummary("Get Available Device Tags"),
		fuego.OptionDescription("Retrieve a list of all available device tags"),
		protectedRouteOptions(),
	)
}

func (f *FuegoAdapter) registerDeviceCertificateRoutes() {
	fuego.Get(f.server, "/api/v1/devices/cert/{guid}", f.getDeviceCertificate,
		fuego.OptionTags("Devices"),
		fuego.OptionSummary("Get Device Certificate"),
		fuego.OptionDescription("Retrieve the certificate for a specific device"),
		fuego.OptionPath("guid", "Device GUID"),
		protectedRouteOptions(),
	)

	fuego.Post(f.server, "/api/v1/devices/cert/{guid}", f.pinDeviceCertificate,
		fuego.OptionTags("Devices"),
		fuego.OptionSummary("Pin Device Certificate"),
		fuego.OptionDescription("Pin the certificate for a specific device"),
		fuego.OptionPath("guid", "Device GUID"),
		protectedRouteOptions(),
	)

	fuego.Delete(f.server, "/api/v1/devices/cert/{guid}", f.deleteDeviceCertificate,
		fuego.OptionTags("Devices"),
		fuego.OptionSummary("Delete Device Certificate"),
		fuego.OptionDescription("Delete the pinned certificate for a specific device"),
		fuego.OptionPath("guid", "Device GUID"),
		protectedRouteOptions(),
	)
}

func (f *FuegoAdapter) registerDeviceMutationRoutes() {
	fuego.Post(f.server, "/api/v1/devices", f.createDevice,
		fuego.OptionTags("Devices"),
		fuego.OptionSummary("Create Device"),
		fuego.OptionDescription("Create a new device"),
		fuego.OptionDefaultStatusCode(http.StatusCreated),
		protectedRouteOptions(),
	)

	fuego.Patch(f.server, "/api/v1/devices", f.updateDevice,
		fuego.OptionTags("Devices"),
		fuego.OptionSummary("Update Device"),
		fuego.OptionDescription("Update an existing device"),
		protectedRouteOptions(),
	)

	fuego.Delete(f.server, "/api/v1/devices/{guid}", f.deleteDevice,
		fuego.OptionTags("Devices"),
		fuego.OptionSummary("Delete Device"),
		fuego.OptionDescription("Delete a device by ID"),
		fuego.OptionPath("guid", "Device GUID"),
		fuego.OptionDefaultStatusCode(http.StatusNoContent),
		protectedRouteOptions(),
	)
}

type AuthorizeRedirectionResponse struct {
	Token string `json:"token"`
}

func (f *FuegoAdapter) login(c fuego.ContextWithBody[dto.Credentials]) (AuthorizeRedirectionResponse, error) {
	_, err := c.Body()
	if err != nil {
		return AuthorizeRedirectionResponse{}, err
	}

	return AuthorizeRedirectionResponse{Token: "example-token"}, nil
}

func (f *FuegoAdapter) loginRedirection(_ fuego.ContextNoBody) (AuthorizeRedirectionResponse, error) {
	return AuthorizeRedirectionResponse{Token: "example-token"}, nil
}

func (f *FuegoAdapter) getDevices(_ fuego.ContextNoBody) (dto.DeviceCountResponse, error) {
	devices := []dto.Device{
		{
			GUID:             "example-guid-1",
			MPSUsername:      "mpsuser1",
			Username:         "admin1",
			Password:         "password1",
			ConnectionStatus: true,
			Hostname:         "device1.example.com",
		},
		{
			GUID:             "example-guid-2",
			MPSUsername:      "mpsuser2",
			Username:         "admin2",
			Password:         "password2",
			ConnectionStatus: false,
			Hostname:         "device2.example.com",
		},
	}

	return dto.DeviceCountResponse{
		Count: len(devices),
		Data:  devices,
	}, nil
}

func (f *FuegoAdapter) getDeviceStats(_ fuego.ContextNoBody) (dto.DeviceStatResponse, error) {
	return dto.DeviceStatResponse{
		TotalCount:        5,
		ConnectedCount:    3,
		DisconnectedCount: 2,
	}, nil
}

type DeviceRedirectStatusResponse struct {
	IsSOLConnected  bool `json:"isSOLConnected"`
	IsIDERConnected bool `json:"isIDERConnected"`
}

func (f *FuegoAdapter) getRedirectStatus(_ fuego.ContextNoBody) (DeviceRedirectStatusResponse, error) {
	return DeviceRedirectStatusResponse{
		IsSOLConnected:  false,
		IsIDERConnected: false,
	}, nil
}

func (f *FuegoAdapter) getDeviceCertificate(_ fuego.ContextNoBody) (dto.Certificate, error) {
	return dto.Certificate{
		GUID:       "example-guid-1",
		CommonName: "device1.example.com",
		NotBefore:  time.Now(),
		NotAfter:   time.Now().Add(365 * 24 * time.Hour),
	}, nil
}

func (f *FuegoAdapter) pinDeviceCertificate(c fuego.ContextWithBody[dto.PinCertificate]) (dto.Device, error) {
	req, err := c.Body()
	if err != nil {
		return dto.Device{}, err
	}

	return dto.Device{
		GUID:     "example-guid-1",
		Hostname: "device1.example.com",
		CertHash: req.SHA256Fingerprint,
		TenantID: "default",
		Tags:     []string{},
		UseTLS:   true,
	}, nil
}

func (f *FuegoAdapter) deleteDeviceCertificate(_ fuego.ContextNoBody) (dto.Device, error) {
	return dto.Device{}, nil
}

func (f *FuegoAdapter) getDeviceByID(_ fuego.ContextNoBody) (dto.Device, error) {
	return dto.Device{
		GUID:             "example-guid-1",
		MPSUsername:      "mpsuser1",
		Username:         "admin1",
		Password:         "password1",
		ConnectionStatus: true,
		Hostname:         "device1.example.com",
	}, nil
}

func (f *FuegoAdapter) getTags(_ fuego.ContextNoBody) ([]string, error) {
	return []string{"tag1", "tag2", "tag3"}, nil
}

func (f *FuegoAdapter) createDevice(c fuego.ContextWithBody[dto.Device]) (dto.Device, error) {
	config, err := c.Body()
	if err != nil {
		return dto.Device{}, err
	}

	return config, nil
}

func (f *FuegoAdapter) updateDevice(c fuego.ContextWithBody[dto.Device]) (dto.Device, error) {
	config, err := c.Body()
	if err != nil {
		return dto.Device{}, err
	}

	return config, nil
}

func (f *FuegoAdapter) deleteDevice(_ fuego.ContextNoBody) (NoContentResponse, error) {
	return NoContentResponse{}, nil
}
