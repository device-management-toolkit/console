// Package v1 provides Redfish-compliant error handling utilities.
package v1

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
)

const (
	// HTTP header constants for Redfish responses
	headerODataVersion = "OData-Version"
	headerContentType  = "Content-Type"
	headerLocation     = "Location"
	headerRetryAfter   = "Retry-After"

	// Header values
	contentTypeJSON = "application/json; charset=utf-8"
	contentTypeXML  = "application/xml"
	odataVersion    = "4.0"

	// Common error messages
	msgInternalServerError = "An internal server error occurred."
)

// ErrUnknownErrorType is returned when an unknown error type is requested
var ErrUnknownErrorType = errors.New("unknown error type")

// registryMgr is the global registry manager instance
var registryMgr = GetRegistryManager()

// ErrorConfig defines configuration for a specific error type
type ErrorConfig struct {
	RegistryKey    string
	StatusCode     int
	CustomMessage  string
	OverrideMsg    bool
	RetryAfter     int
	MessageOveride func(string) string
}

// errorConfigMap maps error types to their registry configuration
var errorConfigMap = map[string]ErrorConfig{
	"Conflict": {
		RegistryKey: "ResourceInUse",
		StatusCode:  http.StatusConflict,
	},
	"SessionConflict": {
		RegistryKey:   "ResourceInUse",
		StatusCode:    http.StatusConflict,
		CustomMessage: "A session already exists for this user. Delete the existing session before creating a new one.",
		OverrideMsg:   true,
	},
	"PowerStateConflict": {
		RegistryKey: "ResourceInUse",
		StatusCode:  http.StatusConflict,
	},
	"MethodNotAllowed": {
		RegistryKey: "MethodNotAllowed",
		StatusCode:  http.StatusMethodNotAllowed,
	},
	"Unauthorized": {
		RegistryKey:   "InsufficientPrivilege",
		StatusCode:    http.StatusUnauthorized,
		CustomMessage: "Unauthorized access",
		OverrideMsg:   true,
	},
	"BadRequest": {
		RegistryKey: "GeneralError",
		StatusCode:  http.StatusBadRequest,
	},
	"Forbidden": {
		RegistryKey:   "InsufficientPrivilege",
		StatusCode:    http.StatusForbidden,
		CustomMessage: "Insufficient privileges to perform operation",
		OverrideMsg:   true,
	},
	"ServiceUnavailable": {
		RegistryKey: "ServiceTemporarilyUnavailable",
		StatusCode:  http.StatusServiceUnavailable,
	},
	"NotFound": {
		RegistryKey: "ResourceMissing",
		StatusCode:  http.StatusNotFound,
		MessageOveride: func(resource string) string {
			return resource + " not found"
		},
	},
	"MalformedJSON": {
		RegistryKey: "MalformedJSON",
		StatusCode:  http.StatusBadRequest,
	},
	"PropertyMissing": {
		RegistryKey: "PropertyMissing",
		StatusCode:  http.StatusBadRequest,
		MessageOveride: func(propertyName string) string {
			return "Missing or empty " + propertyName
		},
	},
	"PropertyValueNotInList": {
		RegistryKey: "PropertyValueNotInList",
		StatusCode:  http.StatusBadRequest,
		MessageOveride: func(propertyName string) string {
			return "Invalid " + propertyName
		},
	},
}

// sendRedfishError is a generic error handler using the error configuration lookup table
func sendRedfishError(c *gin.Context, errorType, customMessage string, args ...interface{}) {
	SetRedfishHeaders(c)

	config, exists := errorConfigMap[errorType]
	if !exists {
		// Fallback to internal error if config not found
		InternalServerError(c, fmt.Errorf("%w: %s", ErrUnknownErrorType, errorType))

		return
	}

	handleRetryAfterHeader(c, config, errorType, args)

	errorResponse, err := createErrorResponse("Base", config.RegistryKey, args...)
	if err != nil {
		// This should never happen since the registry is embedded
		InternalServerError(c, err)

		return
	}

	applyMessageOverrides(errorResponse, customMessage, config, args)

	c.JSON(config.StatusCode, errorResponse)
}

// handleRetryAfterHeader sets the Retry-After header for service unavailable errors
func handleRetryAfterHeader(c *gin.Context, config ErrorConfig, errorType string, args []interface{}) {
	if config.RetryAfter <= 0 && (errorType != "ServiceUnavailable" || len(args) == 0) {
		return
	}

	retryAfter := config.RetryAfter

	if len(args) > 0 {
		if seconds, ok := args[0].(int); ok {
			retryAfter = seconds
		}
	}

	if retryAfter > 0 {
		c.Header(headerRetryAfter, fmt.Sprintf("%d", retryAfter))
	}
}

// applyMessageOverrides applies custom message overrides to the error response
func applyMessageOverrides(errorResponse *generated.RedfishError, customMessage string, config ErrorConfig, args []interface{}) {
	switch {
	case customMessage != "":
		errorResponse.Error.Message = &customMessage
		(*errorResponse.Error.MessageExtendedInfo)[0].Message = &customMessage
	case config.OverrideMsg:
		errorResponse.Error.Message = &config.CustomMessage
	case config.MessageOveride != nil && len(args) > 0:
		if strArg, ok := args[0].(string); ok {
			overrideMsg := config.MessageOveride(strArg)
			errorResponse.Error.Message = &overrideMsg
		}
	}
}

// mapSeverityToResourceHealth converts registry severity strings to generated.ResourceHealth enum
func mapSeverityToResourceHealth(severity string) string {
	switch severity {
	case "Critical":
		return string(generated.Critical)
	case "Warning":
		return string(generated.Warning)
	case "OK":
		return string(generated.OK)
	default:
		// Default to Warning for unknown severity levels
		return string(generated.Warning)
	}
}

