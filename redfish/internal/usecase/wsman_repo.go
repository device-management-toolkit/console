// Package usecase provides a WSMAN-backed implementation of ComputerSystemRepository.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	amtBoot "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/boot"
	cimBoot "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/boot"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/logger"
	"github.com/device-management-toolkit/console/redfish/internal/controller/http/v1/generated"
	redfishv1 "github.com/device-management-toolkit/console/redfish/internal/entity/v1"
)

const (
	// ErrMsgDeviceNotFound is the error message returned by devices use case when device is not found.
	ErrMsgDeviceNotFound = "DevicesUseCase -  - : "

	// Power action constants for AMT/WSMAN power management.
	powerActionPowerUp    = 2  // CIM Power Management Service - Power On
	powerActionPowerCycle = 5  // Power Cycle (off then on)
	powerActionPowerDown  = 8  // Power Down (soft off)
	powerActionReset      = 10 // Reset (reboot)

	// Common constants to eliminate magic values.
	bytesPerGiB          = 1024 * 1024 * 1024
	maxEnabledStateValue = 32767

	// maxSystemsList is the maximum number of systems to retrieve in a single request.
	maxSystemsList = 100

	// Health state constants.
	healthStateOK       = "OK"
	healthStateWarning  = "Warning"
	healthStateCritical = "Critical"

	// CIM OperationalStatus constants for memory health mapping.
	CIMStatusOK                  = 2  // OK
	CIMStatusDegraded            = 3  // Degraded
	CIMStatusError               = 6  // Error
	CIMStatusNonRecoverableError = 7  // Non-Recoverable Error
	CIMStatusStressed            = 10 // Stressed

	// Enabled state constants.
	enabledStateEnabled        = "Enabled"
	enabledStateDisabled       = "Disabled"
	enabledStateStandbyOffline = "StandbyOffline"
	enabledStateInTest         = "InTest"
	enabledStateQuiesced       = "Quiesced"
	enabledStateStarting       = "Starting"

	// CIM health state values.
	cimHealthStateOK        = 5
	cimHealthStateWarning1  = 10
	cimHealthStateWarning2  = 15
	cimHealthStateCritical1 = 20
	cimHealthStateCritical2 = 25
	cimHealthStateCritical3 = 30

	// CIM enabled state values.
	cimEnabledStateOther          = 1
	cimEnabledStateEnabled        = 2
	cimEnabledStateDisabled       = 3
	cimEnabledStateShuttingDown   = 4
	cimEnabledStateNotApplicable  = 5
	cimEnabledStateStandbyOffline = 6
	cimEnabledStateInTest         = 7
	cimEnabledStateDeferred       = 8
	cimEnabledStateQuiesced       = 9
	cimEnabledStateStarting       = 10

	// Maximum items to process in arrays to prevent hangs.
	maxArrayItems = 10
)

var (
	// ErrGetAllNotImplemented is returned when GetAll is called (not yet implemented).
	ErrGetAllNotImplemented = errors.New("GetAll not implemented")

	// ErrUnsupportedPowerState is returned when an unsupported power state is requested.
	ErrUnsupportedPowerState = errors.New("unsupported power state")

	// ErrBootSettingsNotAvailable is returned when boot settings cannot be retrieved.
	ErrBootSettingsNotAvailable = errors.New("boot settings not available")

	// ErrUnsupportedBootTarget is returned when an unsupported boot target is requested.
	ErrUnsupportedBootTarget = errors.New("unsupported boot target")
)

// CIMObjectType represents different types of CIM objects.
type CIMObjectType string

const (
	CIMObjectChassis               CIMObjectType = "chassis"
	CIMObjectComputerSystemPackage CIMObjectType = "computersystem"
	CIMObjectBIOSElement           CIMObjectType = "bioselement"
	CIMObjectPhysicalMemory        CIMObjectType = "physicalmemory"
	CIMObjectProcessor             CIMObjectType = "processor"
	CIMObjectChip                  CIMObjectType = "chip"
)

// PropertyExtractor defines a function signature for custom property transformation.
type PropertyExtractor func(interface{}) interface{}

// CIMPropertyConfig defines the configuration for extracting a property from CIM data.
type CIMPropertyConfig struct {
	CIMObject    CIMObjectType     // Which CIM object to extract from
	CIMProperty  string            // The property name in the CIM object
	StructField  string            // Optional: key name for storing in results map (defaults to CIMProperty)
	Transformer  PropertyExtractor // Optional transformation function
	UseFirstItem bool              // For array responses, use first item (default: true)
}

// WsmanComputerSystemRepo implements ComputerSystemRepository using WSMAN backend.
type WsmanComputerSystemRepo struct {
	usecase *devices.UseCase
	log     logger.Interface
}

// Forward declarations for transformer functions.
var (
	healthStateTransformer  PropertyExtractor
	enabledStateTransformer PropertyExtractor
)

