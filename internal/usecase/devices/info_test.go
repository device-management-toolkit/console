package devices_test

import (
	"context"
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/auditlog"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/messagelog"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/setupandconfiguration"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/software"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	devices "github.com/device-management-toolkit/console/internal/usecase/devices"
	"github.com/device-management-toolkit/console/pkg/logger"
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
					Return(gomock.Any(), nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.HardwareInfo{},
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
			res: dto.HardwareInfo{},
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
					Return(nil, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.HardwareInfo{},
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

			res, err := useCase.GetHardwareInfo(context.Background(), device.GUID)

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
			res: dto.GeneralSettings{Body: gomock.Any()},
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
			res: dto.GeneralSettings{},
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
			res: dto.GeneralSettings{},
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
			res: dto.DiskInfo{},
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
			res: dto.DiskInfo{},
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
			res: dto.DiskInfo{},
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
