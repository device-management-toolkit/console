/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

package entity

type Domain struct {
	ProfileName                   string
	DomainSuffix                  string
	ProvisioningCert              string
	ProvisioningCertStorageFormat string
	ProvisioningCertPassword      string
	ExpirationDate                string
	TenantID                      string
	Version                       string
}
