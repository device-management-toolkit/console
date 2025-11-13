package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func TestKVMDisplaysEndpoints(t *testing.T) {
	t.Parallel()

	t.Run("GET success", func(t *testing.T) {
		t.Parallel()

		mockCtl := gomock.NewController(t)
		defer mockCtl.Finish()

		log := logger.New("error")
		deviceManagement := mocks.NewMockDeviceManagementFeature(mockCtl)
		amtExplorerMock := mocks.NewMockAMTExplorerFeature(mockCtl)
		exporterMock := mocks.NewMockExporter(mockCtl)
		engine := gin.New()
		handler := engine.Group("/api/v1")
		NewAmtRoutes(handler, deviceManagement, amtExplorerMock, exporterMock, log)

		deviceManagement.EXPECT().GetKVMScreenSettings(context.Background(), "guid1").Return(dto.KVMScreenSettings{Displays: []dto.KVMScreenDisplay{{DisplayIndex: 0, IsActive: true}}}, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/amt/kvm/displays/guid1", http.NoBody)
		rr := httptest.NewRecorder()
		engine.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("PUT success", func(t *testing.T) {
		t.Parallel()

		mockCtl := gomock.NewController(t)
		defer mockCtl.Finish()

		log := logger.New("error")
		deviceManagement := mocks.NewMockDeviceManagementFeature(mockCtl)
		amtExplorerMock := mocks.NewMockAMTExplorerFeature(mockCtl)
		exporterMock := mocks.NewMockExporter(mockCtl)
		engine := gin.New()
		handler := engine.Group("/api/v1")
		NewAmtRoutes(handler, deviceManagement, amtExplorerMock, exporterMock, log)

		payload := dto.KVMScreenSettingsRequest{DisplayIndex: 1}
		deviceManagement.EXPECT().SetKVMScreenSettings(context.Background(), "guid2", gomock.Any()).Return(dto.KVMScreenSettings{Displays: []dto.KVMScreenDisplay{{DisplayIndex: 0, IsActive: true}}}, nil)

		b, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/amt/kvm/displays/guid2", bytes.NewReader(b))
		rr := httptest.NewRecorder()
		engine.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
	})
}
