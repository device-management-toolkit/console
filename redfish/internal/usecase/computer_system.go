// Package usecase provides interfaces for accessing Redfish computer system data.
package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
)

var (
	// ErrInvalidPowerState is returned when an invalid power state is requested.
	ErrInvalidPowerState = errors.New("invalid power state")

	// ErrPowerStateConflict is returned when a power state transition is not allowed.
	ErrPowerStateConflict = errors.New("power state transition not allowed")

	// ErrInvalidResetType is returned when an invalid reset type is provided.
	ErrInvalidResetType = errors.New("invalid reset type")

	// ErrInvalidBootSettings is returned when invalid boot settings are provided.
	ErrInvalidBootSettings = errors.New("invalid boot settings")

	// ErrInvalidBootTarget is returned when an invalid boot target is provided.
	ErrInvalidBootTarget = errors.New("invalid boot target")

	// ErrInvalidBootEnabled is returned when an invalid boot enabled setting is provided.
	ErrInvalidBootEnabled = errors.New("invalid boot enabled setting")

	// ErrSystemNotFound is returned when a system is not found.
	ErrSystemNotFound = errors.New("system not found")
)

// OData and schema constants for ComputerSystem.
const (
	// ComputerSystemODataType represents the OData type for ComputerSystem.
	ComputerSystemODataType = "#ComputerSystem.v1_26_0.ComputerSystem"

	// ComputerSystemODataContext represents the OData context for ComputerSystem.
	ComputerSystemODataContext = "/redfish/v1/$metadata#ComputerSystem.ComputerSystem"

	// RedfishSystemsBasePath represents the base path for Systems collection.
	RedfishSystemsBasePath = "/redfish/v1/Systems"

	// Default system type.
	DefaultSystemType = "Physical"
)

// Resource Health constants.
const (
	HealthOK       = "OK"
	HealthWarning  = "Warning"
	HealthCritical = "Critical"
)

// Resource State constants.
const (
	StateEnabled            = "Enabled"
	StateDisabled           = "Disabled"
	StateStandbyOffline     = "StandbyOffline"
	StateStandbySpare       = "StandbySpare"
	StateInTest             = "InTest"
	StateStarting           = "Starting"
	StateAbsent             = "Absent"
	StateUnavailableOffline = "UnavailableOffline"
	StateDeferring          = "Deferring"
	StateQuiesced           = "Quiesced"
	StateUpdating           = "Updating"
	StateDegraded           = "Degraded"
)

// ComputerSystemUseCase provides business logic for ComputerSystem entities.
type ComputerSystemUseCase struct {
	Repo ComputerSystemRepository
}

// GetAll retrieves all ComputerSystem IDs from the repository.
func (uc *ComputerSystemUseCase) GetAll(ctx context.Context) ([]string, error) {
	return uc.Repo.GetAll(ctx)
}

