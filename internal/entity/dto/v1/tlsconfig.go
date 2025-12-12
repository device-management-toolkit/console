/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

package dto

type TLSCerts struct {
	RootCertificate   CertCreationResult `json:"rootCertificate"`
	IssuedCertificate CertCreationResult `json:"issuedCertificate"`
	Version           string             `json:"version"`
}

type CertCreationResult struct {
	H             string `json:"h:"`
	Cert          string `json:"cert"`
	PEM           string `json:"pem"`
	CertBin       string `json:"certBin"`
	PrivateKey    string `json:"privateKey"`
	PrivateKeyBin string `json:"privateKeyBin"`
	Checked       bool   `json:"checked" example:"true"`
	Key           []byte `json:"key"`
}

type SettingDataResponse struct {
	ElementName                   string   `json:"ElementName,omitempty"`
	InstanceID                    string   `json:"InstanceID,omitempty"`
	MutualAuthentication          bool     `json:"MutualAuthentication"`
	Enabled                       bool     `json:"Enabled"`
	TrustedCN                     []string `json:"TrustedCN,omitempty"`
	AcceptNonSecureConnections    bool     `json:"AcceptNonSecureConnections"`
	NonSecureConnectionsSupported *bool    `json:"NonSecureConnectionsSupported"`
}
