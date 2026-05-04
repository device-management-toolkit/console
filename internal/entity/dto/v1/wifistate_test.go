package dto

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"
)

func TestValidateWirelessState(t *testing.T) {
	t.Parallel()

	validate := validator.New()
	require.NoError(t, validate.RegisterValidation("wifistate", ValidateWirelessState))

	tests := []struct {
		name    string
		state   string
		wantErr bool
	}{
		{name: "valid - wifi disabled", state: "WifiDisabled", wantErr: false},
		{name: "valid - wifi enabled s0", state: "WifiEnabledS0", wantErr: false},
		{name: "valid - wifi enabled s0sxac", state: "WifiEnabledS0SxAC", wantErr: false},
		{name: "invalid state", state: "InvalidState", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validate.Var(tc.state, "wifistate")
			if tc.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestWirelessStateChangeRequestValidation(t *testing.T) {
	t.Parallel()

	validate := validator.New()
	validate.SetTagName("binding")
	require.NoError(t, validate.RegisterValidation("wifistate", ValidateWirelessState))

	t.Run("valid request", func(t *testing.T) {
		t.Parallel()

		req := WirelessStateChangeRequest{State: WirelessState("WifiEnabledS0SxAC")}
		require.NoError(t, validate.Struct(req))
	})

	t.Run("invalid request", func(t *testing.T) {
		t.Parallel()

		req := WirelessStateChangeRequest{State: WirelessState("not-valid")}
		require.Error(t, validate.Struct(req))
	})
}
