package mcp

import (
	"context"
	"errors"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/power"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	dtov2 "github.com/device-management-toolkit/console/internal/entity/dto/v2"
	"github.com/device-management-toolkit/console/internal/mocks"
	"github.com/device-management-toolkit/console/pkg/logger"
)

const testGUID = "038d0240-0f01-4a4b-9c8c-6c2f5b1234ab"

// newTestSession wires an in-memory MCP client to a server backed by the mock
// device feature and returns a connected client session for CallTool.
func newTestSession(t *testing.T, dev *mocks.MockDeviceManagementFeature) *sdk.ClientSession {
	t.Helper()

	ctx := context.Background()
	server := NewServer(dev, logger.New("error"))

	clientTransport, serverTransport := sdk.NewInMemoryTransports()

	_, err := server.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)

	client := sdk.NewClient(&sdk.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)

	session, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)

	t.Cleanup(func() { _ = session.Close() })

	return session
}

func callTool(t *testing.T, session *sdk.ClientSession, name string, args any) *sdk.CallToolResult {
	t.Helper()

	res, err := session.CallTool(context.Background(), &sdk.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	require.NoError(t, err)

	return res
}

func TestPowerTools(t *testing.T) {
	t.Parallel()

	t.Run("power_get_state success", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		dev := mocks.NewMockDeviceManagementFeature(ctrl)
		dev.EXPECT().GetPowerState(gomock.Any(), testGUID).
			Return(dto.PowerState{PowerState: 2, OSPowerSavingState: 1}, nil)

		session := newTestSession(t, dev)
		res := callTool(t, session, "power_get_state", map[string]any{"guid": testGUID})

		require.False(t, res.IsError)

		out, ok := res.StructuredContent.(map[string]any)
		require.True(t, ok)
		require.EqualValues(t, 2, out["powerState"])
		require.EqualValues(t, 1, out["osPowerSavingState"])
	})

	t.Run("power_get_state invalid guid", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		dev := mocks.NewMockDeviceManagementFeature(ctrl)
		// No feature call expected — validation fails first.

		session := newTestSession(t, dev)
		res := callTool(t, session, "power_get_state", map[string]any{"guid": "not-a-uuid"})

		require.True(t, res.IsError)
	})

	t.Run("power_get_capabilities success", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		dev := mocks.NewMockDeviceManagementFeature(ctrl)
		dev.EXPECT().GetPowerCapabilities(gomock.Any(), testGUID).
			Return(dto.PowerCapabilities{PowerUp: 2, PowerDown: 8, Reset: 10}, nil)

		session := newTestSession(t, dev)
		res := callTool(t, session, "power_get_capabilities", map[string]any{"guid": testGUID})

		require.False(t, res.IsError)

		out, ok := res.StructuredContent.(map[string]any)
		require.True(t, ok)

		actions, ok := out["supportedActions"].([]any)
		require.True(t, ok)
		require.Len(t, actions, 3)
	})

	t.Run("power_action success", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		dev := mocks.NewMockDeviceManagementFeature(ctrl)
		dev.EXPECT().SendPowerAction(gomock.Any(), testGUID, 8).
			Return(power.PowerActionResponse{ReturnValue: 0}, nil)

		session := newTestSession(t, dev)
		res := callTool(t, session, "power_action", map[string]any{"guid": testGUID, "action": "power_off"})

		require.False(t, res.IsError)

		out, ok := res.StructuredContent.(map[string]any)
		require.True(t, ok)
		require.EqualValues(t, 0, out["returnValue"])
	})

	t.Run("power_action unknown action", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		dev := mocks.NewMockDeviceManagementFeature(ctrl)
		// No SendPowerAction expected.

		session := newTestSession(t, dev)
		res := callTool(t, session, "power_action", map[string]any{"guid": testGUID, "action": "explode"})

		require.True(t, res.IsError)
	})

	t.Run("power_action feature error", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		dev := mocks.NewMockDeviceManagementFeature(ctrl)
		dev.EXPECT().SendPowerAction(gomock.Any(), testGUID, 10).
			Return(power.PowerActionResponse{}, errors.New("device unreachable"))

		session := newTestSession(t, dev)
		res := callTool(t, session, "power_action", map[string]any{"guid": testGUID, "action": "reset"})

		require.True(t, res.IsError)
	})
}

