package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// @Summary     Get Call List
// @Description Get a list of supported WSMAN calls
// @ID          getCallList
// @Tags  	    devices
// @Accept      json
// @Produce     json
// @Success     200 {object} DeviceCountResponse
// @Failure     500 {object} response
// @Router      /api/v1/devices [get]
func (r *deviceManagementRoutes) getCallList(c *gin.Context) {
	items := r.a.GetExplorerSupportedCalls()

	c.JSON(http.StatusOK, items)
}

// @Summary     Execute Call
// @Description Execute a call
// @ID          executeCall
// @Tags  	    amt
// @Accept      json
// @Produce     json
// @Success     200 {object} DeviceCountResponse
// @Failure     500 {object} response
// @Router      /api/v1/devices [get]
func (r *deviceManagementRoutes) executeCall(c *gin.Context) {
	guid := c.Param("guid")
	call := c.Param("call")

	result, err := r.a.ExecuteCall(c.Request.Context(), guid, call, "")
	if err != nil {
		r.l.Error(err, "http - explorer - v1 - executeCall")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, result)
}
