// Package v1 provides Redfish v1 API middleware.
package v1

import (
	"crypto/subtle"
	"encoding/base64"
	"strings"

	"github.com/gin-gonic/gin"
)

const expectedCredentialParts = 2

// BasicAuthValidator validates HTTP Basic Authentication
func BasicAuthValidator(expectedUsername, expectedPassword string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if !strings.HasPrefix(authHeader, "Basic ") {
			UnauthorizedError(c)
			c.Abort()

			return
		}

		// Extract and decode credentials
		credentials := strings.TrimPrefix(authHeader, "Basic ")

		decoded, err := base64.StdEncoding.DecodeString(credentials)
		if err != nil {
			UnauthorizedError(c)
			c.Abort()

			return
		}

		// Split username:password
		parts := strings.SplitN(string(decoded), ":", expectedCredentialParts)
		if len(parts) != expectedCredentialParts {
			UnauthorizedError(c)
			c.Abort()

			return
		}

		username, password := parts[0], parts[1]

		// Constant-time comparison to prevent timing attacks
		usernameMatch := subtle.ConstantTimeCompare([]byte(username), []byte(expectedUsername)) == 1
		passwordMatch := subtle.ConstantTimeCompare([]byte(password), []byte(expectedPassword)) == 1

		if usernameMatch && passwordMatch {
			c.Next()
		} else {
			UnauthorizedError(c)
			c.Abort()
		}
	}
}
