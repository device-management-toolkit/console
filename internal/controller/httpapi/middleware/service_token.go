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

// rpsServiceRoutes lists the method+path pairs RPS calls that a valid service token may
// authenticate instead of a JWT: the configuration reads RPS needs at activation, and the
// device writes RPS performs. Everything else still requires the normal login check.
var rpsServiceRoutes = map[string]map[string]bool{
	http.MethodGet: {
		"/api/v1/admin/domains":                       true,
		"/api/v1/admin/domains/:name":                 true,
		"/api/v1/admin/ciraconfigs":                   true,
		"/api/v1/admin/ciraconfigs/:ciraConfigName":   true,
		"/api/v1/admin/profiles":                      true,
		"/api/v1/admin/profiles/:name":                true,
		"/api/v1/admin/wirelessconfigs":               true,
		"/api/v1/admin/wirelessconfigs/:profileName":  true,
		"/api/v1/admin/ieee8021xconfigs":              true,
		"/api/v1/admin/ieee8021xconfigs/:profileName": true,
	},
	http.MethodPost: {
		"/api/v1/devices": true,
	},
	http.MethodPatch: {
		"/api/v1/devices": true,
	},
	http.MethodDelete: {
		"/api/v1/devices/:guid": true,
	},
}

// IsServiceRoute reports whether method+fullPath is one of the RPS-facing routes eligible for service-token auth.
func IsServiceRoute(method, fullPath string) bool {
	return rpsServiceRoutes[method][fullPath]
}

// HasValidServiceToken reports whether the request carries the configured service token.
func HasValidServiceToken(c *gin.Context, token string) bool {
	if token == "" {
		return false
	}

	supplied := c.GetHeader(ServiceTokenHeader)

	return supplied != "" && subtle.ConstantTimeCompare([]byte(supplied), []byte(token)) == 1
}

// RequireServiceToken gates RPS-only routes independent of AUTH_DISABLED/AUTH_GATEWAY_ENFORCED; fails closed when the token is unconfigured.
func RequireServiceToken(token string, l logger.Interface) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")

		if token == "" {
			l.Warn("auth.serviceToken is not configured; rejecting request to a service-only route")
			c.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		if !HasValidServiceToken(c, token) {
			c.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		c.Next()
	}
}

// RequireJWTOrServiceToken lets the service token stand in for the JWT check on the RPS-facing
// routes in rpsServiceRoutes; every other route under the group still runs jwtMiddleware.
func RequireJWTOrServiceToken(token string, jwtMiddleware gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if IsServiceRoute(c.Request.Method, c.FullPath()) && HasValidServiceToken(c, token) {
			c.Header("Cache-Control", "no-store")
			c.Next()

			return
		}

		jwtMiddleware(c)
	}
}
