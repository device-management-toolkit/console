package v1

import (
	"context"
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	eventLogBatchSize     = 100
	maxDownloadRows       = 50000
	maxDownloadIterations = 1000
	downloadTimeout       = 45 * time.Second
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
	ctx, cancel := context.WithTimeout(c.Request.Context(), downloadTimeout)
	defer cancel()

	startIndex := 1
	iterations := 0
	rowsWritten := 0

	c.Header("Content-Disposition", "attachment; filename=audit_logs.csv")
	c.Header("Content-Type", "text/csv")

	writer := csv.NewWriter(c.Writer)
	if err := writer.Write([]string{"ID", "Time", "Event", "Description"}); err != nil {
		r.l.Error(err, "http - v1 - downloadAuditLog")
		ErrorResponse(c, err)

		return
	}

	for {
		if iterations >= maxDownloadIterations {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, response{Error: "audit log download exceeded max iterations", Message: "audit log download exceeded max iterations"})

			return
		}

		iterations++

		auditLogs, err := r.d.GetAuditLog(ctx, startIndex, guid)
		if err != nil {
			r.l.Error(err, "http - v1 - getAuditLog")
			ErrorResponse(c, err)

			return
		}

		if len(auditLogs.Records) == 0 {
			if rowsWritten >= auditLogs.TotalCount {
				break
			}

			c.AbortWithStatusJSON(http.StatusBadRequest, response{Error: "no progress while downloading audit log", Message: "no progress while downloading audit log"})

			return
		}

		for i := range auditLogs.Records {
			if rowsWritten >= maxDownloadRows {
				c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, response{Error: "audit log download exceeded max rows", Message: "audit log download exceeded max rows"})

				return
			}

			record := auditLogs.Records[i]
			if err := writer.Write([]string{
				strconv.Itoa(record.EventID),
				record.Time.String(),
				record.Event,
				record.ExStr,
			}); err != nil {
				r.l.Error(err, "http - v1 - downloadAuditLog")
				ErrorResponse(c, err)

				return
			}

			rowsWritten++
		}

		writer.Flush()

		if err := writer.Error(); err != nil {
			r.l.Error(err, "http - v1 - downloadAuditLog")
			ErrorResponse(c, err)

			return
		}

		if rowsWritten >= auditLogs.TotalCount {
			break
		}

		startIndex += len(auditLogs.Records)
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
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

	ctx, cancel := context.WithTimeout(c.Request.Context(), downloadTimeout)
	defer cancel()

	startIndex := 0
	iterations := 0
	rowsWritten := 0

	c.Header("Content-Disposition", "attachment; filename=event_logs.csv")
	c.Header("Content-Type", "text/csv")

	writer := csv.NewWriter(c.Writer)
	if err := writer.Write([]string{"Time", "Source", "Event Severity", "Description"}); err != nil {
		r.l.Error(err, "http - v1 - downloadEventLog")
		ErrorResponse(c, err)

		return
	}

	// Keep fetching logs until there are no more records.
	for {
		if iterations >= maxDownloadIterations {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, response{Error: "event log download exceeded max iterations", Message: "event log download exceeded max iterations"})

			return
		}

		iterations++

		eventLogs, err := r.d.GetEventLog(ctx, startIndex, eventLogBatchSize, guid)
		if err != nil {
			r.l.Error(err, "http - v1 - getEventLog")
			ErrorResponse(c, err)

			return
		}

		if len(eventLogs.Records) == 0 && eventLogs.HasMoreRecords {
			c.AbortWithStatusJSON(http.StatusBadRequest, response{Error: "no progress while downloading event log", Message: "no progress while downloading event log"})

			return
		}

		for i := range eventLogs.Records {
			if rowsWritten >= maxDownloadRows {
				c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, response{Error: "event log download exceeded max rows", Message: "event log download exceeded max rows"})

				return
			}

			record := eventLogs.Records[i]
			if err := writer.Write([]string{record.Time, record.Entity, record.EventSeverity, record.Description}); err != nil {
				r.l.Error(err, "http - v1 - downloadEventLog")
				ErrorResponse(c, err)

				return
			}

			rowsWritten++
		}

		writer.Flush()
		if err := writer.Error(); err != nil {
			r.l.Error(err, "http - v1 - downloadEventLog")
			ErrorResponse(c, err)

			return
		}

		// Break when no more records are available from AMT.
		if !eventLogs.HasMoreRecords {
			break
		}

		// Update the startIndex for the next batch
		startIndex += len(eventLogs.Records)
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		r.l.Error(err, "http - v1 - downloadEventLog")
		ErrorResponse(c, err)
	}
}
