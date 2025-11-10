// Package v1 provides Redfish v1 API route setup and configuration.
package v1

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/labstack/gommon/log"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
	"github.com/device-management-toolkit/console/redfish/internal/usecase"
)

// RedfishServer implements the Redfish API handlers
// Add dependencies here if needed (e.g., usecase, presenter, etc.)
type RedfishServer struct {
	ComputerSystemUC *usecase.ComputerSystemUseCase
}

/*
Comment: function not used/invoked
SetupRedfishV1Routes sets up the Redfish v1 routes on the main router
func SetupRedfishV1Routes(router *gin.Engine, devicesUC *devices.UseCase) {
	// Enable HandleMethodNotAllowed to properly distinguish between 404 and 405 errors
	router.HandleMethodNotAllowed = true

	repo := usecase.NewWsmanComputerSystemRepo(devicesUC)
	computerSystemUC := &usecase.ComputerSystemUseCase{Repo: repo}
	redfishServer := &RedfishServer{ComputerSystemUC: computerSystemUC}

	v1Group := router.Group("/redfish/v1")

	// Register the handlers with options
	redfishgenerated.RegisterHandlersWithOptions(v1Group, redfishServer, redfishgenerated.GinServerOptions{
		BaseURL: "",
		ErrorHandler: func(c *gin.Context, err error, statusCode int) {
			switch statusCode {
			case http.StatusUnauthorized:
				UnauthorizedError(c)
			case http.StatusForbidden:
				ForbiddenError(c)
			case http.StatusMethodNotAllowed:
				MethodNotAllowedError(c)
			case http.StatusBadRequest:
				BadRequestError(c, err.Error(), "Base.1.11.GeneralError", "Check your request body and parameters.", "Critical")
			case http.StatusNotFound:
				NotFoundError(c, "Resource")
			case http.StatusConflict:
				ConflictError(c, "Resource", err.Error())
			case http.StatusServiceUnavailable:
				ServiceUnavailableError(c, 60)
			default:
				InternalServerError(c, err)
			}
		},
	})

	// Add Redfish-compliant handler for 405 Method Not Allowed
	router.NoMethod(func(c *gin.Context) {
		MethodNotAllowedError(c)
	})

	// Add Redfish-compliant NoRoute handler for /redfish/v1
	router.NoRoute(func(c *gin.Context) {
		if len(c.Request.URL.Path) >= 11 && c.Request.URL.Path[:11] == "/redfish/v1" {
			NotFoundError(c, "Resource")
		} else {
			c.Next() // fallback to Gin's default
		}
	})

}
*/

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

// SystemTypePtr creates a pointer to a ComputerSystemSystemType value.
func SystemTypePtr(st generated.ComputerSystemSystemType) *generated.ComputerSystemSystemType {
	return &st
}

// GetRedfishV1 returns the service root
func (s *RedfishServer) GetRedfishV1(c *gin.Context) {
	serviceRoot := generated.ServiceRootServiceRoot{
		OdataContext:   StringPtr("/redfish/v1/$metadata#ServiceRoot.ServiceRoot"),
		OdataId:        StringPtr("/redfish/v1"),
		OdataType:      StringPtr("#ServiceRoot.v1_19_0.ServiceRoot"),
		Id:             "RootService",
		Name:           "Root Service",
		RedfishVersion: StringPtr("1.19.0"),
		Systems: &generated.OdataV4IdRef{
			OdataId: StringPtr("/redfish/v1/Systems"),
		},
		Chassis: &generated.OdataV4IdRef{
			OdataId: StringPtr("/redfish/v1/Chassis"),
		},
		Managers: &generated.OdataV4IdRef{
			OdataId: StringPtr("/redfish/v1/Managers"),
		},
	}
	c.JSON(http.StatusOK, serviceRoot)
}

