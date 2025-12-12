/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

package v2

type (
	Version struct {
		Flash               string `json:"flash" example:"<major>.<minor>.<revision>.<build>"`
		Netstack            string `json:"netstack" example:"<major>.<minor>.<revision>.<build>"`
		AMTApps             string `json:"amtApps" example:"<major>.<minor>.<revision>.<build>"`
		AMT                 string `json:"amt" example:"<major>.<minor>.<revision>.<build>"`
		SKU                 string `json:"sku" example:"<major>.<minor>.<revision>.<build>"`
		VendorID            string `json:"vendorID" example:"<major>.<minor>.<revision>.<build>"`
		BuildNumber         string `json:"buildNumber" example:"<major>.<minor>.<revision>.<build>"`
		RecoveryVersion     string `json:"recovery" example:"<major>.<minor>.<revision>.<build>"`
		RecoveryBuildNumber string `json:"recoveryBuildNumber" example:"<major>.<minor>.<revision>.<build>"`
		LegacyMode          *bool  `json:"legacyMode" example:"false"`
		AMTFWCoreVersion    string `json:"amtFWCore" example:"<major>.<minor>.<revision>.<build>"`
	}
)
