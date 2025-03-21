package devices_test

import (
	"context"
	"encoding/xml"
	"testing"

	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/auditlog"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/messagelog"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/amt/setupandconfiguration"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/bios"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/card"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/chassis"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/chip"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/computer"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/mediaaccess"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/physical"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/processor"
	"github.com/open-amt-cloud-toolkit/go-wsman-messages/v2/pkg/wsman/cim/software"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/open-amt-cloud-toolkit/console/internal/entity"
	"github.com/open-amt-cloud-toolkit/console/internal/entity/dto/v1"
	v2 "github.com/open-amt-cloud-toolkit/console/internal/entity/dto/v2"
	"github.com/open-amt-cloud-toolkit/console/internal/mocks"
	devices "github.com/open-amt-cloud-toolkit/console/internal/usecase/devices"
	"github.com/open-amt-cloud-toolkit/console/internal/usecase/devices/wsman"
	"github.com/open-amt-cloud-toolkit/console/pkg/logger"
)

func initInfoTest(t *testing.T) (*devices.UseCase, *mocks.MockWSMAN, *mocks.MockManagement, *mocks.MockDeviceManagementRepository) {
	t.Helper()

	mockCtl := gomock.NewController(t)

	defer mockCtl.Finish()

	repo := mocks.NewMockDeviceManagementRepository(mockCtl)

	wsmanMock := mocks.NewMockWSMAN(mockCtl)
	wsmanMock.EXPECT().Worker().Return().AnyTimes()

	management := mocks.NewMockManagement(mockCtl)

	log := logger.New("error")

	u := devices.New(repo, wsmanMock, mocks.NewMockRedirection(mockCtl), log, mocks.MockCrypto{})

	return u, wsmanMock, management, repo
}