// GetComputerSystem retrieves a ComputerSystem by its systemID and converts it to the generated API type.
func (uc *ComputerSystemUseCase) GetComputerSystem(ctx context.Context, systemID string) (*generated.ComputerSystemComputerSystem, error) {
	// Get device information from repository - this gives us basic device data
	system, err := uc.Repo.GetByID(ctx, systemID)
	if err != nil {
		return nil, err
	}

	// Build the generated type directly with available information
	// Create power state

	var powerState *generated.ComputerSystemComputerSystem_PowerState

	if system.PowerState != "" {
		var redfishPowerState generated.ResourcePowerState

		switch system.PowerState {
		case redfishv1.PowerStateOn:
			redfishPowerState = generated.On
		case redfishv1.PowerStateOff:
			redfishPowerState = generated.Off
		case redfishv1.ResetTypeForceOff, redfishv1.ResetTypeForceRestart, redfishv1.ResetTypePowerCycle:
			redfishPowerState = generated.Off // These reset types default to Off state
		default:
			redfishPowerState = generated.Off // Default to Off for unknown states
		}

		powerState = &generated.ComputerSystemComputerSystem_PowerState{}
		if err := powerState.FromResourcePowerState(redfishPowerState); err != nil {
			// Log error but continue with nil power state
			powerState = nil
		}
	}

	// Convert to string pointers for optional fields (using helper function to reduce allocations)
	manufacturer := stringPtrIfNotEmpty(system.Manufacturer)
	model := stringPtrIfNotEmpty(system.Model)
	serialNumber := stringPtrIfNotEmpty(system.SerialNumber)
	description := stringPtrIfNotEmpty(system.Description)
	biosVersion := stringPtrIfNotEmpty(system.BiosVersion)
	hostName := stringPtrIfNotEmpty(system.HostName)

	// Create system type
	systemType := generated.ComputerSystemSystemType(DefaultSystemType)

	// Create OData fields following the reference pattern
	odataContext := generated.OdataV4Context(ComputerSystemODataContext)
	odataType := generated.OdataV4Type(ComputerSystemODataType)
	odataID := fmt.Sprintf("%s/%s", RedfishSystemsBasePath, systemID)

	// Build Status if present
	status := uc.convertStatusToGenerated(system.Status)

	// Convert Description to union type if present
	var descriptionUnion *generated.ComputerSystemComputerSystem_Description
	if description != nil {
		descriptionUnion = &generated.ComputerSystemComputerSystem_Description{}
		if err := descriptionUnion.FromResourceDescription(*description); err != nil {
			// Log error but continue - don't fail the entire request for Description conversion issues
			descriptionUnion = nil
		}
	}

	// Fetch boot settings
	boot, err := uc.Repo.GetBootSettings(ctx, systemID)
	if err != nil {
		// Log error but don't fail the entire request - boot settings may not be available
		boot = nil
	}
	// Create Actions for this system using the generated Actions type
	actions := uc.createActionsStruct(systemID)

	// Convert MemorySummary if present
	memorySummary := uc.convertMemorySummaryToGenerated(system.MemorySummary)

	// Convert ProcessorSummary if present
	processorSummary := uc.convertProcessorSummaryToGenerated(system.ProcessorSummary)

	result := generated.ComputerSystemComputerSystem{
		OdataContext:     &odataContext,
		OdataId:          &odataID,
		OdataType:        &odataType,
		Id:               systemID,
		Name:             system.Name,
		Description:      descriptionUnion,
		BiosVersion:      biosVersion,
		HostName:         hostName,
		Manufacturer:     manufacturer,
		Model:            model,
		SerialNumber:     serialNumber,
		PowerState:       powerState,
		SystemType:       &systemType,
		Status:           status,
		Boot:             boot,
		Actions:          actions,
		MemorySummary:    memorySummary,
		ProcessorSummary: processorSummary,
	}

	return &result, nil
}

// SetPowerState validates and sets the power state for a ComputerSystem.
func (uc *ComputerSystemUseCase) SetPowerState(ctx context.Context, id string, resetType generated.ResourceResetType) error {
	// Validate the reset type
	switch resetType {
	case generated.ResourceResetTypeOn,
		generated.ResourceResetTypeForceOff,
		generated.ResourceResetTypeForceOn,
		generated.ResourceResetTypeForceRestart,
		generated.ResourceResetTypeGracefulShutdown,
		generated.ResourceResetTypeGracefulRestart,
		generated.ResourceResetTypePowerCycle,
		generated.ResourceResetTypeFullPowerCycle,
		generated.ResourceResetTypeNmi,
		generated.ResourceResetTypePushPowerButton,
		generated.ResourceResetTypePause,
		generated.ResourceResetTypeResume,
		generated.ResourceResetTypeSuspend:
		// Valid reset types
	default:
		return ErrInvalidResetType
	}

	// Convert generated reset type to entity reset type
	entityResetType := convertToEntityResetType(resetType)

	// Set the power state
	return uc.Repo.UpdatePowerState(ctx, id, entityResetType)
}

// StringPtr creates a pointer to a string value.
func StringPtr(s string) *string {
	return &s
}

// stringPtrIfNotEmpty returns a pointer to the string if it's not empty, otherwise nil.
func stringPtrIfNotEmpty(s string) *string {
	if s != "" {
		return &s
	}

	return nil
}

// SystemTypePtr creates a pointer to a ComputerSystemSystemType value.
func SystemTypePtr(st generated.ComputerSystemSystemType) *generated.ComputerSystemSystemType {
	return &st
}

