package devices_test

import (
	"context"
	"encoding/xml"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/publickey"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/credential"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/models"

	"github.com/device-management-toolkit/console/internal/entity"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/mocks"
	devices "github.com/device-management-toolkit/console/internal/usecase/devices"
	wsman "github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
	"github.com/device-management-toolkit/console/pkg/logger"
)

var ErrCertificate = errors.New("certificate error")

func initCertificateTest(t *testing.T) (*devices.UseCase, *mocks.MockWSMAN, *mocks.MockManagement, *mocks.MockDeviceManagementRepository) {
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

func TestGetCertificates(t *testing.T) {
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
					GetCertificates().
					Return(wsman.Certificates{}, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.SecuritySettings{
				ProfileAssociation: []dto.ProfileAssociation(nil),
				CertificateResponse: dto.CertificatePullResponse{
					KeyManagementItems: []dto.RefinedKeyManagementResponse{},
					Certificates:       []dto.RefinedCertificate{},
				},
				KeyResponse: dto.KeyPullResponse{
					Keys: []dto.Key{},
				},
			},
			err: nil,
		},
		{
			name:   "success with CIMCredentialContext",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetCertificates().
					Return(wsman.Certificates{
						CIMCredentialContextResponse: credential.PullResponse{
							XMLName: xml.Name{
								Space: "http://schemas.xmlsoap.org/ws/2004/09/enumeration",
								Local: "PullResponse",
							},
							Items: credential.Items{
								CredentialContextTLS: []credential.CredentialContext{
									{
										ElementInContext: models.AssociationReference{
											Address: "http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous",
											ReferenceParameters: models.ReferenceParametersNoNamespace{
												XMLName: xml.Name{
													Space: "http://schemas.xmlsoap.org/ws/2004/08/addressing",
													Local: "ReferenceParameters",
												},
												ResourceURI: "http://intel.com/wbem/wscim/1/amt-schema/1/AMT_PublicKeyCertificate",
												SelectorSet: models.SelectorNoNamespace{
													XMLName: xml.Name{
														Space: "http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd",
														Local: "SelectorSet",
													},
													Selectors: []models.SelectorResponse{
														{
															XMLName: xml.Name{
																Space: "http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd",
																Local: "Selector",
															},
															Name: "InstanceID",
															Text: "Intel(r) AMT Certificate: Handle: 0",
														},
													},
												},
											},
										},
										ElementProvidingContext: models.AssociationReference{
											Address: "http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous",
											ReferenceParameters: models.ReferenceParametersNoNamespace{
												XMLName: xml.Name{
													Space: "http://schemas.xmlsoap.org/ws/2004/08/addressing",
													Local: "ReferenceParameters",
												},
												ResourceURI: "http://intel.com/wbem/wscim/1/amt-schema/1/AMT_TLSProtocolEndpointCollection",
												SelectorSet: models.SelectorNoNamespace{
													XMLName: xml.Name{
														Space: "http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd",
														Local: "SelectorSet",
													},
													Selectors: []models.SelectorResponse{
														{
															XMLName: xml.Name{
																Space: "http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd",
																Local: "Selector",
															},
															Name: "ElementName",
															Text: "TLSProtocolEndpoint Instances Collection",
														},
													},
												},
											},
										},
									},
								},
							},
							EndOfSequence: xml.Name{
								Space: "http://schemas.xmlsoap.org/ws/2004/09/enumeration",
								Local: "EndOfSequence",
							},
						},
					}, nil)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.SecuritySettings{
				ProfileAssociation: []dto.ProfileAssociation{
					{
						Type:              "TLS",
						ProfileID:         "TLSProtocolEndpoint Instances Collection",
						RootCertificate:   nil,
						ClientCertificate: nil,
						Key:               nil,
					},
				},
				CertificateResponse: dto.CertificatePullResponse{
					KeyManagementItems: []dto.RefinedKeyManagementResponse{},
					Certificates:       []dto.RefinedCertificate{},
				},
				KeyResponse: dto.KeyPullResponse{
					Keys: []dto.Key{},
				},
			},
			err: nil,
		},
		{
			name:   "GetById fails",
			action: 0,
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(nil, ErrGeneral)
			},
			res: dto.SecuritySettings{},
			err: devices.ErrGeneral,
		},
		{
			name:   "GetCertificates fails",
			action: 0,
			manMock: func(man *mocks.MockWSMAN, man2 *mocks.MockManagement) {
				man.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man2)
				man2.EXPECT().
					GetCertificates().
					Return(wsman.Certificates{}, ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			res: dto.SecuritySettings{},
			err: ErrGeneral,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initCertificateTest(t)

			if tc.manMock != nil {
				tc.manMock(wsmanMock, management)
			}

			tc.repoMock(repo)

			res, err := useCase.GetCertificates(context.Background(), device.GUID)

			require.Equal(t, tc.res, res)
			require.IsType(t, tc.err, err)
		})
	}
}

