package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/open-amt-cloud-toolkit/console/internal/entity/dto/v1"
)

func (r *deviceManagementRoutes) getCertificates(c *gin.Context) {
	guid := c.Param("guid")

	certs, err := r.d.GetCertificates(c.Request.Context(), guid)
	if err != nil {
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, certs)
}

func (r *deviceManagementRoutes) getTLSSettingData(c *gin.Context) {
	guid := c.Param("guid")

	tlsSettingData, err := r.d.GetTLSSettingData(c.Request.Context(), guid)
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

	handle, err := r.d.AddCertificate(c.Request.Context(), guid, certInfo)
	if err != nil {
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, handle)
}
