package mcp

import (
	"testing"

	"github.com/stretchr/testify/require"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

func TestPowerActionCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		action   string
		wantCode int
		wantOK   bool
	}{
		{name: "power_on", action: "power_on", wantCode: 2, wantOK: true},
		{name: "power_off", action: "power_off", wantCode: 8, wantOK: true},
		{name: "reset", action: "reset", wantCode: 10, wantOK: true},
		{name: "power_cycle", action: "power_cycle", wantCode: 5, wantOK: true},
		{name: "os_to_full_power", action: "os_to_full_power", wantCode: 500, wantOK: true},
		{name: "unknown", action: "explode", wantCode: 0, wantOK: false},
		{name: "empty", action: "", wantCode: 0, wantOK: false},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			code, ok := powerActionCode(tt.action)
			require.Equal(t, tt.wantOK, ok)
			require.Equal(t, tt.wantCode, code)
		})
	}
}

func TestSortedPowerActionNames(t *testing.T) {
	t.Parallel()

	names := sortedPowerActionNames()
	require.NotEmpty(t, names)
	require.Len(t, names, len(powerActionNames))

	// verify stable, sorted order
	for i := 1; i < len(names); i++ {
		require.Less(t, names[i-1], names[i])
	}
}

func TestCapabilityActionNames(t *testing.T) {
	t.Parallel()

	t.Run("maps non-zero capability codes to names", func(t *testing.T) {
		t.Parallel()

		caps := dto.PowerCapabilities{
			PowerUp:   2,  // power_on
			PowerDown: 8,  // power_off
			Reset:     10, // reset
		}

		names := capabilityActionNames(caps)
		require.ElementsMatch(t, []string{"power_on", "power_off", "reset"}, names)
	})

	t.Run("ignores zero-valued capabilities", func(t *testing.T) {
		t.Parallel()

		names := capabilityActionNames(dto.PowerCapabilities{})
		require.Empty(t, names)
	})

	t.Run("full graceful set", func(t *testing.T) {
		t.Parallel()

		caps := dto.PowerCapabilities{
			PowerUp:    2,
			PowerCycle: 5,
			PowerDown:  8,
			Reset:      10,
			SoftOff:    12,
			SoftReset:  14,
			Sleep:      4,
			Hibernate:  7,
		}

		names := capabilityActionNames(caps)
		require.ElementsMatch(t, []string{
			"power_on", "power_cycle", "power_off", "reset",
			"soft_off", "soft_reset", "sleep", "hibernate",
		}, names)
	})
}