func TestGetVersion(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
	}

	softwares := []software.SoftwareIdentity{
		{
			InstanceID:    "Flash",
			VersionString: "0.0.0",
			IsEntity:      true,
		},
	}

	responses := []setupandconfiguration.SetupAndConfigurationServiceResponse{}

	response := setupandconfiguration.SetupAndConfigurationServiceResponse{
		XMLName:                       xml.Name{Local: "AMT_SetupAndConfigurationService"},
		RequestedState:                1,
		EnabledState:                  1,
		ElementName:                   "SampleElementName",
		SystemCreationClassName:       "SampleSystemCreationClassName",
		SystemName:                    "SampleSystemName",
		CreationClassName:             "SampleCreationClassName",
		Name:                          "SampleName",
		ProvisioningMode:              1,
		ProvisioningState:             1,
		ZeroTouchConfigurationEnabled: true,
		ProvisioningServerOTP:         "SampleProvisioningServerOTP",
		ConfigurationServerFQDN:       "SampleConfigurationServerFQDN",
		PasswordModel:                 1,
		DhcpDNSSuffix:                 "SampleDhcpDNSSuffix",
		TrustedDNSSuffix:              "SampleTrustedDNSSuffix",
	}

	responses = append(responses, response)

	tests := []test{
		{
			name:   "success",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetAMTVersion().
					Return(softwares, nil)

				man2.EXPECT().
					GetSetupAndConfiguration().
					Return(responses, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},

			res: dto.Version{
				CIMSoftwareIdentity: dto.SoftwareIdentityResponses{
					Responses: []dto.SoftwareIdentity{
						{
							InstanceID:    "Flash",
							VersionString: "0.0.0",
							IsEntity:      true,
						},
					},
				}, AMTSetupAndConfigurationService: dto.SetupAndConfigurationServiceResponses{
					Response: dto.SetupAndConfigurationServiceResponse{
						RequestedState:                1,
						EnabledState:                  1,
						ElementName:                   "SampleElementName",
						SystemCreationClassName:       "SampleSystemCreationClassName",
						SystemName:                    "SampleSystemName",
						CreationClassName:             "SampleCreationClassName",
						Name:                          "SampleName",
						ProvisioningMode:              1,
						ProvisioningState:             1,
						ZeroTouchConfigurationEnabled: true,
						ProvisioningServerOTP:         "SampleProvisioningServerOTP",
						ConfigurationServerFQDN:       "SampleConfigurationServerFQDN",
						PasswordModel:                 1,
						DhcpDNSSuffix:                 "SampleDhcpDNSSuffix",
						TrustedDNSSuffix:              "SampleTrustedDNSSuffix",
					},
				},
			},

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

			res: dto.Version{
				CIMSoftwareIdentity: dto.SoftwareIdentityResponses{Responses: []dto.SoftwareIdentity(nil)},
				AMTSetupAndConfigurationService: dto.SetupAndConfigurationServiceResponses{
					Response: dto.SetupAndConfigurationServiceResponse{
						RequestedState:                0,
						EnabledState:                  0,
						ElementName:                   "",
						SystemCreationClassName:       "",
						SystemName:                    "",
						CreationClassName:             "",
						Name:                          "",
						ProvisioningMode:              0,
						ProvisioningState:             0,
						ZeroTouchConfigurationEnabled: false,
						ProvisioningServerOTP:         "",
						ConfigurationServerFQDN:       "",
						PasswordModel:                 0,
						DhcpDNSSuffix:                 "",
						TrustedDNSSuffix:              "",
					},
				},
			},
			err: devices.ErrGeneral,
		},
		{
			name:   "GetAMTVersion fails",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetAMTVersion().
					Return(softwares, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},

			res: dto.Version{},

			err: ErrGeneral,
		},
		{
			name:   "GetSetupAndConfiguration fails",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetAMTVersion().
					Return(softwares, nil)

				man2.EXPECT().
					GetSetupAndConfiguration().
					Return(responses, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},

			res: dto.Version{CIMSoftwareIdentity: dto.SoftwareIdentityResponses{Responses: []dto.SoftwareIdentity(nil)}, AMTSetupAndConfigurationService: dto.SetupAndConfigurationServiceResponses{
				Response: dto.SetupAndConfigurationServiceResponse{
					RequestedState:                0,
					EnabledState:                  0,
					ElementName:                   "",
					SystemCreationClassName:       "",
					SystemName:                    "",
					CreationClassName:             "",
					Name:                          "",
					ProvisioningMode:              0,
					ProvisioningState:             0,
					ZeroTouchConfigurationEnabled: false,
					ProvisioningServerOTP:         "",
					ConfigurationServerFQDN:       "",
					PasswordModel:                 0,
					DhcpDNSSuffix:                 "",
					TrustedDNSSuffix:              "",
				},
			}},
			err: ErrGeneral,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initInfoTest(t)

			tc.manMock(wsmanMock, management)

			tc.repoMock(repo)

			res, _, err := useCase.GetVersion(context.Background(), device.GUID)

			require.Equal(t, tc.res, res)

			require.IsType(t, tc.err, err)
		})
	}
}

