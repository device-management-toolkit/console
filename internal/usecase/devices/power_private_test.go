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

			uc := &UseCase{} // create a dummy UseCase
			res := uc.getBootSource("test-guid", &tc.bootSettings)

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

func Test_determineBootDevice(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		boot     dto.BootSetting
		wantIDER int
		wantUEFI bool
		wantErr  bool
	}{
		{
			name:     "IDER CDROM",
			boot:     dto.BootSetting{Action: 202},
			wantIDER: 1,
			wantUEFI: false,
			wantErr:  false,
		},
		{
			name:     "Power On",
			boot:     dto.BootSetting{Action: 999},
			wantIDER: 0,
			wantUEFI: false,
			wantErr:  false,
		},
		{
			name:     "HTTPS Boot",
			boot:     dto.BootSetting{Action: 105, BootDetails: dto.BootDetails{URL: "https://example.com"}},
			wantIDER: 0,
			wantUEFI: true,
			wantErr:  false,
		},
		{
			name:     "PBA Boot",
			boot:     dto.BootSetting{Action: 107, BootDetails: dto.BootDetails{BootPath: "pba.efi"}},
			wantIDER: 0,
			wantUEFI: true,
			wantErr:  false,
		},
		{
			name:     "WinRE Boot",
			boot:     dto.BootSetting{Action: 109, BootDetails: dto.BootDetails{BootPath: "winre.wim"}},
			wantIDER: 0,
			wantUEFI: true,
			wantErr:  false,
		},
		{
			name:     "HTTPS Boot error (missing URL)",
			boot:     dto.BootSetting{Action: 105, BootDetails: dto.BootDetails{URL: ""}},
			wantIDER: 0,
			wantUEFI: false,
			wantErr:  true,
		},
		{
			name:     "PBA Boot error (missing BootPath)",
			boot:     dto.BootSetting{Action: 107, BootDetails: dto.BootDetails{BootPath: ""}},
			wantIDER: 0,
			wantUEFI: false,
			wantErr:  true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			newData := boot.BootSettingDataRequest{}

			err := determineBootDevice(tc.boot, &newData)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.wantIDER, int(newData.IDERBootDevice))

				if tc.wantUEFI {
					require.True(t, newData.ForcedProgressEvents)
					require.NotEmpty(t, newData.UefiBootParametersArray)
				}
			}
		})
	}
}
