package wsman

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/physical"
)

func TestConvertPhysicalMemorySlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []physical.PhysicalMemory
		expected []interface{}
	}{
		{
			name:     "empty slice",
			input:    []physical.PhysicalMemory{},
			expected: []interface{}{},
		},
		{
			name: "single item",
			input: []physical.PhysicalMemory{
				{
					Capacity:     17179869184, // 16 GB
					Manufacturer: "Kingston",
					PartNumber:   "9905700-101.A00G",
				},
			},
			expected: []interface{}{
				physical.PhysicalMemory{
					Capacity:     17179869184,
					Manufacturer: "Kingston",
					PartNumber:   "9905700-101.A00G",
				},
			},
		},
		{
			name: "multiple items",
			input: []physical.PhysicalMemory{
				{
					Capacity:     8589934592, // 8 GB
					Manufacturer: "Samsung",
					PartNumber:   "M471A1K43CB1-CRC",
				},
				{
					Capacity:     8589934592, // 8 GB
					Manufacturer: "Samsung",
					PartNumber:   "M471A1K43CB1-CRC",
				},
			},
			expected: []interface{}{
				physical.PhysicalMemory{
					Capacity:     8589934592,
					Manufacturer: "Samsung",
					PartNumber:   "M471A1K43CB1-CRC",
				},
				physical.PhysicalMemory{
					Capacity:     8589934592,
					Manufacturer: "Samsung",
					PartNumber:   "M471A1K43CB1-CRC",
				},
			},
		},
		{
			name:     "nil slice",
			input:    nil,
			expected: []interface{}{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := convertPhysicalMemorySlice(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}
