package dto

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

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
