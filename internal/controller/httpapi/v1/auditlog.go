package v1

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/auditlog"

	"github.com/device-management-toolkit/console/internal/controller/httpapi/middleware"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

const (
	eventLogBatchSize = 100
)

func (r *deviceManagementRoutes) getAuditLog(c *gin.Context) {
	guid := c.Param("guid")

	startIndex := c.Query("startIndex")

	startIdx, err := strconv.Atoi(startIndex)
	if err != nil {
		r.l.Error(err, "http - v1 - getAuditLog")
		ErrorResponse(c, err)

		return
	}

	tenantID := middleware.TenantID(c)

	auditLogs, err := r.d.GetAuditLog(c.Request.Context(), startIdx, guid, tenantID)
	if err != nil {
		r.l.Error(err, "http - v1 - getAuditLog")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, auditLogs)
}

func (r *deviceManagementRoutes) downloadAuditLog(c *gin.Context) {
	guid := c.Param("guid")

	var allRecords []auditlog.AuditLogRecord

	startIndex := 1

	tenantID := middleware.TenantID(c)
	for {
		auditLogs, err := r.d.GetAuditLog(c.Request.Context(), startIndex, guid, tenantID)
		if err != nil {
			r.l.Error(err, "http - v1 - getAuditLog")
			ErrorResponse(c, err)

			return
		}

		allRecords = append(allRecords, auditLogs.Records...)

		if len(allRecords) >= auditLogs.TotalCount {
			break
		}

		startIndex += len(auditLogs.Records)
	}

	// Convert logs to CSV
	csvReader, err := r.e.ExportAuditLogsCSV(allRecords)
	if err != nil {
		r.l.Error(err, "http - v1 - downloadAuditLog")
		ErrorResponse(c, err)

		return
	}

	// Serve the CSV file
	c.Header("Content-Disposition", "attachment; filename=audit_logs.csv")
	c.Header("Content-Type", "text/csv")

	_, err = io.Copy(c.Writer, csvReader)
	if err != nil {
		r.l.Error(err, "http - v1 - downloadAuditLog")
		ErrorResponse(c, err)
	}
}

func (r *deviceManagementRoutes) getEventLog(c *gin.Context) {
	guid := c.Param("guid")

	var odata OData
	if err := odata.BindAndValidate(c); err != nil {
		validationErr := ErrValidationProfile.Wrap("get", "BindAndValidate", err)
		ErrorResponse(c, validationErr)

		return
	}

	tenantID := middleware.TenantID(c)
	eventLogs, err := r.d.GetEventLog(c.Request.Context(), odata.Skip, odata.Top, guid, tenantID)
	if err != nil {
		r.l.Error(err, "http - v1 - getEventLog")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, eventLogs)
}

func (r *deviceManagementRoutes) downloadEventLog(c *gin.Context) {
	guid := c.Param("guid")

	var allEventLogs []dto.EventLog

	startIndex := 0

	tenantID := middleware.TenantID(c)
	// Keep fetching logs until there are no more records.
	for {
		eventLogs, err := r.d.GetEventLog(c.Request.Context(), startIndex, eventLogBatchSize, guid, tenantID)
		if err != nil {
			r.l.Error(err, "http - v1 - getEventLog")
			ErrorResponse(c, err)

			return
		}

		// Append the current batch of logs
		allEventLogs = append(allEventLogs, eventLogs.Records...)

		// Break when no more records are available from AMT.
		if !eventLogs.HasMoreRecords {
			break
		}

		// Update the startIndex for the next batch
		startIndex += len(eventLogs.Records)
	}

	// Convert logs to CSV
	csvReader, err := r.e.ExportEventLogsCSV(allEventLogs)
	if err != nil {
		r.l.Error(err, "http - v1 - downloadEventLog")
		ErrorResponse(c, err)

		return
	}

	// Serve the CSV file
	c.Header("Content-Disposition", "attachment; filename=event_logs.csv")
	c.Header("Content-Type", "text/csv")

	_, err = io.Copy(c.Writer, csvReader)
	if err != nil {
		r.l.Error(err, "http - v1 - downloadEventLog")
		ErrorResponse(c, err)
	}
}
