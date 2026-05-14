// Package v1 provides Redfish v1 API handlers for system actions.
package v1

import (
	"errors"
	"fmt"
	"net/http"
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
// This is a stub implementation.
//
//nolint:revive // Method name is generated from OpenAPI spec and cannot be changed
func (s *RedfishServer) PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemCancelKVMConsent(c *gin.Context, computerSystemID string) {
	if err := validateSystemID(computerSystemID); err != nil {
		BadRequestError(c, fmt.Sprintf("Invalid system ID: %s", err.Error()))

		return
	}

	MethodNotAllowedError(c)
}

// PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemGenerateRedirectionToken handles generating a redirection token for a computer system.
// This is a stub implementation.
//
//nolint:revive // Method name is generated from OpenAPI spec and cannot be changed
func (s *RedfishServer) PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemGenerateRedirectionToken(c *gin.Context, computerSystemID string) {
	if err := validateSystemID(computerSystemID); err != nil {
		BadRequestError(c, fmt.Sprintf("Invalid system ID: %s", err.Error()))

		return
	}

	MethodNotAllowedError(c)
}

// PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemRequestKVMConsent handles requesting KVM consent for a computer system.
// This is a stub implementation.
//
//nolint:revive // Method name is generated from OpenAPI spec and cannot be changed
func (s *RedfishServer) PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemRequestKVMConsent(c *gin.Context, computerSystemID string) {
	if err := validateSystemID(computerSystemID); err != nil {
		BadRequestError(c, fmt.Sprintf("Invalid system ID: %s", err.Error()))

		return
	}

	MethodNotAllowedError(c)
}

// PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemSubmitKVMConsentCode handles submitting a KVM consent code for a computer system.
// This is a stub implementation.
//
//nolint:revive // Method name is generated from OpenAPI spec and cannot be changed
func (s *RedfishServer) PostRedfishV1SystemsComputerSystemIdActionsOemIntelComputerSystemSubmitKVMConsentCode(c *gin.Context, computerSystemID string) {
	if err := validateSystemID(computerSystemID); err != nil {
		BadRequestError(c, fmt.Sprintf("Invalid system ID: %s", err.Error()))

		return
	}

	MethodNotAllowedError(c)
}
