package devices

import (
	"testing"

	"github.com/stretchr/testify/require"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

func TestAdoptionFor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		info *dto.DeviceInfo
		want dto.Adoption
	}{
		{"no device info", nil, dto.Adoption{Reason: "unknown"}},
		{"not activated", &dto.DeviceInfo{CurrentMode: "Not Activated", FWSku: "16392"}, dto.Adoption{Reason: "notActivated"}},
		{"empty mode", &dto.DeviceInfo{FWSku: "16392"}, dto.Adoption{Reason: "notActivated"}},
		{"pre-provisioning", &dto.DeviceInfo{CurrentMode: "Pre-Provisioning State", FWSku: "16392"}, dto.Adoption{Reason: "notActivated"}},
		{"missing sku", &dto.DeviceInfo{CurrentMode: "Admin Control Mode"}, dto.Adoption{Reason: "unknown"}},
		{"no AMT data at all", &dto.DeviceInfo{}, dto.Adoption{Reason: "unknown"}},
		{"no sku, not activated", &dto.DeviceInfo{CurrentMode: "Not Activated"}, dto.Adoption{Reason: "notActivated"}},
		{"non-vPro beats not activated", &dto.DeviceInfo{CurrentMode: "Not Activated", FWSku: "1"}, dto.Adoption{Reason: "unsupported"}},
		{"non-vPro with no mode", &dto.DeviceInfo{FWSku: "1"}, dto.Adoption{Reason: "unsupported"}},
		{"non-vPro", &dto.DeviceInfo{CurrentMode: "Admin Control Mode", FWSku: "1"}, dto.Adoption{Reason: "unsupported"}},
		{"unrecognized mode", &dto.DeviceInfo{CurrentMode: "weird", FWSku: "16392"}, dto.Adoption{Reason: "unknown"}},
		{"vPro admin control", &dto.DeviceInfo{CurrentMode: "Admin Control Mode", FWSku: "16392"}, dto.Adoption{Eligible: true}},
		{"ISM client control", &dto.DeviceInfo{CurrentMode: "Client Control Mode", FWSku: "16"}, dto.Adoption{Eligible: true}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.want, *adoptionFor(tc.info))
		})
	}
}
