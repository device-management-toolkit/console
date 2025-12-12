/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

package dto

type PowerState struct {
	PowerState         int `json:"powerstate" example:"0"`
	OSPowerSavingState int `json:"osPowerSavingState" example:"0"`
}
