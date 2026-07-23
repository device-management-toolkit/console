package mcp

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

// --- Tool input/output types ------------------------------------------------

type kvmScreenSettingsOutput struct {
	Displays []dto.KVMScreenDisplay `json:"displays"`
}

type kvmSetDefaultDisplayInput struct {
	GUID         string `json:"guid" jsonschema:"the device GUID (UUID) as registered in Console"`
	DisplayIndex int    `json:"displayIndex" jsonschema:"the display index to set as the default KVM screen (0-based)"`
}

type kvmStatusOutput struct {
	Enabled   bool `json:"enabled" jsonschema:"whether KVM redirection is currently enabled"`
	Available bool `json:"available" jsonschema:"whether the device firmware supports KVM"`
}

type kvmSetEnabledInput struct {
	GUID    string `json:"guid" jsonschema:"the device GUID (UUID) as registered in Console"`
	Enabled bool   `json:"enabled" jsonschema:"true to enable KVM redirection, false to disable it"`
}

// --- Registration -----------------------------------------------------------

func (c *Controller) registerKVMTools(s *sdk.Server) {
	sdk.AddTool(s, &sdk.Tool{
		Name: "kvm_get_screen_settings",
		Description: "Get the KVM screen/display configuration for an AMT device, including each " +
			"display's resolution, geometry, role, and which is the default KVM screen.",
		Annotations: &sdk.ToolAnnotations{ReadOnlyHint: true},
	}, c.kvmGetScreenSettings)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "kvm_set_default_display",
		Description: "Set which display an AMT device presents as the default KVM screen.",
		Annotations: &sdk.ToolAnnotations{
			ReadOnlyHint:    false,
			DestructiveHint: ptr(false),
			IdempotentHint:  true,
		},
	}, c.kvmSetDefaultDisplay)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "kvm_get_status",
		Description: "Report whether KVM redirection is enabled and whether the device supports it.",
		Annotations: &sdk.ToolAnnotations{ReadOnlyHint: true},
	}, c.kvmGetStatus)

	sdk.AddTool(s, &sdk.Tool{
		Name: "kvm_set_enabled",
		Description: "Enable or disable KVM redirection on an AMT device. Other redirection features " +
			"(SOL, IDE-R) and the user-consent setting are left unchanged.",
		Annotations: &sdk.ToolAnnotations{
			ReadOnlyHint:    false,
			DestructiveHint: ptr(false),
			IdempotentHint:  true,
		},
	}, c.kvmSetEnabled)
}

// --- Handlers ---------------------------------------------------------------

func (c *Controller) kvmGetScreenSettings(ctx context.Context, _ *sdk.CallToolRequest, in guidInput) (*sdk.CallToolResult, kvmScreenSettingsOutput, error) {
	if err := validateGUID(in.GUID); err != nil {
		return errorResult(err), kvmScreenSettingsOutput{}, nil
	}

	settings, err := c.devices.GetKVMScreenSettings(ctx, in.GUID)
	if err != nil {
		c.log.Error(err, "mcp - kvm_get_screen_settings")

		return errorResult(err), kvmScreenSettingsOutput{}, nil
	}

	return nil, kvmScreenSettingsOutput{Displays: settings.Displays}, nil
}

func (c *Controller) kvmSetDefaultDisplay(ctx context.Context, _ *sdk.CallToolRequest, in kvmSetDefaultDisplayInput) (*sdk.CallToolResult, kvmScreenSettingsOutput, error) {
	if err := validateGUID(in.GUID); err != nil {
		return errorResult(err), kvmScreenSettingsOutput{}, nil
	}

	settings, err := c.devices.SetKVMScreenSettings(ctx, in.GUID, dto.KVMScreenSettingsRequest{
		DisplayIndex: in.DisplayIndex,
	})
	if err != nil {
		c.log.Error(err, "mcp - kvm_set_default_display")

		return errorResult(err), kvmScreenSettingsOutput{}, nil
	}

	return nil, kvmScreenSettingsOutput{Displays: settings.Displays}, nil
}

func (c *Controller) kvmGetStatus(ctx context.Context, _ *sdk.CallToolRequest, in guidInput) (*sdk.CallToolResult, kvmStatusOutput, error) {
	if err := validateGUID(in.GUID); err != nil {
		return errorResult(err), kvmStatusOutput{}, nil
	}

	features, _, err := c.devices.GetFeatures(ctx, in.GUID)
	if err != nil {
		c.log.Error(err, "mcp - kvm_get_status")

		return errorResult(err), kvmStatusOutput{}, nil
	}

	return nil, kvmStatusOutput{
		Enabled:   features.EnableKVM,
		Available: features.KVMAvailable,
	}, nil
}

func (c *Controller) kvmSetEnabled(ctx context.Context, _ *sdk.CallToolRequest, in kvmSetEnabledInput) (*sdk.CallToolResult, kvmStatusOutput, error) {
	if err := validateGUID(in.GUID); err != nil {
		return errorResult(err), kvmStatusOutput{}, nil
	}

	// Read the current feature set so only EnableKVM is changed; SOL, IDE-R,
	// OCR, remote-erase, and the user-consent mode are preserved.
	current, _, err := c.devices.GetFeatures(ctx, in.GUID)
	if err != nil {
		c.log.Error(err, "mcp - kvm_set_enabled (get)")

		return errorResult(err), kvmStatusOutput{}, nil
	}

	current.EnableKVM = in.Enabled

	updated, _, err := c.devices.SetFeatures(ctx, in.GUID, current)
	if err != nil {
		c.log.Error(err, "mcp - kvm_set_enabled (set)")

		return errorResult(err), kvmStatusOutput{}, nil
	}

	return nil, kvmStatusOutput{
		Enabled:   updated.EnableKVM,
		Available: updated.KVMAvailable,
	}, nil
}
