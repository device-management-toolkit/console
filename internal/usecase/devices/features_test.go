package devices_test

import (
	"context"
	"testing"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/amterror"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/boot"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/redirection"
	cimBoot "github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/boot"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/kvm"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/ips/optin"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

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
	}

	failGetByIDResult := dto.Features{}

	tests := []test{
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
