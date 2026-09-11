package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/internal/controller/httpapi/middleware"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

func (r *deviceManagementRoutes) cancelUserConsentCode(c *gin.Context) {
	guid := c.Param("guid")
	tenantID := middleware.TenantID(c)

	result, err := r.d.CancelUserConsent(c.Request.Context(), guid, tenantID)
	if err != nil {
		r.l.Error(err, "http - v1 - cancelUserConsentCode")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, result)
}

func (r *deviceManagementRoutes) getUserConsentCode(c *gin.Context) {
	guid := c.Param("guid")
	tenantID := middleware.TenantID(c)

	response, err := r.d.GetUserConsentCode(c.Request.Context(), guid, tenantID)
	if err != nil {
		r.l.Error(err, "http - v1 - getUserConsentCode")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, response)
}

func (r *deviceManagementRoutes) sendConsentCode(c *gin.Context) {
	guid := c.Param("guid")

	var userConsent dto.UserConsentCode
	if err := c.ShouldBindJSON(&userConsent); err != nil {
		ErrorResponse(c, err)

		return
	}

	tenantID := middleware.TenantID(c)
	response, err := r.d.SendConsentCode(c.Request.Context(), userConsent, guid, tenantID)
	if err != nil {
		r.l.Error(err, "http - v1 - sendConsentCode")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, response)
}
