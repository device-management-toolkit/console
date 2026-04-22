package devices

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	dtov2 "github.com/device-management-toolkit/console/internal/entity/dto/v2"
)

func TestOcrBootState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ocr      bool
		rpe      bool
		expected int
	}{
		{
			name:     "both disabled",
			ocr:      false,
			rpe:      false,
			expected: enabledStateOCRAndRPEDisabled,
		},
		{
			name:     "OCR only",
			ocr:      true,
			rpe:      false,
			expected: enabledStateOCREnabled,
		},
		{
			name:     "RPE only",
			ocr:      false,
			rpe:      true,
			expected: enabledStateRPEEnabled,
		},
		{
			name:     "both enabled",
			ocr:      true,
			rpe:      true,
			expected: enabledStateOCRAndRPEEnabled,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := ocrBootState(tc.ocr, tc.rpe)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestSyncRPEResults(t *testing.T) {
	t.Parallel()

	src := &dtov2.Features{
		RPE:             true,
		RPESupported:    true,
		RPECaps:         0x45,
		RPESecureErase:  true,
		RPETPMClear:     true,
		RPEClearBIOSNVM: true,
		RPEBIOSReload:   true,
	}

	dst := &dto.Features{}

	syncRPEResults(src, dst)

	require.Equal(t, src.RPE, dst.RPE)
	require.Equal(t, src.RPESupported, dst.RPESupported)
	require.Equal(t, src.RPECaps, dst.RPECaps)
	require.Equal(t, src.RPESecureErase, dst.RPESecureErase)
	require.Equal(t, src.RPETPMClear, dst.RPETPMClear)
	require.Equal(t, src.RPEClearBIOSNVM, dst.RPEClearBIOSNVM)
	require.Equal(t, src.RPEBIOSReload, dst.RPEBIOSReload)
}

func TestSyncRPEResultsZeroValues(t *testing.T) {
	t.Parallel()

	src := &dtov2.Features{}

	dst := &dto.Features{
		RPE:             true,
		RPESupported:    true,
		RPECaps:         0xFF,
		RPESecureErase:  true,
		RPETPMClear:     true,
		RPEClearBIOSNVM: true,
		RPEBIOSReload:   true,
	}

	syncRPEResults(src, dst)

	require.False(t, dst.RPE)
	require.False(t, dst.RPESupported)
	require.Equal(t, 0, dst.RPECaps)
	require.False(t, dst.RPESecureErase)
	require.False(t, dst.RPETPMClear)
	require.False(t, dst.RPEClearBIOSNVM)
	require.False(t, dst.RPEBIOSReload)
}
