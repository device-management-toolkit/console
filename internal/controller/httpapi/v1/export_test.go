package v1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func exportTestHarness(t *testing.T) (*mocks.MockDeviceManagementFeature, *gin.Engine) {
	t.Helper()
	setupTestConfig()

	mockCtl := gomock.NewController(t)
	t.Cleanup(mockCtl.Finish)

	log := logger.New("error")
	device := mocks.NewMockDeviceManagementFeature(mockCtl)

	engine := gin.New()
	handler := engine.Group("/api/v1")

	NewDeviceRoutes(handler, device, log)

	return device, engine
}

func TestExportDevices_Success(t *testing.T) {
	t.Parallel()

	device, engine := exportTestHarness(t)

	firstDiscovered := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	lastSynced := time.Date(2026, 7, 20, 14, 22, 15, 0, time.UTC)
	amtEnabled := true
	dhcp := true
	adapters := 2

	meDevice := dto.Device{
		GUID:         "143e4567-e89b-12d3-a456-426614174000",
		Hostname:     "lab-pc-01",
		FriendlyName: "Lab PC Alpha",
		Tags:         []string{"campus-lab"},
		TenantID:     "tenant1",
		DNSSuffix:    "corp.example.com",
		MPSPassword:  "should-not-appear",
		MEBXPassword: "should-not-appear",
		Password:     "should-not-appear",
		DeviceInfo: &dto.DeviceInfo{
			FWVersion:            "16.1.32",
			FWBuild:              "3400",
			FWSku:                "16392",
			CurrentMode:          "Admin",
			Features:             "AMT Pro Corporate",
			AMTEnabledInBIOS:     &amtEnabled,
			DHCPEnabled:          &dhcp,
			IPAddress:            "10.0.0.12",
			FirstDiscovered:      &firstDiscovered,
			LastSynced:           &lastSynced,
			OSName:               "linux",
			OSIPAddress:          "10.49.76.163",
			CPUModel:             "Intel(R) Core(TM) Ultra 7 165H",
			EthernetAdapterCount: &adapters,
		},
	}

	nonMEDevice := dto.Device{
		GUID:         "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		Hostname:     "standard-pc-05",
		FriendlyName: "Standard Workstation",
		Tags:         nil,
		TenantID:     "tenant1",
		DeviceInfo: &dto.DeviceInfo{
			OSName:      "linux",
			OSIPAddress: "10.49.76.170",
		},
	}

	device.EXPECT().
		Get(gomock.Any(), maxExportRecords, 0, "").
		Return([]dto.Device{meDevice, nonMEDevice}, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/devices/export", http.NoBody)
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "2", w.Header().Get(exportCountHeader))

	var resp dto.DeviceExport
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	require.Equal(t, 2, resp.Summary.TotalCount)
	require.Len(t, resp.Data, 2)
	require.NotEmpty(t, resp.Metadata.SwVersion)
	require.False(t, resp.Metadata.ExportedAt.IsZero())

	// ME device is fully shaped into the nested model.
	me := resp.Data[0]
	require.Equal(t, "143e4567-e89b-12d3-a456-426614174000", me.GUID)
	require.Equal(t, []string{"campus-lab"}, me.Tags)
	require.NotNil(t, me.FirstDiscovered)
	require.NotNil(t, me.LastSynced)
	require.Nil(t, me.LastUpdated)
	require.NotNil(t, me.DeviceInfo.ME)
	require.Equal(t, "corp.example.com", me.DeviceInfo.ME.DNSSuffix)
	require.Equal(t, "16.1.32", me.DeviceInfo.ME.FWVersion)
	require.NotNil(t, me.DeviceInfo.ME.MEBXEnabledInBIOS)
	require.NotNil(t, me.DeviceInfo.ME.Network)
	require.Equal(t, "10.0.0.12", me.DeviceInfo.ME.Network.Wired.IPAddress)
	require.NotNil(t, me.DeviceInfo.OS)
	require.Equal(t, "linux", me.DeviceInfo.OS.Name)
	require.Equal(t, "10.49.76.163", me.DeviceInfo.OS.Network.Wired[0].IPAddress)
	require.NotNil(t, me.DeviceInfo.Platform)
	require.Equal(t, "Intel(R) Core(TM) Ultra 7 165H", me.DeviceInfo.Platform.CPU)
	require.Nil(t, me.DeviceInfo.BMC)

	// Non-ME device has a null me subsystem.
	nonME := resp.Data[1]
	require.Nil(t, nonME.DeviceInfo.ME)
	require.NotNil(t, nonME.DeviceInfo.OS)
	require.Equal(t, []string{}, nonME.Tags)

	// No credential material may ever leak into the export.
	body := w.Body.String()
	require.NotContains(t, body, "should-not-appear")
	require.NotContains(t, body, "mpspassword")
	require.NotContains(t, body, "mebxpassword")
}

func TestExportDevices_DatabaseError(t *testing.T) {
	t.Parallel()

	device, engine := exportTestHarness(t)

	device.EXPECT().
		Get(gomock.Any(), maxExportRecords, 0, "").
		Return(nil, assertErr{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/devices/export", http.NoBody)
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	require.Empty(t, w.Header().Get(exportCountHeader))
}

type assertErr struct{}

func (assertErr) Error() string { return "db failure" }
