package devices

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

// rpc-go reports the OEM serial number as deviceInfo.serialNumber. A sync
// that sends only that sub-field must store it and leave the rest alone.
func TestMergeDeviceInfoSerialNumber(t *testing.T) {
	t.Parallel()

	stored := &dto.Device{DeviceInfo: &dto.DeviceInfo{FWVersion: "16.1.30", SerialNumber: "OLD123"}}
	incoming := &dto.Device{DeviceInfo: &dto.DeviceInfo{FWVersion: "ignored", SerialNumber: "PF4ABCDE"}}

	mergeDeviceInfo(stored, incoming, map[string]bool{
		"deviceinfo":              true,
		"deviceinfo.serialnumber": true,
	})

	require.Equal(t, "PF4ABCDE", stored.DeviceInfo.SerialNumber)
	require.Equal(t, "16.1.30", stored.DeviceInfo.FWVersion)
}

func TestMergeDeviceInfoKeepsSerialNumberWhenNotSent(t *testing.T) {
	t.Parallel()

	stored := &dto.Device{DeviceInfo: &dto.DeviceInfo{FWVersion: "16.1.30", SerialNumber: "PF4ABCDE"}}
	incoming := &dto.Device{DeviceInfo: &dto.DeviceInfo{FWVersion: "16.1.35"}}

	mergeDeviceInfo(stored, incoming, map[string]bool{
		"deviceinfo":           true,
		"deviceinfo.fwversion": true,
	})

	require.Equal(t, "16.1.35", stored.DeviceInfo.FWVersion)
	require.Equal(t, "PF4ABCDE", stored.DeviceInfo.SerialNumber)
}
