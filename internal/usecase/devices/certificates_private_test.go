package devices

import (
	"encoding/xml"
	"testing"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/publickey"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/amt/publicprivate"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/concrete"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/credential"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/models"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/devices/wsman"
)

type certificateTest struct {
	name string
	res  dto.SecuritySettings
	err  error

	profileType string
}

var getResponse wsman.Certificates = wsman.Certificates{
	ConcreteDependencyResponse: concrete.PullResponse{
		XMLName: xml.Name{
			Space: "http://schemas.xmlsoap.org/ws/2004/09/enumeration",
			Local: "PullResponse",
		},
		Items: []concrete.ConcreteDependency{
			{
				Antecedent: models.AssociationReference{
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
				Dependent: models.AssociationReference{
					Address: "http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous",
					ReferenceParameters: models.ReferenceParametersNoNamespace{
						XMLName: xml.Name{
							Space: "http://schemas.xmlsoap.org/ws/2004/08/addressing",
							Local: "ReferenceParameters",
						},
						ResourceURI: "http://intel.com/wbem/wscim/1/amt-schema/1/AMT_PublicPrivateKeyPair",
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
									Text: "Intel(r) AMT Key: Handle: 0",
								},
							},
						},
					},
				},
			},
			{
				Antecedent: models.AssociationReference{
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
									Text: "Intel(r) AMT Certificate: Handle: 1",
								},
							},
						},
					},
				},
				Dependent: models.AssociationReference{
					Address: "http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous",
					ReferenceParameters: models.ReferenceParametersNoNamespace{
						XMLName: xml.Name{
							Space: "http://schemas.xmlsoap.org/ws/2004/08/addressing",
							Local: "ReferenceParameters",
						},
						ResourceURI: "http://intel.com/wbem/wscim/1/amt-schema/1/AMT_PublicPrivateKeyPair",
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
									Text: "Intel(r) AMT Key: Handle: 1",
								},
							},
						},
					},
				},
			},
			{
				Antecedent: models.AssociationReference{
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
									Text: "Intel(r) AMT Certificate: Handle: 3",
								},
							},
						},
					},
				},
				Dependent: models.AssociationReference{
					Address: "http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous",
					ReferenceParameters: models.ReferenceParametersNoNamespace{
						XMLName: xml.Name{
							Space: "http://schemas.xmlsoap.org/ws/2004/08/addressing",
							Local: "ReferenceParameters",
						},
						ResourceURI: "http://intel.com/wbem/wscim/1/amt-schema/1/AMT_PublicPrivateKeyPair",
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
									Text: "Intel(r) AMT Key: Handle: 2",
								},
							},
						},
					},
				},
			},
			{
				Antecedent: models.AssociationReference{
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
									Text: "Intel(r) AMT Certificate: Handle: 4",
								},
							},
						},
					},
				},
				Dependent: models.AssociationReference{
					Address: "http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous",
					ReferenceParameters: models.ReferenceParametersNoNamespace{
						XMLName: xml.Name{
							Space: "http://schemas.xmlsoap.org/ws/2004/08/addressing",
							Local: "ReferenceParameters",
						},
						ResourceURI: "http://intel.com/wbem/wscim/1/amt-schema/1/AMT_PublicPrivateKeyPair",
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
									Text: "Intel(r) AMT Key: Handle: 3",
								},
							},
						},
					},
				},
			},
		},
	},
	PublicKeyCertificateResponse: publickey.RefinedPullResponse{
		KeyManagementItems: []publickey.RefinedKeyManagementResponse{
			{
				CreationClassName:       "",
				ElementName:             "",
				EnabledDefault:          0,
				EnabledState:            0,
				Name:                    "",
				RequestedState:          0,
				SystemCreationClassName: "",
				SystemName:              "",
			},
		},
		PublicKeyCertificateItems: []publickey.RefinedPublicKeyCertificateResponse{
			{
				ElementName:            "Intel(r) AMT Certificate",
				InstanceID:             "Intel(r) AMT Certificate: Handle: 2",
				X509Certificate:        "TestCertRoot",
				TrustedRootCertificate: true,
				Issuer:                 "TestIssuer",
				Subject:                "TestSubject",
				ReadOnlyCertificate:    false,
				PublicKeyHandle:        "",
				AssociatedProfiles:     nil,
			},
			{
				ElementName:            "Intel(r) AMT Certificate",
				InstanceID:             "Intel(r) AMT Certificate: Handle: 0",
				X509Certificate:        "TestCert0",
				TrustedRootCertificate: false,
				Issuer:                 "TestIssuer",
				Subject:                "TestSubject2",
				ReadOnlyCertificate:    false,
				PublicKeyHandle:        "",
				AssociatedProfiles:     nil,
			},
			{
				ElementName:            "Intel(r) AMT Certificate",
				InstanceID:             "Intel(r) AMT Certificate: Handle: 1",
				X509Certificate:        "TestCert1",
				TrustedRootCertificate: false,
				Issuer:                 "TestIssuer",
				Subject:                "TestSubject2",
				ReadOnlyCertificate:    false,
				PublicKeyHandle:        "",
				AssociatedProfiles:     nil,
			},
			{
				ElementName:            "Intel(r) AMT Certificate",
				InstanceID:             "Intel(r) AMT Certificate: Handle: 3",
				X509Certificate:        "TestCert3",
				TrustedRootCertificate: false,
				Issuer:                 "TestIssuer",
				Subject:                "A=Test, CN=CommonName, C=Country",
				ReadOnlyCertificate:    false,
				PublicKeyHandle:        "",
				AssociatedProfiles:     nil,
			},
		},
	},
	PublicPrivateKeyPairResponse: publicprivate.RefinedPullResponse{
		PublicPrivateKeyPairItems: []publicprivate.RefinedPublicPrivateKeyPair{
			{
				ElementName:       "Intel(r) AMT Key",
				InstanceID:        "Intel(r) AMT Key: Handle: 0",
				DERKey:            "Key0",
				CertificateHandle: "",
			},
			{
				ElementName:       "Intel(r) AMT Key",
				InstanceID:        "Intel(r) AMT Key: Handle: 1",
				DERKey:            "Key1",
				CertificateHandle: "",
			},
			{
				ElementName:       "Intel(r) AMT Key",
				InstanceID:        "Intel(r) AMT Key: Handle: 2",
				DERKey:            "Key2",
				CertificateHandle: "",
			},
		},
	},
	CIMCredentialContextResponse: credential.PullResponse{
		XMLName: xml.Name{
			Space: "http://schemas.xmlsoap.org/ws/2004/09/enumeration",
			Local: "PullResponse",
		},
		Items: credential.Items{
			CredentialContext: []credential.CredentialContext{
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
										Text: "Intel(r) AMT Certificate: Handle: 2",
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
							ResourceURI: "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_IEEE8021xSettings",
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
										Text: "Intel(r) AMT:IEEE 802.1x Settings TestWifi8021xTLS",
									},
								},
							},
						},
					},
				},
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
										Text: "Intel(r) AMT Certificate: Handle: 3",
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
							ResourceURI: "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_IEEE8021xSettings",
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
										Text: "Intel(r) AMT:IEEE 802.1x Settings TestWifi8021xTLS",
									},
								},
							},
						},
					},
				},
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
										Text: "Intel(r) AMT Certificate: Handle: 2",
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
							ResourceURI: "http://schemas.dmtf.org/wbem/wscim/1/cim-schema/2/CIM_IEEE8021xSettings",
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
										Text: "Intel(r) AMT:IEEE 802.1x Settings TestWifi8021xTLS2",
									},
								},
							},
						},
					},
				},
			},
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
			CredentialContext8021x: []credential.CredentialContext{
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
										Text: "Intel(r) AMT Certificate: Handle: 1",
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
										Name: "InstanceID",
										Text: "Intel(r) AMT: 8021X Settings",
									},
								},
							},
						},
					},
				},
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
										Text: "Intel(r) AMT Certificate: Handle: 2",
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
										Name: "InstanceID",
										Text: "Intel(r) AMT: 8021X Settings",
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
}

