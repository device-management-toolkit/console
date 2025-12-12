/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

package dto

type (
	SecuritySettings struct {
		ProfileAssociation  []ProfileAssociation    `json:"profileAssociation"`
		CertificateResponse CertificatePullResponse `json:"certificates"`
		KeyResponse         KeyPullResponse         `json:"publicKeys"`
	}

	ProfileAssociation struct {
		Type              string              `json:"type"`
		ProfileID         string              `json:"profileID"`
		RootCertificate   *RefinedCertificate `json:"rootCertificate,omitempty"`
		ClientCertificate *RefinedCertificate `json:"clientCertificate,omitempty"`
		Key               *Key                `json:"publicKey,omitempty"`
	}

	CertificatePullResponse struct {
		KeyManagementItems []RefinedKeyManagementResponse `json:"keyManagementItems,omitempty"`
		Certificates       []RefinedCertificate           `json:"publicKeyCertificateItems,omitempty"`
	}

	KeyPullResponse struct {
		Keys []Key `json:"publicPrivateKeyPairItems,omitempty"`
	}

	RefinedKeyManagementResponse struct {
		CreationClassName       string `json:"creationClassName,omitempty"`
		ElementName             string `json:"elementName,omitempty"`
		EnabledDefault          int    `json:"enabledDefault,omitempty"`
		EnabledState            int    `json:"enabledState,omitempty"`
		Name                    string `json:"name,omitempty"`
		RequestedState          int    `json:"requestedState,omitempty"`
		SystemCreationClassName string `json:"systemCreationClassName,omitempty"`
		SystemName              string `json:"systemName,omitempty"`
	}

	RefinedCertificate struct {
		ElementName            string   `json:"elementName,omitempty"`
		InstanceID             string   `json:"instanceID,omitempty"`
		X509Certificate        string   `json:"x509Certificate,omitempty"`
		TrustedRootCertificate bool     `json:"trustedRootCertificate"`
		Issuer                 string   `json:"issuer,omitempty"`
		Subject                string   `json:"subject,omitempty"`
		ReadOnlyCertificate    bool     `json:"readOnlyCertificate"`
		PublicKeyHandle        string   `json:"publicKeyHandle,omitempty"`
		AssociatedProfiles     []string `json:"associatedProfiles,omitempty"`
		DisplayName            string   `json:"displayName,omitempty"`
	}

	Key struct {
		ElementName       string `json:"elementName,omitempty"`
		InstanceID        string `json:"instanceID,omitempty"`
		DERKey            string `json:"derKey,omitempty"`
		CertificateHandle string `json:"certificateHandle,omitempty"`
	}
)
