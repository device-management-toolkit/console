package export

import (
	"io"

	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/auditlog"

	"github.com/open-amt-cloud-toolkit/console/internal/entity/dto/v1"
)

type Exporter interface {
	ExportAuditLogsCSV(logs []auditlog.AuditLogRecord) (io.Reader, error) // Converts logs to CSV and returns a reader
	ExportEventLogsCSV(logs []dto.EventLog) (io.Reader, error)            // Converts logs to CSV and returns a reader
}
