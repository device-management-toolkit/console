package dto

import (
	"encoding/json"
	"regexp"

	"github.com/go-playground/validator/v10"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/config"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/wifi"
)

const (
	maxWirelessProfilePriority = 255
)

// WirelessProfileRequest carries one wireless profile payload for create/update APIs.
type WirelessProfileRequest struct {
	Profile config.WirelessProfile `json:"-" binding:"wirelessprofile"`
}

// UnmarshalJSON maps a flat profile payload into the dto request wrapper.
func (r *WirelessProfileRequest) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &r.Profile)
}

// ToWirelessProfile converts request payload into WSMan wireless profile config.
func (r WirelessProfileRequest) ToWirelessProfile() config.WirelessProfile {
	return r.Profile
}

// WirelessProfileResponse is the sanitized wireless profile returned by read APIs.
// Sensitive fields (passwords, certificates, and private keys) are intentionally omitted.
type WirelessProfileResponse struct {
	ProfileName          string                     `json:"profileName"`
	SSID                 string                     `json:"ssid"`
	AuthenticationMethod string                     `json:"authenticationMethod"`
	EncryptionMethod     string                     `json:"encryptionMethod"`
	Priority             int                        `json:"priority"`
	IEEE8021x            *WirelessIEEE8021xResponse `json:"ieee8021x,omitempty"`
}

// WirelessIEEE8021xResponse is the sanitized IEEE 802.1x section of a wireless profile.
type WirelessIEEE8021xResponse struct {
	Username               string `json:"username"`
	AuthenticationProtocol int    `json:"authenticationProtocol"`
	PXETimeout             int    `json:"pxeTimeout,omitempty"`
}

// NewWirelessProfileResponse maps a config.WirelessProfile to its sanitized response form.
func NewWirelessProfileResponse(profile config.WirelessProfile) WirelessProfileResponse {
	resp := WirelessProfileResponse{
		ProfileName:          profile.ProfileName,
		SSID:                 profile.SSID,
		AuthenticationMethod: profile.AuthenticationMethod,
		EncryptionMethod:     profile.EncryptionMethod,
		Priority:             profile.Priority,
	}

	if profile.IEEE8021x != nil {
		resp.IEEE8021x = &WirelessIEEE8021xResponse{
			Username:               profile.IEEE8021x.Username,
			AuthenticationProtocol: profile.IEEE8021x.AuthenticationProtocol,
			PXETimeout:             profile.IEEE8021x.PXETimeout,
		}
	}

	return resp
}

// NewWirelessProfileResponses maps a slice of config.WirelessProfile to sanitized responses.
func NewWirelessProfileResponses(profiles []config.WirelessProfile) []WirelessProfileResponse {
	responses := make([]WirelessProfileResponse, 0, len(profiles))
	for i := range profiles {
		responses = append(responses, NewWirelessProfileResponse(profiles[i]))
	}

	return responses
}

var reAlphaNumWirelessProfileName = regexp.MustCompile("^[a-zA-Z0-9]+$")

// ValidateWirelessProfile validates one shared config.WirelessProfile payload.
var ValidateWirelessProfile validator.Func = func(fl validator.FieldLevel) bool {
	profile, ok := fl.Field().Interface().(config.WirelessProfile)
	if !ok {
		return false
	}

	if !isValidWirelessProfileBase(profile) {
		return false
	}

	authMethod, ok := wifi.ParseAuthenticationMethod(profile.AuthenticationMethod)
	if !ok {
		return false
	}

	if _, ok = wifi.ParseEncryptionMethod(profile.EncryptionMethod); !ok {
		return false
	}

	return hasValidWirelessProfileCredentials(profile, authMethod)
}

func isValidWirelessProfileBase(profile config.WirelessProfile) bool {
	if profile.ProfileName == "" || !reAlphaNumWirelessProfileName.MatchString(profile.ProfileName) {
		return false
	}

	if profile.SSID == "" || profile.Priority <= 0 || profile.Priority > maxWirelessProfilePriority {
		return false
	}

	return true
}

func hasValidWirelessProfileCredentials(profile config.WirelessProfile, authMethod wifi.AuthenticationMethod) bool {
	if isPSKAuthenticationMethod(authMethod) {
		return profile.Password != "" && profile.IEEE8021x == nil
	}

	if isIEEE8021xAuthenticationMethod(authMethod) {
		return hasValidIEEE8021xCredentials(profile)
	}

	return false
}

func isPSKAuthenticationMethod(authMethod wifi.AuthenticationMethod) bool {
	return authMethod == wifi.AuthenticationMethodWPAPSK || authMethod == wifi.AuthenticationMethodWPA2PSK
}

func isIEEE8021xAuthenticationMethod(authMethod wifi.AuthenticationMethod) bool {
	return authMethod == wifi.AuthenticationMethodWPAIEEE8021x || authMethod == wifi.AuthenticationMethodWPA2IEEE8021x
}

func hasValidIEEE8021xCredentials(profile config.WirelessProfile) bool {
	if profile.IEEE8021x == nil || profile.Password != "" {
		return false
	}

	return profile.IEEE8021x.AuthenticationProtocol == 0 || profile.IEEE8021x.AuthenticationProtocol == 2
}
