package consoleerrors

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

// ErrorResponse is the standard JSON payload returned to clients for errors.
type ErrorResponse struct {
	Code    string      `json:"code" example:"ERR_INVALID_INPUT"`
	Message string      `json:"message" example:"validation failed"`
	Details interface{} `json:"details,omitempty"`
	TraceID string      `json:"traceId,omitempty"`
}

// HTTPError represents an application error that can map to an HTTP status code.
type HTTPError struct {
	Status  int
	Code    string
	Message string
	Details interface{}
	Err     error
}

func (e *HTTPError) Error() string { // implement error
	if e == nil {
		return ""
	}

	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}

	return e.Message
}

// New creates a new HTTPError with a generated trace id on response.
func NewHTTPError(status int, code, message string, details interface{}, cause error) *HTTPError {
	return &HTTPError{
		Status:  status,
		Code:    code,
		Message: message,
		Details: details,
		Err:     cause,
	}
}

// ToResponse returns an ErrorResponse populated from HTTPError and trace id.
func (e *HTTPError) ToResponse(traceID string) ErrorResponse {
	return ErrorResponse{
		Code:    e.Code,
		Message: e.Message,
		Details: e.Details,
		TraceID: traceID,
	}
}

// TraceID returns a new trace id.
func TraceID() string {
	return uuid.New().String()
}

// MapGeneric maps an unknown error to a generic 500 error.
func MapGeneric(err error) *HTTPError {
	return NewHTTPError(http.StatusInternalServerError, "ERR_INTERNAL", "internal server error", nil, err)
}

// FriendlyError is an interface for errors that provide a user-friendly message.
type FriendlyError interface {
	error
	FriendlyMessage() string
}

// DomainErrorMapper maps domain-specific errors to HTTPError.
type DomainErrorMapper struct {
	mappers []func(error) *HTTPError
}

// NewDomainErrorMapper creates a new DomainErrorMapper with default mappers.
func NewDomainErrorMapper() *DomainErrorMapper {
	return &DomainErrorMapper{
		mappers: make([]func(error) *HTTPError, 0),
	}
}

// Register adds a custom error mapper function.
func (m *DomainErrorMapper) Register(mapper func(error) *HTTPError) {
	m.mappers = append(m.mappers, mapper)
}

// Map converts a domain error to an HTTPError.
// It tries registered mappers first, then falls back to generic mapping.
func (m *DomainErrorMapper) Map(err error) *HTTPError {
	if err == nil {
		return nil
	}

	// If already an HTTPError, return as-is
	if httpErr, ok := err.(*HTTPError); ok {
		return httpErr
	}

	// Try registered mappers
	for _, mapper := range m.mappers {
		if httpErr := mapper(err); httpErr != nil {
			return httpErr
		}
	}

	// Fallback to generic error
	return MapGeneric(err)
}
