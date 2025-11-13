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
	Error string `json:"error" example:"message"`
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
		c.AbortWithStatusJSON(http.StatusNotImplemented, response{notSupportedErr.Console.FriendlyMessage()})
	case errors.As(err, &certExpErr):
		c.AbortWithStatusJSON(http.StatusBadRequest, response{certExpErr.Console.FriendlyMessage()})
	case errors.As(err, &certPasswordErr):
		c.AbortWithStatusJSON(http.StatusBadRequest, response{certPasswordErr.Console.FriendlyMessage()})
	default:
		c.AbortWithStatusJSON(http.StatusInternalServerError, response{"general error"})
	}
}

func netErrorHandle(c *gin.Context, netErr net.Error) {
	c.AbortWithStatusJSON(http.StatusGatewayTimeout, response{netErr.Error()})
}

func notValidErrorHandle(c *gin.Context, err dto.NotValidError) {
	c.AbortWithStatusJSON(http.StatusBadRequest, response{err.Console.FriendlyMessage()})
}

func validatorErrorHandle(c *gin.Context, err validator.ValidationErrors) {
	c.AbortWithStatusJSON(http.StatusBadRequest, response{err.Error()})
}

func notFoundErrorHandle(c *gin.Context, err sqldb.NotFoundError) {
	message := "Error not found"
	if err.Console.FriendlyMessage() != "" {
		message = err.Console.FriendlyMessage()
	}

	c.AbortWithStatusJSON(http.StatusNotFound, response{message})
}

func dbErrorHandle(c *gin.Context, err sqldb.DatabaseError) {
	var notUniqueErr sqldb.NotUniqueError

	var foreignKeyViolationErr sqldb.ForeignKeyViolationError

	if errors.As(err.Console.OriginalError, &notUniqueErr) {
		notUniqueErrorHandle(c, notUniqueErr)

		return
	}

	if errors.As(err.Console.OriginalError, &foreignKeyViolationErr) {
		c.AbortWithStatusJSON(http.StatusBadRequest, response{foreignKeyViolationErr.Console.FriendlyMessage()})

		return
	}

	c.AbortWithStatusJSON(http.StatusBadRequest, response{err.Console.FriendlyMessage()})
}

func amtErrorHandle(c *gin.Context, err devices.AMTError) {
	if strings.Contains(err.Console.Error(), "400 Bad Request") {
		c.AbortWithStatusJSON(http.StatusBadRequest, response{err.Console.FriendlyMessage()})
	} else {
		c.AbortWithStatusJSON(http.StatusInternalServerError, response{err.Console.FriendlyMessage()})
	}
}

func notUniqueErrorHandle(c *gin.Context, err sqldb.NotUniqueError) {
	c.AbortWithStatusJSON(http.StatusBadRequest, response{err.Console.FriendlyMessage()})
}
