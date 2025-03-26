package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
