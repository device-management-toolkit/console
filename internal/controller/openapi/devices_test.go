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

func TestRegisterDeviceRoutes_PasswordChangeContract(t *testing.T) {
	t.Parallel()

	f := newTestAdapter()
	f.RegisterDeviceRoutes()

	specBytes, err := f.GetOpenAPISpec()
	require.NoError(t, err)

	var spec struct {
		Paths map[string]map[string]struct {
			Security    []map[string][]string      `json:"security"`
			RequestBody json.RawMessage            `json:"requestBody"`
			Responses   map[string]json.RawMessage `json:"responses"`
		} `json:"paths"`
	}

	require.NoError(t, json.Unmarshal(specBytes, &spec))
	require.Contains(t, spec.Paths, "/api/v1/authorize/password")
	operation := spec.Paths["/api/v1/authorize/password"]["post"]
	require.NotEmpty(t, operation.Security)
	require.NotEmpty(t, operation.RequestBody)

	for _, status := range []string{"200", "400", "401", "409", "429", "500"} {
		require.Contains(t, operation.Responses, status)
	}

	require.Contains(t, string(specBytes), "currentPassword")
	require.Contains(t, string(specBytes), "newPassword")
	require.Contains(t, string(specBytes), "confirmPassword")
}
