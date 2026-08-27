package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/internal/controller/httpapi/middleware"
	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func explorerTest(t *testing.T) (*mocks.MockAMTExplorerFeature, *gin.Engine) {
	t.Helper()

	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	log := logger.New("error")
	amtExplorer := mocks.NewMockAMTExplorerFeature(mockCtl)

	engine := gin.New()
	engine.Use(middleware.Tenant(logger.New("error")))
	handler := engine.Group("/api/v1")

	NewAmtRoutes(handler, mocks.NewMockDeviceManagementFeature(mockCtl), amtExplorer, mocks.NewMockExporter(mockCtl), log)

	return amtExplorer, engine
}

func executeExplorerCall(t *testing.T, tenantHeader string) *httptest.ResponseRecorder {
	t.Helper()

	amtExplorer, engine := explorerTest(t)

	amtExplorer.EXPECT().
		ExecuteCall(gomock.Any(), "device-guid", "GetGeneralSettings", tenantHeader).
		Return(&dto.Explorer{XMLInput: "<in/>", XMLOutput: "<out/>"}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/amt/explorer/device-guid/GetGeneralSettings", http.NoBody)
	if tenantHeader != "" {
		req.Header.Set(middleware.TenantHeaderName, tenantHeader)
	}

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	return recorder
}

func TestExplorerExecuteCallUsesTenantHeader(t *testing.T) {
	t.Parallel()

	require.Equal(t, http.StatusOK, executeExplorerCall(t, "acme-corp").Code)
}

func TestExplorerExecuteCallWithoutTenantHeader(t *testing.T) {
	t.Parallel()

	require.Equal(t, http.StatusOK, executeExplorerCall(t, "").Code)
}
