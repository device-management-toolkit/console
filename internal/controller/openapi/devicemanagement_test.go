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

func TestGetAlarmOccurrences_ReturnsEmptyArray(t *testing.T) {
	t.Parallel()

	f := newTestAdapter()

	resp, err := f.getAlarmOccurrences(nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Empty(t, resp)

	body, err := json.Marshal(resp)
	require.NoError(t, err)
	require.JSONEq(t, "[]", string(body))
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

func TestRegisterWirelessProfileRoutes_DocumentPEAPUsernameRequirement(t *testing.T) {
	t.Parallel()

	f := newTestAdapter()
	f.RegisterDeviceManagementRoutes()

	specBytes, err := f.GetOpenAPISpec()
	require.NoError(t, err)

	var spec map[string]interface{}
	require.NoError(t, json.Unmarshal(specBytes, &spec))

	paths, ok := spec["paths"].(map[string]interface{})
	require.True(t, ok)

	profilePath, ok := paths["/api/v1/amt/networkSettings/wireless/profile/{guid}"].(map[string]interface{})
	require.True(t, ok, "wireless profile route should be registered")

	for _, method := range []string{"post", "patch"} {
		operation, ok := profilePath[method].(map[string]interface{})
		require.True(t, ok, "wireless profile %s operation should be registered", method)

		description, ok := operation["description"].(string)
		require.True(t, ok)
		require.Contains(t, description, "PEAPv0/EAP-MSCHAPv2 also requires ieee8021x.username and ieee8021x.password", method)
		require.Contains(t, description, "400 Bad Request", method)
	}
}