// allCIMConfigs defines the complete set of CIM property configurations for computer system data extraction.
// This global configuration is used by GetByID to extract all necessary properties in a single call.
// Transformers are statically assigned for computer system status properties.
var allCIMConfigs = []CIMPropertyConfig{
	// Chassis properties
	{CIMObject: CIMObjectChassis, CIMProperty: "Manufacturer", UseFirstItem: true},
	{CIMObject: CIMObjectChassis, CIMProperty: "Model", UseFirstItem: true},
	{CIMObject: CIMObjectChassis, CIMProperty: "SerialNumber", UseFirstItem: true},
	// ComputerSystem properties
	{CIMObject: CIMObjectComputerSystemPackage, CIMProperty: "Description", UseFirstItem: true},
	{CIMObject: CIMObjectComputerSystemPackage, CIMProperty: "DNSHostName", UseFirstItem: true},
	// BIOS properties
	{CIMObject: CIMObjectBIOSElement, CIMProperty: "Version", UseFirstItem: true},
	// Computer System status properties with static transformer functions
	{CIMObject: CIMObjectComputerSystemPackage, CIMProperty: "HealthState", UseFirstItem: true, Transformer: healthStateTransformer},
	{CIMObject: CIMObjectComputerSystemPackage, CIMProperty: "EnabledState", UseFirstItem: true, Transformer: enabledStateTransformer},
	// Memory properties - we extract raw arrays and process them later for aggregation
	{CIMObject: CIMObjectPhysicalMemory, CIMProperty: "Capacity", UseFirstItem: false},
	{CIMObject: CIMObjectPhysicalMemory, CIMProperty: "OperationalStatus", UseFirstItem: false},
	// Processor properties - we extract arrays for aggregation into ProcessorSummary
	{CIMObject: CIMObjectProcessor, CIMProperty: "HealthState", UseFirstItem: false, Transformer: healthStateTransformer},
	{CIMObject: CIMObjectProcessor, CIMProperty: "EnabledState", UseFirstItem: false, Transformer: enabledStateTransformer},
	// Processor model from CIM_Chip
	{CIMObject: CIMObjectChip, CIMProperty: "Version", StructField: "ChipVersion", UseFirstItem: true},
}

// extractStringFromMap safely extracts a string value from a map, returning the value and whether it exists.
func extractStringFromMap(data map[string]interface{}, key string) (string, bool) {
	if value, exists := data[key]; exists {
		if strValue, ok := value.(string); ok {
			return strValue, true
		}
	}

	return "", false
}

