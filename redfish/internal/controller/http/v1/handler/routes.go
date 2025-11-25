// Package v1 provides Redfish v1 API route setup and configuration.
package v1

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/labstack/gommon/log"

	dmtconfig "github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/pkg/logger"
	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	"github.com/device-management-toolkit/console/redfish/internal/usecase"
)

const (
	// Task state constants from Redfish Task.v1_8_0 specification
	taskStateCompleted = "Completed"

	// Registry message IDs
	msgIDBaseSuccess      = "Base.1.22.0.Success"
	msgIDBaseGeneralError = "Base.1.22.0.GeneralError"

	// OData metadata constants - Systems Collection
	odataContextSystems        = "/redfish/v1/$metadata#ComputerSystemCollection.ComputerSystemCollection"
	odataIDSystems             = "/redfish/v1/Systems"
	odataTypeSystemsCollection = "#ComputerSystemCollection.ComputerSystemCollection"
	systemsCollectionName      = "Computer System Collection"
	systemsCollectionDesc      = "Collection of Computer Systems"

	// OData metadata constants - Task
	odataContextTask = "/redfish/v1/$metadata#Task.Task"
	odataTypeTask    = "#Task.v1_6_0.Task"
	taskName         = "System Reset Task"
	taskServiceTasks = "/redfish/v1/TaskService/Tasks/"

	// Systems path patterns
	systemsPath = "/redfish/v1/Systems/"
)

// RedfishServer implements the Redfish API handlers
// Add dependencies here if needed (e.g., usecase, presenter, etc.)
type RedfishServer struct {
	ComputerSystemUC *usecase.ComputerSystemUseCase
	Config           *dmtconfig.Config
	Logger           logger.Interface
}

// Ensure RedfishServer implements generated.ServerInterface
var _ generated.ServerInterface = (*RedfishServer)(nil)

// StringPtr creates a pointer to a string value.
func StringPtr(s string) *string {
	return &s
}

// IntPtr creates a pointer to an int value.
func IntPtr(i int) *int {
	return &i
}

// Int64Ptr creates a pointer to an int64 value.
func Int64Ptr(i int64) *int64 {
	return &i
}

// BoolPtr creates a pointer to a bool value.
func BoolPtr(b bool) *bool {
	return &b
}

// CreateDescription creates a Description from a string using ResourceDescription.
// If an error occurs during description creation, it logs the error and returns nil.
// This allows the calling code to continue with a nil description while ensuring
// the error is captured for debugging purposes.
func CreateDescription(desc string, lgr logger.Interface) *generated.ComputerSystemCollectionComputerSystemCollection_Description {
	description := &generated.ComputerSystemCollectionComputerSystemCollection_Description{}
	if err := description.FromResourceDescription(desc); err != nil {
		if lgr != nil {
			lgr.Error("Failed to create description from resource description: %v, input: %s", err, desc)
		}

		return nil
	}

	return description
}

// SystemTypePtr creates a pointer to a ComputerSystemSystemType value.
func SystemTypePtr(st generated.ComputerSystemSystemType) *generated.ComputerSystemSystemType {
	return &st
}

// GetRedfishV1Systems returns the computer systems collection
func (s *RedfishServer) GetRedfishV1Systems(c *gin.Context) {
	// Get all system IDs from the repository
	systemIDs, err := s.ComputerSystemUC.GetAll(c.Request.Context())
	if err != nil {
		InternalServerError(c, err)

		return
	}

	// Convert system IDs to members array
	members := make([]generated.OdataV4IdRef, 0, len(systemIDs))
	for _, systemID := range systemIDs {
		if systemID != "" {
			members = append(members, generated.OdataV4IdRef{
				OdataId: StringPtr(systemsPath + systemID),
			})
		}
	}

	collection := generated.ComputerSystemCollectionComputerSystemCollection{
		OdataContext:      StringPtr(odataContextSystems),
		OdataId:           StringPtr(odataIDSystems),
		OdataType:         StringPtr(odataTypeSystemsCollection),
		Name:              systemsCollectionName,
		Description:       CreateDescription(systemsCollectionDesc, s.Logger),
		MembersOdataCount: Int64Ptr(int64(len(members))),
		Members:           &members,
	}
	c.JSON(http.StatusOK, collection)
}

// GetRedfishV1SystemsComputerSystemId returns a specific computer system
//
//revive:disable-next-line var-naming. Codegen is using openapi spec for generation which required Id to be Redfish complaint.
func (s *RedfishServer) GetRedfishV1SystemsComputerSystemId(c *gin.Context, computerSystemID string) {
	// Get the computer system from the use case
	system, err := s.ComputerSystemUC.GetComputerSystem(c.Request.Context(), computerSystemID)
	if err != nil {
		if errors.Is(err, usecase.ErrSystemNotFound) {
			NotFoundError(c, "System", computerSystemID)

			return
		}

		InternalServerError(c, err)

		return
	}

	c.JSON(http.StatusOK, system)
}

// PostRedfishV1SystemsComputerSystemIdActionsComputerSystemReset handles the reset action for a computer system
//
//nolint:revive // Method name is generated from OpenAPI spec and cannot be changed
func (s *RedfishServer) PostRedfishV1SystemsComputerSystemIdActionsComputerSystemReset(c *gin.Context, computerSystemID string) {
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

	//nolint:godox // TODO comment is intentional - provides implementation guidance
	// TODO: Add authorization check for 403 Forbidden
	// Example implementation (requires JWT middleware to store user claims in context):
	//
	// userRole, exists := c.Get("user_role")
	// if !exists || userRole == "read-only" {
	//     ForbiddenError(c)
	//     return
	// }
	//
	// More sophisticated example with permission-based checks:
	// permissions, exists := c.Get("user_permissions")
	// if !exists {
	//     ForbiddenError(c)
	//     return
	// }
	// permList := permissions.([]string)
	// if !contains(permList, "system:reset") {
	//     ForbiddenError(c)
	//     return
	// }

	err := s.ComputerSystemUC.SetPowerState(c.Request.Context(), computerSystemID, *req.ResetType)
	if err != nil {
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
		"@odata.context": odataContextTask,
		"@odata.id":      taskServiceTasks + taskID,
		"@odata.type":    odataTypeTask,
		"EndTime":        now,
		"Id":             taskID,
		"Messages": []map[string]interface{}{
			{
				"Message":   successMsg.Message,
				"MessageId": msgIDBaseSuccess,
				"Severity":  string(generated.OK),
			},
		},
		"Name":       taskName,
		"StartTime":  now,
		"TaskState":  taskStateCompleted,
		"TaskStatus": string(generated.OK),
	}
	c.Header(headerLocation, taskServiceTasks+taskID)
	c.JSON(http.StatusAccepted, task)
}