// convertToEntityResetType converts from generated reset type to entity reset type.
func convertToEntityResetType(resetType generated.ResourceResetType) redfishv1.PowerState {
	switch resetType {
	case generated.ResourceResetTypeOn,
		generated.ResourceResetTypeForceOn:
		return redfishv1.PowerStateOn
	case generated.ResourceResetTypeForceOff:
		return redfishv1.ResetTypeForceOff
	case generated.ResourceResetTypeGracefulShutdown:
		return redfishv1.PowerStateOff
	case generated.ResourceResetTypeForceRestart:
		return redfishv1.ResetTypeForceRestart
	case generated.ResourceResetTypeGracefulRestart:
		return redfishv1.PowerStateOff // Map to generic Off since no specific constant
	case generated.ResourceResetTypePowerCycle,
		generated.ResourceResetTypeFullPowerCycle:
		return redfishv1.ResetTypePowerCycle
	case generated.ResourceResetTypeNmi,
		generated.ResourceResetTypePushPowerButton:
		return redfishv1.PowerStateOn
	case generated.ResourceResetTypePause,
		generated.ResourceResetTypeSuspend:
		return redfishv1.PowerStateOff
	case generated.ResourceResetTypeResume:
		return redfishv1.PowerStateOn
	default:
		return redfishv1.PowerStateOff
	}
}

// UpdateBootSettings updates the boot configuration for a ComputerSystem.
func (uc *ComputerSystemUseCase) UpdateBootSettings(ctx context.Context, systemID string, boot *generated.ComputerSystemBoot) error {
	if boot == nil {
		return nil // Nothing to update
	}

	// Validate all boot settings
	if err := uc.validateBootSettings(boot); err != nil {
		return err
	}

	// Update boot settings in repository
	return uc.Repo.UpdateBootSettings(ctx, systemID, boot)
}

// validateBootSettings validates all boot configuration fields.
func (uc *ComputerSystemUseCase) validateBootSettings(boot *generated.ComputerSystemBoot) error {
	if err := uc.validateBootTargetField(boot.BootSourceOverrideTarget); err != nil {
		return err
	}

	if err := uc.validateBootEnabledField(boot.BootSourceOverrideEnabled); err != nil {
		return err
	}

	return uc.validateBootModeField(boot.BootSourceOverrideMode)
}

// validateBootTargetField validates the boot source override target field.
func (uc *ComputerSystemUseCase) validateBootTargetField(targetField *generated.ComputerSystemBoot_BootSourceOverrideTarget) error {
	if targetField == nil {
		return nil
	}

	target, err := targetField.AsComputerSystemBootSource()
	if err != nil {
		return ErrInvalidBootTarget
	}

	return validateBootTarget(target)
}

// validateBootEnabledField validates the boot source override enabled field.
func (uc *ComputerSystemUseCase) validateBootEnabledField(enabledField *generated.ComputerSystemBoot_BootSourceOverrideEnabled) error {
	if enabledField == nil {
		return nil
	}

	enabled, err := enabledField.AsComputerSystemBootSourceOverrideEnabled()
	if err != nil {
		return ErrInvalidBootEnabled
	}

	return validateBootEnabled(enabled)
}

// validateBootModeField validates the boot source override mode field.
func (uc *ComputerSystemUseCase) validateBootModeField(modeField *generated.ComputerSystemBoot_BootSourceOverrideMode) error {
	if modeField == nil {
		return nil
	}

	mode, err := modeField.AsComputerSystemBootSourceOverrideMode()
	if err != nil {
		return nil
	}

	return validateBootMode(mode)
}

// validateBootTarget validates the boot source override target.
func validateBootTarget(target generated.ComputerSystemBootSource) error {
	validTargets := map[generated.ComputerSystemBootSource]bool{
		generated.ComputerSystemBootSourceBiosSetup:    true,
		generated.ComputerSystemBootSourceCd:           true,
		generated.ComputerSystemBootSourceDiags:        true,
		generated.ComputerSystemBootSourceFloppy:       true,
		generated.ComputerSystemBootSourceHdd:          true,
		generated.ComputerSystemBootSourceNone:         true,
		generated.ComputerSystemBootSourcePxe:          true,
		generated.ComputerSystemBootSourceRecovery:     true,
		generated.ComputerSystemBootSourceRemoteDrive:  true,
		generated.ComputerSystemBootSourceSDCard:       true,
		generated.ComputerSystemBootSourceUefiBootNext: true,
		generated.ComputerSystemBootSourceUefiHttp:     true,
		generated.ComputerSystemBootSourceUefiShell:    true,
		generated.ComputerSystemBootSourceUefiTarget:   true,
		generated.ComputerSystemBootSourceUsb:          true,
		generated.ComputerSystemBootSourceUtilities:    true,
	}

	if validTargets[target] {
		return nil
	}

	return fmt.Errorf("%w: invalid boot target %s", ErrInvalidBootSettings, target)
}

