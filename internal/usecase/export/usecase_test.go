package export_test

import (
	"encoding/csv"
	"testing"
	"time"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/auditlog"
	"github.com/stretchr/testify/assert"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/export"
)

func TestExportAuditLogsCSV(t *testing.T) {
	t.Parallel()

	nowTime := time.Now()
	tests := []struct {
		name    string
		logs    []auditlog.AuditLogRecord
		want    [][]string
		wantErr bool
	}{
		{
			name: "successful export",
			logs: []auditlog.AuditLogRecord{
				{EventID: 1, Time: nowTime, Event: "Event1", ExStr: "Description1"},
				{EventID: 2, Time: nowTime, Event: "Event2", ExStr: "Description2"},
			},
			want: [][]string{
				{"ID", "Time", "Event", "Description"},
				{"1", nowTime.String(), "Event1", "Description1"},
				{"2", nowTime.String(), "Event2", "Description2"},
			},
			wantErr: false,
		},
		{
			name:    "empty logs",
			logs:    []auditlog.AuditLogRecord{},
			want:    [][]string{{"ID", "Time", "Event", "Description"}},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			exporter := export.NewFileExporter()
			reader, err := exporter.ExportAuditLogsCSV(tc.logs)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, reader)

				// Read the output and verify the CSV structure
				csvReader := csv.NewReader(reader)
				records, err := csvReader.ReadAll()
				assert.NoError(t, err)
				assert.Equal(t, tc.want, records)
			}
		})
	}
}

func TestExportEventLogsCSV(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		logs    []dto.EventLog
		want    [][]string
		wantErr bool
	}{
		{
			name: "successful export",
			logs: []dto.EventLog{
				{Time: "2025-01-01T10:00:00Z", Entity: "Source1", EventSeverity: "High", Description: "Event1 Description"},
				{Time: "2025-01-02T11:00:00Z", Entity: "Source2", EventSeverity: "Low", Description: "Event2 Description"},
			},
			want: [][]string{
				{"Time", "Source", "Event Severity", "Description"},
				{"2025-01-01T10:00:00Z", "Source1", "High", "Event1 Description"},
				{"2025-01-02T11:00:00Z", "Source2", "Low", "Event2 Description"},
			},
			wantErr: false,
		},
		{
			name:    "empty logs",
			logs:    []dto.EventLog{},
			want:    [][]string{{"Time", "Source", "Event Severity", "Description"}},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			exporter := export.NewFileExporter()
			reader, err := exporter.ExportEventLogsCSV(tc.logs)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, reader)

				// Read the output and verify the CSV structure
				csvReader := csv.NewReader(reader)
				records, err := csvReader.ReadAll()
				assert.NoError(t, err)
				assert.Equal(t, tc.want, records)
			}
		})
	}
}
