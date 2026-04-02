package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (r *deviceManagementRoutes) getNetworkSettings(c *gin.Context) {
	guid := c.Param("guid")

	network, err := r.d.GetNetworkSettings(c.Request.Context(), guid)
	if err != nil {
		r.l.Error(err, "http - v1 - getNetworkSettings")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, network)
}
