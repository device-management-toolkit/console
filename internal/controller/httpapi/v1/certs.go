package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/internal/controller/httpapi/middleware"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

func (r *deviceManagementRoutes) getCertificates(c *gin.Context) {
	guid := c.Param("guid")
	tenantID := middleware.TenantID(c)

	certs, err := r.d.GetCertificates(c.Request.Context(), guid, tenantID)
	if err != nil {
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, certs)
}

func (r *deviceManagementRoutes) getTLSSettingData(c *gin.Context) {
	guid := c.Param("guid")
	tenantID := middleware.TenantID(c)

	tlsSettingData, err := r.d.GetTLSSettingData(c.Request.Context(), guid, tenantID)
	if err != nil {
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, tlsSettingData)
}

func (r *deviceManagementRoutes) addCertificate(c *gin.Context) {
	guid := c.Param("guid")

	var certInfo dto.CertInfo
	if err := c.ShouldBindJSON(&certInfo); err != nil {
		ErrorResponse(c, err)

		return
	}

	tenantID := middleware.TenantID(c)
	handle, err := r.d.AddCertificate(c.Request.Context(), guid, tenantID, certInfo)
	if err != nil {
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, handle)
}

func (r *deviceManagementRoutes) deleteCertificate(c *gin.Context) {
	guid := c.Param("guid")
	instanceID := c.Param("instanceId")
	tenantID := middleware.TenantID(c)

	err := r.d.DeleteCertificate(c.Request.Context(), guid, instanceID, tenantID)
	if err != nil {
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Certificate deleted successfully"})
}
