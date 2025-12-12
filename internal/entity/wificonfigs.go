/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

package entity

type WirelessConfig struct {
	ProfileName          string
	AuthenticationMethod int
	EncryptionMethod     int
	SSID                 string
	PSKValue             int
	PSKPassphrase        string
	LinkPolicy           *string
	TenantID             string
	IEEE8021xProfileName *string
	Version              string
	//	columns to populate from join query IEEE8021xProfileName
	AuthenticationProtocol *int
	PXETimeout             *int
	WiredInterface         *bool
}
