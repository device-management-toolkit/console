package devices_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/amterror"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/boot"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/redirection"
	cimBoot "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/boot"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/kvm"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips/optin"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	dtov2 "github.com/device-management-toolkit/console/internal/entity/dto/v2"
	"github.com/device-management-toolkit/console/internal/mocks"
	devices "github.com/device-management-toolkit/console/internal/usecase/devices"
)

const DestinationUnreachable string = "<?xml version=\"1.0\" encoding=\"UTF-8\"?><a:Envelope xmlns:g=\"http://schemas.dmtf.org/wbem/wsman/1/cimbinding.xsd\" xmlns:f=\"http://schemas.xmlsoap.org/ws/2004/08/eventing\" xmlns:e=\"http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd\" xmlns:d=\"http://schemas.xmlsoap.org/ws/2004/09/transfer\" xmlns:c=\"http://schemas.xmlsoap.org/ws/2004/09/enumeration\" xmlns:b=\"http://schemas.xmlsoap.org/ws/2004/08/addressing\" xmlns:a=\"http://www.w3.org/2003/05/soap-envelope\" xmlns:h=\"http://schemas.xmlsoap.org/ws/2005/02/trust\" xmlns:i=\"http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd\" xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\"><a:Header><b:To>http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</b:To><b:RelatesTo>0</b:RelatesTo><b:Action a:mustUnderstand=\"true\">http://schemas.xmlsoap.org/ws/2004/08/addressing/fault</b:Action><b:MessageID>uuid:00000000-8086-8086-8086-000000000061</b:MessageID></a:Header><a:Body><a:Fault><a:Code><a:Value>a:Sender</a:Value><a:Subcode><a:Value>b:DestinationUnreachable</a:Value></a:Subcode></a:Code><a:Reason><a:Text xml:lang=\"en-US\">No route can be determined to reach the destination role defined by the WSAddressing To.</a:Text></a:Reason><a:Detail></a:Detail></a:Fault></a:Body></a:Envelope>"

