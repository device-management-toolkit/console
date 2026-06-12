// Package v1 provides Redfish v1 API handlers for system actions.
package v1

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/labstack/gommon/log"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	"github.com/device-management-toolkit/console/redfish/internal/usecase"
)

const (
	// JSON field names
	odataContextKey = "@odata.context"
	odataIDKey      = "@odata.id"
	odataTypeKey    = "@odata.type"
	idKey           = "Id"
	nameKey         = "Name"
)

var (
	sixDigitConsentCodeRe = regexp.MustCompile(`^\d{6}$`)
	amtBadRequestRe       = regexp.MustCompile(`(?i)\b400\s+bad\s+request\b`)
)

// PostRedfishV1SystemsComputerSystemIdActionsComputerSystemReset handles reset action for a computer system.
// Validates system ID and reset type before executing power state change.
//
//nolint:revive // Method name is generated from OpenAPI spec and cannot be changed
func (s *RedfishServer) PostRedfishV1SystemsComputerSystemIdActionsComputerSystemReset(c *gin.Context, computerSystemID string) {
	// Validate system ID to prevent injection attacks
	if err := validateSystemID(computerSystemID); err != nil {
		BadRequestError(c, fmt.Sprintf("Invalid system ID: %s", err.Error()))

		return
	}

	var req generated.PostRedfishV1SystemsComputerSystemIdActionsComputerSystemResetJSONRequestBody

	if err := c.ShouldBindJSON(&req); err != nil {
		MalformedJSONError(c)

		return
	}

	if req.ResetType == nil || *req.ResetType == "" {
		PropertyMissingError(c, "ResetType")

		return
	}

	log.Infof("Received reset request for ComputerSystem %s with ResetType %s", computerSystemID, *req.ResetType)

	if err := s.ComputerSystemUC.EnsureSystemExists(c.Request.Context(), computerSystemID); err != nil {
		switch {
		case errors.Is(err, usecase.ErrSystemNotFound):
			NotFoundError(c, "System", computerSystemID)
		default:
			InternalServerError(c, err)
		}

		return
	}

	if err := s.ComputerSystemUC.SetPowerState(c.Request.Context(), computerSystemID, *req.ResetType); err != nil {
		switch {
		case errors.Is(err, usecase.ErrSystemNotFound):
			NotFoundError(c, "System", computerSystemID)
		case errors.Is(err, usecase.ErrInvalidResetType):
			BadRequestError(c, fmt.Sprintf("Invalid reset type: %s", string(*req.ResetType)))
		case errors.Is(err, usecase.ErrPowerStateConflict):
			PowerStateConflictError(c, string(*req.ResetType))
		case errors.Is(err, usecase.ErrUnsupportedPowerState):
			BadRequestError(c, fmt.Sprintf("Unsupported power state: %s", string(*req.ResetType)))
		default:
			InternalServerError(c, err)
		}

		return
	}

	// Generate dynamic Task response
	taskID := fmt.Sprintf("%d", time.Now().UnixNano())
	now := time.Now().UTC().Format(time.RFC3339)

	// Get success message from registry
	successMsg, err := registryMgr.LookupMessage("Base", "Success")
	if err != nil {
		// Fallback if registry lookup fails
		InternalServerError(c, err)

		return
	}

	task := map[string]interface{}{
		odataContextKey: odataContextTask,
		odataIDKey:      taskServiceTasks + taskID,
		odataTypeKey:    odataTypeTask,
		"EndTime":       now,
		idKey:           taskID,
		"Messages": []map[string]interface{}{
			{
				"Message":   successMsg.Message,
				"MessageId": msgIDBaseSuccess,
				"Severity":  string(generated.OK),
			},
		},
		nameKey:      taskName,
		"StartTime":  now,
		"TaskState":  taskStateCompleted,
		"TaskStatus": string(generated.OK),
	}
	c.Header(headerLocation, taskServiceTasks+taskID)
	c.JSON(http.StatusAccepted, task)
}

// PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemCancelKVMConsent handles canceling KVM consent for a computer system.
//
//nolint:revive // Method name is generated from OpenAPI spec and cannot be changed
func (s *RedfishServer) PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemCancelKVMConsent(c *gin.Context, computerSystemID string) {
	if err := validateSystemID(computerSystemID); err != nil {
		BadRequestError(c, fmt.Sprintf("Invalid system ID: %s", err.Error()))

		return
	}

	var req generated.PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemCancelKVMConsentJSONRequestBody

	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		MalformedJSONError(c)

		return
	}

	if err := s.ComputerSystemUC.EnsureSystemExists(c.Request.Context(), computerSystemID); err != nil {
		switch {
		case errors.Is(err, usecase.ErrSystemNotFound):
			NotFoundError(c, "System", computerSystemID)
		default:
			InternalServerError(c, err)
		}

		return
	}

	if err := s.ComputerSystemUC.CancelKVMConsent(c.Request.Context(), computerSystemID); err != nil {
		var consentErr *usecase.ConsentFailedError

		switch {
		case errors.Is(err, usecase.ErrSystemNotFound):
			NotFoundError(c, "System", computerSystemID)
		case errors.As(err, &consentErr):
			BadRequestError(c, consentErr.Error())
		case isAMTBadRequestError(err):
			BadRequestError(c, err.Error())
		default:
			InternalServerError(c, err)
		}

		return
	}

	sendActionSuccessResponse(c)
}

// PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemCancelSolConsent handles canceling SOL consent for a computer system.
//
//nolint:revive // Method name is generated from OpenAPI spec and cannot be changed
func (s *RedfishServer) PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemCancelSolConsent(c *gin.Context, computerSystemID string) {
	if err := validateSystemID(computerSystemID); err != nil {
		BadRequestError(c, fmt.Sprintf("Invalid system ID: %s", err.Error()))

		return
	}

	var req generated.PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemCancelSolConsentJSONRequestBody

	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		MalformedJSONError(c)

		return
	}

	if err := s.ComputerSystemUC.EnsureSystemExists(c.Request.Context(), computerSystemID); err != nil {
		switch {
		case errors.Is(err, usecase.ErrSystemNotFound):
			NotFoundError(c, "System", computerSystemID)
		default:
			InternalServerError(c, err)
		}

		return
	}

	if err := s.ComputerSystemUC.CancelSolConsent(c.Request.Context(), computerSystemID); err != nil {
		var consentErr *usecase.ConsentFailedError

		switch {
		case errors.Is(err, usecase.ErrSystemNotFound):
			NotFoundError(c, "System", computerSystemID)
		case errors.As(err, &consentErr):
			BadRequestError(c, consentErr.Error())
		case isAMTBadRequestError(err):
			BadRequestError(c, err.Error())
		default:
			InternalServerError(c, err)
		}

		return
	}

	sendActionSuccessResponse(c)
}

// PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemGenerateRedirectionToken handles generating a redirection token for a computer system.
//
//nolint:revive // Method name is generated from OpenAPI spec and cannot be changed
func (s *RedfishServer) PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemGenerateRedirectionToken(c *gin.Context, computerSystemID string) {
	if err := validateSystemID(computerSystemID); err != nil {
		BadRequestError(c, fmt.Sprintf("Invalid system ID: %s", err.Error()))

		return
	}

	var req generated.PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemGenerateRedirectionTokenJSONRequestBody

	if err := c.ShouldBindJSON(&req); err != nil {
		MalformedJSONError(c)

		return
	}

	response, err := s.ComputerSystemUC.GenerateRedirectionToken(c.Request.Context(), computerSystemID)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrSystemNotFound):
			NotFoundError(c, "System", computerSystemID)
		default:
			InternalServerError(c, err)
		}

		return
	}

	SetRedfishHeaders(c)
	c.JSON(http.StatusOK, response)
}

// PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemRequestKVMConsent handles requesting KVM consent for a computer system.
//
//nolint:revive // Method name is generated from OpenAPI spec and cannot be changed
func (s *RedfishServer) PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemRequestKVMConsent(c *gin.Context, computerSystemID string) {
	if err := validateSystemID(computerSystemID); err != nil {
		BadRequestError(c, fmt.Sprintf("Invalid system ID: %s", err.Error()))

		return
	}

	var req generated.PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemRequestKVMConsentJSONRequestBody

	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		MalformedJSONError(c)

		return
	}

	if err := s.ComputerSystemUC.EnsureSystemExists(c.Request.Context(), computerSystemID); err != nil {
		switch {
		case errors.Is(err, usecase.ErrSystemNotFound):
			NotFoundError(c, "System", computerSystemID)
		default:
			InternalServerError(c, err)
		}

		return
	}

	if err := s.ComputerSystemUC.RequestKVMConsent(c.Request.Context(), computerSystemID); err != nil {
		var consentErr *usecase.ConsentFailedError

		switch {
		case errors.Is(err, usecase.ErrSystemNotFound):
			NotFoundError(c, "System", computerSystemID)
		case errors.Is(err, usecase.ErrKVMConsentNotRequiredInACM):
			BadRequestError(c, err.Error())
		case errors.As(err, &consentErr):
			BadRequestError(c, consentErr.Error())
		case isAMTBadRequestError(err):
			BadRequestError(c, err.Error())
		default:
			InternalServerError(c, err)
		}

		return
	}

	sendActionSuccessResponse(c)
}

// PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemRequestSolConsent handles requesting SOL consent for a computer system.
//
//nolint:revive // Method name is generated from OpenAPI spec and cannot be changed
func (s *RedfishServer) PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemRequestSolConsent(c *gin.Context, computerSystemID string) {
	if err := validateSystemID(computerSystemID); err != nil {
		BadRequestError(c, fmt.Sprintf("Invalid system ID: %s", err.Error()))

		return
	}

	var req generated.PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemRequestSolConsentJSONRequestBody

	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		MalformedJSONError(c)

		return
	}

	if err := s.ComputerSystemUC.EnsureSystemExists(c.Request.Context(), computerSystemID); err != nil {
		switch {
		case errors.Is(err, usecase.ErrSystemNotFound):
			NotFoundError(c, "System", computerSystemID)
		default:
			InternalServerError(c, err)
		}

		return
	}

	if err := s.ComputerSystemUC.RequestSolConsent(c.Request.Context(), computerSystemID); err != nil {
		var consentErr *usecase.ConsentFailedError

		switch {
		case errors.Is(err, usecase.ErrSystemNotFound):
			NotFoundError(c, "System", computerSystemID)
		case errors.Is(err, usecase.ErrSOLConsentNotRequiredInACM):
			BadRequestError(c, err.Error())
		case errors.As(err, &consentErr):
			BadRequestError(c, consentErr.Error())
		case isAMTBadRequestError(err):
			BadRequestError(c, err.Error())
		default:
			InternalServerError(c, err)
		}

		return
	}

	sendActionSuccessResponse(c)
}

// PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemSubmitKVMConsentCode handles submitting a KVM consent code for a computer system.
//
//nolint:revive // Method name is generated from OpenAPI spec and cannot be changed
func (s *RedfishServer) PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemSubmitKVMConsentCode(c *gin.Context, computerSystemID string) {
	if err := validateSystemID(computerSystemID); err != nil {
		BadRequestError(c, fmt.Sprintf("Invalid system ID: %s", err.Error()))

		return
	}

	var req generated.PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemSubmitKVMConsentCodeJSONRequestBody

	if err := c.ShouldBindJSON(&req); err != nil {
		MalformedJSONError(c)

		return
	}

	consentCode := strings.TrimSpace(req.ConsentCode)
	if consentCode == "" {
		PropertyMissingError(c, "ConsentCode")

		return
	}

	if !sixDigitConsentCodeRe.MatchString(consentCode) {
		BadRequestError(c, "Invalid ConsentCode: must be a six-digit numeric value")

		return
	}

	if err := s.ComputerSystemUC.EnsureSystemExists(c.Request.Context(), computerSystemID); err != nil {
		switch {
		case errors.Is(err, usecase.ErrSystemNotFound):
			NotFoundError(c, "System", computerSystemID)
		default:
			InternalServerError(c, err)
		}

		return
	}

	if err := s.ComputerSystemUC.SubmitKVMConsentCode(c.Request.Context(), computerSystemID, consentCode); err != nil {
		var consentErr *usecase.ConsentFailedError

		switch {
		case errors.Is(err, usecase.ErrSystemNotFound):
			NotFoundError(c, "System", computerSystemID)
		case errors.As(err, &consentErr):
			BadRequestError(c, consentErr.Error())
		case isAMTBadRequestError(err):
			BadRequestError(c, err.Error())
		default:
			InternalServerError(c, err)
		}

		return
	}

	sendActionSuccessResponse(c)
}

// PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemSubmitSolConsentCode handles submitting a SOL consent code for a computer system.
//
//nolint:revive // Method name is generated from OpenAPI spec and cannot be changed
func (s *RedfishServer) PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemSubmitSolConsentCode(c *gin.Context, computerSystemID string) {
	if err := validateSystemID(computerSystemID); err != nil {
		BadRequestError(c, fmt.Sprintf("Invalid system ID: %s", err.Error()))

		return
	}

	var req generated.PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemSubmitSolConsentCodeJSONRequestBody

	if err := c.ShouldBindJSON(&req); err != nil {
		MalformedJSONError(c)

		return
	}

	consentCode := strings.TrimSpace(req.ConsentCode)
	if consentCode == "" {
		PropertyMissingError(c, "ConsentCode")

		return
	}

	if !sixDigitConsentCodeRe.MatchString(consentCode) {
		BadRequestError(c, "Invalid ConsentCode: must be a six-digit numeric value")

		return
	}

	if err := s.ComputerSystemUC.EnsureSystemExists(c.Request.Context(), computerSystemID); err != nil {
		switch {
		case errors.Is(err, usecase.ErrSystemNotFound):
			NotFoundError(c, "System", computerSystemID)
		default:
			InternalServerError(c, err)
		}

		return
	}

	if err := s.ComputerSystemUC.SubmitSolConsentCode(c.Request.Context(), computerSystemID, consentCode); err != nil {
		var consentErr *usecase.ConsentFailedError

		switch {
		case errors.Is(err, usecase.ErrSystemNotFound):
			NotFoundError(c, "System", computerSystemID)
		case errors.As(err, &consentErr):
			BadRequestError(c, consentErr.Error())
		case isAMTBadRequestError(err):
			BadRequestError(c, err.Error())
		default:
			InternalServerError(c, err)
		}

		return
	}

	sendActionSuccessResponse(c)
}

func sendActionSuccessResponse(c *gin.Context) {
	sendActionSuccessResponseWithLookup(c, registryMgr.LookupMessage)
}

func sendActionSuccessResponseWithLookup(c *gin.Context, lookupFn func(string, string) (*RegistryMessage, error)) {
	SetRedfishHeaders(c)

	successMsg, err := lookupFn("Base", "Success")
	if err != nil {
		InternalServerError(c, err)

		return
	}

	messageID := successMsg.MessageID
	message := successMsg.Message
	severity := mapSeverityToResourceHealth(successMsg.Severity)
	resolution := successMsg.Resolution

	c.JSON(http.StatusOK, generated.RedfishError{
		Error: struct {
			MessageExtendedInfo *[]generated.MessageMessage `json:"@Message.ExtendedInfo,omitempty"`
			Code                *string                     `json:"code,omitempty"`
			Message             *string                     `json:"message,omitempty"`
		}{
			Code:    &messageID,
			Message: &message,
			MessageExtendedInfo: &[]generated.MessageMessage{
				{
					MessageId:  &messageID,
					Message:    &message,
					Severity:   &severity,
					Resolution: &resolution,
				},
			},
		},
	})
}

func isAMTBadRequestError(err error) bool {
	if err == nil {
		return false
	}

	return amtBadRequestRe.MatchString(err.Error())
}
