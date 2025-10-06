package export

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/auditlog"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

type FileExporter struct{}

func NewFileExporter() *FileExporter {
	return &FileExporter{}
}

func (e *FileExporter) ExportAuditLogsCSV(logs []auditlog.AuditLogRecord) (io.Reader, error) {
	buffer := &bytes.Buffer{}
	writer := csv.NewWriter(buffer)

	// Assuming logs is a slice of a specific struct
	// Convert it to a slice of string slices for CSV rows
	records := [][]string{{"ID", "Time", "Event", "Description"}} // Replace with actual field names
	for i := range logs {
		records = append(records, []string{
			fmt.Sprintf("%d", logs[i].EventID),
			logs[i].Time.String(),
			logs[i].Event,
			logs[i].ExStr,
		})
	}

	if err := writer.WriteAll(records); err != nil {
		return nil, fmt.Errorf("error writing CSV: %w", err)
	}

	return buffer, nil
}

// ExportEventLogsCSV converts event logs to CSV and returns a reader.
func (e *FileExporter) ExportEventLogsCSV(logs []dto.EventLog) (io.Reader, error) {
	buffer := &bytes.Buffer{}
	writer := csv.NewWriter(buffer)

	// Assuming logs is a slice of a specific struct
	// Convert it to a slice of string slices for CSV rows
	records := [][]string{{"Time", "Source", "Event Severity", "Description"}} // Replace with actual field names
	for i := range logs {
		records = append(records, []string{
			logs[i].Time,
			logs[i].Entity,
			logs[i].EventSeverity,
			logs[i].Description,
		})
	}

	if err := writer.WriteAll(records); err != nil {
		return nil, fmt.Errorf("error writing CSV: %w", err)
	}

	return buffer, nil
}