var parsedCerts = dto.CertificatePullResponse{
	KeyManagementItems: []dto.RefinedKeyManagementResponse{
		{
			CreationClassName:       "",
			ElementName:             "",
			EnabledDefault:          0,
			EnabledState:            0,
			Name:                    "",
			RequestedState:          0,
			SystemCreationClassName: "",
			SystemName:              "",
		},
	},
	Certificates: []dto.RefinedCertificate{
		{
			ElementName:            "Intel(r) AMT Certificate",
			InstanceID:             "Intel(r) AMT Certificate: Handle: 2",
			X509Certificate:        "TestCertRoot",
			TrustedRootCertificate: true,
			Issuer:                 "TestIssuer",
			Subject:                "TestSubject",
			ReadOnlyCertificate:    false,
			PublicKeyHandle:        "",
			AssociatedProfiles:     []string(nil),
			DisplayName:            "Intel(r) AMT Certificate: Handle: 2",
		},
		{
			ElementName:     "Intel(r) AMT Certificate",
			InstanceID:      "Intel(r) AMT Certificate: Handle: 0",
			X509Certificate: "TestCert0", TrustedRootCertificate: false,
			Issuer:              "TestIssuer",
			Subject:             "TestSubject2",
			ReadOnlyCertificate: false,
			PublicKeyHandle:     "",
			AssociatedProfiles:  []string(nil),
			DisplayName:         "Intel(r) AMT Certificate: Handle: 0",
		},
		{
			ElementName:            "Intel(r) AMT Certificate",
			InstanceID:             "Intel(r) AMT Certificate: Handle: 1",
			X509Certificate:        "TestCert1",
			TrustedRootCertificate: false,
			Issuer:                 "TestIssuer",
			Subject:                "TestSubject2",
			ReadOnlyCertificate:    false,
			PublicKeyHandle:        "", AssociatedProfiles: []string(nil),
			DisplayName: "Intel(r) AMT Certificate: Handle: 1",
		},
		{
			ElementName:     "Intel(r) AMT Certificate",
			InstanceID:      "Intel(r) AMT Certificate: Handle: 3",
			X509Certificate: "TestCert3", TrustedRootCertificate: false,
			Issuer:              "TestIssuer",
			Subject:             "A=Test, CN=CommonName, C=Country",
			ReadOnlyCertificate: false,
			PublicKeyHandle:     "",
			AssociatedProfiles:  []string(nil),
			DisplayName:         "CommonName",
		},
	},
}

