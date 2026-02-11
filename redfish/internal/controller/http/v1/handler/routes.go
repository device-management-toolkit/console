// Package v1 provides Redfish v1 API route setup and configuration.
package v1

import (
	dmtconfig "github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/pkg/logger"
	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	"github.com/device-management-toolkit/console/redfish/internal/usecase"
	"github.com/device-management-toolkit/console/redfish/internal/usecase/sessions"
)

const (
	// Task state constants from Redfish Task.v1_8_0 specification
	taskStateCompleted = "Completed"

	// Registry message IDs
	msgIDBaseSuccess = "Base.1.22.0.Success"

	// OData metadata constants - Task
	odataContextTask = "/redfish/v1/$metadata#Task.Task"
	odataTypeTask    = "#Task.v1_6_0.Task"
	taskName         = "System Reset Task"
	taskServiceTasks = "/redfish/v1/TaskService/Tasks/"
)

// RedfishServer implements the Redfish API handlers and delegates operations to specialized handlers
type RedfishServer struct {
	ComputerSystemUC *usecase.ComputerSystemUseCase
	SessionUC        *sessions.UseCase
	Config           *dmtconfig.Config
	Logger           logger.Interface
	Services         []ODataService // Cached OData services loaded from OpenAPI spec
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

// SystemTypePtr creates a pointer to a ComputerSystemSystemType value.
func SystemTypePtr(st generated.ComputerSystemSystemType) *generated.ComputerSystemSystemType {
	return &st
}
