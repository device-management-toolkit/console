/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

package entity

type TLSCerts struct {
	RootCertificate   CertCreationResult
	IssuedCertificate CertCreationResult
	Version           string
}

type CertCreationResult struct {
	H             string
	Cert          string
	PEM           string
	CertBin       string
	PrivateKey    string
	PrivateKeyBin string
	Checked       bool
	Key           []byte
}