func TestGetHardwareInfo(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
	}

	tests := []test{
		{
			name:   "success",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetHardwareInfo().
					Return(wsman.HWResults{}, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			// res: dto.HardwareInfoResults{},
			res: dto.HardwareInfoResults{CIMComputerSystemPackage: dto.CIMComputerSystemPackage{Response: "", Responses: ""}, CIMSystemPackage: dto.CIMSystemPackage{Responses: []dto.CIMSystemPackagingResponses(nil)}, CIMChassis: dto.CIMChassis{Response: dto.CIMChassisResponse{Version: "", SerialNumber: "", Model: "", Manufacturer: "", ElementName: "", CreationClassName: "", Tag: "", OperationalStatus: []int(nil), PackageType: 0, ChassisPackageType: 0}, Responses: []interface{}(nil)}, CIMChip: dto.CIMChips{Responses: []dto.CIMChipGet{{CanBeFRUed: false, CreationClassName: "", ElementName: "", Manufacturer: "", OperationalStatus: []int(nil), Tag: "", Version: ""}}}, CIMCard: dto.CIMCard{Response: dto.CIMCardResponseGet{CanBeFRUed: false, CreationClassName: "", ElementName: "", Manufacturer: "", Model: "", OperationalStatus: []int(nil), PackageType: 0, SerialNumber: "", Tag: "", Version: ""}, Responses: []interface{}(nil)}, CIMBIOSElement: dto.CIMBIOSElement{Response: dto.CIMBIOSElementResponse{TargetOperatingSystem: 0, SoftwareElementID: "", SoftwareElementState: 0, Name: "", OperationalStatus: []int(nil), ElementName: "", Version: "", Manufacturer: "", PrimaryBIOS: false, ReleaseDate: dto.Time{DateTime: ""}}, Responses: []interface{}(nil)}, CIMProcessor: dto.CIMProcessor{Responses: []dto.CIMProcessorResponse{{DeviceID: "", CreationClassName: "", SystemName: "", SystemCreationClassName: "", ElementName: "", OperationalStatus: []int(nil), HealthState: 0, EnabledState: 0, RequestedState: 0, Role: "", Family: 0, OtherFamilyDescription: "", UpgradeMethod: 0, MaxClockSpeed: 0, CurrentClockSpeed: 0, Stepping: "", CPUStatus: 0, ExternalBusClockSpeed: 0}}}, CIMPhysicalPackage: dto.CIMPhysicalPackage{Responses: []dto.CIMPhysicalPackageResponses(nil)}, CIMPhysicalMemory: dto.CIMPhysicalMemory{Responses: []dto.CIMPhysicalMemoryResponse(nil)}, CIMMediaAccessDevice: dto.CIMMediaAccessDevice{Pull: []interface{}(nil), Get: struct {
				Capabilities            []int
				CreationClassName       string
				DeviceID                string
				ElementName             string
				EnabledDefault          int
				EnabledState            int
				MaxMediaSize            int
				OperationalStatus       []int
				RequestedState          int
				Security                int
				SystemCreationClassName string
				SystemName              string
			}{Capabilities: []int(nil), CreationClassName: "", DeviceID: "", ElementName: "", EnabledDefault: 0, EnabledState: 0, MaxMediaSize: 0, OperationalStatus: []int(nil), RequestedState: 0, Security: 0, SystemCreationClassName: "", SystemName: ""}}},
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
			res: dto.HardwareInfoResults{},
			err: devices.ErrGeneral,
		},
		{
			name:   "GetFeatures fails",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetHardwareInfo().
					Return(wsman.HWResults{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.HardwareInfoResults{},
			err: ErrGeneral,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initInfoTest(t)

			tc.manMock(wsmanMock, management)

			tc.repoMock(repo)

			res, _, err := useCase.GetHardwareInfo(context.Background(), device.GUID)

			require.Equal(t, tc.res, res)
			require.IsType(t, tc.err, err)
		})
	}
}

func TestGetAuditLog(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID: "device-guid-123", TenantID: "tenant-id-456",
	}

	tests := []test{
		{
			name:   "success",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetAuditLog(1).
					Return(auditlog.Response{}, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.AuditLog{},
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
			res: dto.AuditLog{
				TotalCount: 0,
				Records:    nil,
			},
			err: devices.ErrGeneral,
		},
		{
			name:   "GetFeatures fails",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetAuditLog(1).
					Return(auditlog.Response{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.AuditLog{TotalCount: 0, Records: nil},
			err: ErrGeneral,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initInfoTest(t)

			tc.manMock(wsmanMock, management)

			tc.repoMock(repo)

			res, err := useCase.GetAuditLog(context.Background(), 1, device.GUID)

			require.Equal(t, tc.res, res)
			require.IsType(t, tc.err, err)
		})
	}
}

func TestGetEventLog(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID: "device-guid-123", TenantID: "tenant-id-456",
	}

	tests := []test{
		{
			name:   "success - with no more records",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetEventLog(1, 10).
					Return(messagelog.GetRecordsResponse{NoMoreRecords: true}, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.EventLogs{},
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
			res: dto.EventLogs{},
			err: devices.ErrGeneral,
		},
		{
			name:   "GetFeatures fails",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetEventLog(1, 10).
					Return(messagelog.GetRecordsResponse{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.EventLogs{},
			err: ErrGeneral,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initInfoTest(t)

			tc.manMock(wsmanMock, management)

			tc.repoMock(repo)

			res, err := useCase.GetEventLog(context.Background(), 1, 10, device.GUID)

			require.Equal(t, tc.res, res)
			require.IsType(t, tc.err, err)
		})
	}
}

func TestGetGeneralSettings(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID: "device-guid-123", TenantID: "tenant-id-456",
	}

	tests := []test{
		{
			name:   "success",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetGeneralSettings().
					Return(gomock.Any(), nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: map[string]interface{}{"Body": gomock.Any()},
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
			res: nil,
			err: devices.ErrGeneral,
		},
		{
			name:   "GetFeatures fails",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetGeneralSettings().
					Return(nil, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: nil,
			err: ErrGeneral,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initInfoTest(t)

			tc.manMock(wsmanMock, management)

			tc.repoMock(repo)

			res, err := useCase.GetGeneralSettings(context.Background(), device.GUID)

			require.Equal(t, tc.res, res)
			require.IsType(t, tc.err, err)
		})
	}
}

func TestGetDiskInfo(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
	}

	tests := []test{
		{
			name:   "success",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetDiskInfo().
					Return(gomock.Any(), nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: gomock.Any(),
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
			res: nil,
			err: devices.ErrGeneral,
		},
		{
			name:   "GetDiskInfo fails",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetDiskInfo().
					Return(nil, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: nil,
			err: ErrGeneral,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initInfoTest(t)

			tc.manMock(wsmanMock, management)

			tc.repoMock(repo)

			res, err := useCase.GetDiskInfo(context.Background(), device.GUID)

			require.Equal(t, tc.res, res)
			require.IsType(t, tc.err, err)
		})
	}
}

func TestChipItemsToDTOv2(t *testing.T) {
	t.Parallel()

	d := []chip.PackageResponse{{CanBeFRUed: false, CreationClassName: "", ElementName: "", Manufacturer: "", OperationalStatus: nil, Tag: "", Version: ""}}

	res := []v2.ChipItems{{CanBeFRUed: false, CreationClassName: "", ElementName: "", Manufacturer: "", OperationalStatus: []int(nil), Tag: "", Version: ""}}

	x := devices.ChipItemsToDTOv2(d)

	require.Equal(t, x, res)
}

func TestCardItemsToDTOv2(t *testing.T) {
	t.Parallel()

	d := []card.PackageResponse{{CanBeFRUed: false, CreationClassName: "", ElementName: "", Manufacturer: "", Model: "", OperationalStatus: nil, PackageType: 0, SerialNumber: "", Tag: "", Version: ""}}
	res := []v2.CardItems{{CanBeFRUed: false, CreationClassName: "", ElementName: "", Manufacturer: "", Model: "", OperationalStatus: []int(nil), PackageType: 0, SerialNumber: "", Tag: "", Version: ""}}

	x := devices.CardItemsToDTOv2(d)

	require.Equal(t, x, res)
}

func TestProcessorItemsToDTOv2(t *testing.T) {
	t.Parallel()

	d := []processor.PackageResponse{{DeviceID: "", CreationClassName: "", SystemName: "", SystemCreationClassName: "", ElementName: "", OperationalStatus: nil, HealthState: 0, EnabledState: 0, RequestedState: 0, Role: "", Family: 0, OtherFamilyDescription: "", UpgradeMethod: 0, MaxClockSpeed: 0, CurrentClockSpeed: 0, Stepping: "", CPUStatus: 0, ExternalBusClockSpeed: 0}}
	res := []v2.ProcessorItems{{DeviceID: "", CreationClassName: "", SystemName: "", SystemCreationClassName: "", ElementName: "", OperationalStatus: []int(nil), HealthState: 0, EnabledState: 0, RequestedState: 0, Role: "", Family: 0, OtherFamilyDescription: "", UpgradeMethod: 0, MaxClockSpeed: 0, CurrentClockSpeed: 0, Stepping: "", CPUStatus: 0, ExternalBusClockSpeed: 0}}

	x := devices.ProcessorItemsToDTOv2(d)

	require.Equal(t, x, res)
}

func TestPhysicalMemoryToDTOv2(t *testing.T) {
	t.Parallel()

	d := []physical.PhysicalMemory{{
		PartNumber:                 "",
		SerialNumber:               "",
		Manufacturer:               "",
		ElementName:                "",
		CreationClassName:          "",
		Tag:                        "",
		OperationalStatus:          nil,
		FormFactor:                 0,
		MemoryType:                 0,
		Speed:                      0,
		Capacity:                   0,
		BankLabel:                  "",
		ConfiguredMemoryClockSpeed: 0,
		IsSpeedInMhz:               false,
		MaxMemorySpeed:             0,
	}}
	res := []v2.PhysicalMemory{{
		PartNumber:                 "",
		SerialNumber:               "",
		Manufacturer:               "",
		ElementName:                "",
		CreationClassName:          "",
		Tag:                        "",
		OperationalStatus:          []int(nil),
		FormFactor:                 0,
		MemoryType:                 0,
		Speed:                      0,
		Capacity:                   0,
		BankLabel:                  "",
		ConfiguredMemoryClockSpeed: 0,
		IsSpeedInMhz:               false,
		MaxMemorySpeed:             0,
	}}

	x := devices.PhysicalMemoryToDTOv2(d)

	require.Equal(t, x, res)
}

func TestPpPullResponseCardToDTOv2(t *testing.T) {
	t.Parallel()

	d := physical.PullResponse{
		XMLName:     xml.Name{},
		MemoryItems: []physical.PhysicalMemory{},
		Card:        []card.PackageResponse{},
		PhysicalPackage: []physical.PhysicalPackage{
			{
				CanBeFRUed:           false,
				VendorEquipmentType:  "",
				ManufactureDate:      "",
				OtherIdentifyingInfo: "",
				SerialNumber:         "",
				SKU:                  "",
				Model:                "",
				Manufacturer:         "",
				ElementName:          "",
				CreationClassName:    "",
				Tag:                  "",
				OperationalStatus:    nil,
				PackageType:          0,
			},
		},
		Chassis:       []chassis.PackageResponse{},
		EndOfSequence: xml.Name{},
	}
	res := []v2.CardItems{{
		CanBeFRUed:        false,
		CreationClassName: "",
		ElementName:       "",
		Manufacturer:      "",
		Model:             "",
		OperationalStatus: []int(nil),
		PackageType:       0,
		SerialNumber:      "",
		Tag:               "",
	}}

	x := devices.PpPullResponseCardToDTOv2(d)

	require.Equal(t, x, res)
}

func TestPpPullResponseMemoryToDTOv2(t *testing.T) {
	t.Parallel()

	d := physical.PullResponse{
		XMLName: xml.Name{},
		MemoryItems: []physical.PhysicalMemory{
			{
				PartNumber:                 "",
				SerialNumber:               "",
				Manufacturer:               "",
				ElementName:                "",
				CreationClassName:          "",
				Tag:                        "",
				OperationalStatus:          nil,
				FormFactor:                 0,
				MemoryType:                 0,
				Speed:                      0,
				Capacity:                   0,
				BankLabel:                  "",
				ConfiguredMemoryClockSpeed: 0,
				IsSpeedInMhz:               false,
				MaxMemorySpeed:             0,
			},
		},
		Card:            []card.PackageResponse{},
		PhysicalPackage: []physical.PhysicalPackage{},
		Chassis:         []chassis.PackageResponse{},
		EndOfSequence:   xml.Name{},
	}
	res := []v2.PhysicalMemory{{
		PartNumber:                 "",
		SerialNumber:               "",
		Manufacturer:               "",
		ElementName:                "",
		CreationClassName:          "",
		Tag:                        "",
		OperationalStatus:          []int(nil),
		FormFactor:                 0,
		MemoryType:                 0,
		Speed:                      0,
		Capacity:                   0,
		BankLabel:                  "",
		ConfiguredMemoryClockSpeed: 0,
		IsSpeedInMhz:               false,
		MaxMemorySpeed:             0,
	}}

	x := devices.PpPullResponseMemoryToDTOv2(d)

	require.Equal(t, x, res)
}

func TestMediaAccessDeviceToDTOv2(t *testing.T) {
	t.Parallel()

	d := []mediaaccess.MediaAccessDevice{{
		CreationClassName:       "",
		DeviceID:                "",
		ElementName:             "",
		EnabledDefault:          0,
		EnabledState:            0,
		MaxMediaSize:            0,
		OperationalStatus:       nil,
		RequestedState:          0,
		Security:                0,
		SystemCreationClassName: "",
		SystemName:              "",
	}}
	res := []v2.MediaAccessDevice{{
		CreationClassName:       "",
		DeviceID:                "",
		ElementName:             "",
		EnabledDefault:          0,
		EnabledState:            0,
		MaxMediaSize:            0,
		OperationalStatus:       []int(nil),
		RequestedState:          0,
		Security:                0,
		SystemCreationClassName: "",
		SystemName:              "",
	}}

	x := devices.MediaAccessDeviceToDTOv2(d)

	require.Equal(t, x, res)
}

func TestCimChipArray(t *testing.T) {
	t.Parallel()

	d := wsman.HWResults{
		CSPResult:             computer.Response{},
		ChassisResult:         chassis.Response{},
		ChipResult:            chip.Response{},
		CardResult:            card.Response{},
		PhysicalMemoryResult:  physical.Response{},
		MediaAccessPullResult: mediaaccess.Response{},
		PPPullResult:          physical.Response{},
		BiosResult:            bios.Response{},
		ProcessorResult:       processor.Response{},
	}

	res := []dto.CIMChipGet{{
		CanBeFRUed:        false,
		CreationClassName: "",
		ElementName:       "",
		Manufacturer:      "",
		OperationalStatus: []int(nil),
		Tag:               "",
		Version:           "",
	}}

	x := devices.CimChipArray(&d)

	require.Equal(t, x, res)
}

func TestCimPhysicalMemoryArray(t *testing.T) {
	t.Parallel()

	d := wsman.HWResults{
		CSPResult:     computer.Response{},
		ChassisResult: chassis.Response{},
		ChipResult:    chip.Response{},
		CardResult:    card.Response{},
		PhysicalMemoryResult: physical.Response{
			Body: physical.Body{
				PullResponse: physical.PullResponse{
					MemoryItems: []physical.PhysicalMemory{{
						PartNumber:                 "",
						SerialNumber:               "",
						Manufacturer:               "",
						ElementName:                "",
						CreationClassName:          "",
						Tag:                        "",
						OperationalStatus:          nil,
						FormFactor:                 0,
						MemoryType:                 0,
						Speed:                      0,
						Capacity:                   0,
						BankLabel:                  "",
						ConfiguredMemoryClockSpeed: 0,
						IsSpeedInMhz:               false,
						MaxMemorySpeed:             0,
					}},
				},
			},
		},
		MediaAccessPullResult: mediaaccess.Response{},
		PPPullResult:          physical.Response{},
		BiosResult:            bios.Response{},
		ProcessorResult:       processor.Response{},
	}

	res := []dto.CIMPhysicalMemoryResponse{{
		PartNumber:                 "",
		SerialNumber:               "",
		Manufacturer:               "",
		ElementName:                "",
		CreationClassName:          "",
		Tag:                        "",
		OperationalStatus:          []int(nil),
		FormFactor:                 0,
		MemoryType:                 0,
		Speed:                      0,
		Capacity:                   0,
		BankLabel:                  "",
		ConfiguredMemoryClockSpeed: 0,
		IsSpeedInMhz:               false,
		MaxMemorySpeed:             0,
	}}

	x := devices.CimPhysicalMemoryArray(&d)

	require.Equal(t, x, res)
}

func TestProcessorArray(t *testing.T) {
	t.Parallel()

	d := wsman.HWResults{
		CSPResult:             computer.Response{},
		ChassisResult:         chassis.Response{},
		ChipResult:            chip.Response{},
		CardResult:            card.Response{},
		PhysicalMemoryResult:  physical.Response{},
		MediaAccessPullResult: mediaaccess.Response{},
		PPPullResult:          physical.Response{},
		BiosResult:            bios.Response{},
		ProcessorResult:       processor.Response{},
	}

	res := []dto.CIMProcessorResponse{{
		DeviceID:                "",
		CreationClassName:       "",
		SystemName:              "",
		SystemCreationClassName: "",
		ElementName:             "",
		OperationalStatus:       nil,
		HealthState:             0,
		EnabledState:            0,
		RequestedState:          0,
		Role:                    "",
		Family:                  0,
		OtherFamilyDescription:  "",
		UpgradeMethod:           0,
		MaxClockSpeed:           0,
		CurrentClockSpeed:       0,
		Stepping:                "",
		CPUStatus:               0,
		ExternalBusClockSpeed:   0,
	}}

	x := devices.CimProcessorArray(&d)

	require.Equal(t, x, res)
}
