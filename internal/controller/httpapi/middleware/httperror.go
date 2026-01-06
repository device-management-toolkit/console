package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/pkg/consoleerrors"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// ErrorHandling returns a Gin middleware that recovers panics and
// converts errors into standardized JSON responses. It expects handlers to
// set an error on the context via c.Error(err) or return an *consoleerrors.HTTPError.
func ErrorHandling(l logger.Interface) gin.HandlerFunc {
	// Initialize error mapper with all domain error mappers
	errorMapper := NewErrorMapper()

	return func(c *gin.Context) {
		// create trace id for this request
		traceID := consoleerrors.TraceID()

		// set trace id on context so handlers can use it
		c.Set("traceId", traceID)

		defer func() {
			if rec := recover(); rec != nil {
				// panic recovered; log and return 500
				l.Error("panic recovered: %v, traceId=%s", rec, traceID)
				resp := consoleerrors.NewHTTPError(http.StatusInternalServerError, "ERR_PANIC", "internal server error", nil, nil)
				c.AbortWithStatusJSON(resp.Status, resp.ToResponse(traceID))
			}
		}()

		c.Next()

		// If errors were added to the context, pick the last one
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			// Use the error mapper to convert domain errors to HTTPError
			httpErr := errorMapper.Map(err)

			if httpErr.Status >= 500 {
				l.Error("request error: %s traceId=%s", httpErr.Error(), traceID)
			} else {
				l.Warn("request error: %s traceId=%s", httpErr.Error(), traceID)
			}

			c.AbortWithStatusJSON(httpErr.Status, httpErr.ToResponse(traceID))
		}
	}
}
