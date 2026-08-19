package v1

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

const (
	// maxExportRecords is the hard cap on records returned by a single export.
	maxExportRecords = 10000

	// exportTimeout bounds how long an export may take before it is abandoned.
	exportTimeout = 60 * time.Second

	// exportCountHeader tells the client how many records are in the response.
	exportCountHeader = "X-Total-Count"

	outcomeSuccess  = "success"
	outcomeError    = "error"
	tenantClaimKey  = "tenantId"
	subjectClaimKey = "sub"
)

// export handles GET /api/v1/devices/export. It returns a tenant-scoped,
// snapshot of every device details stored in deatabase.
func (dr *deviceRoutes) export(c *gin.Context) {
	start := time.Now()
	tenantID, userID := exportIdentity(c)

	ctx, cancel := context.WithTimeout(c.Request.Context(), exportTimeout)
	defer cancel()

	devicesList, err := dr.t.Get(ctx, maxExportRecords, 0, tenantID)
	if err != nil {
		dr.logExportAudit(c, userID, tenantID, 0, time.Since(start), outcomeError+": "+err.Error())
		dr.l.Error(err, "http - devices - v1 - export")
		// Never emit a partial file: discard everything and signal unavailability.
		c.JSON(http.StatusServiceUnavailable, gin.H{errorKey: "export unavailable"})

		return
	}

	records := make([]dto.DeviceExportRecord, 0, len(devicesList))
	for i := range devicesList {
		records = append(records, flatToNested(&devicesList[i]))
	}

	resp := dto.DeviceExport{
		Metadata: dto.ExportMetadata{
			ExportedAt: time.Now().UTC(),
			SwVersion:  swVersion(),
		},
		Summary: dto.ExportSummary{TotalCount: len(records)},
		Data:    records,
	}

	c.Header(exportCountHeader, strconv.Itoa(len(records)))
	dr.logExportAudit(c, userID, tenantID, len(records), time.Since(start), outcomeSuccess)
	c.JSON(http.StatusOK, resp)
}

// swVersion formats the running software version for the export metadata.
func swVersion() string {
	version := "unknown"
	if config.ConsoleConfig != nil {
		version = config.ConsoleConfig.Version
	}

	return "console " + version
}

// exportIdentity reads the tenant and subject from the already-verified JWT. The
// export is tenant-scoped: only devices for the caller's tenant are returned
func exportIdentity(c *gin.Context) (tenantID, userID string) {
	token := resolveToken(c)
	if token == "" {
		return "", ""
	}

	claims := jwt.MapClaims{}
	if _, _, err := jwt.NewParser().ParseUnverified(token, claims); err != nil {
		return "", ""
	}

	if v, ok := claims[tenantClaimKey].(string); ok {
		tenantID = v
	}

	if v, ok := claims[subjectClaimKey].(string); ok {
		userID = v
	}

	return tenantID, userID
}

// logExportAudit writes a server-side audit record for every export attempt,
func (dr *deviceRoutes) logExportAudit(c *gin.Context, userID, tenantID string, count int, dur time.Duration, outcome string) {
	serverHostname, _ := os.Hostname()

	msg := fmt.Sprintf("device export | outcome=%q userId=%q tenantId=%q sourceIp=%q serverHostname=%q deviceCount=%d durationMs=%d",
		outcome, userID, tenantID, c.ClientIP(), serverHostname, count, dur.Milliseconds())

	dr.l.Info(msg)
}

// flatToNested maps a flat stored device into the nested export shape.
// This is the single place where flat->nested shaping happens, so SQL and
// MongoDB backends produce an identical response.
func flatToNested(d *dto.Device) dto.DeviceExportRecord {
	tags := d.Tags
	if tags == nil {
		tags = []string{}
	}

	record := dto.DeviceExportRecord{
		GUID:         d.GUID,
		Hostname:     d.Hostname,
		FriendlyName: d.FriendlyName,
		Tags:         tags,
		TenantID:     d.TenantID,
		DeviceInfo: dto.DeviceExportInfo{
			BMC: nil, // export-gap: high-level BMC data is not collected yet.
		},
	}

	info := d.DeviceInfo
	if info == nil {
		return record
	}

	record.FirstDiscovered = info.FirstDiscovered
	record.LastSynced = info.LastSynced
	// export-gap: lastUpdated has no writer yet; add once the device DTO gains a lastUpdated field.

	record.DeviceInfo.ME = buildExportME(d, info)
	record.DeviceInfo.OS = buildExportOS(info)
	record.DeviceInfo.Platform = buildExportPlatform(info)

	return record
}

// buildExportME returns the ME subsystem, or nil for devices without a
// Management Engine
func buildExportME(d *dto.Device, info *dto.DeviceInfo) *dto.ExportME {
	hasME := info.FWVersion != "" || info.FWSku != "" || info.CurrentMode != "" ||
		(info.AMTEnabledInBIOS != nil && *info.AMTEnabledInBIOS)
	if !hasME {
		return nil
	}

	me := &dto.ExportME{
		DNSSuffix:         d.DNSSuffix,
		CurrentMode:       info.CurrentMode,
		MEBXEnabledInBIOS: info.AMTEnabledInBIOS,
		FWVersion:         info.FWVersion,
		FWBuild:           info.FWBuild,
		FWSku:             info.FWSku,
		Features:          info.Features,
		TLSMode:           info.TLSMode,
		DHCPEnabled:       info.DHCPEnabled,
		CertHashes:        info.CertHashes,
		UPID:              info.UPID,
	}

	if info.IPAddress != "" {
		// export-gap: only the wired ME IP is stored today; wireless has no source field yet, so it stays null.
		me.Network = &dto.ExportMENetwork{
			Wired: &dto.ExportMEInterface{
				IPAddress:   info.IPAddress,
				DHCPEnabled: info.DHCPEnabled,
			},
		}
	}

	return me
}

// buildExportOS returns the OS subsystem, or nil when no OS data was reported.
func buildExportOS(info *dto.DeviceInfo) *dto.ExportOS {
	hasOS := info.OSName != "" || info.OSVersion != "" || info.OSDistro != "" ||
		info.OSIPAddress != "" || info.LMSInstalled != nil || info.LMSVersion != "" ||
		info.MEInterfaceVersion != "" || info.MonitorConnected != nil || info.IEEE8021XEnabled != nil
	if !hasOS {
		return nil
	}

	osInfo := &dto.ExportOS{
		Name:               info.OSName,
		Version:            info.OSVersion,
		Distro:             info.OSDistro,
		LMSInstalled:       info.LMSInstalled,
		LMSVersion:         info.LMSVersion,
		MEInterfaceVersion: info.MEInterfaceVersion,
		MonitorConnected:   info.MonitorConnected,
		IEEE8021XEnabled:   info.IEEE8021XEnabled,
	}

	if info.OSIPAddress != "" {
		// export-gap: only the wired OS IP is stored today; wireless has no source field yet, so it stays null.
		osInfo.Network = &dto.ExportOSNetwork{
			Wired: []dto.ExportOSInterface{{IPAddress: info.OSIPAddress}},
		}
	}

	return osInfo
}

// buildExportPlatform returns the platform subsystem, or nil when no platform
// data was reported.
func buildExportPlatform(info *dto.DeviceInfo) *dto.ExportPlatform {
	if info.CPUModel == "" && info.EthernetAdapterCount == nil {
		return nil
	}

	// export-gap: adapters has no source fields yet, so it stays null.
	return &dto.ExportPlatform{
		CPU:                  info.CPUModel,
		EthernetAdapterCount: info.EthernetAdapterCount,
	}
}
