package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/open-amt-cloud-toolkit/console/internal/entity/dto/v1"
)

func (r *deviceManagementRoutes) cancelUserConsentCode(c *gin.Context) {
	guid := c.Param("guid")

	result, err := r.d.CancelUserConsent(c.Request.Context(), guid)
	if err != nil {
		r.l.Error(err, "http - v1 - cancelUserConsentCode")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, result)
}

func (r *deviceManagementRoutes) getUserConsentCode(c *gin.Context) {
	guid := c.Param("guid")

	response, err := r.d.GetUserConsentCode(c.Request.Context(), guid)
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

	response, err := r.d.SendConsentCode(c.Request.Context(), userConsent, guid)
	if err != nil {
		r.l.Error(err, "http - v1 - sendConsentCode")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, response)
}
