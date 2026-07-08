package v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func TestDownloadEventLogPaginatesWithStartIndex(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	log := logger.New("error")
	deviceManagement := mocks.NewMockDeviceManagementFeature(ctrl)
	amtExplorer := mocks.NewMockAMTExplorerFeature(ctrl)
	exporter := mocks.NewMockExporter(ctrl)

	engine := gin.New()
	handler := engine.Group("/api/v1")
	NewAmtRoutes(handler, deviceManagement, amtExplorer, exporter, log)

	firstBatch := dto.EventLogs{
		Records: []dto.EventLog{
			{Time: "t1", Entity: "BIOS", EventSeverity: "Monitor", Description: "first"},
			{Time: "t2", Entity: "BIOS", EventSeverity: "Monitor", Description: "second"},
		},
		HasMoreRecords: true,
	}

	secondBatch := dto.EventLogs{
		Records: []dto.EventLog{
			{Time: "t3", Entity: "Intel(r) ME", EventSeverity: "Monitor", Description: "third"},
		},
		HasMoreRecords: false,
	}

	deviceManagement.EXPECT().GetEventLog(context.Background(), 0, eventLogBatchSize, "valid-guid").Return(firstBatch, nil)
	deviceManagement.EXPECT().GetEventLog(context.Background(), 2, eventLogBatchSize, "valid-guid").Return(secondBatch, nil)

	expectedLogs := append(append([]dto.EventLog{}, firstBatch.Records...), secondBatch.Records...)
	exporter.EXPECT().ExportEventLogsCSV(expectedLogs).Return(strings.NewReader("Time,Source,Event Severity,Description\n"), nil)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/amt/log/event/valid-guid/download", http.NoBody)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Header().Get("Content-Type"), "text/csv")
	require.Equal(t, "Time,Source,Event Severity,Description\n", w.Body.String())
}
