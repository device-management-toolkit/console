package v1

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/wifi"

	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func TestWiFiStateHandlers(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	t.Run("request wireless state change", func(t *testing.T) {
		t.Parallel()

		mockCtl := gomock.NewController(t)
		defer mockCtl.Finish()

		devMock := mocks.NewMockDeviceManagementFeature(mockCtl)
		engine := gin.New()
		handler := engine.Group("/api/v1")
		NewAmtRoutes(handler, devMock, nil, nil, logger.New("error"))

		devMock.EXPECT().
			RequestWirelessStateChange(gomock.Any(), "my-guid", wifi.RequestedStateWifiEnabledS0SxAC).
			Return(wifi.RequestedStateWifiEnabledS0SxAC, nil)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/amt/networkSettings/wireless/state/my-guid", bytes.NewBufferString(`{"state":"WifiEnabledS0SxAC"}`))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.JSONEq(t, `{"state":"WifiEnabledS0SxAC"}`, w.Body.String())
	})

	t.Run("request wireless state change - invalid state type", func(t *testing.T) {
		t.Parallel()

		mockCtl := gomock.NewController(t)
		defer mockCtl.Finish()

		devMock := mocks.NewMockDeviceManagementFeature(mockCtl)
		engine := gin.New()
		handler := engine.Group("/api/v1")
		NewAmtRoutes(handler, devMock, nil, nil, logger.New("error"))

		req := httptest.NewRequest(http.MethodPost, "/api/v1/amt/networkSettings/wireless/state/my-guid", bytes.NewBufferString(`{"state":"invalid"}`))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("request wireless state change - invalid json", func(t *testing.T) {
		t.Parallel()

		mockCtl := gomock.NewController(t)
		defer mockCtl.Finish()

		devMock := mocks.NewMockDeviceManagementFeature(mockCtl)
		engine := gin.New()
		handler := engine.Group("/api/v1")
		NewAmtRoutes(handler, devMock, nil, nil, logger.New("error"))

		req := httptest.NewRequest(http.MethodPost, "/api/v1/amt/networkSettings/wireless/state/my-guid", bytes.NewBufferString(`{"state":`))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("request wireless state change - service failure", func(t *testing.T) {
		t.Parallel()

		mockCtl := gomock.NewController(t)
		defer mockCtl.Finish()

		devMock := mocks.NewMockDeviceManagementFeature(mockCtl)
		engine := gin.New()
		handler := engine.Group("/api/v1")
		NewAmtRoutes(handler, devMock, nil, nil, logger.New("error"))

		devMock.EXPECT().
			RequestWirelessStateChange(gomock.Any(), "my-guid", wifi.RequestedStateWifiEnabledS0SxAC).
			Return(wifi.RequestedState(0), ErrGeneral)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/amt/networkSettings/wireless/state/my-guid", bytes.NewBufferString(`{"state":"WifiEnabledS0SxAC"}`))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("request wireless state change - unsupported requested state conversion", func(t *testing.T) {
		t.Parallel()

		mockCtl := gomock.NewController(t)
		defer mockCtl.Finish()

		devMock := mocks.NewMockDeviceManagementFeature(mockCtl)
		engine := gin.New()
		handler := engine.Group("/api/v1")
		NewAmtRoutes(handler, devMock, nil, nil, logger.New("error"))

		devMock.EXPECT().
			RequestWirelessStateChange(gomock.Any(), "my-guid", wifi.RequestedStateWifiEnabledS0SxAC).
			Return(wifi.RequestedState(1), nil)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/amt/networkSettings/wireless/state/my-guid", bytes.NewBufferString(`{"state":"WifiEnabledS0SxAC"}`))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("get wireless state", func(t *testing.T) {
		t.Parallel()

		mockCtl := gomock.NewController(t)
		defer mockCtl.Finish()

		devMock := mocks.NewMockDeviceManagementFeature(mockCtl)
		engine := gin.New()
		handler := engine.Group("/api/v1")
		NewAmtRoutes(handler, devMock, nil, nil, logger.New("error"))

		devMock.EXPECT().
			GetWirelessState(gomock.Any(), "my-guid").
			Return(wifi.EnabledStateWifiEnabledS0SxAC, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/amt/networkSettings/wireless/state/my-guid", http.NoBody)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.JSONEq(t, `{"state":"WifiEnabledS0SxAC"}`, w.Body.String())
	})

	t.Run("get wireless state - service failure", func(t *testing.T) {
		t.Parallel()

		mockCtl := gomock.NewController(t)
		defer mockCtl.Finish()

		devMock := mocks.NewMockDeviceManagementFeature(mockCtl)
		engine := gin.New()
		handler := engine.Group("/api/v1")
		NewAmtRoutes(handler, devMock, nil, nil, logger.New("error"))

		devMock.EXPECT().
			GetWirelessState(gomock.Any(), "my-guid").
			Return(wifi.EnabledState(0), ErrGeneral)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/amt/networkSettings/wireless/state/my-guid", http.NoBody)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("get wireless state - no wifi port", func(t *testing.T) {
		t.Parallel()

		mockCtl := gomock.NewController(t)
		defer mockCtl.Finish()

		devMock := mocks.NewMockDeviceManagementFeature(mockCtl)
		engine := gin.New()
		handler := engine.Group("/api/v1")
		NewAmtRoutes(handler, devMock, nil, nil, logger.New("error"))

		devMock.EXPECT().
			GetWirelessState(gomock.Any(), "my-guid").
			Return(wifi.EnabledState(0), wsman.ErrNoWiFiPort)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/amt/networkSettings/wireless/state/my-guid", http.NoBody)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("get wireless state - unsupported enabled state conversion", func(t *testing.T) {
		t.Parallel()

		mockCtl := gomock.NewController(t)
		defer mockCtl.Finish()

		devMock := mocks.NewMockDeviceManagementFeature(mockCtl)
		engine := gin.New()
		handler := engine.Group("/api/v1")
		NewAmtRoutes(handler, devMock, nil, nil, logger.New("error"))

		devMock.EXPECT().
			GetWirelessState(gomock.Any(), "my-guid").
			Return(wifi.EnabledState(1), nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/amt/networkSettings/wireless/state/my-guid", http.NoBody)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		require.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
