package dto

import (
	"encoding/json"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/config"
	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/wsman/cim/wifi"
)

func TestValidateWirelessProfile(t *testing.T) {
	t.Parallel()

	validate := validator.New()
	require.NoError(t, validate.RegisterValidation("wirelessprofile", ValidateWirelessProfile))

	type profileWrapper struct {
		Profile config.WirelessProfile `validate:"wirelessprofile"`
	}

	tests := []struct {
		name    string
		profile config.WirelessProfile
		wantErr bool
	}{
		{
			name: "valid psk profile",
			profile: config.WirelessProfile{
				ProfileName:          "Office",
				SSID:                 "CorpNet",
				Priority:             1,
				AuthenticationMethod: "WPA2PSK",
				EncryptionMethod:     "CCMP",
				Password:             "password123",
			},
			wantErr: false,
		},
		{
			name: "valid ieee8021x profile",
			profile: config.WirelessProfile{
				ProfileName:          "OfficeEAP",
				SSID:                 "CorpNet",
				Priority:             1,
				AuthenticationMethod: "WPA2IEEE8021x",
				EncryptionMethod:     "CCMP",
				IEEE8021x: &config.IEEE8021x{
					AuthenticationProtocol: 0,
					Username:               "user",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid profile name",
			profile: config.WirelessProfile{
				ProfileName:          "office-net",
				SSID:                 "CorpNet",
				Priority:             1,
				AuthenticationMethod: "WPA2PSK",
				EncryptionMethod:     "CCMP",
				Password:             "password123",
			},
			wantErr: true,
		},
		{
			name: "invalid auth method",
			profile: config.WirelessProfile{
				ProfileName:          "Office",
				SSID:                 "CorpNet",
				Priority:             1,
				AuthenticationMethod: "OpenSystem",
				EncryptionMethod:     "CCMP",
				Password:             "password123",
			},
			wantErr: true,
		},
		{
			name: "auth parse failure",
			profile: config.WirelessProfile{
				ProfileName:          "Office",
				SSID:                 "CorpNet",
				Priority:             1,
				AuthenticationMethod: "NotRealAuthMethod",
				EncryptionMethod:     "CCMP",
				Password:             "password123",
			},
			wantErr: true,
		},
		{
			name: "encryption method accepted by parser",
			profile: config.WirelessProfile{
				ProfileName:          "Office",
				SSID:                 "CorpNet",
				Priority:             1,
				AuthenticationMethod: "WPA2PSK",
				EncryptionMethod:     "WEP",
				Password:             "password123",
			},
			wantErr: false,
		},
		{
			name: "encryption parse failure",
			profile: config.WirelessProfile{
				ProfileName:          "Office",
				SSID:                 "CorpNet",
				Priority:             1,
				AuthenticationMethod: "WPA2PSK",
				EncryptionMethod:     "NotRealEncryptionMethod",
				Password:             "password123",
			},
			wantErr: true,
		},
		{
			name: "invalid ssid priority guard",
			profile: config.WirelessProfile{
				ProfileName:          "Office",
				SSID:                 "",
				Priority:             1,
				AuthenticationMethod: "WPA2PSK",
				EncryptionMethod:     "CCMP",
				Password:             "password123",
			},
			wantErr: true,
		},
		{
			name: "invalid priority out of range",
			profile: config.WirelessProfile{
				ProfileName:          "Office",
				SSID:                 "CorpNet",
				Priority:             256,
				AuthenticationMethod: "WPA2PSK",
				EncryptionMethod:     "CCMP",
				Password:             "password123",
			},
			wantErr: true,
		},
		{
			name: "psk auth missing password",
			profile: config.WirelessProfile{
				ProfileName:          "Office",
				SSID:                 "CorpNet",
				Priority:             1,
				AuthenticationMethod: "WPA2PSK",
				EncryptionMethod:     "CCMP",
			},
			wantErr: true,
		},
		{
			name: "ieee8021x auth missing settings",
			profile: config.WirelessProfile{
				ProfileName:          "OfficeEAP",
				SSID:                 "CorpNet",
				Priority:             1,
				AuthenticationMethod: "WPA2IEEE8021x",
				EncryptionMethod:     "CCMP",
			},
			wantErr: true,
		},
		{
			name: "ieee8021x invalid authentication protocol",
			profile: config.WirelessProfile{
				ProfileName:          "OfficeEAP",
				SSID:                 "CorpNet",
				Priority:             1,
				AuthenticationMethod: "WPA2IEEE8021x",
				EncryptionMethod:     "CCMP",
				IEEE8021x: &config.IEEE8021x{
					AuthenticationProtocol: 1,
					Username:               "user",
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validate.Struct(profileWrapper{Profile: tc.profile})
			if tc.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestValidateWirelessProfileTypeAssertionFailure(t *testing.T) {
	t.Parallel()

	validate := validator.New()
	require.NoError(t, validate.RegisterValidation("wirelessprofile", ValidateWirelessProfile))

	type wrongProfileWrapper struct {
		Profile interface{} `validate:"wirelessprofile"`
	}

	err := validate.Struct(wrongProfileWrapper{Profile: "not-a-wireless-profile"})
	require.Error(t, err)
}

func TestHasValidWirelessProfileCredentialsUnsupportedAuthMethod(t *testing.T) {
	t.Parallel()

	profile := config.WirelessProfile{
		ProfileName: "Office",
		SSID:        "CorpNet",
		Priority:    1,
	}

	require.False(t, hasValidWirelessProfileCredentials(profile, wifi.AuthenticationMethod(255)))
}

func TestWirelessProfileRequestValidation(t *testing.T) {
	t.Parallel()

	validate := validator.New()
	validate.SetTagName("binding")
	require.NoError(t, validate.RegisterValidation("wirelessprofile", ValidateWirelessProfile))

	t.Run("valid request", func(t *testing.T) {
		t.Parallel()

		var req WirelessProfileRequest

		err := json.Unmarshal([]byte(`{"profileName":"Office","ssid":"CorpNet","priority":1,"authenticationMethod":"WPA2PSK","encryptionMethod":"CCMP","password":"password123"}`), &req)
		require.NoError(t, err)

		require.NoError(t, validate.Struct(req))
	})

	t.Run("invalid request", func(t *testing.T) {
		t.Parallel()

		var req WirelessProfileRequest

		err := json.Unmarshal([]byte(`{"profileName":"office-net","ssid":"","priority":0,"authenticationMethod":"bad-auth","encryptionMethod":"bad-encryption"}`), &req)
		require.NoError(t, err)

		require.Error(t, validate.Struct(req))
	})
}

func TestWirelessProfileRequestToWirelessProfile(t *testing.T) {
	t.Parallel()

	req := WirelessProfileRequest{
		Profile: config.WirelessProfile{
			ProfileName:          "Office",
			SSID:                 "CorpNet",
			Password:             "password123",
			AuthenticationMethod: "WPA2PSK",
			EncryptionMethod:     "CCMP",
			Priority:             1,
			IEEE8021x:            &config.IEEE8021x{AuthenticationProtocol: 0, Username: "user"},
		},
	}

	profile := req.ToWirelessProfile()

	require.Equal(t, req.Profile.ProfileName, profile.ProfileName)
	require.Equal(t, req.Profile.SSID, profile.SSID)
	require.Equal(t, req.Profile.Password, profile.Password)
	require.Equal(t, req.Profile.AuthenticationMethod, profile.AuthenticationMethod)
	require.Equal(t, req.Profile.EncryptionMethod, profile.EncryptionMethod)
	require.Equal(t, req.Profile.Priority, profile.Priority)
	require.Equal(t, req.Profile.IEEE8021x, profile.IEEE8021x)
}
