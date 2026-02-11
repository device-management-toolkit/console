// Package usecase provides interfaces for accessing Redfish computer system data.
package usecase

import (
	"context"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
)

// ComputerSystemRepository defines the interface for computer system data access.
type ComputerSystemRepository interface {
	GetAll(ctx context.Context) ([]string, error)
	GetByID(ctx context.Context, systemID string) (*redfishv1.ComputerSystem, error)
	UpdatePowerState(ctx context.Context, systemID string, state redfishv1.PowerState) error
	GetBootSettings(ctx context.Context, systemID string) (*generated.ComputerSystemBoot, error)
	UpdateBootSettings(ctx context.Context, systemID string, boot *generated.ComputerSystemBoot) error
}