// validateBootEnabled validates the boot source override enabled setting.
func validateBootEnabled(enabled generated.ComputerSystemBootSourceOverrideEnabled) error {
	switch enabled {
	case generated.ComputerSystemBootSourceOverrideEnabledContinuous,
		generated.ComputerSystemBootSourceOverrideEnabledDisabled,
		generated.ComputerSystemBootSourceOverrideEnabledOnce:
		return nil
	default:
		return fmt.Errorf("%w: invalid boot enabled setting %s", ErrInvalidBootSettings, enabled)
	}
}

// validateBootMode validates the boot source override mode.
func validateBootMode(mode generated.ComputerSystemBootSourceOverrideMode) error {
	switch mode {
	case generated.Legacy, generated.UEFI:
		return nil
	default:
		return fmt.Errorf("%w: invalid boot mode %s", ErrInvalidBootSettings, mode)
	}
}

// convertStatusToGenerated converts entity Status to generated ResourceStatus.
func (uc *ComputerSystemUseCase) convertStatusToGenerated(status *redfishv1.Status) *generated.ResourceStatus {
	if status == nil {
		return nil
	}

	healthPtr := uc.convertHealthToGenerated(status.Health)
	healthRollupPtr := uc.convertHealthToGenerated(status.HealthRollup)
	statePtr := uc.convertStateToGenerated(status.State)

	// Only create Status if we have at least one field
	if healthPtr == nil && healthRollupPtr == nil && statePtr == nil {
		return nil
	}

	result := &generated.ResourceStatus{
		Health: healthPtr,
		State:  statePtr,
	}

	// Set HealthRollup using the same conversion as Health
	if healthRollupPtr != nil {
		healthEnum, err := healthRollupPtr.AsResourceHealth()
		if err == nil {
			healthRollup := &generated.ResourceStatus_HealthRollup{}
			if err := healthRollup.FromResourceHealth(healthEnum); err == nil {
				result.HealthRollup = healthRollup
			}
		}
	}

	return result
}

// convertHealthToGenerated converts Health string to generated ResourceStatus_Health.
func (uc *ComputerSystemUseCase) convertHealthToGenerated(health string) *generated.ResourceStatus_Health {
	if health == "" {
		return nil
	}

	var healthEnum generated.ResourceHealth

	switch health {
	case HealthOK:
		healthEnum = generated.OK
	case HealthWarning:
		healthEnum = generated.Warning
	case HealthCritical:
		healthEnum = generated.Critical
	default:
		return nil
	}

	healthObj := generated.ResourceStatus_Health{}
	if err := healthObj.FromResourceHealth(healthEnum); err != nil {
		return nil
	}

	return &healthObj
}

// convertStateToGenerated converts State string to generated ResourceStatus_State.
func (uc *ComputerSystemUseCase) convertStateToGenerated(state string) *generated.ResourceStatus_State {
	var stateEnum generated.ResourceState

	switch state {
	case StateEnabled:
		stateEnum = generated.ResourceStateEnabled
	case StateDisabled:
		stateEnum = generated.ResourceStateDisabled
	case StateStandbyOffline:
		stateEnum = generated.ResourceStateStandbyOffline
	case StateStandbySpare:
		stateEnum = generated.ResourceStateStandbySpare
	case StateInTest:
		stateEnum = generated.ResourceStateInTest
	case StateStarting:
		stateEnum = generated.ResourceStateStarting
	case StateAbsent:
		stateEnum = generated.ResourceStateAbsent
	case StateUnavailableOffline:
		stateEnum = generated.ResourceStateUnavailableOffline
	case StateDeferring:
		stateEnum = generated.ResourceStateDeferring
	case StateQuiesced:
		stateEnum = generated.ResourceStateQuiesced
	case StateUpdating:
		stateEnum = generated.ResourceStateUpdating
	case StateDegraded:
		stateEnum = generated.ResourceStateDegraded
	default:
		return nil // Don't create state if unknown value
	}

	stateObj := generated.ResourceStatus_State{}
	if err := stateObj.FromResourceState(stateEnum); err != nil {
		return nil
	}

	return &stateObj
}