// GetRedfishV1Metadata returns the OData metadata
func (s *RedfishServer) GetRedfishV1Metadata(c *gin.Context) {
	metadata := ""

	c.Header("Content-Type", "application/xml")
	c.String(http.StatusOK, metadata)
}

// GetRedfishV1Systems returns the computer systems collection
func (s *RedfishServer) GetRedfishV1Systems(c *gin.Context) {
	collection := generated.ComputerSystemCollectionComputerSystemCollection{
		OdataContext:      StringPtr("/redfish/v1/$metadata#ComputerSystemCollection.ComputerSystemCollection"),
		OdataId:           StringPtr("/redfish/v1/Systems"),
		OdataType:         StringPtr("#ComputerSystemCollection.ComputerSystemCollection"),
		Name:              "Computer System Collection",
		Description:       nil,
		MembersOdataCount: Int64Ptr(1),
		Members: &[]generated.OdataV4IdRef{
			{OdataId: StringPtr("/redfish/v1/Systems/System1")},
		},
	}
	c.JSON(http.StatusOK, collection)
}

// GetRedfishV1SystemsComputerSystemId returns a specific computer system
//
//revive:disable-next-line var-naming. Codegen is using openapi spec for generation which required Id to be Redfish complaint.
func (s *RedfishServer) GetRedfishV1SystemsComputerSystemId(c *gin.Context, computerSystemID string) {
	if computerSystemID != "System1" {
		NotFoundError(c, "System")

		return
	}

	system := generated.ComputerSystemComputerSystem{
		OdataContext: StringPtr("/redfish/v1/$metadata#ComputerSystem.ComputerSystem"),
		OdataId:      StringPtr("/redfish/v1/Systems/System1"),
		OdataType:    StringPtr("#ComputerSystem.v1_26_0.ComputerSystem"),
		Id:           "System1",
		Name:         "Computer System",
		SerialNumber: StringPtr("SN123456789"),
		Manufacturer: StringPtr("Intel Corporation"),
		Model:        StringPtr("Example System"),
		SystemType:   SystemTypePtr(generated.Physical),
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

	err := s.ComputerSystemUC.SetPowerState(computerSystemID, *req.ResetType)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidPowerState) {
			PropertyValueNotInListError(c, "ResetType")

			return
		}

		if errors.Is(err, usecase.ErrPowerStateConflict) {
			PowerStateConflictError(c, string(*req.ResetType))

			return
		}
		// Robust backend not found error
		if err.Error() == "system not found" {
			NotFoundError(c, "System")

			return
		}
		// Check for service unavailability errors (503)
		// This catches cases where the backend service (WSMAN/AMT) is unreachable
		errMsg := err.Error()
		if errMsg == "connection refused" || errMsg == "connection timeout" ||
			errMsg == "service unavailable" || errMsg == "device not responding" {
			ServiceUnavailableError(c, redfishv1.ServiceUnavailableRetryAfterSeconds)

			return
		}

		InternalServerError(c, err)

		return
	}

	// Generate dynamic Task response
	taskID := fmt.Sprintf("%d", time.Now().UnixNano())
	now := time.Now().UTC().Format(time.RFC3339)
	task := map[string]interface{}{
		"@odata.context": "/redfish/v1/$metadata#Task.Task",
		"@odata.id":      "/redfish/v1/TaskService/Tasks/" + taskID,
		"@odata.type":    "#Task.v1_6_0.Task",
		"EndTime":        now,
		"Id":             taskID,
		"Messages": []map[string]interface{}{
			{
				"Message":   "The request completed successfully.",
				"MessageId": "Base.1.11.0.Success",
				"Severity":  "OK",
			},
		},
		"Name":       "System Reset Task",
		"StartTime":  now,
		"TaskState":  "Completed",
		"TaskStatus": "OK",
	}
	c.Header("Location", "/redfish/v1/TaskService/Tasks/"+taskID)
	c.JSON(http.StatusAccepted, task)
}
