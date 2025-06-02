package dto

import "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/auditlog"

type AuditLog struct {
	TotalCount int                       `json:"totalCnt" binding:"required" example:"0"`
	Records    []auditlog.AuditLogRecord `json:"records" binding:"required"`
}