// createActionsStruct builds the Actions property using generated types.
func (uc *ComputerSystemUseCase) createActionsStruct(systemID string) *generated.ComputerSystemActions {
	// Create the target URI for the Reset action
	target := fmt.Sprintf("/redfish/v1/Systems/%s/Actions/ComputerSystem.Reset", systemID)
	title := "Reset"

	// Create the ComputerSystem.Reset action
	resetAction := &generated.ComputerSystemReset{
		Target: &target,
		Title:  &title,
	}

	// Create and return the Actions structure
	return &generated.ComputerSystemActions{
		HashComputerSystemReset: resetAction,
	}
}

// convertMemorySummaryToGenerated converts entity ComputerSystemMemorySummary to generated ComputerSystemMemorySummary.
func (uc *ComputerSystemUseCase) convertMemorySummaryToGenerated(memorySummary *redfishv1.ComputerSystemMemorySummary) *generated.ComputerSystemMemorySummary {
	if memorySummary == nil {
		return nil
	}

	return &generated.ComputerSystemMemorySummary{
		TotalSystemMemoryGiB: memorySummary.TotalSystemMemoryGiB,
		Status:               uc.convertStatusToGenerated(memorySummary.Status),
		MemoryMirroring:      uc.convertMemoryMirroringToGenerated(memorySummary.MemoryMirroring),
	}
}

// convertMemoryMirroringToGenerated converts MemoryMirroring enum to generated type.
func (uc *ComputerSystemUseCase) convertMemoryMirroringToGenerated(mirroring redfishv1.MemoryMirroring) *generated.ComputerSystemMemorySummary_MemoryMirroring {
	if mirroring == "" {
		return nil
	}

	// Validate the MemoryMirroring value against known enum values
	if !uc.isValidMemoryMirroring(mirroring) {
		return nil // Return nil for invalid mirroring types
	}

	memoryMirroring := &generated.ComputerSystemMemorySummary_MemoryMirroring{}

	mirroringType := generated.ComputerSystemMemoryMirroring(string(mirroring))
	if err := memoryMirroring.FromComputerSystemMemoryMirroring(mirroringType); err != nil {
		return nil // Return nil on conversion error
	}

	return memoryMirroring
}

// isValidMemoryMirroring validates if the MemoryMirroring value is one of the defined enum values.
func (uc *ComputerSystemUseCase) isValidMemoryMirroring(mirroring redfishv1.MemoryMirroring) bool {
	switch mirroring {
	case redfishv1.MemoryMirroringSystem, redfishv1.MemoryMirroringDIMM,
		redfishv1.MemoryMirroringHybrid, redfishv1.MemoryMirroringNone:
		return true
	default:
		return false
	}
}

// convertProcessorSummaryToGenerated converts entity ComputerSystemProcessorSummary to generated ComputerSystemProcessorSummary.
func (uc *ComputerSystemUseCase) convertProcessorSummaryToGenerated(processorSummary *redfishv1.ComputerSystemProcessorSummary) *generated.ComputerSystemProcessorSummary {
	if processorSummary == nil {
		return nil
	}

	var metrics *generated.OdataV4IdRef
	if processorSummary.Metrics != nil {
		metrics = &generated.OdataV4IdRef{
			OdataId: processorSummary.Metrics,
		}
	}

	// Convert *int to *int64 for count fields
	var count *int64

	if processorSummary.Count != nil {
		c := int64(*processorSummary.Count)
		count = &c
	}

	var coreCount *int64

	if processorSummary.CoreCount != nil {
		cc := int64(*processorSummary.CoreCount)
		coreCount = &cc
	}

	var logicalProcessorCount *int64

	if processorSummary.LogicalProcessorCount != nil {
		lpc := int64(*processorSummary.LogicalProcessorCount)
		logicalProcessorCount = &lpc
	}

	return &generated.ComputerSystemProcessorSummary{
		Count:                 count,
		CoreCount:             coreCount,
		LogicalProcessorCount: logicalProcessorCount,
		Metrics:               metrics,
		Model:                 processorSummary.Model,
		Status:                uc.convertStatusToGenerated(processorSummary.Status),
		ThreadingEnabled:      processorSummary.ThreadingEnabled,
	}
}