// convertToInt safely converts interface{} values to int, handling both int and float64 types.
func convertToInt(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

// createHealthStateTransformer creates the health state transformation function.
func createHealthStateTransformer() PropertyExtractor {
	return func(value interface{}) interface{} {
		healthState, ok := convertToInt(value)
		if !ok {
			return nil
		}

		// Use constants for validation and conversion
		if healthState < 0 || healthState > cimHealthStateCritical3 {
			return nil // Invalid range
		}

		switch healthState {
		case 0:
			return nil // Unknown
		case cimHealthStateOK:
			return healthStateOK
		case cimHealthStateWarning1, cimHealthStateWarning2:
			return healthStateWarning
		case cimHealthStateCritical1, cimHealthStateCritical2:
			return healthStateCritical
		case cimHealthStateCritical3:
			return healthStateCritical
		default:
			return nil
		}
	}
}

// createEnabledStateTransformer creates the enabled state transformation function.
func createEnabledStateTransformer() PropertyExtractor {
	return func(value interface{}) interface{} {
		enabledState, ok := convertToInt(value)
		if !ok {
			return nil
		}

		// Use constants for validation and conversion
		if enabledState < 0 || enabledState > maxEnabledStateValue {
			return nil // Invalid range
		}

		switch enabledState {
		case cimEnabledStateOther, cimEnabledStateEnabled, cimEnabledStateNotApplicable:
			return enabledStateEnabled
		case cimEnabledStateDisabled, cimEnabledStateShuttingDown, cimEnabledStateDeferred:
			return enabledStateDisabled
		case cimEnabledStateStandbyOffline:
			return enabledStateStandbyOffline
		case cimEnabledStateInTest:
			return enabledStateInTest
		case cimEnabledStateQuiesced:
			return enabledStateQuiesced
		case cimEnabledStateStarting:
			return enabledStateStarting
		default:
			return nil
		}
	}
}

// initializeTransformers initializes the global transformer functions.
func initializeTransformers() {
	healthStateTransformer = createHealthStateTransformer()
	enabledStateTransformer = createEnabledStateTransformer()
}

// NewWsmanComputerSystemRepo creates a new WSMAN-backed computer system repository.
func NewWsmanComputerSystemRepo(uc *devices.UseCase, log logger.Interface) *WsmanComputerSystemRepo {
	// Ensure transformers are initialized
	if healthStateTransformer == nil || enabledStateTransformer == nil {
		initializeTransformers()
	}

	return &WsmanComputerSystemRepo{
		usecase: uc,
		log:     log,
	}
}

// getCIMProperties extracts multiple CIM properties in a single call.
func (r *WsmanComputerSystemRepo) getCIMProperties(ctx context.Context, systemID string, configs []CIMPropertyConfig) (map[string]interface{}, dto.HardwareInfo, error) {
	results := make(map[string]interface{})

	// Get hardware info only once to avoid multiple WSMAN calls
	hwInfo, err := r.usecase.GetHardwareInfo(ctx, systemID)
	if err != nil {
		r.log.Error("Failed to get hardware info", "systemID", systemID, "error", err)

		return results, hwInfo, err
	}

	for _, config := range configs {
		if value := r.extractPropertyFromHardwareInfo(hwInfo, config); value != nil {
			// Use StructField as key if specified, otherwise use CIMProperty
			key := config.CIMProperty
			if config.StructField != "" {
				key = config.StructField
			}

			results[key] = value
		}
	}

	return results, hwInfo, nil
}

// extractPropertyFromHardwareInfo extracts a single property from pre-fetched hardware info.
func (r *WsmanComputerSystemRepo) extractPropertyFromHardwareInfo(hwInfo dto.HardwareInfo, config CIMPropertyConfig) interface{} {
	// Select the appropriate CIM object
	response := r.selectCIMObject(hwInfo, config)
	if response == nil {
		return nil
	}

	// Extract the property value using optimized method
	value := r.extractFromResponse(response, config)

	// Apply transformation if provided
	if config.Transformer != nil && value != nil {
		if transformed := config.Transformer(value); transformed != nil {
			return transformed
		}

		r.log.Warn("Transformer returned nil", "property", config.CIMProperty, "original_value", value)
	}

	return value
}

// selectCIMObject selects the appropriate CIM object from hardware info based on config.
func (r *WsmanComputerSystemRepo) selectCIMObject(hwInfo dto.HardwareInfo, config CIMPropertyConfig) interface{} {
	switch config.CIMObject {
	case CIMObjectChassis:
		return hwInfo.CIMChassis.Response
	case CIMObjectComputerSystemPackage:
		return hwInfo.CIMComputerSystemPackage.Response
	case CIMObjectBIOSElement:
		return hwInfo.CIMBIOSElement.Response
	case CIMObjectPhysicalMemory:
		return hwInfo.CIMPhysicalMemory.Response
	case CIMObjectProcessor:
		if len(hwInfo.CIMProcessor.Responses) > 0 {
			return hwInfo.CIMProcessor.Responses
		}
	case CIMObjectChip:
		if len(hwInfo.CIMChip.Responses) > 0 {
			return hwInfo.CIMChip.Responses
		}
	default:
		r.log.Warn("Unknown CIM object type", "type", config.CIMObject, "property", config.CIMProperty)
	}

	return nil
}

// extractFromResponse handles both struct and map response formats using a unified approach.
func (r *WsmanComputerSystemRepo) extractFromResponse(response interface{}, config CIMPropertyConfig) interface{} {
	// Handle array responses - respect UseFirstItem configuration
	if responseArray, ok := response.([]interface{}); ok && len(responseArray) > 0 {
		if config.UseFirstItem {
			// Extract from first item only
			return r.extractFromSingleResponse(responseArray[0], config)
		}
		// Process all items (existing behavior for non-UseFirstItem)
		return r.processItemsArray(responseArray, config.CIMProperty)
	}

	// Handle single responses
	return r.extractFromSingleResponse(response, config)
}

// extractFromSingleResponse handles extraction from a single response (not array).
func (r *WsmanComputerSystemRepo) extractFromSingleResponse(response interface{}, config CIMPropertyConfig) interface{} {
	// First try reflection-based extraction (works for structs)
	if value := r.extractUsingReflection(response, config.CIMProperty); value != nil {
		return value
	}
	// Fall back to map-based extraction for generic structures
	return r.extractFromMap(response, config)
}

// extractUsingReflection extracts a property using reflection.
func (r *WsmanComputerSystemRepo) extractUsingReflection(response interface{}, fieldName string) interface{} {
	responseValue := reflect.ValueOf(response)
	// Handle pointer types by dereferencing
	if responseValue.Kind() == reflect.Ptr {
		if responseValue.IsNil() {
			return nil
		}

		responseValue = responseValue.Elem()
	}
	// Ensure we have a struct
	if responseValue.Kind() != reflect.Struct {
		return nil
	}
	// Get the field by name
	fieldValue := responseValue.FieldByName(fieldName)
	if !fieldValue.IsValid() || !fieldValue.CanInterface() {
		return nil
	}

	return fieldValue.Interface()
}

// extractFromSingleItem extracts property from a single map item.
func (r *WsmanComputerSystemRepo) extractFromSingleItem(itemMap map[string]interface{}, propertyName string) interface{} {
	if len(itemMap) == 0 {
		return nil
	}

	if value, exists := itemMap[propertyName]; exists {
		return value
	}

	return nil
}

// processItemsArray processes an array of items and returns the first matching property.
func (r *WsmanComputerSystemRepo) processItemsArray(items []interface{}, propertyName string) interface{} {
	if len(items) == 0 {
		return nil
	}
	// Limit iterations to prevent hanging on large arrays
	for i, item := range items {
		if i >= maxArrayItems {
			break
		}

		if itemMap, ok := item.(map[string]interface{}); ok {
			if value := r.extractFromSingleItem(itemMap, propertyName); value != nil {
				return value
			}
		}
	}

	return nil
}

// extractFromMap extracts specific CIM property values from a WSMAN response.
// It handles multiple response formats:
//   - Direct array responses containing CIM items
//   - Map responses with nested structures following common CIM response patterns
//   - Single item responses without array wrapping
//
// The function attempts to locate the Items array by traversing common CIM response
// paths (PullResponse, Body.PullResponse, etc.). If an Items array is found, it
// processes each item to extract the specified CIM property. Otherwise, it falls
// back to extracting the property from a single item response.
//
// Parameters:
//   - response: The raw WSMAN response, typically a map[string]interface{} or []interface{}
//   - config: Configuration specifying which CIM property to extract
//
// Returns:
//   - The extracted property value(s), or nil if the response is invalid or the property is not found
func (r *WsmanComputerSystemRepo) extractFromMap(response interface{}, config CIMPropertyConfig) interface{} {
	if response == nil {
		return nil
	}
	// Handle array response directly
	if itemsArray, ok := response.([]interface{}); ok {
		return r.processItemsArray(itemsArray, config.CIMProperty)
	}
	// Handle map response
	responseMap, ok := response.(map[string]interface{})
	if !ok {
		return nil
	}
	// Try common CIM response paths
	paths := [][]string{{"PullResponse"}, {}, {"Body", "PullResponse"}}
	for _, path := range paths {
		current := responseMap
		for _, key := range path {
			if next, ok := current[key].(map[string]interface{}); ok {
				current = next
			} else {
				goto nextPath
			}
		}

		if items, ok := current["Items"].([]interface{}); ok {
			return r.processItemsArray(items, config.CIMProperty)
		}

	nextPath:
	}
	// Fallback to single item extraction
	return r.extractFromSingleItem(responseMap, config.CIMProperty)
}

// isDeviceNotFoundError checks if the error indicates a device was not found.
func (r *WsmanComputerSystemRepo) isDeviceNotFoundError(err error) bool {
	return err != nil && err.Error() == ErrMsgDeviceNotFound
}

// mapCIMPowerStateToRedfish converts CIM power state to Redfish PowerState.
func (r *WsmanComputerSystemRepo) mapCIMPowerStateToRedfish(cimState int) redfishv1.PowerState {
	// Validate input range - CIM power states should be positive values
	if cimState < 0 {
		return redfishv1.PowerStateOff // Invalid negative value defaults to Off
	}

	switch cimState {
	case redfishv1.CIMPowerStateOn:
		return redfishv1.PowerStateOn
	case redfishv1.CIMPowerStateOffSoft, redfishv1.CIMPowerStateOffHard:
		return redfishv1.PowerStateOff
	default:
		return redfishv1.PowerStateOff // Default to Off for unknown states
	}
}

// mapRedfishPowerStateToAction converts Redfish PowerState to WSMAN power action.
// Note: Graceful operations (GracefulShutdown, GracefulRestart) should use
// IPS_PowerManagementService.RequestOSPowerSavingStateChange instead of CIM power management.
func (r *WsmanComputerSystemRepo) mapRedfishPowerStateToAction(state redfishv1.PowerState) (int, error) {
	switch state {
	case redfishv1.PowerStateOn:
		return devices.CIMPMSPowerOn, nil // Power On = 2
	case redfishv1.PowerStateOff:
		return powerActionPowerDown, nil
	case redfishv1.ResetTypeForceOff:
		return powerActionPowerDown, nil
	case redfishv1.ResetTypeForceRestart:
		return powerActionReset, nil
	case redfishv1.ResetTypePowerCycle:
		return powerActionPowerCycle, nil
	// Note: GracefulShutdown and GracefulRestart are handled in the computer_system.go usecase
	// using IPS_PowerManagementService.RequestOSPowerSavingStateChange
	default:
		return 0, ErrUnsupportedPowerState
	}
}

// buildComputerSystemFromCIMData creates a ComputerSystem entity from CIM data and hardware info.
func (r *WsmanComputerSystemRepo) buildComputerSystemFromCIMData(systemID string, powerState redfishv1.PowerState, cimData map[string]interface{}, hwInfo dto.HardwareInfo) *redfishv1.ComputerSystem {
	// Extract CIM properties using helper function
	manufacturer, _ := extractStringFromMap(cimData, "Manufacturer")
	model, _ := extractStringFromMap(cimData, "Model")
	serialNumber, _ := extractStringFromMap(cimData, "SerialNumber")
	description, _ := extractStringFromMap(cimData, "Description")
	biosVersion, _ := extractStringFromMap(cimData, "Version")
	hostNameFromCIM, _ := extractStringFromMap(cimData, "DNSHostName")

	// Build Status from extracted health and state data using common function
	health, hasHealth := extractStringFromMap(cimData, "HealthState")
	state, hasState := extractStringFromMap(cimData, "EnabledState")
	status := r.buildComponentStatus(health, hasHealth, state, hasState, false, string(CIMObjectComputerSystemPackage))

	// Build ComputerSystem using only CIM data
	system := &redfishv1.ComputerSystem{
		ID:           systemID,
		Name:         systemID, // Use systemID as default name
		Status:       status,
		PowerState:   powerState,
		Manufacturer: manufacturer,
		Model:        model,
		SerialNumber: serialNumber,
		BiosVersion:  biosVersion,
		SystemType:   redfishv1.SystemTypePhysical,
		ODataID:      "/redfish/v1/Systems/" + systemID,
		ODataType:    "#ComputerSystem.v1_26_0.ComputerSystem",
	}

	// Set optional properties only if we have actual CIM data
	if description != "" {
		system.Description = description
	}

	if hostNameFromCIM != "" {
		system.HostName = hostNameFromCIM
		system.Name = hostNameFromCIM // Use CIM hostname as the name
	}

	// Build MemorySummary from memory data
	memorySummary := r.buildMemorySummaryFromCIMData(cimData)
	if memorySummary != nil {
		system.MemorySummary = memorySummary
	}

	// Build ProcessorSummary from processor data
	processorSummary := r.buildProcessorSummaryFromCIMData(cimData, hwInfo)
	if processorSummary != nil {
		system.ProcessorSummary = processorSummary
	}

	return system
}

// buildMemorySummaryFromCIMData creates a MemorySummary from CIM memory data.
func (r *WsmanComputerSystemRepo) buildMemorySummaryFromCIMData(cimData map[string]interface{}) *redfishv1.ComputerSystemMemorySummary {
	// Extract memory capacity data (array of capacity values)
	capacityData, hasCapacity := cimData["Capacity"]
	operationalStatusData, hasStatus := cimData["OperationalStatus"]

	if !hasCapacity && !hasStatus {
		return nil // No memory data available
	}

	// Process capacity data to calculate total memory in GiB
	var totalMemoryGiB float32
	if hasCapacity {
		totalMemoryGiB = r.calculateTotalMemoryGiB(capacityData)
	}

	// Process operational status to determine worst health state
	var memoryHealth string
	if hasStatus {
		memoryHealth = r.calculateMemoryHealth(operationalStatusData)
	}

	// Build memory summary using internal entity type - only populate with actual data
	memorySummary := &redfishv1.ComputerSystemMemorySummary{}

	// Only set TotalSystemMemoryGiB if we have capacity data
	if hasCapacity && totalMemoryGiB > 0 {
		memorySummary.TotalSystemMemoryGiB = &totalMemoryGiB
	}

	// MemoryMirroring is not set as AMT doesn't provide this information
	// It will remain empty unless we have actual mirroring data from hardware

	// Build status only if we have actual CIM status data (OperationalStatus)
	// Don't create status just because we have capacity data
	if hasStatus && memoryHealth != "" {
		memorySummary.Status = &redfishv1.Status{
			Health: memoryHealth,
			// State is left empty as PhysicalMemory CIM doesn't provide EnabledState
		}
	}

	// Only return memorySummary if we have at least some memory data
	if memorySummary.TotalSystemMemoryGiB == nil && memorySummary.Status == nil {
		return nil // No memory data available
	}

	return memorySummary
}

// calculateTotalMemoryGiB sums up memory capacity from all memory modules and converts to GiB.
func (r *WsmanComputerSystemRepo) calculateTotalMemoryGiB(capacityData interface{}) float32 {
	var totalBytes int64

	switch data := capacityData.(type) {
	case []interface{}:
		for _, capacity := range data {
			if bytes, ok := convertToInt(capacity); ok {
				totalBytes += int64(bytes)
			}
		}
	default:
		if bytes, ok := convertToInt(data); ok {
			totalBytes = int64(bytes)
		}
	}

	// Convert bytes to GiB using constant
	return float32(totalBytes) / bytesPerGiB
}

// calculateMemoryHealth determines the worst health state from all memory modules.
func (r *WsmanComputerSystemRepo) calculateMemoryHealth(statusData interface{}) string {
	var worstHealth string

	switch data := statusData.(type) {
	case []interface{}:
		for _, status := range data {
			if health := r.convertOperationalStatusToHealth(status); health != "" {
				if worstHealth == "" {
					worstHealth = health
				} else {
					worstHealth = r.getWorseHealth(worstHealth, health)
				}
			}
		}
	default:
		worstHealth = r.convertOperationalStatusToHealth(statusData)
	}

	return worstHealth // Returns empty string if no valid CIM data found
}

// convertOperationalStatusToHealth converts CIM operational status to Redfish health.
func (r *WsmanComputerSystemRepo) convertOperationalStatusToHealth(status interface{}) string {
	operationalStatus, ok := convertToInt(status)
	if !ok {
		return ""
	}

	switch operationalStatus {
	case CIMStatusOK:
		return healthStateOK
	case CIMStatusDegraded, CIMStatusStressed:
		return healthStateWarning
	case CIMStatusError, CIMStatusNonRecoverableError:
		return healthStateCritical
	default:
		return "" // Unknown status - don't default to any value
	}
}

// getWorseHealth returns the worse of two health states.
func (r *WsmanComputerSystemRepo) getWorseHealth(current, next string) string {
	// Critical is worst, then Warning, then OK
	if current == healthStateCritical || next == healthStateCritical {
		return healthStateCritical
	}

	if current == healthStateWarning || next == healthStateWarning {
		return healthStateWarning
	}

	return healthStateOK
}

// buildProcessorSummaryFromCIMData creates a ProcessorSummary from CIM processor data.
func (r *WsmanComputerSystemRepo) buildProcessorSummaryFromCIMData(cimData map[string]interface{}, hwInfo dto.HardwareInfo) *redfishv1.ComputerSystemProcessorSummary {
	// Extract processor health and state data from CIM properties
	healthStateData, hasHealthState := cimData["HealthState"]
	enabledStateData, hasEnabledState := cimData["EnabledState"]

	// Compute processor count from actual hardware enumeration
	processorCount := len(hwInfo.CIMProcessor.Responses)
	hasProcessorCount := processorCount > 0

	// Check processor model availability from CIM_Chip.Version
	_, hasProcessorModel := cimData["ChipVersion"]

	// Check if we have any processor data available (status info, count info, or model info)
	if !hasHealthState && !hasEnabledState && !hasProcessorCount && !hasProcessorModel {
		return nil // No processor data available
	}

	// Initialize processor summary with basic properties
	processorSummary := &redfishv1.ComputerSystemProcessorSummary{
		// CoreCount, LogicalProcessorCount, and ThreadingEnabled are set to nil
		// because CIM_Processor doesn't provide these properties in Intel AMT WSMAN
		CoreCount:             nil,
		LogicalProcessorCount: nil,
		ThreadingEnabled:      nil,
	}

	// Extract processor model from pre-extracted CIM_Chip.Version data
	if processorModel, ok := cimData["ChipVersion"].(string); ok && processorModel != "" {
		processorSummary.Model = &processorModel
	} else {
		processorSummary.Model = nil
	}

	// Set processor count if available
	if hasProcessorCount {
		processorSummary.Count = &processorCount
	}

	// Build status from CIM data using the common function
	processorSummary.Status = r.buildComponentStatus(healthStateData, hasHealthState, enabledStateData, hasEnabledState, hasProcessorCount, string(CIMObjectProcessor))

	// Set Redfish deprecation annotation for Status property
	deprecationMessage := "Please migrate to use Status in the individual Processor resources"
	processorSummary.StatusRedfishDeprecated = &deprecationMessage

	// Return processorSummary if we have any processor data (Count, Status, or Model)
	if processorSummary.Count == nil && processorSummary.Status == nil && processorSummary.Model == nil {
		return nil // No processor data available
	}

	return processorSummary
}

// buildComponentStatus creates a common status object with health and state information.
// This eliminates code duplication between memory and processor status builders.
func (r *WsmanComputerSystemRepo) buildComponentStatus(healthData interface{}, hasHealth bool, stateData interface{}, hasState, hasComponentData bool, componentName string) *redfishv1.Status {
	componentHealth, componentState := r.extractHealthAndState(healthData, hasHealth, stateData, hasState)

	return r.buildStatusWithDefaults(componentHealth, componentState, hasHealth, hasState, hasComponentData, componentName)
}

// extractHealthAndState extracts health and state strings from interface data.
func (r *WsmanComputerSystemRepo) extractHealthAndState(healthData interface{}, hasHealth bool, stateData interface{}, hasState bool) (componentHealth, componentState string) {
	if hasHealth {
		if health, ok := healthData.(string); ok && health != "" {
			componentHealth = health
		}
	}

	if hasState {
		if state, ok := stateData.(string); ok && state != "" {
			componentState = state
		}
	}

	return componentHealth, componentState
}

// buildStatusWithDefaults creates status object with optional defaults for processors.
func (r *WsmanComputerSystemRepo) buildStatusWithDefaults(componentHealth, componentState string, hasHealth, hasState, hasComponentData bool, componentName string) *redfishv1.Status {
	// If we have valid health or state data, return complete status
	if componentHealth != "" || componentState != "" {
		status := &redfishv1.Status{Health: componentHealth, State: componentState}
		// Set HealthRollup for processors
		if componentName == string(CIMObjectProcessor) && componentHealth != "" {
			status.HealthRollup = componentHealth
		}

		return status
	}

	// For processors, provide default status if we have processor count but no CIM status data
	if hasComponentData && componentName == string(CIMObjectProcessor) {
		status := &redfishv1.Status{}
		if !hasHealth {
			status.Health = healthStateOK
			status.HealthRollup = healthStateOK
		}

		if !hasState {
			status.State = enabledStateEnabled
		}

		return status
	}

	return nil
}

// GetAll retrieves all computer system IDs from the WSMAN backend.
func (r *WsmanComputerSystemRepo) GetAll(ctx context.Context) ([]string, error) {
	// Get devices from the device use case
	items, err := r.usecase.Get(ctx, maxSystemsList, 0, "")
	if err != nil {
		return nil, err
	}

	// Extract just the GUIDs from devices
	systemIDs := make([]string, 0, len(items))
	for i := range items { // avoid value copy
		device := &items[i]
		if device.GUID != "" { // Only append non-empty GUIDs
			systemIDs = append(systemIDs, device.GUID)
		}
	}

	return systemIDs, nil
}

// GetByID retrieves a computer system by its ID from the WSMAN backend.
func (r *WsmanComputerSystemRepo) GetByID(ctx context.Context, systemID string) (*redfishv1.ComputerSystem, error) {
	// Verify device exists first
	device, err := r.usecase.GetByID(ctx, systemID, "", true)
	if r.isDeviceNotFoundError(err) {
		return nil, ErrSystemNotFound
	}

	if err != nil {
		return nil, err
	}

	if device == nil {
		return nil, ErrSystemNotFound
	}

	// Get power state from devices use case
	powerState, err := r.usecase.GetPowerState(ctx, systemID)
	if r.isDeviceNotFoundError(err) {
		return nil, ErrSystemNotFound
	}

	if err != nil {
		return nil, err
	}

	// Map the integer power state to Redfish PowerState
	redfishPowerState := r.mapCIMPowerStateToRedfish(powerState.PowerState)

	// Extract CIM data using the global configuration with static transformers
	cimData, hwInfo, err := r.getCIMProperties(ctx, systemID, allCIMConfigs)
	if err != nil {
		return nil, err
	}

	// Build and return the complete ComputerSystem using CIM data and hardware info
	system := r.buildComputerSystemFromCIMData(systemID, redfishPowerState, cimData, hwInfo)

	return system, nil
}

// UpdatePowerState sends a power action command to the specified system via WSMAN.
func (r *WsmanComputerSystemRepo) UpdatePowerState(ctx context.Context, systemID string, resetType redfishv1.PowerState) error {
	// Get the current power state for logging and validation
	currentSystem, err := r.GetByID(ctx, systemID)
	if err != nil {
		return err
	}

	// For certain reset types like PowerCycle and ForceRestart, we don't check current state
	// because they are valid operations regardless of current power state
	if resetType != redfishv1.ResetTypePowerCycle && resetType != redfishv1.ResetTypeForceRestart {
		// Check if the requested state matches the current state
		if currentSystem.PowerState == resetType {
			return ErrPowerStateConflict
		}
	}

	// Map Redfish reset type to WSMAN action
	action, err := r.mapRedfishPowerStateToAction(resetType)
	if err != nil {
		return err
	}

	// Send power action command
	_, err = r.usecase.SendPowerAction(ctx, systemID, action)
	if r.isDeviceNotFoundError(err) {
		return ErrSystemNotFound
	}

	return err
}

// GetBootSettings retrieves the current boot configuration for a system.
func (r *WsmanComputerSystemRepo) GetBootSettings(ctx context.Context, systemID string) (*generated.ComputerSystemBoot, error) {
	// Get current boot data from AMT via devices use case
	bootData, err := r.usecase.GetBootData(ctx, systemID)
	if err != nil {
		if r.isDeviceNotFoundError(err) {
			return nil, ErrSystemNotFound
		}

		r.log.Warn("Failed to get boot data from device", "systemID", systemID, "error", err)

		return nil, ErrBootSettingsNotAvailable
	}

	// Map AMT boot data to Redfish Boot structure
	boot := &generated.ComputerSystemBoot{}

	// Map BootSourceOverrideEnabled
	if bootData.BootMediaIndex == 0 {
		// No override, boot from normal sources
		enabled := generated.ComputerSystemBoot_BootSourceOverrideEnabled{}
		_ = enabled.FromComputerSystemBootSourceOverrideEnabled(generated.ComputerSystemBootSourceOverrideEnabledDisabled)
		boot.BootSourceOverrideEnabled = &enabled
	} else {
		// Boot override is active - assume "Once" for AMT
		enabled := generated.ComputerSystemBoot_BootSourceOverrideEnabled{}
		_ = enabled.FromComputerSystemBootSourceOverrideEnabled(generated.ComputerSystemBootSourceOverrideEnabledOnce)
		boot.BootSourceOverrideEnabled = &enabled
	}

	// Map boot target based on boot configuration
	target := generated.ComputerSystemBoot_BootSourceOverrideTarget{}

	switch {
	case bootData.BIOSSetup:
		_ = target.FromComputerSystemBootSource(generated.ComputerSystemBootSourceBiosSetup)
	case bootData.UseIDER:
		// IDER can be CD or Floppy
		if bootData.IDERBootDevice == 1 {
			_ = target.FromComputerSystemBootSource(generated.ComputerSystemBootSourceCd)
		} else {
			_ = target.FromComputerSystemBootSource(generated.ComputerSystemBootSourceFloppy)
		}
	default:
		// Default or PXE boot - would need additional logic to determine exact source
		_ = target.FromComputerSystemBootSource(generated.ComputerSystemBootSourceNone)
	}

	boot.BootSourceOverrideTarget = &target

	// Map boot mode based on UEFI boot parameters
	mode := generated.ComputerSystemBoot_BootSourceOverrideMode{}

	if bootData.UEFILocalPBABootEnabled || bootData.UEFIHTTPSBootEnabled || len(bootData.UEFIBootParametersArray) > 0 {
		_ = mode.FromComputerSystemBootSourceOverrideMode(generated.UEFI)
	} else {
		_ = mode.FromComputerSystemBootSourceOverrideMode(generated.Legacy)
	}

	boot.BootSourceOverrideMode = &mode

	return boot, nil
}

// UpdateBootSettings updates the boot configuration for a system.
func (r *WsmanComputerSystemRepo) UpdateBootSettings(ctx context.Context, systemID string, boot *generated.ComputerSystemBoot) error {
	// Get current boot data to preserve settings
	bootData, err := r.usecase.GetBootData(ctx, systemID)
	if err != nil {
		if r.isDeviceNotFoundError(err) {
			return ErrSystemNotFound
		}

		return fmt.Errorf("failed to get current boot data: %w", err)
	}

	// Create new boot settings based on current data
	newBootData := r.createBootDataRequest(bootData)

	// Parse and apply boot target
	bootSource, err := r.applyBootTarget(boot, &newBootData)
	if err != nil {
		return err
	}

	// Apply boot mode if specified
	r.applyBootMode(boot, systemID)

	// Use devices use case methods to configure boot
	if err := r.usecase.SetBootData(ctx, systemID, newBootData); err != nil {
		return fmt.Errorf("failed to set boot settings: %w", err)
	}

	// Set boot order if we have a boot source
	if bootSource != "" {
		if err := r.usecase.ChangeBootOrder(ctx, systemID, bootSource); err != nil {
			return fmt.Errorf("failed to change boot order: %w", err)
		}
	}

	r.log.Info("Boot settings updated successfully",
		"systemID", systemID,
		"target", boot.BootSourceOverrideTarget,
		"enabled", boot.BootSourceOverrideEnabled,
		"mode", boot.BootSourceOverrideMode,
	)

	return nil
}

// createBootDataRequest creates a new boot data request from current settings.
func (r *WsmanComputerSystemRepo) createBootDataRequest(bootData amtBoot.BootSettingDataResponse) amtBoot.BootSettingDataRequest {
	return amtBoot.BootSettingDataRequest{
		BIOSLastStatus:         bootData.BIOSLastStatus,
		BIOSPause:              false,
		BIOSSetup:              false,
		BootMediaIndex:         0,
		ConfigurationDataReset: false,
		ElementName:            bootData.ElementName,
		EnforceSecureBoot:      bootData.EnforceSecureBoot,
		FirmwareVerbosity:      bootData.FirmwareVerbosity,
		ForcedProgressEvents:   false,
		InstanceID:             bootData.InstanceID,
		LockKeyboard:           false,
		LockPowerButton:        false,
		LockResetButton:        false,
		LockSleepButton:        false,
		OptionsCleared:         true,
		OwningEntity:           bootData.OwningEntity,
		ReflashBIOS:            false,
		UseIDER:                false,
		UseSOL:                 bootData.UseSOL,
		UseSafeMode:            false,
		UserPasswordBypass:     false,
		SecureErase:            false,
	}
}

// applyBootTarget applies the boot target to the boot data and returns the boot source.
func (r *WsmanComputerSystemRepo) applyBootTarget(boot *generated.ComputerSystemBoot, newBootData *amtBoot.BootSettingDataRequest) (string, error) {
	if boot.BootSourceOverrideTarget == nil {
		return "", nil
	}

	target, err := boot.BootSourceOverrideTarget.AsComputerSystemBootSource()
	if err != nil {
		return "", nil
	}

	switch target {
	case generated.ComputerSystemBootSourceBiosSetup:
		newBootData.BIOSSetup = true

		return "", nil // Clear boot order for BIOS setup
	case generated.ComputerSystemBootSourcePxe:
		return string(cimBoot.PXE), nil
	case generated.ComputerSystemBootSourceCd:
		newBootData.UseIDER = true
		newBootData.IDERBootDevice = 1 // CD-ROM

		return string(cimBoot.CD), nil
	case generated.ComputerSystemBootSourceFloppy:
		newBootData.UseIDER = true
		newBootData.IDERBootDevice = 0 // Floppy

		return "", nil
	case generated.ComputerSystemBootSourceHdd, generated.ComputerSystemBootSourceNone:
		return "", nil // Default boot or clear override
	case generated.ComputerSystemBootSourceUsb:
		return "", ErrUnsupportedBootTarget
	case generated.ComputerSystemBootSourceDiags, generated.ComputerSystemBootSourceRecovery,
		generated.ComputerSystemBootSourceRemoteDrive, generated.ComputerSystemBootSourceSDCard,
		generated.ComputerSystemBootSourceUefiBootNext, generated.ComputerSystemBootSourceUefiHttp,
		generated.ComputerSystemBootSourceUefiShell, generated.ComputerSystemBootSourceUefiTarget,
		generated.ComputerSystemBootSourceUtilities:
		return "", ErrUnsupportedBootTarget
	default:
		return "", ErrUnsupportedBootTarget
	}
}

// applyBootMode logs the requested boot mode.
func (r *WsmanComputerSystemRepo) applyBootMode(boot *generated.ComputerSystemBoot, systemID string) {
	if boot.BootSourceOverrideMode == nil {
		return
	}

	mode, err := boot.BootSourceOverrideMode.AsComputerSystemBootSourceOverrideMode()
	if err != nil {
		return
	}

	switch mode {
	case generated.UEFI:
		r.log.Info("UEFI boot mode requested", "systemID", systemID)
	case generated.Legacy:
		r.log.Info("Legacy boot mode requested", "systemID", systemID)
	}
}
