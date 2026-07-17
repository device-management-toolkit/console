package v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	wsAuditLog "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/auditlog"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func TestDownloadAuditLogStreamsCSVWithPagination(t *testing.T) {
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

	ts := time.Unix(0, 0).UTC()
	firstBatch := dto.AuditLog{
		TotalCount: 3,
		Records: []wsAuditLog.AuditLogRecord{
			{EventID: 1, Time: ts, Event: "first", ExStr: "desc-1"},
			{EventID: 2, Time: ts, Event: "second", ExStr: "desc-2"},
		},
	}

	secondBatch := dto.AuditLog{
		TotalCount: 3,
		Records: []wsAuditLog.AuditLogRecord{
			{EventID: 3, Time: ts, Event: "third", ExStr: "desc-3"},
		},
	}

	deviceManagement.EXPECT().GetAuditLog(gomock.Any(), 1, "valid-guid").Return(firstBatch, nil)
	deviceManagement.EXPECT().GetAuditLog(gomock.Any(), 3, "valid-guid").Return(secondBatch, nil)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/amt/log/audit/valid-guid/download", http.NoBody)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Header().Get("Content-Type"), "text/csv")
	body := w.Body.String()
	require.Contains(t, body, "ID,Time,Event,Description")
	require.Contains(t, body, "1,1970-01-01 00:00:00 +0000 UTC,first,desc-1")
	require.Contains(t, body, "2,1970-01-01 00:00:00 +0000 UTC,second,desc-2")
	require.Contains(t, body, "3,1970-01-01 00:00:00 +0000 UTC,third,desc-3")
}

func TestDownloadEventLogStreamsCSVWithPagination(t *testing.T) {
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

	deviceManagement.EXPECT().GetEventLog(gomock.Any(), 0, eventLogBatchSize, "valid-guid").Return(firstBatch, nil)
	deviceManagement.EXPECT().GetEventLog(gomock.Any(), 2, eventLogBatchSize, "valid-guid").Return(secondBatch, nil)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/amt/log/event/valid-guid/download", http.NoBody)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Header().Get("Content-Type"), "text/csv")
	body := w.Body.String()
	require.Contains(t, body, "Time,Source,Event Severity,Description")
	require.Contains(t, body, "t1,BIOS,Monitor,first")
	require.Contains(t, body, "t2,BIOS,Monitor,second")
	require.Contains(t, body, "t3,Intel(r) ME,Monitor,third")
}

func TestDownloadEventLogReturnsBadRequestOnZeroProgress(t *testing.T) {
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

	deviceManagement.EXPECT().GetEventLog(gomock.Any(), 0, eventLogBatchSize, "valid-guid").Return(dto.EventLogs{
		Records:        []dto.EventLog{},
		HasMoreRecords: true,
	}, nil)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/amt/log/event/valid-guid/download", http.NoBody)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, strings.ToLower(w.Body.String()), "no progress")
}
