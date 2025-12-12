/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

package dto

type PowerCapabilities struct {
	PowerUp    int `json:"Power up,omitempty" example:"0"`
	PowerCycle int `json:"Power cycle,omitempty" example:"0"`
	PowerDown  int `json:"Power down,omitempty" example:"0"`
	Reset      int `json:"Reset,omitempty" example:"0"`

	SoftOff   int `json:"Soft-off,omitempty" example:"0"`
	SoftReset int `json:"Soft-reset,omitempty" example:"0"`
	Sleep     int `json:"Sleep,omitempty" example:"0"`
	Hibernate int `json:"Hibernate,omitempty" example:"0"`

	PowerOnToBIOS int `json:"Power up to BIOS,omitempty" example:"0"`
	ResetToBIOS   int `json:"Reset to BIOS,omitempty" example:"0"`

	ResetToSecureErase int `json:"Reset to Secure Erase,omitempty" example:"0"`

	ResetToIDERFloppy   int `json:"Reset to IDE-R Floppy,omitempty" example:"0"`
	PowerOnToIDERFloppy int `json:"Power on to IDE-R Floppy,omitempty" example:"0"`
	ResetToIDERCDROM    int `json:"Reset to IDE-R CDROM,omitempty" example:"0"`
	PowerOnToIDERCDROM  int `json:"Power on to IDE-R CDROM,omitempty" example:"0"`

	PowerOnToDiagnostic int `json:"Power on to diagnostic,omitempty" example:"0"`
	ResetToDiagnostic   int `json:"Reset to diagnostic,omitempty" example:"0"`

	ResetToPXE   int `json:"Reset to PXE,omitempty" example:"0"`
	PowerOnToPXE int `json:"Power on to PXE,omitempty" example:"0"`
}
