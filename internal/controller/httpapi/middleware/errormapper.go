package middleware

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/internal/usecase/domains"
	"github.com/device-management-toolkit/console/internal/usecase/sqldb"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

// NewErrorMapper creates a DomainErrorMapper with all domain error mappers registered.
func NewErrorMapper() *consoleerrors.DomainErrorMapper {
	mapper := consoleerrors.NewDomainErrorMapper()

	// Register mappers in order of specificity
	mapper.Register(mapNetError)
	mapper.Register(mapValidatorError)
	mapper.Register(mapNotValidError)
	mapper.Register(mapNotFoundError)
	mapper.Register(mapNotUniqueError)
	mapper.Register(mapForeignKeyViolationError)
	mapper.Register(mapDatabaseError)
	mapper.Register(mapAMTError)
	mapper.Register(mapExplorerError)
	mapper.Register(mapNotSupportedError)
	mapper.Register(mapValidationError)
	mapper.Register(mapCertExpirationError)
	mapper.Register(mapCertPasswordError)

	return mapper
}

func mapNetError(err error) *consoleerrors.HTTPError {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return consoleerrors.NewHTTPError(
			http.StatusGatewayTimeout,
			consoleerrors.ErrCodeTimeout,
			netErr.Error(),
			nil,
			err,
		)
	}
	return nil
}

// ValidationErrorDetail represents a single field validation error.
type ValidationErrorDetail struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Value   string `json:"value,omitempty"`
	Message string `json:"message"`
}

func mapValidatorError(err error) *consoleerrors.HTTPError {
	var validatorErr validator.ValidationErrors
	if errors.As(err, &validatorErr) {
		details := make([]ValidationErrorDetail, 0, len(validatorErr))
		for _, fe := range validatorErr {
			details = append(details, ValidationErrorDetail{
				Field:   fe.Field(),
				Tag:     fe.Tag(),
				Value:   fmt.Sprintf("%v", fe.Value()),
				Message: fe.Error(),
			})
		}

		return consoleerrors.NewHTTPError(
			http.StatusBadRequest,
			consoleerrors.ErrCodeValidation,
			"validation failed",
			details,
			err,
		)
	}
	return nil
}

func mapNotValidError(err error) *consoleerrors.HTTPError {
	var notValidErr dto.NotValidError
	if errors.As(err, &notValidErr) {
		return consoleerrors.NewHTTPError(
			http.StatusBadRequest,
			consoleerrors.ErrCodeValidation,
			notValidErr.Console.FriendlyMessage(),
			nil,
			err,
		)
	}
	return nil
}

func mapNotFoundError(err error) *consoleerrors.HTTPError {
	var nfErr sqldb.NotFoundError
	if errors.As(err, &nfErr) {
		message := "resource not found"
		if nfErr.Console.FriendlyMessage() != "" {
			message = nfErr.Console.FriendlyMessage()
		}
		return consoleerrors.NewHTTPError(
			http.StatusNotFound,
			consoleerrors.ErrCodeNotFound,
			message,
			nil,
			err,
		)
	}
	return nil
}

func mapNotUniqueError(err error) *consoleerrors.HTTPError {
	var notUniqueErr sqldb.NotUniqueError
	if errors.As(err, &notUniqueErr) {
		return consoleerrors.NewHTTPError(
			http.StatusBadRequest,
			consoleerrors.ErrCodeNotUnique,
			notUniqueErr.Console.FriendlyMessage(),
			nil,
			err,
		)
	}
	return nil
}

func mapForeignKeyViolationError(err error) *consoleerrors.HTTPError {
	var fkErr sqldb.ForeignKeyViolationError
	if errors.As(err, &fkErr) {
		return consoleerrors.NewHTTPError(
			http.StatusBadRequest,
			consoleerrors.ErrCodeForeignKey,
			fkErr.Console.FriendlyMessage(),
			nil,
			err,
		)
	}
	return nil
}

func mapDatabaseError(err error) *consoleerrors.HTTPError {
	var dbErr sqldb.DatabaseError
	if errors.As(err, &dbErr) {
		// Check for nested errors first
		var notUniqueErr sqldb.NotUniqueError
		var fkErr sqldb.ForeignKeyViolationError

		if errors.As(dbErr.Console.OriginalError, &notUniqueErr) {
			return mapNotUniqueError(dbErr.Console.OriginalError)
		}
		if errors.As(dbErr.Console.OriginalError, &fkErr) {
			return mapForeignKeyViolationError(dbErr.Console.OriginalError)
		}

		return consoleerrors.NewHTTPError(
			http.StatusBadRequest,
			consoleerrors.ErrCodeDatabase,
			dbErr.Console.FriendlyMessage(),
			nil,
			err,
		)
	}
	return nil
}

func mapAMTError(err error) *consoleerrors.HTTPError {
	var amtErr devices.AMTError
	if errors.As(err, &amtErr) {
		status := http.StatusInternalServerError
		if strings.Contains(amtErr.Console.Error(), "400 Bad Request") {
			status = http.StatusBadRequest
		}
		return consoleerrors.NewHTTPError(
			status,
			consoleerrors.ErrCodeAMT,
			amtErr.Console.FriendlyMessage(),
			nil,
			err,
		)
	}
	return nil
}

func mapExplorerError(err error) *consoleerrors.HTTPError {
	var explorerErr devices.ExplorerError
	if errors.As(err, &explorerErr) {
		return consoleerrors.NewHTTPError(
			http.StatusInternalServerError,
			consoleerrors.ErrCodeAMT,
			explorerErr.Console.FriendlyMessage(),
			nil,
			err,
		)
	}
	return nil
}

func mapNotSupportedError(err error) *consoleerrors.HTTPError {
	var notSupportedErr devices.NotSupportedError
	if errors.As(err, &notSupportedErr) {
		return consoleerrors.NewHTTPError(
			http.StatusNotImplemented,
			consoleerrors.ErrCodeNotSupported,
			notSupportedErr.Console.FriendlyMessage(),
			nil,
			err,
		)
	}
	return nil
}

func mapValidationError(err error) *consoleerrors.HTTPError {
	var validationErr devices.ValidationError
	if errors.As(err, &validationErr) {
		return consoleerrors.NewHTTPError(
			http.StatusBadRequest,
			consoleerrors.ErrCodeValidation,
			validationErr.Console.FriendlyMessage(),
			nil,
			err,
		)
	}
	return nil
}

func mapCertExpirationError(err error) *consoleerrors.HTTPError {
	var certExpErr domains.CertExpirationError
	if errors.As(err, &certExpErr) {
		return consoleerrors.NewHTTPError(
			http.StatusBadRequest,
			consoleerrors.ErrCodeCertExpired,
			certExpErr.Console.FriendlyMessage(),
			nil,
			err,
		)
	}
	return nil
}

func mapCertPasswordError(err error) *consoleerrors.HTTPError {
	var certPwdErr domains.CertPasswordError
	if errors.As(err, &certPwdErr) {
		return consoleerrors.NewHTTPError(
			http.StatusBadRequest,
			consoleerrors.ErrCodeCertPassword,
			certPwdErr.Console.FriendlyMessage(),
			nil,
			err,
		)
	}
	return nil
}
