package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/internal/usecase/amtexplorer"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/internal/usecase/export"
	"github.com/device-management-toolkit/console/pkg/logger"
)

type deviceManagementRoutes struct {
	d devices.Feature
	a amtexplorer.Feature
	e export.Exporter
	l logger.Interface
}

func NewAmtRoutes(handler *gin.RouterGroup, d devices.Feature, amt amtexplorer.Feature, e export.Exporter, l logger.Interface) {
	r := &deviceManagementRoutes{d, amt, e, l}

	h := handler.Group("/amt")
	{
		h.GET("version/:guid", r.getVersion)

		h.GET("features/:guid", r.getFeatures)
		h.POST("features/:guid", r.setFeatures)

		h.GET("alarmOccurrences/:guid", r.getAlarmOccurrences)
		h.POST("alarmOccurrences/:guid", r.createAlarmOccurrences)
		h.DELETE("alarmOccurrences/:guid", r.deleteAlarmOccurrences)

		h.GET("hardwareInfo/:guid", r.getHardwareInfo)
		h.GET("diskInfo/:guid", r.getDiskInfo)
		h.GET("power/state/:guid", r.getPowerState)
		h.POST("power/action/:guid", r.powerAction)
		h.POST("power/on/:guid", r.powerOn)
		h.POST("power/off/:guid", r.powerOff)
		h.POST("power/bootOptions/:guid", r.setBootOptions)
		h.POST("power/bootoptions/:guid", r.setBootOptions)
		h.GET("power/bootSources/:guid", r.getBootSources)
		h.GET("power/capabilities/:guid", r.getPowerCapabilities)

		h.GET("log/audit/:guid", r.getAuditLog)
		h.GET("log/audit/:guid/download", r.downloadAuditLog)
		h.GET("log/event/:guid", r.getEventLog)
		h.GET("log/event/:guid/download", r.downloadEventLog)
		h.GET("generalSettings/:guid", r.getGeneralSettings)

		h.GET("userConsentCode/cancel/:guid", r.cancelUserConsentCode)
		h.GET("userConsentCode/:guid", r.getUserConsentCode)
		h.POST("userConsentCode/:guid", r.sendConsentCode)

		h.GET("networkSettings/:guid", r.getNetworkSettings)

		h.GET("explorer", r.getCallList)
		h.GET("explorer/:guid/:call", r.executeCall)
		h.GET("tls/:guid", r.getTLSSettingData)

		h.GET("certificates/:guid", r.getCertificates)
		h.POST("certificates/:guid", r.addCertificate)

		// KVM display settings
		h.GET("kvm/displays/:guid", r.getKVMDisplays)
		h.PUT("kvm/displays/:guid", r.setKVMDisplays)
	}
}
