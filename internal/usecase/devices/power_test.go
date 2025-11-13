package devices_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/boot"
	cimBoot "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/boot"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/power"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/service"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/software"
	ipspower "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips/power"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	devices "github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/logger"
)

var ErrGeneral = errors.New("general error")

type test struct {
	name     string
	manMock  func(*mocks.MockWSMAN, *mocks.MockManagement)
	repoMock func(*mocks.MockDeviceManagementRepository)
	res      any
	resV2    any
	err      error

	action int
}

func initPowerTest(t *testing.T) (*devices.UseCase, *mocks.MockWSMAN, *mocks.MockManagement, *mocks.MockDeviceManagementRepository) {
	t.Helper()

	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	repo := mocks.NewMockDeviceManagementRepository(mockCtl)
	wsmanMock := mocks.NewMockWSMAN(mockCtl)
	wsmanMock.EXPECT().Worker().Return().AnyTimes()

	managementMock := mocks.NewMockManagement(mockCtl)
	log := logger.New("error")
	u := devices.New(repo, wsmanMock, mocks.NewMockRedirection(mockCtl), log, mocks.MockCrypto{})

	return u, wsmanMock, managementMock, repo
}

func TestSendPowerAction(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		Password: "encrypted",
		TenantID: "tenant-id-456",
	}

	powerActionRes := power.PowerActionResponse{
		ReturnValue: power.ReturnValue(0),
	}

	ipsPowerActionRes := ipspower.PowerActionResponse{
		ReturnValue: ipspower.ReturnValue(0),
	}

	tests := []test{
		{
			name:   "success for Action 0",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm)
				hmm.EXPECT().
					SendPowerAction(0).
					Return(powerActionRes, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: powerActionRes,
			err: nil,
		},
		{
			name:   "success for Action 2",
			action: 2,
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm)
				hmm.EXPECT().
					GetOSPowerSavingState().
					Return(ipspower.OSPowerSavingState(3), nil) // It emulates to be in SAVING MODE
				hmm.EXPECT().
					RequestOSPowerSavingStateChange(ipspower.OSPowerSavingState(2)).
					Return(ipsPowerActionRes, nil)
				hmm.EXPECT().
					SendPowerAction(2).
					Return(powerActionRes, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: powerActionRes,
			err: nil,
		},
		{
			name:   "success for Action 500 (OSToFullPower)",
			action: 500,
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm)
				hmm.EXPECT().
					GetOSPowerSavingState().
					Return(ipspower.OSPowerSavingState(3), nil) // It emulates to be in SAVING MODE
				hmm.EXPECT().
					RequestOSPowerSavingStateChange(ipspower.OSPowerSavingState(2)).
					Return(ipsPowerActionRes, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: powerActionRes,
			err: nil,
		},
		{
			name:   "success for Action 501 (OSToPowerSaving)",
			action: 501,
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm)
				hmm.EXPECT().
					GetOSPowerSavingState().
					Return(ipspower.OSPowerSavingState(2), nil) // It emulates to be in FULL POWER
				hmm.EXPECT().
					RequestOSPowerSavingStateChange(ipspower.OSPowerSavingState(3)).
					Return(ipsPowerActionRes, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: powerActionRes,
			err: nil,
		},
		{
			name:    "GetById fails",
			action:  0,
			manMock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(nil, ErrGeneral)
			},
			res: power.PowerActionResponse{},
			err: devices.ErrGeneral,
		},
		{
			name:   "SendPowerAction fails",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm)
				hmm.EXPECT().
					SendPowerAction(0).
					Return(power.PowerActionResponse{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: power.PowerActionResponse{},
			err: ErrGeneral,
		},
		{
			name:   "SendPowerAction fails (action = 2)",
			action: 2,
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm)
				hmm.EXPECT().
					SendPowerAction(2).
					Return(power.PowerActionResponse{}, ErrGeneral)
				hmm.EXPECT().
					GetOSPowerSavingState().
					Return(ipspower.OSPowerSavingState(3), nil) // It emulates to be in SAVING MODE
				hmm.EXPECT().
					RequestOSPowerSavingStateChange(ipspower.OSPowerSavingState(2)). // Go to FULL POWER
					Return(ipspower.PowerActionResponse{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: powerActionRes,
			err: nil,
		},
		{
			name:   "SendPowerAction fails (OSToFullPower, action = 500)",
			action: 500,
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm)
				hmm.EXPECT().
					GetOSPowerSavingState().
					Return(ipspower.OSPowerSavingState(3), nil) // It emulates to be in SAVING MODE
				hmm.EXPECT().
					RequestOSPowerSavingStateChange(ipspower.OSPowerSavingState(2)). // Go to FULL POWER
					Return(ipspower.PowerActionResponse{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: power.PowerActionResponse{},
			err: ErrGeneral,
		},
		{
			name:   "SendPowerAction fails (OSToPowerSaving, action = 501)",
			action: 501,
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm)
				hmm.EXPECT().
					GetOSPowerSavingState().
					Return(ipspower.OSPowerSavingState(2), nil) // It emulates to be in FULL POWER
				hmm.EXPECT().
					RequestOSPowerSavingStateChange(ipspower.OSPowerSavingState(3)). // Go to SAVING MODE
					Return(ipspower.PowerActionResponse{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: powerActionRes,
			err: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initPowerTest(t)

			tc.manMock(wsmanMock, management)
			tc.repoMock(repo)

			res, err := useCase.SendPowerAction(context.Background(), device.GUID, tc.action)

			require.Equal(t, tc.res, res)

			if tc.err != nil {
				assert.Equal(t, err, tc.err)
			}
		})
	}
}

func TestGetPowerState(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
	}

	tests := []test{
		{
			name: "success",
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm)
				hmm.EXPECT().
					GetPowerState().
					Return([]service.CIM_AssociatedPowerManagementService{{PowerState: 0}}, nil)
				hmm.EXPECT().
					GetOSPowerSavingState().
					Return(ipspower.OSPowerSavingState(3), nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.PowerState{
				PowerState:         0,
				OSPowerSavingState: 3,
			},
			err: nil,
		},
		{
			name:    "GetById fails",
			manMock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(nil, ErrGeneral)
			},
			res: dto.PowerState{},
			err: devices.ErrGeneral,
		},
		{
			name: "GetPowerState fails",
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm)
				hmm.EXPECT().
					GetPowerState().
					Return([]service.CIM_AssociatedPowerManagementService{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.PowerState{},
			err: ErrGeneral,
		},
		{
			name: "GetOSPowerSavingState fails",
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm)
				hmm.EXPECT().
					GetPowerState().
					Return([]service.CIM_AssociatedPowerManagementService{{PowerState: 0}}, nil)
				hmm.EXPECT().
					GetOSPowerSavingState().
					Return(ipspower.OSPowerSavingState(0), ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.PowerState{
				PowerState:         0,
				OSPowerSavingState: 0, // UNKNOWN
			},
			err: ErrGeneral,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initPowerTest(t)

			tc.manMock(wsmanMock, management)
			tc.repoMock(repo)

			res, err := useCase.GetPowerState(context.Background(), device.GUID)

			require.Equal(t, tc.res, res)
			require.IsType(t, tc.err, err)
		})
	}
}

