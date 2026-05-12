package dto

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateProfileName(t *testing.T) {
	t.Parallel()

	validate := validator.New()
	require.NoError(t, validate.RegisterValidation("profilename", ValidateProfileName))

	tests := []struct {
		name        string
		profileName string
		wantErr     bool
	}{
		{name: "alphanumeric", profileName: "MyProfile1", wantErr: false},
		{name: "spaces, hyphen, underscore, dots allowed", profileName: "My-cool_profile v2.1", wantErr: false},
		{name: "punctuation, braces, quotes allowed", profileName: `Prof!le {v1} "x" |a| $@*~^`, wantErr: false},
		{name: "unicode allowed", profileName: "プロファイル_日本", wantErr: false},
		// '/', '?', '#', '%' break the ":name" path-segment routes — the only rejected characters.
		{name: "rejects slash", profileName: "my/profile", wantErr: true},
		{name: "rejects question mark", profileName: "my?profile", wantErr: true},
		{name: "rejects hash", profileName: "my#profile", wantErr: true},
		{name: "rejects percent", profileName: "my%profile", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			type testStruct struct {
				ProfileName string `validate:"profilename"`
			}

			err := validate.Struct(testStruct{ProfileName: tt.profileName})

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAMTPasswordComplexity(t *testing.T) {
	t.Parallel()

	validate := validator.New()
	require.NoError(t, validate.RegisterValidation("amtpasswordcomplexity", ValidateAMTPasswordComplexity))

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{name: "lower upper digit special", password: "P@ssw0rd", wantErr: false},
		{name: "all required classes present", password: "Abcdef1!", wantErr: false},
		{name: "underscore allowed as filler when a required special is present", password: "P@ss_w0rd", wantErr: false},
		{name: "underscore alone is not a required special", password: "Passw0rd_", wantErr: true},
		{name: "rejects empty", password: "", wantErr: true},
		{name: "missing uppercase", password: "passw0rd!", wantErr: true},
		{name: "missing lowercase", password: "PASSW0RD!", wantErr: true},
		{name: "missing digit", password: "Password!", wantErr: true},
		{name: "missing special", password: "Passw0rd1", wantErr: true},
		{name: "the gap example", password: "password!", wantErr: true},
		{name: "rejects disallowed char (space)", password: "P@ss w0rd", wantErr: true},
		{name: "rejects disallowed char (pipe)", password: "P@ssw0rd|", wantErr: true},
		{name: "rejects non-ascii", password: "P@ssw0rdé", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			type testStruct struct {
				Password string `validate:"amtpasswordcomplexity"`
			}

			err := validate.Struct(testStruct{Password: tt.password})

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateWiFiDHCP(t *testing.T) {
	t.Parallel()

	validate := validator.New()
	err := validate.RegisterValidation("wifidhcp", ValidateWiFiDHCP)
	assert.NoError(t, err)

	tests := []struct {
		name        string
		dhcpEnabled bool
		wifiConfigs []ProfileWiFiConfigs
		wantErr     bool
	}{
		{
			name:        "valid with WiFi configs and DHCP enabled",
			dhcpEnabled: true,
			wifiConfigs: []ProfileWiFiConfigs{
				{
					Priority:            1,
					WirelessProfileName: "MyWiFiProfile",
					ProfileName:         "MyProfile",
					TenantID:            "tenant1",
				},
			},
			wantErr: false,
		},
		{
			name:        "invalid with WiFi configs and DHCP disabled",
			dhcpEnabled: false,
			wifiConfigs: []ProfileWiFiConfigs{
				{
					Priority:            1,
					WirelessProfileName: "MyWiFiProfile",
					ProfileName:         "MyProfile",
					TenantID:            "tenant1",
				},
			},
			wantErr: true,
		},
		{
			name:        "valid with no WiFi configs and DHCP enabled",
			dhcpEnabled: true,
			wifiConfigs: []ProfileWiFiConfigs{},
			wantErr:     false,
		},
		{
			name:        "valid with no WiFi configs and DHCP disabled",
			dhcpEnabled: false,
			wifiConfigs: []ProfileWiFiConfigs{},
			wantErr:     false,
		},
		{
			name:        "valid with multiple WiFi configs and DHCP enabled",
			dhcpEnabled: true,
			wifiConfigs: []ProfileWiFiConfigs{
				{
					Priority:            1,
					WirelessProfileName: "WiFiProfile1",
					ProfileName:         "Profile1",
					TenantID:            "tenant1",
				},
				{
					Priority:            2,
					WirelessProfileName: "WiFiProfile2",
					ProfileName:         "Profile2",
					TenantID:            "tenant1",
				},
			},
			wantErr: false,
		},
		{
			name:        "invalid with multiple WiFi configs and DHCP disabled",
			dhcpEnabled: false,
			wifiConfigs: []ProfileWiFiConfigs{
				{
					Priority:            1,
					WirelessProfileName: "WiFiProfile1",
					ProfileName:         "Profile1",
					TenantID:            "tenant1",
				},
				{
					Priority:            2,
					WirelessProfileName: "WiFiProfile2",
					ProfileName:         "Profile2",
					TenantID:            "tenant1",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			type testStruct struct {
				DHCPEnabled bool                 `validate:"omitempty"`
				WiFiConfigs []ProfileWiFiConfigs `validate:"wifidhcp"`
			}

			s := testStruct{
				DHCPEnabled: tt.dhcpEnabled,
				WiFiConfigs: tt.wifiConfigs,
			}
			err := validate.Struct(s)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
