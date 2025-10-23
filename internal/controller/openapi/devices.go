package openapi

import (
	"time"

	"github.com/go-fuego/fuego"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

func (f *FuegoAdapter) RegisterDeviceRoutes() {
	fuego.Get(f.server, "/api/v1/admin/devices", f.getDevices,
		fuego.OptionTags("Devices"),
		fuego.OptionSummary("List Devices"),
		fuego.OptionDescription("Retrieve all devices with optional pagination and filtering"),
		fuego.OptionQueryInt("$top", "Number of records to return"),
		fuego.OptionQueryInt("$skip", "Number of records to skip"),
		fuego.OptionQueryBool("$count", "Include total count"),
		fuego.OptionQuery("tags", "Comma-separated list of tags to filter devices"),
		fuego.OptionQuery("method", "Method to filter tags (any/all)"),
	)

	fuego.Get(f.server, "/api/v1/admin/devices/stats", f.getDeviceStats,
		fuego.OptionTags("Devices"),
		fuego.OptionSummary("Get Device Statistics"),
		fuego.OptionDescription("Retrieve statistics for devices"),
	)

	fuego.Get(f.server, "/api/v1/admin/devices/cert/{id}", f.getDeviceCertificate,
		fuego.OptionTags("Devices"),
		fuego.OptionSummary("Get Device Certificate"),
		fuego.OptionDescription("Retrieve the certificate for a specific device"),
		fuego.OptionPath("id", "Device ID"),
	)

	fuego.Post(f.server, "/api/v1/admin/devices/cert/{id}", f.pinDeviceCertificate,
		fuego.OptionTags("Devices"),
		fuego.OptionSummary("Pin Device Certificate"),
		fuego.OptionDescription("Pin the certificate for a specific device"),
		fuego.OptionPath("id", "Device ID"),
	)

	fuego.Get(f.server, "/api/v1/admin/devices/{id}", f.getDeviceByID,
		fuego.OptionTags("Devices"),
		fuego.OptionSummary("Get Device by ID"),
		fuego.OptionDescription("Retrieve a specific device by ID"),
		fuego.OptionPath("id", "Device ID"),
	)

	fuego.Get(f.server, "/api/v1/admin/devices/tags", f.getTags,
		fuego.OptionTags("Devices"),
		fuego.OptionSummary("Get Available Device Tags"),
		fuego.OptionDescription("Retrieve a list of all available device tags"),
	)

	fuego.Post(f.server, "/api/v1/admin/devices", f.createDevice,
		fuego.OptionTags("Devices"),
		fuego.OptionSummary("Create Device"),
		fuego.OptionDescription("Create a new device"),
	)

	fuego.Patch(f.server, "/api/v1/admin/devices", f.updateDevice,
		fuego.OptionTags("Devices"),
		fuego.OptionSummary("Update Device"),
		fuego.OptionDescription("Update an existing device"),
	)

	fuego.Delete(f.server, "/api/v1/admin/devices/{id}", f.deleteDevice,
		fuego.OptionTags("Devices"),
		fuego.OptionSummary("Delete Device"),
		fuego.OptionDescription("Delete a device by ID"),
		fuego.OptionPath("id", "Device ID"),
	)
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

func (f *FuegoAdapter) getDeviceCertificate(_ fuego.ContextNoBody) (dto.Certificate, error) {
	return dto.Certificate{
		GUID:       "example-guid-1",
		CommonName: "device1.example.com",
		NotBefore:  time.Now(),
		NotAfter:   time.Now().Add(365 * 24 * time.Hour),
	}, nil
}

func (f *FuegoAdapter) pinDeviceCertificate(_ fuego.ContextNoBody) (any, error) {
	return map[string]string{
		"message": "Certificate pinned successfully",
	}, nil
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

func (f *FuegoAdapter) deleteDevice(_ fuego.ContextNoBody) (any, error) {
	return nil, nil
}
