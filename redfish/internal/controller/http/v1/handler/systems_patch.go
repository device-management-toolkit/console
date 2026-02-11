package v1

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	"github.com/device-management-toolkit/console/redfish/internal/usecase"
)

// PatchRedfishV1SystemsComputerSystemId handles PATCH requests to modify a ComputerSystem resource.
// This endpoint supports updating boot settings and other system properties.
//
//revive:disable-next-line var-naming. Codegen is using openapi spec for generation which required Id to be Redfish compliant.
func (s *RedfishServer) PatchRedfishV1SystemsComputerSystemId(c *gin.Context, computerSystemID string) {
	ctx := c.Request.Context()

	var req generated.ComputerSystemComputerSystem

	// Validate system ID
	if err := validateSystemID(computerSystemID); err != nil {
		BadRequestError(c, fmt.Sprintf("Invalid system ID: %s", err.Error()))

		return
	}

	// Parse request body
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestError(c, fmt.Sprintf("Invalid request body: %s", err.Error()))

		return
	}

	// Update boot settings if provided
	if req.Boot != nil {
		if err := s.ComputerSystemUC.UpdateBootSettings(ctx, computerSystemID, req.Boot); err != nil {
			s.handlePatchSystemError(c, err, computerSystemID)

			return
		}
	}

	// Return updated system
	updatedSystem, err := s.ComputerSystemUC.GetComputerSystem(ctx, computerSystemID)
	if err != nil {
		s.handleGetSystemError(c, err, computerSystemID)

		return
	}

	c.JSON(http.StatusOK, updatedSystem)
}

// handlePatchSystemError handles errors from PATCH operations on ComputerSystem.
func (s *RedfishServer) handlePatchSystemError(c *gin.Context, err error, systemID string) {
	switch {
	case errors.Is(err, usecase.ErrSystemNotFound):
		NotFoundError(c, "System", systemID)
	case errors.Is(err, usecase.ErrInvalidBootSettings),
		errors.Is(err, usecase.ErrInvalidBootTarget),
		errors.Is(err, usecase.ErrInvalidBootEnabled):
		BadRequestError(c, fmt.Sprintf("Invalid boot settings: %s", err.Error()))
	default:
		if s.Logger != nil {
			s.Logger.Error("Failed to update computer system",
				"systemID", systemID,
				"error", err)
		}

		InternalServerError(c, err)
	}
}