func TestGetPowerCapabilities(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
	}

	tests := []test{
		{
			name: "success",
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm)
				hmm.EXPECT().
					GetAMTVersion().
					Return([]software.SoftwareIdentity{}, nil)
				hmm.EXPECT().
					GetPowerCapabilities().
					Return(boot.BootCapabilitiesResponse{}, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.PowerCapabilities{
				PowerUp:             2,
				PowerCycle:          5,
				PowerDown:           8,
				Reset:               10,
				ResetToIDERFloppy:   200,
				PowerOnToIDERFloppy: 201,
				ResetToIDERCDROM:    202,
				PowerOnToIDERCDROM:  203,
				ResetToPXE:          400,
				PowerOnToPXE:        401,
			},
			err: nil,
		},
		{
			name:    "GetById fails",
			manMock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(nil, ErrGeneral)
			},
			res: dto.PowerCapabilities{},
			err: devices.ErrGeneral,
		},
		{
			name: "GetPowerCapabilities fails",
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm)
				hmm.EXPECT().
					GetPowerCapabilities().
					Return(boot.BootCapabilitiesResponse{}, ErrGeneral)
				hmm.EXPECT().
					GetAMTVersion().
					Return(nil, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.PowerCapabilities{},
			err: ErrGeneral,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initPowerTest(t)
			tc.manMock(wsmanMock, management)
			tc.repoMock(repo)

			res, err := useCase.GetPowerCapabilities(context.Background(), device.GUID)

			require.Equal(t, tc.res, res)
			require.IsType(t, tc.err, err)
		})
	}
}

