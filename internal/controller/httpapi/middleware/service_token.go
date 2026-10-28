/*********************************************************************
 * Copyright (c) Intel Corporation 2026
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/pkg/logger"
)

// ServiceTokenHeader authenticates the calling service (RPS), not a human JWT session.
const ServiceTokenHeader = "X-Service-Token"

// RequireServiceToken gates RPS-only routes independent of AUTH_DISABLED/AUTH_GATEWAY_ENFORCED; fails closed when the token is unconfigured.
func RequireServiceToken(token string, l logger.Interface) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")

		if token == "" {
			l.Warn("auth.serviceToken is not configured; rejecting request to a service-only route")
			c.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		supplied := c.GetHeader(ServiceTokenHeader)
		if supplied == "" || subtle.ConstantTimeCompare([]byte(supplied), []byte(token)) != 1 {
			c.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		c.Next()
	}
}
