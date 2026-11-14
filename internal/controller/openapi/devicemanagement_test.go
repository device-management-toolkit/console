package openapi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	dtov2 "github.com/device-management-toolkit/console/internal/entity/dto/v2"
	"github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func newTestAdapter() *FuegoAdapter {
	log := logger.New("error")

	return NewFuegoAdapter(usecase.Usecases{}, log)
}

func TestSetRemoteEraseOptions_ReturnsEmptyResponse(t *testing.T) {
	t.Parallel()

	f := newTestAdapter()

	resp, err := f.setRemoteEraseOptions(nil)
	require.NoError(t, err)
	require.Equal(t, NoContentResponse{}, resp)
}

func TestGetRemoteEraseStatus_ReturnsEmptyResponse(t *testing.T) {
	t.Parallel()

	f := newTestAdapter()

	resp, err := f.getRemoteEraseStatus(nil)
	require.NoError(t, err)
	require.Equal(t, dtov2.RemoteEraseStatus{}, resp)
}

func TestRemoteEraseStatusSchemaEnum(t *testing.T) {
	t.Parallel()

	f := newTestAdapter()
	f.RegisterDeviceManagementRoutes()

	spec, err := f.GetOpenAPISpec()
	require.NoError(t, err)

	var document struct {
		Components struct {
			Schemas map[string]struct {
				Properties map[string]struct {
					Enum []string `json:"enum"`
				} `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}

	require.NoError(t, json.Unmarshal(spec, &document))
	require.Equal(t, []string{"pending", "succeeded", "failed", "unknown"},
		document.Components.Schemas["RemoteEraseStatus"].Properties["status"].Enum)
}

func TestRegisterPowerRoutes_IncludesBootEndpoints(t *testing.T) {
	t.Parallel()

	f := newTestAdapter()
	f.RegisterDeviceManagementRoutes()

	specBytes, err := f.GetOpenAPISpec()
	require.NoError(t, err)

	var spec map[string]interface{}
	require.NoError(t, json.Unmarshal(specBytes, &spec))

	paths, ok := spec["paths"].(map[string]interface{})
	require.True(t, ok)

	require.Contains(t, paths, "/api/v1/amt/boot/remoteErase/{guid}", "remote erase route should be registered")

	remoteErasePath, ok := paths["/api/v1/amt/boot/remoteErase/{guid}"].(map[string]interface{})
	require.True(t, ok)
	require.Contains(t, remoteErasePath, "get", "remote erase GET operation should be registered")
	require.Contains(t, remoteErasePath, "post", "remote erase POST operation should be registered")
}