func TestGetFeatures(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
	}

	featureSet := dto.Features{
		UserConsent:           "kvm",
		EnableSOL:             true,
		EnableIDER:            true,
		EnableKVM:             true,
		Redirection:           true,
		KVMAvailable:          true,
		OptInState:            1,
		OCR:                   true,
		HTTPSBootSupported:    true,
		WinREBootSupported:    false,
		LocalPBABootSupported: false,
	}

	featureSetNoKVM := dto.Features{
		UserConsent:           "kvm",
		EnableSOL:             true,
		EnableIDER:            true,
		EnableKVM:             false,
		Redirection:           true,
		KVMAvailable:          false,
		OptInState:            1,
		OCR:                   true,
		HTTPSBootSupported:    true,
		WinREBootSupported:    false,
		LocalPBABootSupported: false,
	}

	featureSetNoOCR := dto.Features{
		UserConsent:           "kvm",
		EnableSOL:             true,
		EnableIDER:            true,
		EnableKVM:             true,
		Redirection:           true,
		KVMAvailable:          true,
		OptInState:            1,
		OCR:                   false,
		HTTPSBootSupported:    false,
		WinREBootSupported:    false,
		LocalPBABootSupported: false,
	}

	featureSetV2 := dtov2.Features{
		UserConsent:           "kvm",
		EnableSOL:             true,
		EnableIDER:            true,
		EnableKVM:             true,
		Redirection:           true,
		KVMAvailable:          true,
		OptInState:            1,
		OCR:                   true,
		HTTPSBootSupported:    true,
		WinREBootSupported:    false,
		LocalPBABootSupported: false,
	}

	featureSetV2NoKVM := dtov2.Features{
		UserConsent:           "kvm",
		EnableSOL:             true,
		EnableIDER:            true,
		EnableKVM:             false,
		Redirection:           true,
		KVMAvailable:          false,
		OptInState:            1,
		OCR:                   true,
		HTTPSBootSupported:    true,
		WinREBootSupported:    false,
		LocalPBABootSupported: false,
	}

	featureSetV2NoOCR := dtov2.Features{
		UserConsent:           "kvm",
		EnableSOL:             true,
		EnableIDER:            true,
		EnableKVM:             true,
		Redirection:           true,
		KVMAvailable:          true,
		OptInState:            1,
		OCR:                   false,
		HTTPSBootSupported:    false,
		WinREBootSupported:    false,
		LocalPBABootSupported: false,
	}

	tests := []test{
		{
			name:    "Device not found - nil device",
			action:  0,
			manMock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(nil, nil)
			},
			res:   dto.Features{},
			resV2: dtov2.Features{},
			err:   devices.ErrNotFound,
		},
		{
			name:    "Device not found - empty GUID",
			action:  0,
			manMock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				emptyDevice := &entity.Device{
					GUID:     "",
					TenantID: "tenant-id-456",
				}
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(emptyDevice, nil)
			},
			res:   dto.Features{},
			resV2: dtov2.Features{},
			err:   devices.ErrNotFound,
		},
		{
			name:   "success",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetAMTRedirectionService().
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					GetIPSOptInService().
					Return(optin.Response{
						Body: optin.Body{
							GetAndPutResponse: optin.OptInServiceResponse{
								OptInRequired: 1,
								OptInState:    1,
							},
						},
					}, nil)
				man2.EXPECT().
					GetKVMRedirection().
					Return(kvm.Response{
						Body: kvm.Body{
							GetResponse: kvm.KVMRedirectionSAP{
								EnabledState: kvm.EnabledState(redirection.Enabled),
							},
						},
					}, nil)
				man2.EXPECT().
					GetBootService().
					Return(cimBoot.BootService{
						EnabledState: 32769,
					}, nil)
				man2.EXPECT().
					GetCIMBootSourceSetting().
					Return(cimBoot.Response{
						Body: cimBoot.Body{
							PullResponse: cimBoot.PullResponse{
								BootSourceSettingItems: []cimBoot.BootSourceSetting{
									{
										InstanceID: "Intel(r) AMT: Force OCR UEFI HTTPS Boot",
									},
									{
										InstanceID: "Intel(r) AMT: Force OCR UEFI Boot Option",
									},
								},
							},
						},
					}, nil)
				man2.EXPECT().
					GetPowerCapabilities().
					Return(boot.BootCapabilitiesResponse{
						ForceUEFIHTTPSBoot:    true,
						ForceWinREBoot:        true,
						ForceUEFILocalPBABoot: true,
					}, nil)
				man2.EXPECT().
					GetBootData().
					Return(boot.BootSettingDataResponse{
						UEFIHTTPSBootEnabled:    true,
						WinREBootEnabled:        true,
						UEFILocalPBABootEnabled: true,
					}, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res:   featureSet,
			resV2: featureSetV2,
			err:   nil,
		},
		{
			name:   "success with OCR supported but disabled",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetAMTRedirectionService().
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					GetIPSOptInService().
					Return(optin.Response{
						Body: optin.Body{
							GetAndPutResponse: optin.OptInServiceResponse{
								OptInRequired: 1,
								OptInState:    1,
							},
						},
					}, nil)
				man2.EXPECT().
					GetKVMRedirection().
					Return(kvm.Response{
						Body: kvm.Body{
							GetResponse: kvm.KVMRedirectionSAP{
								EnabledState: kvm.EnabledState(redirection.Enabled),
							},
						},
					}, nil)
				man2.EXPECT().
					GetBootService().
					Return(cimBoot.BootService{
						EnabledState: 32768, // Disabled state
					}, nil)
				man2.EXPECT().
					GetCIMBootSourceSetting().
					Return(cimBoot.Response{
						Body: cimBoot.Body{
							PullResponse: cimBoot.PullResponse{
								BootSourceSettingItems: []cimBoot.BootSourceSetting{
									{
										InstanceID: "Intel(r) AMT: Force OCR UEFI HTTPS Boot",
									},
									{
										InstanceID: "Intel(r) AMT: Force OCR UEFI Boot Option",
									},
								},
							},
						},
					}, nil)
				man2.EXPECT().
					GetPowerCapabilities().
					Return(boot.BootCapabilitiesResponse{
						ForceUEFIHTTPSBoot:    true,
						ForceWinREBoot:        true,
						ForceUEFILocalPBABoot: true,
					}, nil)
				man2.EXPECT().
					GetBootData().
					Return(boot.BootSettingDataResponse{
						UEFIHTTPSBootEnabled:    true,
						WinREBootEnabled:        true,
						UEFILocalPBABootEnabled: true,
					}, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.Features{
				UserConsent:           "kvm",
				EnableSOL:             true,
				EnableIDER:            true,
				EnableKVM:             true,
				Redirection:           true,
				KVMAvailable:          true,
				OptInState:            1,
				OCR:                   false,
				HTTPSBootSupported:    true,
				WinREBootSupported:    false,
				LocalPBABootSupported: false,
				RemoteErase:           false,
			},
			resV2: dtov2.Features{
				UserConsent:           "kvm",
				EnableSOL:             true,
				EnableIDER:            true,
				EnableKVM:             true,
				Redirection:           true,
				KVMAvailable:          true,
				OptInState:            1,
				OCR:                   false,
				HTTPSBootSupported:    true,
				WinREBootSupported:    false,
				LocalPBABootSupported: false,
			},
			err: nil,
		},
		{
			name:   "success with OCR not supported",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetAMTRedirectionService().
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					GetIPSOptInService().
					Return(optin.Response{
						Body: optin.Body{
							GetAndPutResponse: optin.OptInServiceResponse{
								OptInRequired: 1,
								OptInState:    1,
							},
						},
					}, nil)
				man2.EXPECT().
					GetKVMRedirection().
					Return(kvm.Response{
						Body: kvm.Body{
							GetResponse: kvm.KVMRedirectionSAP{
								EnabledState: kvm.EnabledState(redirection.Enabled),
							},
						},
					}, nil)
				man2.EXPECT().
					GetBootService().
					Return(cimBoot.BootService{
						EnabledState: 32768,
					}, nil)
				man2.EXPECT().
					GetCIMBootSourceSetting().
					Return(cimBoot.Response{
						Body: cimBoot.Body{
							PullResponse: cimBoot.PullResponse{
								BootSourceSettingItems: []cimBoot.BootSourceSetting{
									{
										InstanceID: "Some other boot option", // No OCR boot option
									},
								},
							},
						},
					}, nil)
				man2.EXPECT().
					GetPowerCapabilities().
					Return(boot.BootCapabilitiesResponse{
						ForceUEFIHTTPSBoot:    false,
						ForceWinREBoot:        false,
						ForceUEFILocalPBABoot: false,
					}, nil)
				man2.EXPECT().
					GetBootData().
					Return(boot.BootSettingDataResponse{
						UEFIHTTPSBootEnabled:    false,
						WinREBootEnabled:        false,
						UEFILocalPBABootEnabled: false,
					}, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res:   featureSetNoOCR,
			resV2: featureSetV2NoOCR,
			err:   nil,
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
			res:   dto.Features{},
			resV2: dtov2.Features{},
			err:   devices.ErrGeneral,
		},
		{
			name:   "GetFeatures fails on redirection service",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetAMTRedirectionService().
					Return(redirection.Response{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res:   dto.Features{},
			resV2: dtov2.Features{},
			err:   ErrGeneral,
		},
		{
			name:   "GetFeatures fails on user consent",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetAMTRedirectionService().
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					GetIPSOptInService().
					Return(optin.Response{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.Features{
				EnableSOL:   true,
				EnableIDER:  true,
				Redirection: true,
			},
			resV2: dtov2.Features{
				EnableSOL:   true,
				EnableIDER:  true,
				Redirection: true,
			},
			err: ErrGeneral,
		},
		{
			name:   "GetFeatures fails on KVM with non-AMT error",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetAMTRedirectionService().
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					GetIPSOptInService().
					Return(optin.Response{
						Body: optin.Body{
							GetAndPutResponse: optin.OptInServiceResponse{
								OptInRequired: 1,
								OptInState:    1,
							},
						},
					}, nil)
				man2.EXPECT().
					GetKVMRedirection().
					Return(kvm.Response{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.Features{
				UserConsent: "kvm",
				EnableSOL:   true,
				EnableIDER:  true,
				Redirection: true,
				OptInState:  1,
			},
			resV2: dtov2.Features{
				UserConsent: "kvm",
				EnableSOL:   true,
				EnableIDER:  true,
				Redirection: true,
				OptInState:  1,
			},
			err: ErrGeneral,
		},
		{
			name:   "GetFeatures fails immediately on OCR",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetAMTRedirectionService().
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					GetIPSOptInService().
					Return(optin.Response{
						Body: optin.Body{
							GetAndPutResponse: optin.OptInServiceResponse{
								OptInRequired: 1,
								OptInState:    1,
							},
						},
					}, nil)
				man2.EXPECT().
					GetKVMRedirection().
					Return(kvm.Response{
						Body: kvm.Body{
							GetResponse: kvm.KVMRedirectionSAP{
								EnabledState: kvm.EnabledState(redirection.Enabled),
							},
						},
					}, nil)
				man2.EXPECT().
					GetBootService().
					Return(cimBoot.BootService{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res:   dto.Features{},
			resV2: dtov2.Features{},
			err:   ErrGeneral,
		},
		{
			name:   "GetFeatures fails on boot service",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetAMTRedirectionService().
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					GetIPSOptInService().
					Return(optin.Response{
						Body: optin.Body{
							GetAndPutResponse: optin.OptInServiceResponse{
								OptInRequired: 1,
								OptInState:    1,
							},
						},
					}, nil)
				man2.EXPECT().
					GetKVMRedirection().
					Return(kvm.Response{
						Body: kvm.Body{
							GetResponse: kvm.KVMRedirectionSAP{
								EnabledState: kvm.EnabledState(redirection.Enabled),
							},
						},
					}, nil)
				man2.EXPECT().
					GetBootService().
					Return(cimBoot.BootService{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res:   dto.Features{},
			resV2: dtov2.Features{},
			err:   ErrGeneral,
		},
		{
			name:   "GetFeatures fails on boot source setting",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetAMTRedirectionService().
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					GetIPSOptInService().
					Return(optin.Response{
						Body: optin.Body{
							GetAndPutResponse: optin.OptInServiceResponse{
								OptInRequired: 1,
								OptInState:    1,
							},
						},
					}, nil)
				man2.EXPECT().
					GetKVMRedirection().
					Return(kvm.Response{
						Body: kvm.Body{
							GetResponse: kvm.KVMRedirectionSAP{
								EnabledState: kvm.EnabledState(redirection.Enabled),
							},
						},
					}, nil)
				man2.EXPECT().
					GetBootService().
					Return(cimBoot.BootService{
						EnabledState: 32769,
					}, nil)
				man2.EXPECT().
					GetCIMBootSourceSetting().
					Return(cimBoot.Response{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res:   dto.Features{},
			resV2: dtov2.Features{},
			err:   ErrGeneral,
		},
		{
			name:   "GetFeatures fails on power capabilities",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetAMTRedirectionService().
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					GetIPSOptInService().
					Return(optin.Response{
						Body: optin.Body{
							GetAndPutResponse: optin.OptInServiceResponse{
								OptInRequired: 1,
								OptInState:    1,
							},
						},
					}, nil)
				man2.EXPECT().
					GetKVMRedirection().
					Return(kvm.Response{
						Body: kvm.Body{
							GetResponse: kvm.KVMRedirectionSAP{
								EnabledState: kvm.EnabledState(redirection.Enabled),
							},
						},
					}, nil)
				man2.EXPECT().
					GetBootService().
					Return(cimBoot.BootService{
						EnabledState: 32769,
					}, nil)
				man2.EXPECT().
					GetCIMBootSourceSetting().
					Return(cimBoot.Response{
						Body: cimBoot.Body{
							PullResponse: cimBoot.PullResponse{
								BootSourceSettingItems: []cimBoot.BootSourceSetting{
									{
										InstanceID: "Intel(r) AMT: Force OCR UEFI HTTPS Boot",
									},
									{
										InstanceID: "Intel(r) AMT: Force OCR UEFI Boot Option",
									},
								},
							},
						},
					}, nil)
				man2.EXPECT().
					GetPowerCapabilities().
					Return(boot.BootCapabilitiesResponse{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res:   dto.Features{},
			resV2: dtov2.Features{},
			err:   ErrGeneral,
		},
		{
			name:   "GetFeatures fails on boot data",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetAMTRedirectionService().
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					GetIPSOptInService().
					Return(optin.Response{
						Body: optin.Body{
							GetAndPutResponse: optin.OptInServiceResponse{
								OptInRequired: 1,
								OptInState:    1,
							},
						},
					}, nil)
				man2.EXPECT().
					GetKVMRedirection().
					Return(kvm.Response{
						Body: kvm.Body{
							GetResponse: kvm.KVMRedirectionSAP{
								EnabledState: kvm.EnabledState(redirection.Enabled),
							},
						},
					}, nil)
				man2.EXPECT().
					GetBootService().
					Return(cimBoot.BootService{
						EnabledState: 32769,
					}, nil)
				man2.EXPECT().
					GetCIMBootSourceSetting().
					Return(cimBoot.Response{
						Body: cimBoot.Body{
							PullResponse: cimBoot.PullResponse{
								BootSourceSettingItems: []cimBoot.BootSourceSetting{
									{
										InstanceID: "Intel(r) AMT: Force OCR UEFI HTTPS Boot",
									},
									{
										InstanceID: "Intel(r) AMT: Force OCR UEFI Boot Option",
									},
								},
							},
						},
					}, nil)
				man2.EXPECT().
					GetPowerCapabilities().
					Return(boot.BootCapabilitiesResponse{
						ForceUEFIHTTPSBoot:    true,
						ForceWinREBoot:        true,
						ForceUEFILocalPBABoot: true,
					}, nil)
				man2.EXPECT().
					GetBootData().
					Return(boot.BootSettingDataResponse{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res:   dto.Features{},
			resV2: dtov2.Features{},
			err:   ErrGeneral,
		},
		{
			name:   "GetFeatures on ISM",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetAMTRedirectionService().
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					GetIPSOptInService().
					Return(optin.Response{
						Body: optin.Body{
							GetAndPutResponse: optin.OptInServiceResponse{
								OptInState:    1,
								OptInRequired: 1,
							},
						},
					}, nil)
				man2.EXPECT().
					GetKVMRedirection().
					Return(kvm.Response{}, amterror.DecodeAMTErrorString(DestinationUnreachable))
				man2.EXPECT().
					GetBootService().
					Return(cimBoot.BootService{
						EnabledState: 32769,
					}, nil)
				man2.EXPECT().
					GetCIMBootSourceSetting().
					Return(cimBoot.Response{
						Body: cimBoot.Body{
							PullResponse: cimBoot.PullResponse{
								BootSourceSettingItems: []cimBoot.BootSourceSetting{
									{
										InstanceID: "Intel(r) AMT: Force OCR UEFI HTTPS Boot",
									},
									{
										InstanceID: "Intel(r) AMT: Force OCR UEFI Boot Option",
									},
								},
							},
						},
					}, nil)
				man2.EXPECT().
					GetPowerCapabilities().
					Return(boot.BootCapabilitiesResponse{
						ForceUEFIHTTPSBoot:    true,
						ForceWinREBoot:        true,
						ForceUEFILocalPBABoot: true,
					}, nil)
				man2.EXPECT().
					GetBootData().
					Return(boot.BootSettingDataResponse{
						UEFIHTTPSBootEnabled:    true,
						WinREBootEnabled:        true,
						UEFILocalPBABootEnabled: true,
					}, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res:   featureSetNoKVM,
			resV2: featureSetV2NoKVM,
			err:   nil,
		},
		{
			name:   "OCR with different EnabledState values - 32771",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetAMTRedirectionService().
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					GetIPSOptInService().
					Return(optin.Response{
						Body: optin.Body{
							GetAndPutResponse: optin.OptInServiceResponse{
								OptInRequired: 1,
								OptInState:    1,
							},
						},
					}, nil)
				man2.EXPECT().
					GetKVMRedirection().
					Return(kvm.Response{
						Body: kvm.Body{
							GetResponse: kvm.KVMRedirectionSAP{
								EnabledState: kvm.EnabledState(redirection.Enabled),
							},
						},
					}, nil)
				man2.EXPECT().
					GetBootService().
					Return(cimBoot.BootService{
						EnabledState: 32771, // Different enabled state that should also enable OCR
					}, nil)
				man2.EXPECT().
					GetCIMBootSourceSetting().
					Return(cimBoot.Response{
						Body: cimBoot.Body{
							PullResponse: cimBoot.PullResponse{
								BootSourceSettingItems: []cimBoot.BootSourceSetting{
									{
										InstanceID: "Intel(r) AMT: Force OCR UEFI HTTPS Boot",
									},
								},
							},
						},
					}, nil)
				man2.EXPECT().
					GetPowerCapabilities().
					Return(boot.BootCapabilitiesResponse{
						ForceUEFIHTTPSBoot: true,
					}, nil)
				man2.EXPECT().
					GetBootData().
					Return(boot.BootSettingDataResponse{
						UEFIHTTPSBootEnabled: true,
					}, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.Features{
				UserConsent:           "kvm",
				EnableSOL:             true,
				EnableIDER:            true,
				EnableKVM:             true,
				Redirection:           true,
				KVMAvailable:          true,
				OptInState:            1,
				OCR:                   true, // Should be true for EnabledState 32771
				HTTPSBootSupported:    true,
				WinREBootSupported:    false,
				LocalPBABootSupported: false,
			},
			resV2: dtov2.Features{
				UserConsent:           "kvm",
				EnableSOL:             true,
				EnableIDER:            true,
				EnableKVM:             true,
				Redirection:           true,
				KVMAvailable:          true,
				OptInState:            1,
				OCR:                   true,
				HTTPSBootSupported:    true,
				WinREBootSupported:    false,
				LocalPBABootSupported: false,
			},
			err: nil,
		},
		{
			name:   "OCR with mixed boot support",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetAMTRedirectionService().
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					GetIPSOptInService().
					Return(optin.Response{
						Body: optin.Body{
							GetAndPutResponse: optin.OptInServiceResponse{
								OptInRequired: 1,
								OptInState:    1,
							},
						},
					}, nil)
				man2.EXPECT().
					GetKVMRedirection().
					Return(kvm.Response{
						Body: kvm.Body{
							GetResponse: kvm.KVMRedirectionSAP{
								EnabledState: kvm.EnabledState(redirection.Enabled),
							},
						},
					}, nil)
				man2.EXPECT().
					GetBootService().
					Return(cimBoot.BootService{
						EnabledState: 32769,
					}, nil)
				man2.EXPECT().
					GetCIMBootSourceSetting().
					Return(cimBoot.Response{
						Body: cimBoot.Body{
							PullResponse: cimBoot.PullResponse{
								BootSourceSettingItems: []cimBoot.BootSourceSetting{
									{
										InstanceID:     "Intel(r) AMT: Force OCR UEFI Boot Option",
										BIOSBootString: "WinRe Recovery", // Only WinRE, no PBA or HTTPS
										BootString:     "winre.wim",
									},
								},
							},
						},
					}, nil)
				man2.EXPECT().
					GetPowerCapabilities().
					Return(boot.BootCapabilitiesResponse{
						ForceWinREBoot: true, // Only WinRE supported
					}, nil)
				man2.EXPECT().
					GetBootData().
					Return(boot.BootSettingDataResponse{
						WinREBootEnabled: true, // Only WinRE enabled
					}, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.Features{
				UserConsent:           "kvm",
				EnableSOL:             true,
				EnableIDER:            true,
				EnableKVM:             true,
				Redirection:           true,
				KVMAvailable:          true,
				OptInState:            1,
				OCR:                   true,
				HTTPSBootSupported:    false, // No HTTPS support
				WinREBootSupported:    true,  // Only WinRE supported
				LocalPBABootSupported: false, // No PBA support
			},
			resV2: dtov2.Features{
				UserConsent:           "kvm",
				EnableSOL:             true,
				EnableIDER:            true,
				EnableKVM:             true,
				Redirection:           true,
				KVMAvailable:          true,
				OptInState:            1,
				OCR:                   true,
				HTTPSBootSupported:    false,
				WinREBootSupported:    true,
				LocalPBABootSupported: false,
			},
			err: nil,
		},
		{
			name:   "GetFeatures fails immediately on OCR setup",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetAMTRedirectionService().
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					GetIPSOptInService().
					Return(optin.Response{
						Body: optin.Body{
							GetAndPutResponse: optin.OptInServiceResponse{
								OptInRequired: 1,
								OptInState:    1,
							},
						},
					}, nil)
				man2.EXPECT().
					GetKVMRedirection().
					Return(kvm.Response{
						Body: kvm.Body{
							GetResponse: kvm.KVMRedirectionSAP{
								EnabledState: kvm.EnabledState(redirection.Enabled),
							},
						},
					}, nil)
				man2.EXPECT().
					GetBootService().
					Return(cimBoot.BootService{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res:   dto.Features{},
			resV2: dtov2.Features{},
			err:   ErrGeneral,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initInfoTest(t)

			tc.manMock(wsmanMock, management)

			tc.repoMock(repo)

			v1, v2, err := useCase.GetFeatures(context.Background(), device.GUID)

			require.Equal(t, tc.res, v1)

			require.Equal(t, tc.resV2, v2)

			require.IsType(t, tc.err, err)
		})
	}
}

func TestSetFeatures(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
	}

	featureSet := dto.Features{
		UserConsent: "kvm",
		EnableSOL:   true,
		EnableIDER:  true,
		EnableKVM:   true,
		OCR:         true,
	}

	featureSetDisabledOCR := dto.Features{
		UserConsent: "kvm",
		EnableSOL:   true,
		EnableIDER:  true,
		EnableKVM:   true,
		Redirection: true,
		OCR:         false,
	}

	featureSetV2 := dtov2.Features{
		UserConsent:           "kvm",
		EnableSOL:             true,
		EnableIDER:            true,
		EnableKVM:             true,
		Redirection:           true,
		KVMAvailable:          true,
		OCR:                   true,
		HTTPSBootSupported:    true,
		WinREBootSupported:    false,
		LocalPBABootSupported: false,
		RemoteErase:           false,
	}

	featureSetDisabledOCRResult := dto.Features{
		UserConsent:           "kvm",
		EnableSOL:             true,
		EnableIDER:            true,
		EnableKVM:             true,
		Redirection:           true,
		OCR:                   false,
		HTTPSBootSupported:    true,
		WinREBootSupported:    false,
		LocalPBABootSupported: false,
		RemoteErase:           false,
	}

	featureSetV2DisabledOCR := dtov2.Features{
		UserConsent:           "kvm",
		EnableSOL:             true,
		EnableIDER:            true,
		EnableKVM:             true,
		Redirection:           true,
		KVMAvailable:          true,
		OCR:                   false,
		HTTPSBootSupported:    true,
		WinREBootSupported:    false,
		LocalPBABootSupported: false,
		RemoteErase:           false,
	}

	failGetByIDResult := dto.Features{}

	tests := []test{
		{
			name:    "Device not found - nil device",
			action:  0,
			manMock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(nil, nil) // Returns nil device, no error
			},
			res:   dto.Features{},
			resV2: dtov2.Features{},
			err:   devices.ErrNotFound,
		},
		{
			name:    "Device not found - empty GUID",
			action:  0,
			manMock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				emptyDevice := &entity.Device{
					GUID:     "", // Empty GUID
					TenantID: "tenant-id-456",
				}
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(emptyDevice, nil)
			},
			res:   dto.Features{},
			resV2: dtov2.Features{},
			err:   devices.ErrNotFound,
		},
		{
			name:   "success",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					RequestAMTRedirectionServiceStateChange(true, true).
					Return(redirection.EnableIDERAndSOL, 1, nil)
				man2.EXPECT().
					SetKVMRedirection(true).
					Return(1, nil)
				man2.EXPECT().
					GetAMTRedirectionService().
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					SetAMTRedirectionService(&redirection.RedirectionRequest{
						EnabledState:    redirection.EnabledState(redirection.EnableIDERAndSOL),
						ListenerEnabled: true,
					}).
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					GetIPSOptInService().
					Return(optin.Response{
						Body: optin.Body{
							GetAndPutResponse: optin.OptInServiceResponse{
								OptInRequired: 1,
								OptInState:    0,
							},
						},
					}, nil)
				man2.EXPECT().
					SetIPSOptInService(optin.OptInServiceRequest{
						OptInRequired: 1,
						OptInState:    0,
					}).
					Return(nil)
				man2.EXPECT().
					BootServiceStateChange(32769). // OCR enabled
					Return(cimBoot.BootService{}, nil)
				man2.EXPECT().
					GetBootService().
					Return(cimBoot.BootService{
						EnabledState: 32769,
					}, nil)
				man2.EXPECT().
					GetCIMBootSourceSetting().
					Return(cimBoot.Response{
						Body: cimBoot.Body{
							PullResponse: cimBoot.PullResponse{
								BootSourceSettingItems: []cimBoot.BootSourceSetting{
									{
										InstanceID: "Intel(r) AMT: Force OCR UEFI HTTPS Boot",
									},
									{
										InstanceID: "Intel(r) AMT: Force OCR UEFI Boot Option",
									},
								},
							},
						},
					}, nil)
				man2.EXPECT().
					GetPowerCapabilities().
					Return(boot.BootCapabilitiesResponse{
						ForceUEFIHTTPSBoot:    true,
						ForceWinREBoot:        false,
						ForceUEFILocalPBABoot: false,
					}, nil)
				man2.EXPECT().
					GetBootData().
					Return(boot.BootSettingDataResponse{
						UEFIHTTPSBootEnabled:    true,
						WinREBootEnabled:        false,
						UEFILocalPBABootEnabled: false,
					}, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.Features{
				UserConsent:           "kvm",
				EnableSOL:             true,
				EnableIDER:            true,
				EnableKVM:             true,
				Redirection:           true,
				OCR:                   true,
				HTTPSBootSupported:    true,
				WinREBootSupported:    false,
				LocalPBABootSupported: false,
				RemoteErase:           false,
			},
			resV2: featureSetV2,
			err:   nil,
		},
		{
			name:   "success with OCR disabled",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					RequestAMTRedirectionServiceStateChange(true, true).
					Return(redirection.EnableIDERAndSOL, 1, nil)
				man2.EXPECT().
					SetKVMRedirection(true).
					Return(1, nil)
				man2.EXPECT().
					GetAMTRedirectionService().
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					SetAMTRedirectionService(&redirection.RedirectionRequest{
						EnabledState:    redirection.EnabledState(redirection.EnableIDERAndSOL),
						ListenerEnabled: true,
					}).
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					GetIPSOptInService().
					Return(optin.Response{
						Body: optin.Body{
							GetAndPutResponse: optin.OptInServiceResponse{
								OptInRequired: 1,
								OptInState:    0,
							},
						},
					}, nil)
				man2.EXPECT().
					SetIPSOptInService(optin.OptInServiceRequest{
						OptInRequired: 1,
						OptInState:    0,
					}).
					Return(nil)
				man2.EXPECT().
					BootServiceStateChange(32768).
					Return(cimBoot.BootService{}, nil)
				man2.EXPECT().
					GetBootService().
					Return(cimBoot.BootService{
						EnabledState: 32768,
					}, nil)
				man2.EXPECT().
					GetCIMBootSourceSetting().
					Return(cimBoot.Response{
						Body: cimBoot.Body{
							PullResponse: cimBoot.PullResponse{
								BootSourceSettingItems: []cimBoot.BootSourceSetting{
									{
										InstanceID: "Intel(r) AMT: Force OCR UEFI HTTPS Boot",
									},
									{
										InstanceID: "Intel(r) AMT: Force OCR UEFI Boot Option",
									},
								},
							},
						},
					}, nil)
				man2.EXPECT().
					GetPowerCapabilities().
					Return(boot.BootCapabilitiesResponse{
						ForceUEFIHTTPSBoot:    true,
						ForceWinREBoot:        false,
						ForceUEFILocalPBABoot: false,
					}, nil)
				man2.EXPECT().
					GetBootData().
					Return(boot.BootSettingDataResponse{
						UEFIHTTPSBootEnabled:    true,
						WinREBootEnabled:        false,
						UEFILocalPBABootEnabled: false,
					}, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res:   featureSetDisabledOCRResult,
			resV2: featureSetV2DisabledOCR,
			err:   nil,
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
			res:   failGetByIDResult,
			resV2: dtov2.Features{},
			err:   devices.ErrGeneral,
		},
		{
			name:   "SetFeatures fails on redirection service",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					RequestAMTRedirectionServiceStateChange(true, true).
					Return(redirection.RequestedState(0), 0, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res:   failGetByIDResult,
			resV2: dtov2.Features{},
			err:   ErrGeneral,
		},
		{
			name:   "SetFeatures fails on KVM with non-AMT error",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					RequestAMTRedirectionServiceStateChange(true, true).
					Return(redirection.EnableIDERAndSOL, 1, nil)
				man2.EXPECT().
					SetKVMRedirection(true).
					Return(0, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			err: ErrGeneral,
		},
		{
			name:   "SetFeatures fails on KVM with AMT error",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					RequestAMTRedirectionServiceStateChange(true, true).
					Return(redirection.EnableIDERAndSOL, 1, nil)
				man2.EXPECT().
					SetKVMRedirection(true).
					Return(0, amterror.DecodeAMTErrorString(DestinationUnreachable))
				man2.EXPECT().
					GetAMTRedirectionService().
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					SetAMTRedirectionService(&redirection.RedirectionRequest{
						EnabledState:    redirection.EnabledState(redirection.EnableIDERAndSOL),
						ListenerEnabled: true,
					}).
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					GetIPSOptInService().
					Return(optin.Response{
						Body: optin.Body{
							GetAndPutResponse: optin.OptInServiceResponse{
								OptInRequired: 1,
								OptInState:    0,
							},
						},
					}, nil)
				man2.EXPECT().
					SetIPSOptInService(optin.OptInServiceRequest{
						OptInRequired: 1,
						OptInState:    0,
					}).
					Return(nil)
				man2.EXPECT().
					BootServiceStateChange(32769).
					Return(cimBoot.BootService{}, nil)
				man2.EXPECT().
					GetBootService().
					Return(cimBoot.BootService{
						EnabledState: 32769,
					}, nil)
				man2.EXPECT().
					GetCIMBootSourceSetting().
					Return(cimBoot.Response{
						Body: cimBoot.Body{
							PullResponse: cimBoot.PullResponse{
								BootSourceSettingItems: []cimBoot.BootSourceSetting{
									{
										InstanceID: "Intel(r) AMT: Force OCR UEFI HTTPS Boot",
									},
									{
										InstanceID: "Intel(r) AMT: Force OCR UEFI Boot Option",
									},
								},
							},
						},
					}, nil)
				man2.EXPECT().
					GetPowerCapabilities().
					Return(boot.BootCapabilitiesResponse{
						ForceUEFIHTTPSBoot:    true,
						ForceWinREBoot:        false,
						ForceUEFILocalPBABoot: false,
					}, nil)
				man2.EXPECT().
					GetBootData().
					Return(boot.BootSettingDataResponse{
						UEFIHTTPSBootEnabled:    true,
						WinREBootEnabled:        false,
						UEFILocalPBABootEnabled: false,
					}, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.Features{
				UserConsent:           "kvm",
				EnableSOL:             true,
				EnableIDER:            true,
				EnableKVM:             false,
				Redirection:           true,
				OCR:                   true,
				HTTPSBootSupported:    true,
				WinREBootSupported:    false,
				LocalPBABootSupported: false,
				RemoteErase:           false,
			},
			resV2: dtov2.Features{
				UserConsent:           "kvm",
				EnableSOL:             true,
				EnableIDER:            true,
				EnableKVM:             false,
				Redirection:           true,
				KVMAvailable:          false,
				OCR:                   true,
				HTTPSBootSupported:    true,
				WinREBootSupported:    false,
				LocalPBABootSupported: false,
			},
			err: nil,
		},
		{
			name:   "SetFeatures fails on redirection service set",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					RequestAMTRedirectionServiceStateChange(true, true).
					Return(redirection.EnableIDERAndSOL, 1, nil)
				man2.EXPECT().
					SetKVMRedirection(true).
					Return(1, nil)
				man2.EXPECT().
					GetAMTRedirectionService().
					Return(redirection.Response{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.Features{
				EnableSOL:  true,
				EnableIDER: true,
				EnableKVM:  true,
			},
			resV2: dtov2.Features{
				EnableSOL:    true,
				EnableIDER:   true,
				EnableKVM:    true,
				KVMAvailable: true,
			},
			err: ErrGeneral,
		},
		{
			name:   "SetFeatures fails on user consent",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					RequestAMTRedirectionServiceStateChange(true, true).
					Return(redirection.EnableIDERAndSOL, 1, nil)
				man2.EXPECT().
					SetKVMRedirection(true).
					Return(1, nil)
				man2.EXPECT().
					GetAMTRedirectionService().
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					SetAMTRedirectionService(&redirection.RedirectionRequest{
						EnabledState:    redirection.EnabledState(redirection.EnableIDERAndSOL),
						ListenerEnabled: true,
					}).
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					GetIPSOptInService().
					Return(optin.Response{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.Features{
				EnableSOL:   true,
				EnableIDER:  true,
				EnableKVM:   true,
				Redirection: true,
			},
			resV2: dtov2.Features{
				EnableSOL:    true,
				EnableIDER:   true,
				EnableKVM:    true,
				Redirection:  true,
				KVMAvailable: true,
			},
			err: ErrGeneral,
		},
		{
			name:   "SetFeatures fails on boot service state change",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					RequestAMTRedirectionServiceStateChange(true, true).
					Return(redirection.EnableIDERAndSOL, 1, nil)
				man2.EXPECT().
					SetKVMRedirection(true).
					Return(1, nil)
				man2.EXPECT().
					GetAMTRedirectionService().
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					SetAMTRedirectionService(&redirection.RedirectionRequest{
						EnabledState:    redirection.EnabledState(redirection.EnableIDERAndSOL),
						ListenerEnabled: true,
					}).
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					GetIPSOptInService().
					Return(optin.Response{
						Body: optin.Body{
							GetAndPutResponse: optin.OptInServiceResponse{
								OptInRequired: 1,
								OptInState:    0,
							},
						},
					}, nil)
				man2.EXPECT().
					SetIPSOptInService(optin.OptInServiceRequest{
						OptInRequired: 1,
						OptInState:    0,
					}).
					Return(nil)
				man2.EXPECT().
					BootServiceStateChange(32769).
					Return(cimBoot.BootService{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.Features{
				UserConsent: "kvm",
				EnableSOL:   true,
				EnableIDER:  true,
				EnableKVM:   true,
				Redirection: true,
			},
			resV2: dtov2.Features{
				UserConsent:  "kvm",
				EnableSOL:    true,
				EnableIDER:   true,
				EnableKVM:    true,
				Redirection:  true,
				KVMAvailable: true,
			},
			err: nil,
		},
		{
			name:   "SetFeatures fails on OCR settings retrieval after successful state change",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					RequestAMTRedirectionServiceStateChange(true, true).
					Return(redirection.EnableIDERAndSOL, 1, nil)
				man2.EXPECT().
					SetKVMRedirection(true).
					Return(1, nil)
				man2.EXPECT().
					GetAMTRedirectionService().
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					SetAMTRedirectionService(&redirection.RedirectionRequest{
						EnabledState:    redirection.EnabledState(redirection.EnableIDERAndSOL),
						ListenerEnabled: true,
					}).
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					GetIPSOptInService().
					Return(optin.Response{
						Body: optin.Body{
							GetAndPutResponse: optin.OptInServiceResponse{
								OptInRequired: 1,
								OptInState:    0,
							},
						},
					}, nil)
				man2.EXPECT().
					SetIPSOptInService(optin.OptInServiceRequest{
						OptInRequired: 1,
						OptInState:    0,
					}).
					Return(nil)
				man2.EXPECT().
					BootServiceStateChange(32769).
					Return(cimBoot.BootService{}, nil)
				man2.EXPECT().
					GetBootService().
					Return(cimBoot.BootService{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res:   dto.Features{},
			resV2: dtov2.Features{},
			err:   ErrGeneral,
		},
		{
			name:   "SetFeatures fails on setRedirectionService - SetAMTRedirectionService error",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					RequestAMTRedirectionServiceStateChange(true, true).
					Return(redirection.EnableIDERAndSOL, 1, nil)
				man2.EXPECT().
					SetKVMRedirection(true).
					Return(1, nil)
				man2.EXPECT().
					GetAMTRedirectionService().
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					SetAMTRedirectionService(&redirection.RedirectionRequest{
						EnabledState:    redirection.EnabledState(redirection.EnableIDERAndSOL),
						ListenerEnabled: true,
					}).
					Return(redirection.Response{}, ErrGeneral) // SetAMTRedirectionService fails
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.Features{
				EnableSOL:  true,
				EnableIDER: true,
				EnableKVM:  true,
			},
			resV2: dtov2.Features{
				EnableSOL:    true,
				EnableIDER:   true,
				EnableKVM:    true,
				KVMAvailable: true,
			},
			err: ErrGeneral,
		},
		{
			name:   "SetFeatures fails on setUserConsent - SetIPSOptInService error",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					RequestAMTRedirectionServiceStateChange(true, true).
					Return(redirection.EnableIDERAndSOL, 1, nil)
				man2.EXPECT().
					SetKVMRedirection(true).
					Return(1, nil)
				man2.EXPECT().
					GetAMTRedirectionService().
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					SetAMTRedirectionService(&redirection.RedirectionRequest{
						EnabledState:    redirection.EnabledState(redirection.EnableIDERAndSOL),
						ListenerEnabled: true,
					}).
					Return(redirection.Response{
						Body: redirection.Body{
							GetAndPutResponse: redirection.RedirectionResponse{
								EnabledState:    32771,
								ListenerEnabled: true,
							},
						},
					}, nil)
				man2.EXPECT().
					GetIPSOptInService().
					Return(optin.Response{
						Body: optin.Body{
							GetAndPutResponse: optin.OptInServiceResponse{
								OptInRequired: 1,
								OptInState:    0,
							},
						},
					}, nil)
				man2.EXPECT().
					SetIPSOptInService(optin.OptInServiceRequest{
						OptInRequired: 1,
						OptInState:    0,
					}).
					Return(ErrGeneral) // SetIPSOptInService fails
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.Features{
				EnableSOL:   true,
				EnableIDER:  true,
				EnableKVM:   true,
				Redirection: true,
			},
			resV2: dtov2.Features{
				EnableSOL:    true,
				EnableIDER:   true,
				EnableKVM:    true,
				Redirection:  true,
				KVMAvailable: true,
			},
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

			// Use the appropriate input for the OCR disabled test
			var inputFeatures dto.Features
			if tc.name == "success with OCR disabled" {
				inputFeatures = featureSetDisabledOCR
			} else {
				inputFeatures = featureSet
			}

			v1, v2, err := useCase.SetFeatures(context.Background(), device.GUID, inputFeatures)

			if tc.err == nil {
				require.Equal(t, tc.res, v1)
				require.Equal(t, tc.resV2, v2)
			} else {
				require.IsType(t, tc.err, err)
			}
		})
	}
}

func TestFindBootSettingInstances(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		settings []cimBoot.BootSourceSetting
		expected dtov2.BootSettings
	}{
		{
			name: "All boot instances found",
			settings: []cimBoot.BootSourceSetting{
				{
					InstanceID:     "Intel(r) AMT: Force OCR UEFI HTTPS Boot",
					BIOSBootString: "HTTPS Boot",
					BootString:     "https://example.com",
				},
				{
					InstanceID:     "Intel(r) AMT: Force OCR UEFI Boot Option",
					BIOSBootString: "WinRe Recovery",
					BootString:     "winre.wim",
				},
				{
					InstanceID:     "Intel(r) AMT: Force OCR UEFI Boot Option",
					BIOSBootString: "PBA Boot",
					BootString:     "pba.efi",
				},
			},
			expected: dtov2.BootSettings{
				IsHTTPSBootExists: true,
				IsWinREExists:     true,
				IsPBAExists:       true,
			},
		},
		{
			name: "Only HTTPS boot found",
			settings: []cimBoot.BootSourceSetting{
				{
					InstanceID:     "Intel(r) AMT: Force OCR UEFI HTTPS Boot",
					BIOSBootString: "HTTPS Boot",
					BootString:     "https://example.com",
				},
			},
			expected: dtov2.BootSettings{
				IsHTTPSBootExists: true,
				IsWinREExists:     false,
				IsPBAExists:       false,
			},
		},
		{
			name: "Multiple PBA instances",
			settings: []cimBoot.BootSourceSetting{
				{
					InstanceID:     "Intel(r) AMT: Force OCR UEFI Boot Option",
					BIOSBootString: "PBA Boot 1",
					BootString:     "pba1.efi",
				},
				{
					InstanceID:     "Intel(r) AMT: Force OCR UEFI Boot Option",
					BIOSBootString: "PBA Boot 2",
					BootString:     "pba2.efi",
				},
			},
			expected: dtov2.BootSettings{
				IsHTTPSBootExists: false,
				IsWinREExists:     false,
				IsPBAExists:       true,
			},
		},
		{
			name: "No matching boot instances",
			settings: []cimBoot.BootSourceSetting{
				{
					InstanceID:     "Some Other Boot Option",
					BIOSBootString: "Other Boot",
					BootString:     "other.efi",
				},
			},
			expected: dtov2.BootSettings{
				IsHTTPSBootExists: false,
				IsWinREExists:     false,
				IsPBAExists:       false,
			},
		},
		{
			name: "Early break when all found",
			settings: []cimBoot.BootSourceSetting{
				{
					InstanceID:     "Intel(r) AMT: Force OCR UEFI HTTPS Boot",
					BIOSBootString: "HTTPS Boot",
					BootString:     "https://example.com",
				},
				{
					InstanceID:     "Intel(r) AMT: Force OCR UEFI Boot Option",
					BIOSBootString: "WinRe Recovery",
					BootString:     "winre.wim",
				},
				{
					InstanceID:     "Intel(r) AMT: Force OCR UEFI Boot Option",
					BIOSBootString: "PBA Boot",
					BootString:     "pba.efi",
				},
				{
					InstanceID:     "Should Not Process This",
					BIOSBootString: "Extra Boot",
					BootString:     "extra.efi",
				},
			},
			expected: dtov2.BootSettings{
				IsHTTPSBootExists: true,
				IsWinREExists:     true,
				IsPBAExists:       true,
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := devices.FindBootSettingInstances(tc.settings)
			require.Equal(t, tc.expected, result)
		})
	}
}