func TestSetBootOptions(t *testing.T) {
	t.Parallel()

	bootResponse := boot.BootSettingDataResponse{
		BIOSLastStatus:           []uint16{2, 0},
		BIOSPause:                false,
		BIOSSetup:                false,
		BootMediaIndex:           0,
		BootguardStatus:          127,
		ConfigurationDataReset:   false,
		ElementName:              "Intel(r) AMT Boot Configuration Settings",
		EnforceSecureBoot:        false,
		FirmwareVerbosity:        0,
		ForcedProgressEvents:     false,
		IDERBootDevice:           0,
		InstanceID:               "Intel(r) AMT:BootSettingData 0",
		LockKeyboard:             false,
		LockPowerButton:          false,
		LockResetButton:          false,
		LockSleepButton:          false,
		OptionsCleared:           true,
		OwningEntity:             "Intel(r) AMT",
		PlatformErase:            false,
		RPEEnabled:               false,
		RSEPassword:              "",
		ReflashBIOS:              false,
		SecureBootControlEnabled: false,
		SecureErase:              false,
		UEFIHTTPSBootEnabled:     false,
		UEFILocalPBABootEnabled:  false,
		UefiBootNumberOfParams:   0,
		UseIDER:                  false,
		UseSOL:                   false,
		UseSafeMode:              false,
		UserPasswordBypass:       false,
		WinREBootEnabled:         false,
	}

	device := &entity.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
	}

	bootSetting := dto.BootSetting{
		Action: 400,
		UseSOL: true,
	}

	powerActionRes := power.PowerActionResponse{ReturnValue: 5}

	tests := []test{
		{
			name: "success",
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm)
				hmm.EXPECT().
					GetBootData().
					Return(bootResponse, nil)
				hmm.EXPECT().
					ChangeBootOrder("").
					Return(cimBoot.ChangeBootOrder_OUTPUT{}, nil)
				hmm.EXPECT().
					SetBootData(gomock.Any()).
					Return(nil, nil)
				hmm.EXPECT().
					SetBootConfigRole(1).
					Return(powerActionRes, nil)
				hmm.EXPECT().
					ChangeBootOrder(string(cimBoot.PXE)).
					Return(cimBoot.ChangeBootOrder_OUTPUT{}, nil)
				hmm.EXPECT().
					SendPowerAction(10).
					Return(powerActionRes, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: powerActionRes,
			err: nil,
		},
		{
			name:    "GetById fails",
			manMock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(nil, ErrGeneral)
			},
			res: power.PowerActionResponse{},
			err: devices.ErrGeneral,
		},
		{
			name: "GetBootData fails",
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm)
				hmm.EXPECT().
					GetBootData().
					Return(boot.BootSettingDataResponse{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: power.PowerActionResponse{},
			err: ErrGeneral,
		},
		{
			name: "First ChangeBootOrder fails",
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm)
				hmm.EXPECT().
					GetBootData().
					Return(bootResponse, nil)
				hmm.EXPECT().
					ChangeBootOrder("").
					Return(cimBoot.ChangeBootOrder_OUTPUT{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: power.PowerActionResponse{},
			err: ErrGeneral,
		},
		{
			name: "SetBootData fails",
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm)
				hmm.EXPECT().
					GetBootData().
					Return(bootResponse, nil)
				hmm.EXPECT().
					ChangeBootOrder("").
					Return(cimBoot.ChangeBootOrder_OUTPUT{}, nil)
				hmm.EXPECT().
					SetBootData(gomock.Any()).
					Return(nil, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: power.PowerActionResponse{},
			err: ErrGeneral,
		},
		{
			name: "SetBootConfigRole fails",
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm)
				hmm.EXPECT().
					GetBootData().
					Return(bootResponse, nil)
				hmm.EXPECT().
					ChangeBootOrder("").
					Return(cimBoot.ChangeBootOrder_OUTPUT{}, nil)
				hmm.EXPECT().
					SetBootData(gomock.Any()).
					Return(nil, nil)
				hmm.EXPECT().
					SetBootConfigRole(1).
					Return(powerActionRes, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: power.PowerActionResponse{},
			err: ErrGeneral,
		},
		{
			name: "Second ChangeBootOrder fails",
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm)
				hmm.EXPECT().
					GetBootData().
					Return(bootResponse, nil)
				hmm.EXPECT().
					ChangeBootOrder("").
					Return(cimBoot.ChangeBootOrder_OUTPUT{}, nil)
				hmm.EXPECT().
					SetBootData(gomock.Any()).
					Return(nil, nil)
				hmm.EXPECT().
					SetBootConfigRole(1).
					Return(powerActionRes, nil)
				hmm.EXPECT().
					ChangeBootOrder(string(cimBoot.PXE)).
					Return(cimBoot.ChangeBootOrder_OUTPUT{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: power.PowerActionResponse{},
			err: ErrGeneral,
		},
		{
			name: "SendPowerAction fails",
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(hmm)
				hmm.EXPECT().
					GetBootData().
					Return(bootResponse, nil)
				hmm.EXPECT().
					ChangeBootOrder("").
					Return(cimBoot.ChangeBootOrder_OUTPUT{}, nil)
				hmm.EXPECT().
					SetBootData(gomock.Any()).
					Return(nil, nil)
				hmm.EXPECT().
					SetBootConfigRole(1).
					Return(powerActionRes, nil)
				hmm.EXPECT().
					ChangeBootOrder(string(cimBoot.PXE)).
					Return(cimBoot.ChangeBootOrder_OUTPUT{}, nil)
				hmm.EXPECT().
					SendPowerAction(10).
					Return(powerActionRes, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: power.PowerActionResponse{},
			err: ErrGeneral,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initPowerTest(t)
			tc.manMock(wsmanMock, management)
			tc.repoMock(repo)

			res, err := useCase.SetBootOptions(context.Background(), device.GUID, bootSetting)

			require.Equal(t, tc.res, res)
			require.IsType(t, tc.err, err)
		})
	}
}

func TestGetBootSourceSetting(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
	}

	bootSourceSettings := []cimBoot.BootSourceSetting{
		{InstanceID: "PXE", BootString: "PXE Boot Path", BIOSBootString: "PXE BIOS String", StructuredBootString: "PXE Structured String"},
		{InstanceID: "CD", BootString: "CD Boot Path", BIOSBootString: "CD BIOS String", StructuredBootString: "CD Structured String"},
	}

	settingsResponse := cimBoot.Response{
		Body: cimBoot.Body{
			PullResponse: cimBoot.PullResponse{
				BootSourceSettingItems: bootSourceSettings,
			},
		},
	}

	expected := []dto.BootSources{
		{
			InstanceID:           "PXE",
			BootString:           "PXE Boot Path",
			BIOSBootString:       "PXE BIOS String",
			StructuredBootString: "PXE Structured String",
			ElementName:          "",
			FailThroughSupported: 0,
		},
		{
			InstanceID:           "CD",
			BootString:           "CD Boot Path",
			BIOSBootString:       "CD BIOS String",
			StructuredBootString: "CD Structured String",
			ElementName:          "",
			FailThroughSupported: 0,
		},
	}

	tests := []struct {
		name     string
		manMock  func(*mocks.MockWSMAN, *mocks.MockManagement)
		repoMock func(*mocks.MockDeviceManagementRepository)
		want     []dto.BootSources
		wantErr  error
	}{
		{
			name: "success",
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().SetupWsmanClient(gomock.Any(), false, true).Return(hmm)
				hmm.EXPECT().GetCIMBootSourceSetting().Return(settingsResponse, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			want:    expected,
			wantErr: nil,
		},
		{
			name:    "not found",
			manMock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(nil, devices.ErrNotFound)
			},
			want:    nil,
			wantErr: devices.ErrNotFound,
		},
		{
			name: "GetCIMBootSourceSetting error",
			manMock: func(man *mocks.MockWSMAN, hmm *mocks.MockManagement) {
				man.EXPECT().SetupWsmanClient(gomock.Any(), false, true).Return(hmm)
				hmm.EXPECT().GetCIMBootSourceSetting().Return(settingsResponse, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil)
			},
			want:    nil,
			wantErr: ErrGeneral,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			useCase, wsmanMock, management, repo := initPowerTest(t)
			tc.manMock(wsmanMock, management)
			tc.repoMock(repo)

			result, err := useCase.GetBootSourceSetting(context.Background(), device.GUID)
			assert.Equal(t, tc.want, result)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

func TestValidateHTTPBootParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		url            string
		username       string
		password       string
		expectedParams int
		expectError    bool
		errorType      error
	}{
		{
			name:           "valid URL only",
			url:            "https://example.com/boot.efi",
			username:       "",
			password:       "",
			expectedParams: 2, // network path + sync root CA
			expectError:    false,
		},
		{
			name:        "empty URL",
			url:         "",
			username:    "",
			password:    "",
			expectError: true,
		},
		{
			name:        "URL too long",
			url:         "https://example.com/" + string(make([]byte, 300)), // Over 300 byte limit
			username:    "",
			password:    "",
			expectError: true,
		},
		{
			name:        "username too long",
			url:         "https://example.com/boot.efi",
			username:    string(make([]byte, 301)), // Over 300 byte limit
			password:    "",
			expectError: true,
		},
		{
			name:        "password too long",
			url:         "https://example.com/boot.efi",
			username:    "testuser",
			password:    string(make([]byte, 301)), // Over 300 byte limit
			expectError: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			buffer, paramCount, err := devices.ValidateHTTPBootParams(tc.url, tc.username, tc.password)

			if tc.expectError {
				require.Error(t, err)
				require.Nil(t, buffer)
				require.Equal(t, 0, paramCount)
			} else {
				require.NoError(t, err)
				require.NotNil(t, buffer)
				require.Equal(t, tc.expectedParams, paramCount)
				require.Greater(t, len(buffer), 0)
			}
		})
	}
}

func TestValidatePBAWinReBootParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		file           string
		expectedParams int
		expectError    bool
		errorType      error
	}{
		{
			name:           "valid file path",
			file:           "/boot/winre.wim",
			expectedParams: 2, // file path + file length
			expectError:    false,
		},
		{
			name:           "empty file path",
			file:           "",
			expectedParams: 2,
			expectError:    false,
		},
		{
			name:           "short file path",
			file:           "/boot/test.wim",
			expectedParams: 2,
			expectError:    false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			buffer, paramCount, err := devices.ValidatePBAWinReBootParams(tc.file)

			if tc.expectError {
				require.Error(t, err)

				if tc.errorType != nil {
					// Check if it's the specific error we expect
					require.Equal(t, tc.errorType, err)
				}

				require.Nil(t, buffer)
				require.Equal(t, 0, paramCount)
			} else {
				require.NoError(t, err)
				require.NotNil(t, buffer)
				require.Equal(t, tc.expectedParams, paramCount)
				require.Greater(t, len(buffer), 0)
			}
		})
	}
}