func TestAddCertificate(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
	}

	validCertPEM := "-----BEGIN CERTIFICATE-----\nMIIDtTM=\n-----END CERTIFICATE-----"

	tests := []struct {
		name     string
		certInfo dto.CertInfo
		mock     func(m *mocks.MockWSMAN, man *mocks.MockManagement)
		repoMock func(repo *mocks.MockDeviceManagementRepository)
		expected string
		err      error
	}{
		{
			name: "get device by ID fails",
			certInfo: dto.CertInfo{
				Cert:      validCertPEM,
				IsTrusted: true,
			},
			mock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(nil, ErrGeneral)
			},
			expected: "",
			err:      ErrGeneral,
		},
		{
			name: "device not found",
			certInfo: dto.CertInfo{
				Cert:      validCertPEM,
				IsTrusted: true,
			},
			mock: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(nil, nil)
			},
			expected: "",
			err:      devices.ErrNotFound,
		},
		{
			name: "base64 decode fails",
			certInfo: dto.CertInfo{
				Cert:      validCertPEM,
				IsTrusted: true,
			},
			mock: func(m *mocks.MockWSMAN, man *mocks.MockManagement) {
				m.EXPECT().
					SetupWsmanClient(gomock.Any(), false, true).
					Return(man)
				man.EXPECT().
					AddTrustedRootCert(gomock.Any()).
					Return("", ErrGeneral)
			},
			repoMock: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().
					GetByID(context.Background(), device.GUID, "").
					Return(device, nil)
			},
			expected: "",
			err:      ErrCertificate,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initCertificateTest(t)

			tc.mock(wsmanMock, management)
			tc.repoMock(repo)

			result, err := useCase.AddCertificate(context.Background(), device.GUID, tc.certInfo)

			require.Equal(t, tc.expected, result)

			if tc.err != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDeleteCertificate(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
	}

	tests := []struct {
		name       string
		instanceID string
		mockRepo   func(*mocks.MockDeviceManagementRepository)
		mockWsman  func(*mocks.MockWSMAN, *mocks.MockManagement)
		err        error
	}{
		{
			name:       "device not found - repository error",
			instanceID: "Intel(r) AMT Certificate: Handle: 1",
			mockRepo: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(nil, errors.New("device not found"))
			},
			mockWsman: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {
				// No WSMAN calls expected
			},
			err: errors.New("device not found"),
		},
		{
			name:       "device not found - nil device",
			instanceID: "Intel(r) AMT Certificate: Handle: 1",
			mockRepo: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(nil, nil)
			},
			mockWsman: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {
				// No WSMAN calls expected
			},
			err: devices.ErrNotFound,
		},
		{
			name:       "device found but empty GUID",
			instanceID: "Intel(r) AMT Certificate: Handle: 1",
			mockRepo: func(repo *mocks.MockDeviceManagementRepository) {
				emptyDevice := &entity.Device{GUID: "", TenantID: "tenant-id-456"}
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(emptyDevice, nil)
			},
			mockWsman: func(_ *mocks.MockWSMAN, _ *mocks.MockManagement) {
				// No WSMAN calls expected
			},
			err: devices.ErrNotFound,
		},
		{
			name:       "GetCertificates fails",
			instanceID: "Intel(r) AMT Certificate: Handle: 1",
			mockRepo: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil).Times(2) // Called twice: once by DeleteCertificate, once by GetCertificates
			},
			mockWsman: func(wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
				wsmanMock.EXPECT().SetupWsmanClient(*device, false, true).Return(management)
				management.EXPECT().GetCertificates().Return(wsman.Certificates{}, errors.New("wsman error"))
			},
			err: errors.New("wsman error"),
		},
		{
			name:       "certificate not found in response",
			instanceID: "Intel(r) AMT Certificate: Handle: 999",
			mockRepo: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil).Times(2) // Called twice: once by DeleteCertificate, once by GetCertificates
			},
			mockWsman: func(wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
				wsmanMock.EXPECT().SetupWsmanClient(*device, false, true).Return(management)
				// Return empty certificates response
				management.EXPECT().GetCertificates().Return(wsman.Certificates{}, nil)
			},
			err: devices.ErrNotFound,
		},
		{
			name:       "certificate associated with profiles",
			instanceID: "Intel(r) AMT Certificate: Handle: 1",
			mockRepo: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil).Times(2) // Called twice: once by DeleteCertificate, once by GetCertificates
			},
			mockWsman: func(wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
				wsmanMock.EXPECT().SetupWsmanClient(*device, false, true).Return(management)
				// Return certificate with associated profiles
				certificates := wsman.Certificates{
					PublicKeyCertificateResponse: publickey.RefinedPullResponse{
						PublicKeyCertificateItems: []publickey.RefinedPublicKeyCertificateResponse{
							{
								InstanceID:             "Intel(r) AMT Certificate: Handle: 1",
								Subject:                "CN=Test Certificate",
								Issuer:                 "CN=Test CA",
								TrustedRootCertificate: false,
								ReadOnlyCertificate:    false,
								AssociatedProfiles:     []string{"TLS - profile1"},
							},
						},
					},
				}
				management.EXPECT().GetCertificates().Return(certificates, nil)
			},
			err: &dto.NotValidError{},
		},
		{
			name:       "certificate is read-only",
			instanceID: "Intel(r) AMT Certificate: Handle: 1",
			mockRepo: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil).Times(2) // Called twice: once by DeleteCertificate, once by GetCertificates
			},
			mockWsman: func(wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
				wsmanMock.EXPECT().SetupWsmanClient(*device, false, true).Return(management)
				// Return read-only certificate
				certificates := wsman.Certificates{
					PublicKeyCertificateResponse: publickey.RefinedPullResponse{
						PublicKeyCertificateItems: []publickey.RefinedPublicKeyCertificateResponse{
							{
								InstanceID:             "Intel(r) AMT Certificate: Handle: 1",
								Subject:                "CN=Test Certificate",
								Issuer:                 "CN=Test CA",
								TrustedRootCertificate: false,
								ReadOnlyCertificate:    true,
								AssociatedProfiles:     []string{},
							},
						},
					},
				}
				management.EXPECT().GetCertificates().Return(certificates, nil)
			},
			err: &dto.NotValidError{},
		},
		{
			name:       "DeleteCertificate fails",
			instanceID: "Intel(r) AMT Certificate: Handle: 1",
			mockRepo: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil).Times(2) // Called twice: once by DeleteCertificate, once by GetCertificates
			},
			mockWsman: func(wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
				wsmanMock.EXPECT().SetupWsmanClient(*device, false, true).Return(management).Times(2) // Called twice: once for GetCertificates, once for DeleteCertificate
				// Return valid certificate that can be deleted
				certificates := wsman.Certificates{
					PublicKeyCertificateResponse: publickey.RefinedPullResponse{
						PublicKeyCertificateItems: []publickey.RefinedPublicKeyCertificateResponse{
							{
								InstanceID:             "Intel(r) AMT Certificate: Handle: 1",
								Subject:                "CN=Test Certificate",
								Issuer:                 "CN=Test CA",
								TrustedRootCertificate: false,
								ReadOnlyCertificate:    false,
								AssociatedProfiles:     []string{},
							},
						},
					},
				}
				management.EXPECT().GetCertificates().Return(certificates, nil)
				management.EXPECT().DeleteCertificate("Intel(r) AMT Certificate: Handle: 1").Return(errors.New("wsman delete error"))
			},
			err: devices.ErrDeviceUseCase,
		},
		{
			name:       "successful certificate deletion",
			instanceID: "Intel(r) AMT Certificate: Handle: 1",
			mockRepo: func(repo *mocks.MockDeviceManagementRepository) {
				repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil).Times(2) // Called twice: once by DeleteCertificate, once by GetCertificates
			},
			mockWsman: func(wsmanMock *mocks.MockWSMAN, management *mocks.MockManagement) {
				wsmanMock.EXPECT().SetupWsmanClient(*device, false, true).Return(management).Times(2) // Called twice: once for GetCertificates, once for DeleteCertificate
				// Return valid certificate that can be deleted
				certificates := wsman.Certificates{
					PublicKeyCertificateResponse: publickey.RefinedPullResponse{
						PublicKeyCertificateItems: []publickey.RefinedPublicKeyCertificateResponse{
							{
								InstanceID:             "Intel(r) AMT Certificate: Handle: 1",
								Subject:                "CN=Test Certificate",
								Issuer:                 "CN=Test CA",
								TrustedRootCertificate: false,
								ReadOnlyCertificate:    false,
								AssociatedProfiles:     []string{},
							},
						},
					},
				}
				management.EXPECT().GetCertificates().Return(certificates, nil)
				management.EXPECT().DeleteCertificate("Intel(r) AMT Certificate: Handle: 1").Return(nil)
			},
			err: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			useCase, wsmanMock, management, repo := initCertificateTest(t)

			tc.mockRepo(repo)
			tc.mockWsman(wsmanMock, management)

			err := useCase.DeleteCertificate(context.Background(), device.GUID, tc.instanceID)

			if tc.err != nil {
				require.Error(t, err)
				// Check for specific error types where applicable
				var notValidErr dto.NotValidError
				if errors.As(tc.err, &notValidErr) {
					// Match found, check assertion
					require.ErrorAs(t, err, &notValidErr)
				}

				// Check that error messages contain expected content for validation errors
				if tc.name == "certificate associated with profiles" || tc.name == "certificate is read-only" {
					var validationErr dto.NotValidError
					require.ErrorAs(t, err, &validationErr)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestDeleteCertificate_Integration tests the full flow by creating a custom usecase with mocked GetCertificates.
func TestDeleteCertificate_Integration(t *testing.T) {
	t.Parallel()

	device := &entity.Device{
		GUID:     "device-guid-123",
		TenantID: "tenant-id-456",
	}

	// Test case: certificate exists and can be deleted
	t.Run("certificate exists and can be deleted", func(t *testing.T) {
		t.Parallel()

		useCase, wsmanMock, management, repo := initCertificateTest(t)

		repo.EXPECT().GetByID(context.Background(), device.GUID, "").Return(device, nil).Times(2) // Called twice: once by DeleteCertificate, once by GetCertificates
		wsmanMock.EXPECT().SetupWsmanClient(*device, false, true).Return(management).Times(2)     // Called twice: once for GetCertificates, once for DeleteCertificate setup

		// Mock GetCertificates to return a certificate that can be deleted
		certificates := wsman.Certificates{
			PublicKeyCertificateResponse: publickey.RefinedPullResponse{
				PublicKeyCertificateItems: []publickey.RefinedPublicKeyCertificateResponse{
					{
						InstanceID:             "Intel(r) AMT Certificate: Handle: 1",
						Subject:                "CN=Integration Test Certificate",
						Issuer:                 "CN=Test CA",
						TrustedRootCertificate: false,
						ReadOnlyCertificate:    false,
						AssociatedProfiles:     []string{}, // No associated profiles, so it can be deleted
					},
				},
			},
		}
		management.EXPECT().GetCertificates().Return(certificates, nil)

		// Mock DeleteCertificate
		management.EXPECT().DeleteCertificate("Intel(r) AMT Certificate: Handle: 1").Return(nil)

		err := useCase.DeleteCertificate(context.Background(), device.GUID, "Intel(r) AMT Certificate: Handle: 1")
		require.NoError(t, err) // Should succeed now
	})
}