// SetRedfishHeaders sets Redfish-compliant headers
func SetRedfishHeaders(c *gin.Context) {
	c.Header(headerContentType, contentTypeJSON)
	c.Header(headerODataVersion, odataVersion)
	c.Header("Cache-Control", "no-cache")
}

// createErrorResponse creates a Redfish error response using registry lookup.
// Note: registryName currently always receives "Base", but the parameter is kept for
// future extensibility when additional registries (Task, Update, ResourceEvent) are added.
func createErrorResponse(registryName, messageKey string, args ...interface{}) (*generated.RedfishError, error) {
	regMsg, err := registryMgr.LookupMessage(registryName, messageKey)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup message %s.%s: %w", registryName, messageKey, err)
	}

	messageStr := regMsg.FormatMessage(args...)
	messageID := regMsg.MessageID
	resolution := regMsg.Resolution
	severity := mapSeverityToResourceHealth(regMsg.Severity)

	errorResponse := &generated.RedfishError{
		Error: struct {
			MessageExtendedInfo *[]generated.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
			Code                *string                     `json:"code,omitempty"`
			Message             *string                     `json:"message,omitempty"`
		}{
			Code:    &[]string{regMsg.RegistryPrefix + "." + regMsg.RegistryVersion + ".GeneralError"}[0],
			Message: &messageStr,
			MessageExtendedInfo: &[]generated.MessageMessage{
				{
					MessageId:  &messageID,
					Message:    &messageStr,
					Severity:   &severity,
					Resolution: &resolution,
				},
			},
		},
	}

	return errorResponse, nil
}

// ConflictError returns a Redfish-compliant 409 error
func ConflictError(c *gin.Context, _, message string) {
	sendRedfishError(c, "Conflict", message)
}

// SessionConflictError returns a Redfish-compliant 409 error for duplicate sessions
func SessionConflictError(c *gin.Context) {
	sendRedfishError(c, "SessionConflict", "")
}

// PowerStateConflictError returns a Redfish-compliant 409 error for power state conflicts
func PowerStateConflictError(c *gin.Context, _ string) {
	sendRedfishError(c, "PowerStateConflict", "")
}

// MethodNotAllowedError returns a Redfish-compliant 405 error
func MethodNotAllowedError(c *gin.Context) {
	sendRedfishError(c, "MethodNotAllowed", "")
}

// UnauthorizedError returns a Redfish-compliant 401 error
func UnauthorizedError(c *gin.Context) {
	sendRedfishError(c, "Unauthorized", "")
}

// BadRequestError returns a Redfish-compliant 400 error
func BadRequestError(c *gin.Context, customMessage string) {
	sendRedfishError(c, "BadRequest", customMessage)
}

// ForbiddenError returns a Redfish-compliant 403 error for insufficient privileges
func ForbiddenError(c *gin.Context) {
	sendRedfishError(c, "Forbidden", "")
}

// ServiceUnavailableError returns a Redfish-compliant 503 error
func ServiceUnavailableError(c *gin.Context, retryAfterSeconds int) {
	sendRedfishError(c, "ServiceUnavailable", "", retryAfterSeconds, fmt.Sprintf("%d", retryAfterSeconds))
}

// NotFoundError returns a Redfish-compliant 404 error
func NotFoundError(c *gin.Context, resource string, id ...string) {
	identifier := resource
	if len(id) > 0 {
		identifier = id[0]
	}

	sendRedfishError(c, "NotFound", "", resource, identifier)
}

// InternalServerError returns a Redfish-compliant 500 error
func InternalServerError(c *gin.Context, err error) {
	SetRedfishHeaders(c)

	errorResponse, regErr := createErrorResponse("Base", "InternalError")
	if regErr != nil {
		// Ultimate fallback - if even the registry lookup fails, return a minimal error
		errorMessage := msgInternalServerError
		errMsg := err.Error()
		c.JSON(http.StatusInternalServerError, generated.RedfishError{
			Error: struct {
				MessageExtendedInfo *[]generated.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
				Code                *string                     `json:"code,omitempty"`
				Message             *string                     `json:"message,omitempty"`
			}{
				Code:    &[]string{"Base.1.22.0.InternalError"}[0],
				Message: &errorMessage,
				MessageExtendedInfo: &[]generated.MessageMessage{
					{
						MessageId: &[]string{"Base.1.22.0.InternalError"}[0],
						Message:   &errMsg,
						Severity:  &[]string{string(generated.Critical)}[0],
					},
				},
			},
		})

		return
	}

	// Override message with actual error
	errMsg := err.Error()
	errorMessage := msgInternalServerError
	errorResponse.Error.Message = &errorMessage
	(*errorResponse.Error.MessageExtendedInfo)[0].Message = &errMsg

	c.JSON(http.StatusInternalServerError, errorResponse)
}

// MalformedJSONError returns a Redfish-compliant 400 error for malformed JSON
func MalformedJSONError(c *gin.Context) {
	sendRedfishError(c, "MalformedJSON", "")
}

// PropertyMissingError returns a Redfish-compliant 400 error for missing required property
func PropertyMissingError(c *gin.Context, propertyName string) {
	sendRedfishError(c, "PropertyMissing", "", propertyName)
}

// PropertyValueNotInListError returns a Redfish-compliant 400 error for invalid property value
func PropertyValueNotInListError(c *gin.Context, propertyName string) {
	sendRedfishError(c, "PropertyValueNotInList", "", "invalid", propertyName)
}
