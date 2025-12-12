/*********************************************************************
 * Copyright (c) Intel Corporation 2023
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

package dto

import "github.com/go-playground/validator/v10"

type WirelessConfigCountResponse struct {
	Count int              `json:"totalCount"`
	Data  []WirelessConfig `json:"data"`
}

type WirelessConfig struct {
	ProfileName            string           `json:"profileName,omitempty" example:"My Profile"`
	AuthenticationMethod   int              `json:"authenticationMethod" binding:"required,oneof=4 5 6 7,authforieee8021x" example:"1"`
	EncryptionMethod       int              `json:"encryptionMethod" binding:"oneof=3 4" example:"3"`
	SSID                   string           `json:"ssid" binding:"max=32" example:"abc"`
	PSKValue               int              `json:"pskValue" example:"3"`
	PSKPassphrase          string           `json:"pskPassphrase,omitempty" binding:"omitempty,min=8,max=32" example:"abc"`
	LinkPolicy             []int            `json:"linkPolicy"`
	TenantID               string           `json:"tenantId" example:"abc123"`
	IEEE8021xProfileName   *string          `json:"ieee8021xProfileName,omitempty" example:"My Profile"`
	IEEE8021xProfileObject *IEEE8021xConfig `json:"ieee8021xProfileObject,omitempty"`
	Version                string           `json:"version"`
}

var ValidateAuthandIEEE validator.Func = func(fl validator.FieldLevel) bool {
	authMethod, _ := fl.Parent().FieldByName("AuthenticationMethod").Interface().(int)
	profName, _ := fl.Parent().FieldByName("IEEE8021xProfileName").Interface().(*string)
	if authMethod == 5 || authMethod == 7 {
		if profName == nil || *profName == "" {
			return false
		}
	}

	if authMethod == 4 || authMethod == 6 {
		if profName != nil && *profName != "" {
			return false
		}
	}

	return true
}
