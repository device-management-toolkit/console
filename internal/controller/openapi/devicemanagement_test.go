package openapi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func newTestAdapter() *FuegoAdapter {
	log := logger.New("error")

	return NewFuegoAdapter(usecase.Usecases{}, log)
}

func TestGetBootCapabilities(t *testing.T) {
	t.Parallel()

	f := newTestAdapter()

	result, err := f.getBootCapabilities(nil)

	require.NoError(t, err)
	require.Equal(t, dto.BootCapabilities{}, result)
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

	require.Contains(t, paths, "/api/v1/amt/boot/capabilities/{guid}", "boot capabilities route should be registered")
	require.Contains(t, paths, "/api/v1/amt/boot/remoteErase/{guid}", "set RPE enabled route should be registered")
}
