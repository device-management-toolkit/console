package v1

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/auditlog"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
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

	auditLogs, err := r.d.GetAuditLog(c.Request.Context(), startIdx, guid)
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

	for {
		auditLogs, err := r.d.GetAuditLog(c.Request.Context(), startIndex, guid)
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
	if err := c.ShouldBindQuery(&odata); err != nil {
		validationErr := ErrValidationProfile.Wrap("get", "ShouldBindQuery", err)
		ErrorResponse(c, validationErr)

		return
	}

	eventLogs, err := r.d.GetEventLog(c.Request.Context(), odata.Skip, odata.Top, guid)
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

	// Keep fetching logs until NoMoreRecords is true
	for {
		eventLogs, err := r.d.GetEventLog(c.Request.Context(), 0, 0, guid)
		if err != nil {
			r.l.Error(err, "http - v1 - getEventLog")
			ErrorResponse(c, err)

			return
		}

		// Append the current batch of logs
		allEventLogs = append(allEventLogs, eventLogs.Records...)

		// Break if no more records
		if eventLogs.HasMoreRecords {
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
