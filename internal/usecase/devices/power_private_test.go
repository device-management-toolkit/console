package devices

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/boot"
	cimBoot "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/boot"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/software"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

type powerTest struct {
	name string
	res  any
	err  error

	amtVersion   int
	capabilities boot.BootCapabilitiesResponse
	bootSettings dto.BootSetting
	version      []software.SoftwareIdentity
}

func TestDeterminePowerCapabilities(t *testing.T) {
	t.Parallel()

	tests := []powerTest{
		{
			name:       "AMT version 10",
			amtVersion: 10,
			capabilities: boot.BootCapabilitiesResponse{
				BIOSReflash:         true,
				BIOSSetup:           false,
				SecureErase:         false,
				ForceDiagnosticBoot: true,
			},
			res: dto.PowerCapabilities{
				PowerUp:             2,
				PowerCycle:          5,
				PowerDown:           8,
				Reset:               10,
				SoftOff:             12,
				SoftReset:           14,
				Sleep:               4,
				Hibernate:           7,
				ResetToIDERFloppy:   200,
				PowerOnToIDERFloppy: 201,
				ResetToIDERCDROM:    202,
				PowerOnToIDERCDROM:  203,
				PowerOnToDiagnostic: 300,
				ResetToDiagnostic:   301,
				ResetToPXE:          400,
				PowerOnToPXE:        401,
			},
		},
		{
			name:       "AMT version 7",
			amtVersion: 7,
			capabilities: boot.BootCapabilitiesResponse{
				BIOSReflash:         false,
				BIOSSetup:           true,
				SecureErase:         true,
				ForceDiagnosticBoot: false,
			},
			res: dto.PowerCapabilities{
				PowerUp:             2,
				PowerCycle:          5,
				PowerDown:           8,
				Reset:               10,
				PowerOnToBIOS:       100,
				ResetToBIOS:         101,
				ResetToSecureErase:  104,
				ResetToIDERFloppy:   200,
				PowerOnToIDERFloppy: 201,
				ResetToIDERCDROM:    202,
				PowerOnToIDERCDROM:  203,
				ResetToPXE:          400,
				PowerOnToPXE:        401,
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res := determinePowerCapabilities(tc.amtVersion, tc.capabilities)

			require.Equal(t, tc.res, res)
		})
	}
}

func TestDetermineIDERBootDevice(t *testing.T) {
	t.Parallel()

	tests := []powerTest{
		{
			name: "Master Bus Reset",
			res: boot.BootSettingDataRequest{
				IDERBootDevice: 1,
			},
			bootSettings: dto.BootSetting{
				Action: 202,
			},
		},
		{
			name: "Power On",
			res: boot.BootSettingDataRequest{
				IDERBootDevice: 0,
			},
			bootSettings: dto.BootSetting{
				Action: 999,
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := boot.BootSettingDataRequest{
				IDERBootDevice: 999,
			}

			err := determineBootDevice(tc.bootSettings, &result)

			require.NoError(t, err)
			require.Equal(t, tc.res, result)
		})
	}
}

func TestGetBootSource(t *testing.T) {
	t.Parallel()

	tests := []powerTest{
		{
			name: "Action 400",
			res:  string(cimBoot.PXE),
			bootSettings: dto.BootSetting{
				Action: 400,
			},
		},
		{
			name: "Action 202",
			res:  string(cimBoot.CD),
			bootSettings: dto.BootSetting{
				Action: 202,
			},
		},
		{
			name: "Action 999",
			res:  "",
			bootSettings: dto.BootSetting{
				Action: 999,
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res := getBootSource(tc.bootSettings)

			require.Equal(t, tc.res, res)
		})
	}
}

func TestDetermineBootAction(t *testing.T) {
	t.Parallel()

	tests := []powerTest{
		{
			name: "Master Bus Reset",
			res:  10,
			bootSettings: dto.BootSetting{
				Action: 200,
			},
		},
		{
			name: "Power On",
			res:  2,
			bootSettings: dto.BootSetting{
				Action: 999,
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			determineBootAction(&tc.bootSettings)

			require.Equal(t, tc.res, tc.bootSettings.Action)
		})
	}
}

func TestParseVersion(t *testing.T) {
	t.Parallel()

	tests := []powerTest{
		{
			name: "success",
			res:  12,
			err:  nil,
			version: []software.SoftwareIdentity{
				{
					InstanceID:    "AMT",
					VersionString: "12.2.67",
				},
			},
		},
		{
			name: "Instance id not AMT",
			res:  0,
			err:  nil,
			version: []software.SoftwareIdentity{
				{
					InstanceID:    "NOT",
					VersionString: "12.2.67",
				},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res, err := parseVersion(tc.version)

			require.Equal(t, tc.res, res)
			require.Equal(t, tc.err, err)
		})
	}
}
