package consoleerrors

import (
	"errors"
	"net"
	"net/http"
	"strings"
)

// ErrorCode constants for standardized error codes.
const (
	ErrCodeNotFound           = "ERR_NOT_FOUND"
	ErrCodeValidation         = "ERR_VALIDATION"
	ErrCodeNotUnique          = "ERR_NOT_UNIQUE"
	ErrCodeForeignKey         = "ERR_FOREIGN_KEY_VIOLATION"
	ErrCodeDatabase           = "ERR_DATABASE"
	ErrCodeAMT                = "ERR_AMT"
	ErrCodeNotSupported       = "ERR_NOT_SUPPORTED"
	ErrCodeCertExpired        = "ERR_CERT_EXPIRED"
	ErrCodeCertPassword       = "ERR_CERT_PASSWORD"
	ErrCodeTimeout            = "ERR_TIMEOUT"
	ErrCodeInternal           = "ERR_INTERNAL"
)

// ConsoleError is an interface that all domain errors implement.
type ConsoleError interface {
	error
	GetConsole() InternalError
}

// MapNetError maps network errors to HTTPError.
func MapNetError(err error) *HTTPError {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return NewHTTPError(
			http.StatusGatewayTimeout,
			ErrCodeTimeout,
			netErr.Error(),
			nil,
			err,
		)
	}
	return nil
}

// MapNotFoundError maps not found errors to HTTPError.
func MapNotFoundError(err error, checker func(error) (string, bool)) *HTTPError {
	if msg, ok := checker(err); ok {
		message := "resource not found"
		if msg != "" {
			message = msg
		}
		return NewHTTPError(
			http.StatusNotFound,
			ErrCodeNotFound,
			message,
			nil,
			err,
		)
	}
	return nil
}

// MapValidationError maps validation errors to HTTPError.
func MapValidationError(err error, checker func(error) (string, bool)) *HTTPError {
	if msg, ok := checker(err); ok {
		return NewHTTPError(
			http.StatusBadRequest,
			ErrCodeValidation,
			msg,
			nil,
			err,
		)
	}
	return nil
}

// MapNotUniqueError maps unique constraint errors to HTTPError.
func MapNotUniqueError(err error, checker func(error) (string, bool)) *HTTPError {
	if msg, ok := checker(err); ok {
		return NewHTTPError(
			http.StatusBadRequest,
			ErrCodeNotUnique,
			msg,
			nil,
			err,
		)
	}
	return nil
}

// MapForeignKeyError maps foreign key violation errors to HTTPError.
func MapForeignKeyError(err error, checker func(error) (string, bool)) *HTTPError {
	if msg, ok := checker(err); ok {
		return NewHTTPError(
			http.StatusBadRequest,
			ErrCodeForeignKey,
			msg,
			nil,
			err,
		)
	}
	return nil
}

// MapDatabaseError maps database errors to HTTPError.
func MapDatabaseError(err error, checker func(error) (string, bool)) *HTTPError {
	if msg, ok := checker(err); ok {
		return NewHTTPError(
			http.StatusBadRequest,
			ErrCodeDatabase,
			msg,
			nil,
			err,
		)
	}
	return nil
}

// MapAMTError maps AMT errors to HTTPError.
func MapAMTError(err error, checker func(error) (string, bool)) *HTTPError {
	if msg, ok := checker(err); ok {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "400 Bad Request") {
			status = http.StatusBadRequest
		}
		return NewHTTPError(
			status,
			ErrCodeAMT,
			msg,
			nil,
			err,
		)
	}
	return nil
}

// MapNotSupportedError maps not supported errors to HTTPError.
func MapNotSupportedError(err error, checker func(error) (string, bool)) *HTTPError {
	if msg, ok := checker(err); ok {
		return NewHTTPError(
			http.StatusNotImplemented,
			ErrCodeNotSupported,
			msg,
			nil,
			err,
		)
	}
	return nil
}

// MapCertExpirationError maps certificate expiration errors to HTTPError.
func MapCertExpirationError(err error, checker func(error) (string, bool)) *HTTPError {
	if msg, ok := checker(err); ok {
		return NewHTTPError(
			http.StatusBadRequest,
			ErrCodeCertExpired,
			msg,
			nil,
			err,
		)
	}
	return nil
}

// MapCertPasswordError maps certificate password errors to HTTPError.
func MapCertPasswordError(err error, checker func(error) (string, bool)) *HTTPError {
	if msg, ok := checker(err); ok {
		return NewHTTPError(
			http.StatusBadRequest,
			ErrCodeCertPassword,
			msg,
			nil,
			err,
		)
	}
	return nil
}
