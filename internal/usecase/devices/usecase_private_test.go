package devices

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/internal/entity"
	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	crypto "github.com/device-management-toolkit/console/internal/mocks/crypto"
	"github.com/device-management-toolkit/console/pkg/logger"
)

func TestDtoToEntity_DeviceInfoSerialization(t *testing.T) {
	t.Parallel()

	uc := &UseCase{log: logger.New("error"), safeRequirements: crypto.MockCrypto{}}
	lms := true

	t.Run("returns empty DeviceInfo when nil", func(t *testing.T) {
		t.Parallel()

		d := &dto.Device{GUID: "g1", TenantID: "t1", DeviceInfo: nil}
		ent, err := uc.dtoToEntity(d)
		require.NoError(t, err)
		require.Empty(t, ent.DeviceInfo)
	})

	t.Run("serializes LMSInstalled", func(t *testing.T) {
		t.Parallel()

		d := &dto.Device{GUID: "g1", TenantID: "t1", DeviceInfo: &dto.DeviceInfo{LMSInstalled: &lms}}
		ent, err := uc.dtoToEntity(d)
		require.NoError(t, err)
		require.Contains(t, ent.DeviceInfo, `"lmsInstalled":true`)
	})

	t.Run("serializes DeviceInfo with multiple fields", func(t *testing.T) {
		t.Parallel()

		d := &dto.Device{
			GUID:       "g2",
			TenantID:   "t1",
			DeviceInfo: &dto.DeviceInfo{FWVersion: "16.1.25", LMSInstalled: &lms},
		}
		ent, err := uc.dtoToEntity(d)
		require.NoError(t, err)
		require.Contains(t, ent.DeviceInfo, `"lmsInstalled":true`)
		require.Contains(t, ent.DeviceInfo, `"fwVersion":"16.1.25"`)
	})
}

func TestDtoToEntity_PreservesDiscoveryState(t *testing.T) {
	t.Parallel()

	uc := &UseCase{log: logger.New("error"), safeRequirements: crypto.MockCrypto{}}
	discovered := true
	managed := false

	tests := []struct {
		name        string
		currentMode string
		discovered  *bool
	}{
		{name: "RPC discovery remains discovered while pre-provisioning", currentMode: "not activated", discovered: &discovered},
		{name: "RPC activation is managed", currentMode: "admin control mode", discovered: &managed},
		{name: "synced ACM device retains its discovery flag", currentMode: "admin control mode", discovered: &discovered},
		{name: "synced CCM device retains its discovery flag", currentMode: "client control mode", discovered: &discovered},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			entityDevice, err := uc.dtoToEntity(&dto.Device{
				GUID:       "g1",
				TenantID:   "t1",
				DeviceInfo: &dto.DeviceInfo{CurrentMode: test.currentMode, Discovered: test.discovered},
			})
			require.NoError(t, err)
			require.NotNil(t, entityDevice.Discovered)
			require.Equal(t, *test.discovered, *entityDevice.Discovered)
			require.Contains(t, entityDevice.DeviceInfo, `"discovered":`)
		})
	}
}

func TestEntityToDTO_DeviceInfoDeserialization(t *testing.T) {
	t.Parallel()

	uc := &UseCase{log: logger.New("error"), safeRequirements: crypto.MockCrypto{}}

	ent := &entity.Device{
		GUID:       "guid-1",
		TenantID:   "t-1",
		DeviceInfo: `{"fwVersion":"16.1.25","lmsInstalled":true}`,
	}

	d, err := uc.entityToDTO(ent)
	require.NoError(t, err)
	require.NotNil(t, d.DeviceInfo)
	require.Equal(t, "16.1.25", d.DeviceInfo.FWVersion)
	require.NotNil(t, d.DeviceInfo.LMSInstalled)
	require.True(t, *d.DeviceInfo.LMSInstalled)
}

func TestMergeDeviceFields_NewSetters(t *testing.T) {
	t.Parallel()

	lms := true
	info := &dto.DeviceInfo{FWVersion: "16.1.25", LMSInstalled: &lms}
	src := &dto.Device{DeviceInfo: info}
	dst := &dto.Device{}

	mergeDeviceFields(dst, src, map[string]bool{"deviceinfo": true})
	require.Equal(t, info, dst.DeviceInfo)
}
