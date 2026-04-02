package v1

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	gomock "go.uber.org/mock/gomock"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func TestSetLinkPreferenceHandler(t *testing.T) {
	t.Parallel()

	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	devMock := mocks.NewMockDeviceManagementFeature(mockCtl)

	gin.SetMode(gin.TestMode)

	engine := gin.New()

	handler := engine.Group("/api/v1")

	// Use NewAmtRoutes to register the route
	NewAmtRoutes(handler, devMock, nil, nil, logger.New("error"))

	// Success case -> device.SetLinkPreference returns ReturnValue 0
	devMock.EXPECT().
		SetLinkPreference(gomock.Any(), "my-guid", dto.LinkPreferenceRequest{LinkPreference: 1, Timeout: 60}).
		Return(dto.LinkPreferenceResponse{ReturnValue: 0}, nil)

	body := `{"linkPreference":1,"timeout":60}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/amt/network/linkPreference/my-guid", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d body=%s", w.Code, w.Body.String())
	}

	// AMT error case -> non-zero return value maps to 400
	devMock = mocks.NewMockDeviceManagementFeature(mockCtl)
	engine = gin.New()
	handler = engine.Group("/api/v1")
	NewAmtRoutes(handler, devMock, nil, nil, logger.New("error"))

	devMock.EXPECT().
		SetLinkPreference(gomock.Any(), "my-guid", dto.LinkPreferenceRequest{LinkPreference: 1, Timeout: 60}).
		Return(dto.LinkPreferenceResponse{ReturnValue: 5}, nil)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/amt/network/linkPreference/my-guid", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d body=%s", w.Code, w.Body.String())
	}

	// No WiFi port case -> SetLinkPreference returns ErrNoWiFiPort -> handler returns 404
	devMock = mocks.NewMockDeviceManagementFeature(mockCtl)
	engine = gin.New()
	handler = engine.Group("/api/v1")
	NewAmtRoutes(handler, devMock, nil, nil, logger.New("error"))

	devMock.EXPECT().
		SetLinkPreference(gomock.Any(), "my-guid", dto.LinkPreferenceRequest{LinkPreference: 1, Timeout: 60}).
		Return(dto.LinkPreferenceResponse{ReturnValue: -1}, wsman.ErrNoWiFiPort)

	req = httptest.NewRequest(http.MethodPost, "/api/v1/amt/network/linkPreference/my-guid", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()

	engine.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d body=%s", w.Code, w.Body.String())
	}
}