func TestKVMTools(t *testing.T) {
	t.Parallel()

	t.Run("kvm_get_screen_settings success", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		dev := mocks.NewMockDeviceManagementFeature(ctrl)
		dev.EXPECT().GetKVMScreenSettings(gomock.Any(), testGUID).
			Return(dto.KVMScreenSettings{Displays: []dto.KVMScreenDisplay{{DisplayIndex: 0, IsActive: true}}}, nil)

		session := newTestSession(t, dev)
		res := callTool(t, session, "kvm_get_screen_settings", map[string]any{"guid": testGUID})

		require.False(t, res.IsError)
	})

	t.Run("kvm_set_default_display success", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		dev := mocks.NewMockDeviceManagementFeature(ctrl)
		dev.EXPECT().SetKVMScreenSettings(gomock.Any(), testGUID, dto.KVMScreenSettingsRequest{DisplayIndex: 1}).
			Return(dto.KVMScreenSettings{Displays: []dto.KVMScreenDisplay{{DisplayIndex: 1, IsDefault: true}}}, nil)

		session := newTestSession(t, dev)
		res := callTool(t, session, "kvm_set_default_display", map[string]any{"guid": testGUID, "displayIndex": 1})

		require.False(t, res.IsError)
	})

	t.Run("kvm_get_status success", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		dev := mocks.NewMockDeviceManagementFeature(ctrl)
		dev.EXPECT().GetFeatures(gomock.Any(), testGUID).
			Return(dto.Features{EnableKVM: true, KVMAvailable: true}, dtov2.Features{}, nil)

		session := newTestSession(t, dev)
		res := callTool(t, session, "kvm_get_status", map[string]any{"guid": testGUID})

		require.False(t, res.IsError)

		out, ok := res.StructuredContent.(map[string]any)
		require.True(t, ok)
		require.Equal(t, true, out["enabled"])
		require.Equal(t, true, out["available"])
	})

	t.Run("kvm_set_enabled preserves other features", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		dev := mocks.NewMockDeviceManagementFeature(ctrl)
		// Current features: KVM off, SOL on. Enabling KVM must keep SOL on.
		dev.EXPECT().GetFeatures(gomock.Any(), testGUID).
			Return(dto.Features{EnableKVM: false, EnableSOL: true, KVMAvailable: true}, dtov2.Features{}, nil)
		dev.EXPECT().SetFeatures(gomock.Any(), testGUID, dto.Features{EnableKVM: true, EnableSOL: true, KVMAvailable: true}).
			Return(dto.Features{EnableKVM: true, EnableSOL: true, KVMAvailable: true}, dtov2.Features{}, nil)

		session := newTestSession(t, dev)
		res := callTool(t, session, "kvm_set_enabled", map[string]any{"guid": testGUID, "enabled": true})

		require.False(t, res.IsError)

		out, ok := res.StructuredContent.(map[string]any)
		require.True(t, ok)
		require.Equal(t, true, out["enabled"])
	})
}

func TestListTools(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dev := mocks.NewMockDeviceManagementFeature(ctrl)
	session := newTestSession(t, dev)

	res, err := session.ListTools(context.Background(), nil)
	require.NoError(t, err)

	names := make(map[string]bool)
	for _, tool := range res.Tools {
		names[tool.Name] = true
	}

	for _, want := range []string{
		"power_get_state", "power_get_capabilities", "power_get_boot_sources", "power_action",
		"kvm_get_screen_settings", "kvm_set_default_display", "kvm_get_status", "kvm_set_enabled",
	} {
		require.True(t, names[want], "expected tool %q to be registered", want)
	}
}
