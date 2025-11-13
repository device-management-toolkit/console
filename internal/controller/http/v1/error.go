package v1

import (
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/internal/usecase/domains"
	"github.com/device-management-toolkit/console/internal/usecase/sqldb"
)

type response struct {
	Error   string `json:"error,omitempty" example:"message"`
	Message string `json:"message,omitempty" example:"message"`
}

func ErrorResponse(c *gin.Context, err error) {
	var (
		validatorErr    validator.ValidationErrors
		nfErr           sqldb.NotFoundError
		notValidErr     dto.NotValidError
		dbErr           sqldb.DatabaseError
		NotUniqueErr    sqldb.NotUniqueError
		amtErr          devices.AMTError
		notSupportedErr devices.NotSupportedError
		certExpErr      domains.CertExpirationError
		certPasswordErr domains.CertPasswordError
		netErr          net.Error
	)

	switch {
	case errors.As(err, &netErr):
		netErrorHandle(c, netErr)
	case errors.As(err, &notValidErr):
		notValidErrorHandle(c, notValidErr)
	case errors.As(err, &validatorErr):
		validatorErrorHandle(c, validatorErr)
	case errors.As(err, &nfErr):
		notFoundErrorHandle(c, nfErr)
	case errors.As(err, &NotUniqueErr):
		notUniqueErrorHandle(c, NotUniqueErr)
	case errors.As(err, &dbErr):
		dbErrorHandle(c, dbErr)
	case errors.As(err, &amtErr):
		amtErrorHandle(c, amtErr)
	case errors.As(err, &notSupportedErr):
		msg := notSupportedErr.Console.FriendlyMessage()
		c.AbortWithStatusJSON(http.StatusNotImplemented, response{Error: msg, Message: msg})
	case errors.As(err, &certExpErr):
		msg := certExpErr.Console.FriendlyMessage()
		c.AbortWithStatusJSON(http.StatusBadRequest, response{Error: msg, Message: msg})
	case errors.As(err, &certPasswordErr):
		msg := certPasswordErr.Console.FriendlyMessage()
		c.AbortWithStatusJSON(http.StatusBadRequest, response{Error: msg, Message: msg})
	default:
		c.AbortWithStatusJSON(http.StatusInternalServerError, response{Error: "general error", Message: "general error"})
	}
}

func netErrorHandle(c *gin.Context, netErr net.Error) {
	msg := netErr.Error()
	c.AbortWithStatusJSON(http.StatusGatewayTimeout, response{Error: msg, Message: msg})
}

func notValidErrorHandle(c *gin.Context, err dto.NotValidError) {
	msg := err.Console.FriendlyMessage()
	c.AbortWithStatusJSON(http.StatusBadRequest, response{Error: msg, Message: msg})
}

func validatorErrorHandle(c *gin.Context, err validator.ValidationErrors) {
	msg := err.Error()
	c.AbortWithStatusJSON(http.StatusBadRequest, response{Error: msg, Message: msg})
}

func notFoundErrorHandle(c *gin.Context, err sqldb.NotFoundError) {
	message := "Error not found"
	if err.Console.FriendlyMessage() != "" {
		message = err.Console.FriendlyMessage()
	}

	c.AbortWithStatusJSON(http.StatusNotFound, response{Error: message, Message: message})
}

func dbErrorHandle(c *gin.Context, err sqldb.DatabaseError) {
	var notUniqueErr sqldb.NotUniqueError

	var foreignKeyViolationErr sqldb.ForeignKeyViolationError

	if errors.As(err.Console.OriginalError, &notUniqueErr) {
		notUniqueErrorHandle(c, notUniqueErr)

		return
	}

	if errors.As(err.Console.OriginalError, &foreignKeyViolationErr) {
		msg := foreignKeyViolationErr.Console.FriendlyMessage()
		c.AbortWithStatusJSON(http.StatusBadRequest, response{Error: msg, Message: msg})

		return
	}

	msg := err.Console.FriendlyMessage()
	c.AbortWithStatusJSON(http.StatusBadRequest, response{Error: msg, Message: msg})
}

func amtErrorHandle(c *gin.Context, err devices.AMTError) {
	msg := err.Console.FriendlyMessage()
	if strings.Contains(err.Console.Error(), "400 Bad Request") {
		c.AbortWithStatusJSON(http.StatusBadRequest, response{Error: msg, Message: msg})
	} else {
		c.AbortWithStatusJSON(http.StatusInternalServerError, response{Error: msg, Message: msg})
	}
}

func notUniqueErrorHandle(c *gin.Context, err sqldb.NotUniqueError) {
	msg := err.Console.FriendlyMessage()
	c.AbortWithStatusJSON(http.StatusBadRequest, response{Error: msg, Message: msg})
}
