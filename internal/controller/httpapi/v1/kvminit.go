package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// getKVMInitData returns all data needed to initialize a KVM session.
// This combines display settings, power state, redirection status, and features
// into a single API call to reduce latency.
func (r *deviceManagementRoutes) getKVMInitData(c *gin.Context) {
	guid := c.Param("guid")

	initData, err := r.d.GetKVMInitData(c.Request.Context(), guid)
	if err != nil {
		r.l.Error(err, "http - v1 - getKVMInitData")
		ErrorResponse(c, err)

		return
	}

	c.JSON(http.StatusOK, initData)
}
