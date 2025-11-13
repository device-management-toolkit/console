package dto

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

type Profile struct {
	ProfileName                string               `json:"profileName,omitempty" binding:"required" example:"My Profile"`
	AMTPassword                string               `json:"amtPassword,omitempty" binding:"required_if=GenerateRandomPassword false,omitempty,len=0|min=8,max=32,containsany=$@$!%*#?&-_~^" example:"my_password"`
	CreationDate               string               `json:"creationDate,omitempty" example:"2021-07-01T00:00:00Z"`
	CreatedBy                  string               `json:"created_by,omitempty" example:"admin"`
	GenerateRandomPassword     bool                 `json:"generateRandomPassword" binding:"omitempty,genpasswordwone" example:"true"`
	CIRAConfigName             *string              `json:"ciraConfigName,omitempty" example:"My CIRA Config"`
	Activation                 string               `json:"activation" binding:"required,oneof=ccmactivate acmactivate" example:"activate"`
	MEBXPassword               string               `json:"mebxPassword,omitempty" binding:"required_if=Activation acmactivate|required_if=GenerateRandomMEBxPassword false,omitempty,len=0|min=8,max=32,containsany=$@$!%*#?&-_~^" example:"my_password"`
	GenerateRandomMEBxPassword bool                 `json:"generateRandomMEBxPassword" example:"true"`
	CIRAConfigObject           *CIRAConfig          `json:"ciraConfigObject,omitempty"`
	Tags                       []string             `json:"tags,omitempty"`
	DHCPEnabled                bool                 `json:"dhcpEnabled" example:"true"`
	IPSyncEnabled              bool                 `json:"ipSyncEnabled" example:"true"`
	LocalWiFiSyncEnabled       bool                 `json:"localWifiSyncEnabled" example:"true"`
	WiFiConfigs                []ProfileWiFiConfigs `json:"wifiConfigs,omitempty" binding:"excluded_if=DHCPEnabled false,dive"`
	TenantID                   string               `json:"tenantId" example:"abc123"`
	TLSMode                    int                  `json:"tlsMode,omitempty" binding:"omitempty,min=1,max=4,ciraortls" example:"1"`
	TLSCerts                   *TLSCerts            `json:"tlsCerts,omitempty"`
	TLSSigningAuthority        string               `json:"tlsSigningAuthority,omitempty" binding:"omitempty,oneof=SelfSigned MicrosoftCA" example:"SelfSigned"`
	UserConsent                string               `json:"userConsent,omitempty" binding:"omitempty" default:"All" example:"All"`
	IDEREnabled                bool                 `json:"iderEnabled" example:"true"`
	KVMEnabled                 bool                 `json:"kvmEnabled" example:"true"`
	SOLEnabled                 bool                 `json:"solEnabled" example:"true"`
	IEEE8021xProfileName       *string              `json:"ieee8021xProfileName,omitempty" example:"My Profile"`
	IEEE8021xProfile           *IEEE8021xConfig     `json:"ieee8021xProfile,omitempty"`
	Version                    string               `json:"version,omitempty" example:"1.0.0"`
	UEFIWiFiSyncEnabled        bool                 `json:"uefiWifiSyncEnabled" example:"true"`
}

var ValidateCIRAOrTLS validator.Func = func(fl validator.FieldLevel) bool {
	ciraConfigField := fl.Parent().FieldByName("CIRAConfigName")
	tlsModeField := fl.Parent().FieldByName("TLSMode")

	ciraConfigName, _ := ciraConfigField.Interface().(*string)
	tlsMode, _ := tlsModeField.Interface().(int)

	if ciraConfigName != nil && *ciraConfigName != "" && tlsMode != 0 {
		return false
	}

	return true
}

var ValidateAMTPassOrGenRan validator.Func = func(fl validator.FieldLevel) bool {
	amtPass := fl.Parent().FieldByName("AMTPassword").String()

	return amtPass == ""
}

var ValidateUserConsent validator.Func = func(fl validator.FieldLevel) bool {
	userConsent := strings.ToLower(fl.Field().String())
	activation := fl.Parent().FieldByName("Activation").String()
	if activation == "ccmactivate" && userConsent != "All" {
		return false
	}

	return userConsent == "none" || userConsent == "kvm" || userConsent == "all"
}

type ProfileCountResponse struct {
	Count int       `json:"totalCount"`
	Data  []Profile `json:"data"`
}

type ProfileExportResponse struct {
	Content  string `json:"content"`
	Filename string `json:"filename"`
	Key      string `json:"key"`
}
