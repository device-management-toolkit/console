// Package v1 implements Redfish API v1 error handling and utilities.
package v1

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/device-management-toolkit/console/config"
)

// Redfish Base Message Registry v1.11.0 Message IDs
const (
	BaseSuccessMessageID         = "Base.1.11.0.Success"
	BaseErrorMessageID           = "Base.1.11.0.GeneralError"
	BaseMalformedJSONID          = "Base.1.11.0.MalformedJSON"
	BasePropertyMissingID        = "Base.1.11.0.PropertyMissing"
	BasePropertyValueNotInListID = "Base.1.11.0.PropertyValueNotInList"
	BaseResourceNotFoundID       = "Base.1.11.0.ResourceNotFound"
	BaseOperationNotAllowedID    = "Base.1.11.0.OperationNotAllowed"
	BaseActionNotSupportedID     = "Base.1.11.0.ActionNotSupported"
	BaseNoValidSessionID         = "Base.1.11.0.NoValidSession"
	BaseInsufficientPrivilegeID  = "Base.1.11.0.InsufficientPrivilege"
	BaseNotAcceptableID          = "Base.1.11.0.NotAcceptable"
)

// redfishError creates a standard Redfish error response structure
func redfishError(messageID, message, severity, resolution string, messageArgs []string) map[string]any {
	extendedInfo := map[string]any{
		"MessageId":  messageID,
		"Message":    message,
		"Severity":   severity,
		"Resolution": resolution,
	}

	// Only add MessageArgs if provided and not empty
	if len(messageArgs) > 0 {
		extendedInfo["MessageArgs"] = messageArgs
	}

	return map[string]any{
		"error": map[string]any{
			"@Message.ExtendedInfo": []map[string]any{extendedInfo},
			"code":                  messageID,
			"message":               message,
		},
	}
}

// SetRedfishHeaders sets standard Redfish-compliant HTTP headers
func SetRedfishHeaders(c *gin.Context) {
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Header("OData-Version", "4.0")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Frame-Options", "DENY")
	c.Header("Content-Security-Policy", "default-src 'self'")
}

// redfishErrorResponse sends a Redfish error response with proper headers
func redfishErrorResponse(c *gin.Context, statusCode int, messageID, message, severity, resolution string, messageArgs []string) {
	SetRedfishHeaders(c)
	c.JSON(statusCode, redfishError(messageID, message, severity, resolution, messageArgs))
}

// MalformedJSONError returns a Redfish-compliant error for malformed JSON requests
func MalformedJSONError(c *gin.Context) {
	redfishErrorResponse(c, http.StatusBadRequest,
		BaseMalformedJSONID,
		"The request body submitted was malformed JSON and could not be parsed by the receiving service.",
		"Critical",
		"Ensure that the request body is valid JSON and resubmit the request.",
		nil)
}

// PropertyMissingError returns a Redfish-compliant error for missing required properties
func PropertyMissingError(c *gin.Context, propertyName string) {
	redfishErrorResponse(c, http.StatusBadRequest,
		BasePropertyMissingID,
		fmt.Sprintf("The property %s is a required property and must be included in the request.", propertyName),
		"Warning",
		"Ensure that the property is in the request body and has a valid value and resubmit the request.",
		[]string{propertyName})
}

// PropertyValueNotInListError returns a Redfish-compliant error for invalid enum values
func PropertyValueNotInListError(c *gin.Context, value, propertyName string) {
	redfishErrorResponse(c, http.StatusBadRequest,
		BasePropertyValueNotInListID,
		fmt.Sprintf("The value '%s' for the property %s is not in the list of acceptable values.", value, propertyName),
		"Warning",
		"Choose a value from the enumeration list that the implementation can support and resubmit the request if the operation failed.",
		[]string{value, propertyName})
}

// ResourceNotFoundError returns a Redfish-compliant error for missing resources
func ResourceNotFoundError(c *gin.Context, resourceType, resourceID string) {
	redfishErrorResponse(c, http.StatusNotFound,
		BaseResourceNotFoundID,
		fmt.Sprintf("The requested resource of type %s named '%s' was not found.", resourceType, resourceID),
		"Critical",
		"Provide a valid resource identifier and resubmit the request.",
		[]string{resourceType, resourceID})
}

// OperationNotAllowedError returns a Redfish-compliant error for operations not allowed due to resource state
func OperationNotAllowedError(c *gin.Context) {
	redfishErrorResponse(c, http.StatusConflict,
		BaseOperationNotAllowedID,
		"The operation was not successful because the resource is in a state that does not allow this operation.",
		"Critical",
		"The operation was not successful because the resource is in a state that does not allow this operation.",
		nil)
}

// MethodNotAllowedError returns a Redfish-compliant error for HTTP method not allowed (405)
func MethodNotAllowedError(c *gin.Context, action, allowedMethods string) {
	// Set the required Allow header for 405 responses
	c.Header("Allow", allowedMethods)

	redfishErrorResponse(c, http.StatusMethodNotAllowed,
		BaseActionNotSupportedID,
		fmt.Sprintf("The action %s is not supported by the resource.", action),
		"Critical",
		"The action supplied cannot be resubmitted to the implementation. Perhaps the action was invalid, the wrong resource was the target or the implementation documentation may be of assistance.",
		[]string{action})
}

