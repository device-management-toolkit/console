package openapi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegisterDeviceRoutes_IncludesExportEndpoint(t *testing.T) {
	t.Parallel()

	f := newTestAdapter()
	f.RegisterDeviceRoutes()

	specBytes, err := f.GetOpenAPISpec()
	require.NoError(t, err)

	var spec map[string]interface{}
	require.NoError(t, json.Unmarshal(specBytes, &spec))

	paths, ok := spec["paths"].(map[string]interface{})
	require.True(t, ok)

	require.Contains(t, paths, "/api/v1/devices/export", "device export route should be registered")

	exportPath, ok := paths["/api/v1/devices/export"].(map[string]interface{})
	require.True(t, ok)
	require.Contains(t, exportPath, "get", "device export GET operation should be registered")
}

func TestExportDevices_ReturnsExampleRecord(t *testing.T) {
	t.Parallel()

	f := newTestAdapter()

	resp, err := f.exportDevices(nil)
	require.NoError(t, err)
	require.Equal(t, 1, resp.Summary.TotalCount)
	require.Len(t, resp.Data, 1)
	require.Equal(t, exampleDeviceGUID, resp.Data[0].GUID)
}

func TestRegisterDeviceRoutes_IncludesAdoptEndpoint(t *testing.T) {
	t.Parallel()

	f := newTestAdapter()
	f.RegisterDeviceRoutes()

	specBytes, err := f.GetOpenAPISpec()
	require.NoError(t, err)

	var spec map[string]interface{}
	require.NoError(t, json.Unmarshal(specBytes, &spec))

	paths, ok := spec["paths"].(map[string]interface{})
	require.True(t, ok)

	adoptPath, ok := paths["/api/v1/devices/{guid}/adopt"].(map[string]interface{})
	require.True(t, ok, "adopt route should be registered")

	adopt, ok := adoptPath["post"].(map[string]interface{})
	require.True(t, ok, "adopt POST operation should be registered")

	responses, ok := adopt["responses"].(map[string]interface{})
	require.True(t, ok)
	require.Contains(t, responses, "403", "rejected credentials should be documented as 403")
}
