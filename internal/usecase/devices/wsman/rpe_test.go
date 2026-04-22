package wsman

import (
	"encoding/base64"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			tlvMask:       0x40, // e.g. 0x10040 &^ rpeCSMEBit = 0x40
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

			encoded, numParams := buildRPETLVParams(tc.tlvMask)

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