// HTTPMethodNotAllowedError returns a Redfish-compliant error for unsupported HTTP methods (405)
func HTTPMethodNotAllowedError(c *gin.Context, method, resourceType, allowedMethods string) {
	// Set the required Allow header for 405 responses
	c.Header("Allow", allowedMethods)

	redfishErrorResponse(c, http.StatusMethodNotAllowed,
		BaseOperationNotAllowedID,
		fmt.Sprintf("The HTTP method %s is not allowed on this resource.", method),
		"Critical",
		fmt.Sprintf("The operation is not allowed. The %s method is not supported for %s resources. Use one of the allowed methods: %s.", method, resourceType, allowedMethods),
		[]string{method, resourceType})
}

// NoValidSessionError returns a Redfish-compliant error for missing or invalid authentication (401)
func NoValidSessionError(c *gin.Context) {
	redfishErrorResponse(c, http.StatusUnauthorized,
		BaseNoValidSessionID,
		"There is no valid session established with the implementation.",
		"Critical",
		"Establish a valid session before attempting any operations.",
		nil)
}

// InsufficientPrivilegeError returns a Redfish-compliant error for insufficient permissions (403)
func InsufficientPrivilegeError(c *gin.Context) {
	redfishErrorResponse(c, http.StatusForbidden,
		BaseInsufficientPrivilegeID,
		"There are insufficient privileges for the account or credentials associated with the current session to perform the requested operation.",
		"Critical",
		"Either abandon the operation or change the associated access rights and resubmit the request if the operation failed for authorization reasons.",
		nil)
}

// NotAcceptableError returns a Redfish-compliant error for unsupported media type (406)
func NotAcceptableError(c *gin.Context, requestedType string) {
	redfishErrorResponse(c, http.StatusNotAcceptable,
		BaseNotAcceptableID,
		fmt.Sprintf("The requested media type '%s' is not acceptable. This service only supports 'application/json'.", requestedType),
		"Warning",
		"Resubmit the request with a supported media type in the Accept header.",
		[]string{requestedType})
}

// RedfishJWTAuthMiddleware provides Redfish-compliant authentication error responses
func RedfishJWTAuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		tokenString = strings.Replace(tokenString, "Bearer ", "", 1)

		if tokenString == "" {
			NoValidSessionError(c)
			c.Abort()

			return
		}

		// if clientID is set, use the oidc verifier (this would need the verifier passed in)
		if cfg.ClientID != "" {
			// For OAuth/OIDC, we'd need to pass the verifier or handle differently
			// For now, return a general authentication error
			NoValidSessionError(c)
			c.Abort()

			return
		}

		claims := &jwt.MapClaims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(_ *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWTKey), nil
		})

		if err != nil || !token.Valid {
			NoValidSessionError(c)
			c.Abort()

			return
		}

		c.Next()
	}
}

// GeneralError returns a Redfish-compliant error for general internal errors
func GeneralError(c *gin.Context) {
	redfishErrorResponse(c, http.StatusInternalServerError,
		BaseErrorMessageID,
		"A general error has occurred. See ExtendedInfo for more information.",
		"Critical",
		"None.",
		nil)
}

// BadGatewayError returns a Redfish-compliant error for upstream service communication failures (502 Bad Gateway)
func BadGatewayError(c *gin.Context) {
	redfishErrorResponse(c, http.StatusBadGateway,
		BaseErrorMessageID,
		"The upstream service or managed device is unavailable or unreachable.",
		"Critical",
		"Verify network connectivity to the managed device and ensure the device is powered on and accessible.",
		nil)
}

// ServiceUnavailableError returns a Redfish-compliant error for upstream service communication failures (502 Bad Gateway)
// Deprecated: Use BadGatewayError for 502 errors or ServiceTemporarilyUnavailableError for 503 errors
func ServiceUnavailableError(c *gin.Context) {
	redfishErrorResponse(c, http.StatusBadGateway,
		BaseErrorMessageID,
		"The upstream service or managed device is unavailable or unreachable.",
		"Critical",
		"Verify network connectivity to the managed device and ensure the device is powered on and accessible.",
		nil)
}

// ServiceTemporarilyUnavailableError returns a Redfish-compliant error for temporary service unavailability (503 Service Unavailable)
func ServiceTemporarilyUnavailableError(c *gin.Context) {
	c.Header("Retry-After", "30") // Suggest retry after 30 seconds
	redfishErrorResponse(c, http.StatusServiceUnavailable,
		BaseErrorMessageID,
		"The service is temporarily unavailable due to overloading or maintenance. Please retry the request after some time.",
		"Critical",
		"Wait for the specified retry period and resubmit the request.",
		nil)
}

// RedfishRecoveryMiddleware provides Redfish-compliant error responses for panics (500)
func RedfishRecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic for debugging (using logger instead of fmt.Printf)
				// Note: Proper logger integration needed for production use
				_ = err // Acknowledge the error without printing

				// Check if response was already written
				if !c.Writer.Written() {
					GeneralError(c)
				}

				c.Abort()
			}
		}()

		c.Next()
	}
}
