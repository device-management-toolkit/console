package wsman

import (
	"encoding/base64"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/power"
)

const (
	intelVendorPrefix = 0x8086
	rpeTLVValueLen    = 4
)

func TestBuildRPETLVParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		tlvMask       int
		wantNumParams int
		wantVendor    uint16
		wantTypeID    uint16
		wantValueLen  uint32
		wantMaskInTLV uint32
	}{
		{
			name:          "TPM Clear only (0x40)",
			tlvMask:       0x40,
			wantNumParams: 1,
			wantVendor:    intelVendorPrefix,
			wantTypeID:    1,
			wantValueLen:  rpeTLVValueLen,
			wantMaskInTLV: 0x40,
		},
		{
			name:          "BIOS Reload only (0x4000000)",
			tlvMask:       0x4000000,
			wantNumParams: 1,
			wantVendor:    intelVendorPrefix,
			wantTypeID:    1,
			wantValueLen:  rpeTLVValueLen,
			wantMaskInTLV: 0x4000000,
		},
		{
			name:          "Clear BIOS NVM only (0x2000000)",
			tlvMask:       0x2000000,
			wantNumParams: 1,
			wantVendor:    intelVendorPrefix,
			wantTypeID:    1,
			wantValueLen:  rpeTLVValueLen,
			wantMaskInTLV: 0x2000000,
		},
		{
			name:          "SSD secure erase only (0x04)",
			tlvMask:       0x04,
			wantNumParams: 1,
			wantVendor:    intelVendorPrefix,
			wantTypeID:    1,
			wantValueLen:  rpeTLVValueLen,
			wantMaskInTLV: 0x04,
		},
		{
			name:          "TPM + SSD + BIOS NVM + BIOS Reload combined",
			tlvMask:       0x40 | 0x04 | 0x2000000 | 0x4000000,
			wantNumParams: 1,
			wantVendor:    intelVendorPrefix,
			wantTypeID:    1,
			wantValueLen:  rpeTLVValueLen,
			wantMaskInTLV: 0x40 | 0x04 | 0x2000000 | 0x4000000,
		},
		{
			name:          "CSME bit already stripped (0x10000 absent from tlvMask)",
			tlvMask:       0x40, // e.g. 0x10040 &^ RPEConfigurationDataResetSignalBit = 0x40
			wantNumParams: 1,
			wantVendor:    intelVendorPrefix,
			wantTypeID:    1,
			wantValueLen:  rpeTLVValueLen,
			wantMaskInTLV: 0x40,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			encoded, numParams, err := buildRPETLVParams(tc.tlvMask, RPECredentials{})
			require.NoError(t, err)

			assert.Equal(t, tc.wantNumParams, numParams)

			raw, err := base64.StdEncoding.DecodeString(encoded)
			require.NoError(t, err, "encoded params must be valid base64")
			require.Len(t, raw, 12, "TLV buffer must be exactly 12 bytes")

			gotVendor := binary.LittleEndian.Uint16(raw[0:2])
			gotTypeID := binary.LittleEndian.Uint16(raw[2:4])
			gotValueLen := binary.LittleEndian.Uint32(raw[4:8])
			gotMask := binary.LittleEndian.Uint32(raw[8:12])

			assert.Equal(t, tc.wantVendor, gotVendor, "vendor prefix")
			assert.Equal(t, tc.wantTypeID, gotTypeID, "ParameterTypeID")
			assert.Equal(t, tc.wantValueLen, gotValueLen, "value length")
			assert.Equal(t, tc.wantMaskInTLV, gotMask, "device bitmask in TLV value")
		})
	}
}

func TestBuildRPETLVParamsCredentials(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		password   string
		psid       string
		wantTypes  []uint16
		wantValues []string
	}{
		{"password", "master-password", "", []uint16{20}, []string{"master-password"}},
		{"pyrite PSID", "", "drive-psid", []uint16{10}, []string{"drive-psid"}},
		{"both", "password", "psid", []uint16{10, 20}, []string{"psid", "password"}},
		{"maximum multibyte UTF-8 length", strings.Repeat("\xc3\xa9", 32), "", []uint16{20}, []string{strings.Repeat("\xc3\xa9", 32)}},
	} {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			encoded, count, err := buildRPETLVParams(4, RPECredentials{SSDPassword: tc.password, PSID: tc.psid})
			require.NoError(t, err)
			require.Equal(t, 1+len(tc.wantTypes), count)

			raw, err := base64.StdEncoding.DecodeString(encoded)
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(raw), 12)
			require.Equal(t, uint32(4), binary.LittleEndian.Uint32(raw[8:12]))

			remaining := raw[12:]
			for index, typeID := range tc.wantTypes {
				require.GreaterOrEqual(t, len(remaining), 8)
				assert.Equal(t, uint16(intelVendorPrefix), binary.LittleEndian.Uint16(remaining[:2]))
				assert.Equal(t, typeID, binary.LittleEndian.Uint16(remaining[2:4]))

				length := int(binary.LittleEndian.Uint32(remaining[4:8]))
				require.Equal(t, len(tc.wantValues[index]), length)
				require.GreaterOrEqual(t, len(remaining), 8+length)
				assert.Equal(t, tc.wantValues[index], string(remaining[8:8+length]))
				remaining = remaining[8+length:]
			}

			require.Empty(t, remaining)
		})
	}
}

func TestSelectRPEPowerAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		state        int
		expectsPower power.PowerState
	}{
		{
			name:         "off hard selects power on (2)",
			state:        currentPowerStateOffHard,
			expectsPower: power.PowerOn,
		},
		{
			name:         "off soft selects power on (2)",
			state:        currentPowerStateOffSoft,
			expectsPower: power.PowerOn,
		},
		{
			name:         "off soft graceful selects power on (2)",
			state:        currentPowerStatePowerOffSoftGraceful,
			expectsPower: power.PowerOn,
		},
		{
			name:         "off hard graceful selects power on (2)",
			state:        currentPowerStatePowerOffHardGraceful,
			expectsPower: power.PowerOn,
		},
		{
			name:         "on selects hard cycle (5)",
			state:        2,
			expectsPower: power.PowerCycleOffHard,
		},
		{
			name:         "sleep selects hard cycle (5)",
			state:        3,
			expectsPower: power.PowerCycleOffHard,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expectsPower, selectRPEPowerAction(tc.state))
		})
	}
}
