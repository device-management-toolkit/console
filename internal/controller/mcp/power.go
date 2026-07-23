package mcp

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

// --- Tool input/output types ------------------------------------------------

type guidInput struct {
	GUID string `json:"guid" jsonschema:"the device GUID (UUID) as registered in Console"`
}

type powerStateOutput struct {
	PowerState         int `json:"powerState" jsonschema:"CIM power state value"`
	OSPowerSavingState int `json:"osPowerSavingState" jsonschema:"OS power-saving state (0 = unknown)"`
}

type powerCapabilitiesOutput struct {
	SupportedActions []string `json:"supportedActions" jsonschema:"power_action names this device supports"`
}

type powerActionInput struct {
	GUID   string `json:"guid" jsonschema:"the device GUID (UUID) as registered in Console"`
	Action string `json:"action" jsonschema:"the power action to perform"`
}

type powerActionOutput struct {
	ReturnValue int `json:"returnValue" jsonschema:"WSMAN return code; 0 indicates success"`
}

type bootSourceOutput struct {
	Sources []dto.BootSources `json:"sources"`
}

// --- Registration -----------------------------------------------------------

func (c *Controller) registerPowerTools(s *sdk.Server) {
	sdk.AddTool(s, &sdk.Tool{
		Name:        "power_get_state",
		Description: "Get the current power state of an AMT device, including its OS power-saving state.",
		Annotations: &sdk.ToolAnnotations{ReadOnlyHint: true},
	}, c.powerGetState)

	sdk.AddTool(s, &sdk.Tool{
		Name: "power_get_capabilities",
		Description: "List the power actions this AMT device supports. " +
			"Call this before power_action to discover valid action names for the device.",
		Annotations: &sdk.ToolAnnotations{ReadOnlyHint: true},
	}, c.powerGetCapabilities)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "power_get_boot_sources",
		Description: "List the configured boot sources (boot order entries) for an AMT device.",
		Annotations: &sdk.ToolAnnotations{ReadOnlyHint: true},
	}, c.powerGetBootSources)

	sdk.AddTool(s, &sdk.Tool{
		Name: "power_action",
		Description: "Perform a power action on physical AMT hardware (e.g. power_on, power_off, " +
			"reset, power_cycle). WARNING: this reboots or powers off a real machine and can " +
			"interrupt a running operating system. Valid actions: " +
			strings.Join(sortedPowerActionNames(), ", ") + ".",
		Annotations: &sdk.ToolAnnotations{
			ReadOnlyHint:    false,
			DestructiveHint: ptr(true),
			IdempotentHint:  false,
		},
	}, c.powerAction)
}

// --- Handlers ---------------------------------------------------------------

func (c *Controller) powerGetState(ctx context.Context, _ *sdk.CallToolRequest, in guidInput) (*sdk.CallToolResult, powerStateOutput, error) {
	if err := validateGUID(in.GUID); err != nil {
		return errorResult(err), powerStateOutput{}, nil
	}

	state, err := c.devices.GetPowerState(ctx, in.GUID)
	if err != nil {
		c.log.Error(err, "mcp - power_get_state")

		return errorResult(err), powerStateOutput{}, nil
	}

	return nil, powerStateOutput{
		PowerState:         state.PowerState,
		OSPowerSavingState: state.OSPowerSavingState,
	}, nil
}

func (c *Controller) powerGetCapabilities(ctx context.Context, _ *sdk.CallToolRequest, in guidInput) (*sdk.CallToolResult, powerCapabilitiesOutput, error) {
	if err := validateGUID(in.GUID); err != nil {
		return errorResult(err), powerCapabilitiesOutput{}, nil
	}

	caps, err := c.devices.GetPowerCapabilities(ctx, in.GUID)
	if err != nil {
		c.log.Error(err, "mcp - power_get_capabilities")

		return errorResult(err), powerCapabilitiesOutput{}, nil
	}

	return nil, powerCapabilitiesOutput{SupportedActions: capabilityActionNames(caps)}, nil
}

func (c *Controller) powerGetBootSources(ctx context.Context, _ *sdk.CallToolRequest, in guidInput) (*sdk.CallToolResult, bootSourceOutput, error) {
	if err := validateGUID(in.GUID); err != nil {
		return errorResult(err), bootSourceOutput{}, nil
	}

	sources, err := c.devices.GetBootSourceSetting(ctx, in.GUID)
	if err != nil {
		c.log.Error(err, "mcp - power_get_boot_sources")

		return errorResult(err), bootSourceOutput{}, nil
	}

	return nil, bootSourceOutput{Sources: sources}, nil
}

func (c *Controller) powerAction(ctx context.Context, _ *sdk.CallToolRequest, in powerActionInput) (*sdk.CallToolResult, powerActionOutput, error) {
	if err := validateGUID(in.GUID); err != nil {
		return errorResult(err), powerActionOutput{}, nil
	}

	code, ok := powerActionCode(in.Action)
	if !ok {
		err := fmt.Errorf("%w %q; valid actions: %s",
			ErrUnknownPowerAction, in.Action, strings.Join(sortedPowerActionNames(), ", "))

		return errorResult(err), powerActionOutput{}, nil
	}

	resp, err := c.devices.SendPowerAction(ctx, in.GUID, code)
	if err != nil {
		c.log.Error(err, "mcp - power_action")

		return errorResult(err), powerActionOutput{}, nil
	}

	return nil, powerActionOutput{ReturnValue: int(resp.ReturnValue)}, nil
}
