package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ciraDisabledMessage is returned when CIRA is disabled (APP_DISABLE_CIRA=true).
const ciraDisabledMessage = "CIRA is disabled on this instance"

// ciraDisabledMiddleware returns 404 for CIRA endpoints when CIRA is disabled:
// the CIRA surface is not mounted on this instance, so the resource does not exist.
func ciraDisabledMiddleware(disabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if disabled {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{errorKey: ciraDisabledMessage})

			return
		}

		c.Next()
	}
}
