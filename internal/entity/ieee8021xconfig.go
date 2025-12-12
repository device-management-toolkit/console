/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

package entity

type IEEE8021xConfig struct {
	ProfileName            string
	AuthenticationProtocol int
	PXETimeout             *int
	WiredInterface         bool
	TenantID               string
	Version                string
}
