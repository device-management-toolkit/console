/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

package dto

type (
	SoftwareIdentityResponses struct {
		Responses []SoftwareIdentity `json:"responses"`
	}
	SetupAndConfigurationServiceResponses struct {
		Response SetupAndConfigurationServiceResponse `json:"response"`
	}
	SoftwareIdentity struct {
		InstanceID    string `json:"InstanceID"`
		VersionString string `json:"VersionString" example:"<major>.<minor>.<revision>.<build>"`
		IsEntity      bool   `json:"IsEntity" example:"true"`
	}

	Version struct {
		CIMSoftwareIdentity             SoftwareIdentityResponses             `json:"CIM_SoftwareIdentity"`
		AMTSetupAndConfigurationService SetupAndConfigurationServiceResponses `json:"AMT_SetupAndConfigurationService"`
	}
)
