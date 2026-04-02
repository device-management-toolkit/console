package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (r *deviceManagementRoutes) getCallList(c *gin.Context) {
	items := r.a.GetExplorerSupportedCalls()

	c.JSON(http.StatusOK, items)
}

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