var parsedKeys = dto.KeyPullResponse{
	Keys: []dto.Key{
		{
			ElementName:       "Intel(r) AMT Key",
			InstanceID:        "Intel(r) AMT Key: Handle: 0",
			DERKey:            "Key0",
			CertificateHandle: "",
		},
		{
			ElementName:       "Intel(r) AMT Key",
			InstanceID:        "Intel(r) AMT Key: Handle: 1",
			DERKey:            "Key1",
			CertificateHandle: "",
		},
		{
			ElementName:       "Intel(r) AMT Key",
			InstanceID:        "Intel(r) AMT Key: Handle: 2",
			DERKey:            "Key2",
			CertificateHandle: "",
		},
	},
}

func TestCertificatesToDTO(t *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying env variables.
	certs := CertificatesToDTO(&getResponse.PublicKeyCertificateResponse)
	require.IsType(t, parsedCerts, certs)
}

func TestKeysToDTO(t *testing.T) { //nolint:paralleltest // cannot have simultaneous tests modifying env variables.
	keys := KeysToDTO(&getResponse.PublicPrivateKeyPairResponse)
	require.IsType(t, parsedKeys, keys)
}

func TestProcessCertificates(t *testing.T) {
	t.Parallel()

	securitySettings := dto.SecuritySettings{
		CertificateResponse: parsedCerts,
		KeyResponse:         parsedKeys,
	}

	tests := []certificateTest{
		{
			name: "success",
			res: dto.SecuritySettings{
				ProfileAssociation: []dto.ProfileAssociation{
					{
						Type:      "Wireless",
						ProfileID: "TestWifi8021xTLS",
						RootCertificate: &dto.RefinedCertificate{
							ElementName:            "Intel(r) AMT Certificate",
							InstanceID:             "Intel(r) AMT Certificate: Handle: 2",
							X509Certificate:        "TestCertRoot",
							TrustedRootCertificate: true,
							ReadOnlyCertificate:    false,
							PublicKeyHandle:        "",
							AssociatedProfiles:     nil,
						},
						ClientCertificate: &dto.RefinedCertificate{
							ElementName:            "Intel(r) AMT Certificate",
							InstanceID:             "Intel(r) AMT Certificate: Handle: 3",
							X509Certificate:        "TestCert3",
							TrustedRootCertificate: false,
							ReadOnlyCertificate:    false,
							PublicKeyHandle:        "",
							AssociatedProfiles:     nil,
						},
						Key: &dto.Key{
							ElementName:       "Intel(r) AMT Key",
							InstanceID:        "Intel(r) AMT Key: Handle: 2",
							DERKey:            "Key2",
							CertificateHandle: "",
						},
					},
					{
						Type:      "Wireless",
						ProfileID: "TestWifi8021xTLS2",
						RootCertificate: &dto.RefinedCertificate{
							ElementName:            "Intel(r) AMT Certificate",
							InstanceID:             "Intel(r) AMT Certificate: Handle: 2",
							X509Certificate:        "TestCertRoot",
							TrustedRootCertificate: true,
							ReadOnlyCertificate:    false,
							PublicKeyHandle:        "",
							AssociatedProfiles: []string{
								"Wireless - TestWifi8021xTLS",
								"Wireless - TestWifi8021xTLS2",
							},
						},
						ClientCertificate: &dto.RefinedCertificate{},
						Key:               &dto.Key{},
					},
				},
				CertificateResponse: dto.CertificatePullResponse{
					KeyManagementItems: nil,
					Certificates: []dto.RefinedCertificate{
						{
							ElementName:            "Intel(r) AMT Certificate",
							InstanceID:             "Intel(r) AMT Certificate: Handle: 2",
							X509Certificate:        "TestRootCert",
							TrustedRootCertificate: true,
							ReadOnlyCertificate:    false,
							PublicKeyHandle:        "",
							AssociatedProfiles: []string{
								"Wireless - exampleWifi8021xTLS",
								"Wireless - exampleWifi8021xTLS2",
							},
						},
						{
							ElementName:            "Intel(r) AMT Certificate",
							InstanceID:             "Intel(r) AMT Certificate: Handle: 0",
							X509Certificate:        "TestCert0",
							TrustedRootCertificate: false,
							ReadOnlyCertificate:    false,
							PublicKeyHandle:        "Intel(r) AMT Key: Handle: 0",
							AssociatedProfiles:     nil,
						},
						{
							ElementName:            "Intel(r) AMT Certificate",
							InstanceID:             "Intel(r) AMT Certificate: Handle: 1",
							X509Certificate:        "TestCert1",
							TrustedRootCertificate: false,
							ReadOnlyCertificate:    false,
							PublicKeyHandle:        "",
							AssociatedProfiles:     nil,
						},
						{
							ElementName:            "Intel(r) AMT Certificate",
							InstanceID:             "Intel(r) AMT Certificate: Handle: 3",
							X509Certificate:        "TestCert3",
							TrustedRootCertificate: false,
							ReadOnlyCertificate:    false,
							PublicKeyHandle:        "",
							AssociatedProfiles: []string{
								"Wireless - TestWifi8021xTLS",
							},
						},
					},
				},
				KeyResponse: dto.KeyPullResponse{
					Keys: []dto.Key{
						{
							ElementName:       "Intel(r) AMT Key",
							InstanceID:        "Intel(r) AMT Key: Handle: 0",
							DERKey:            "Key0",
							CertificateHandle: "",
						},
						{
							ElementName:       "Intel(r) AMT Key",
							InstanceID:        "Intel(r) AMT Key: Handle: 1",
							DERKey:            "Key1",
							CertificateHandle: "",
						},
						{
							ElementName:       "Intel(r) AMT Key",
							InstanceID:        "Intel(r) AMT Key: Handle: 2",
							DERKey:            "Key2",
							CertificateHandle: "Intel(r) AMT Certificate: Handle: 3",
						},
					},
				},
			},
			err:         nil,
			profileType: "Wireless",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			processCertificates(getResponse.CIMCredentialContextResponse.Items.CredentialContext, getResponse, tc.profileType, &securitySettings)

			require.Equal(t, tc.res.ProfileAssociation[0].Type, securitySettings.ProfileAssociation[0].Type)
			require.Equal(t, tc.res.ProfileAssociation[0].ClientCertificate.InstanceID, securitySettings.ProfileAssociation[0].ClientCertificate.InstanceID)
			require.Equal(t, tc.res.ProfileAssociation[1].RootCertificate.AssociatedProfiles, securitySettings.ProfileAssociation[1].RootCertificate.AssociatedProfiles)
			require.Equal(t, tc.res.CertificateResponse.Certificates[3].AssociatedProfiles, securitySettings.CertificateResponse.Certificates[3].AssociatedProfiles)
			require.Equal(t, tc.res.KeyResponse.Keys[2].CertificateHandle, securitySettings.KeyResponse.Keys[2].CertificateHandle)
		})
	}
}
