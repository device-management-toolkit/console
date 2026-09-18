package dto

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"
)

// A package built for "activate" without a profile would point rpc-go at a
// profile-export URL with an empty name, which the server answers with 404.
func TestPackageRequestProfileRequiredForActivate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
		profile string
		wantErr bool
	}{
		{"activate without profile is invalid", "activate", "", true},
		{"activate with profile is valid", "activate", "profile1", false},
		{"deactivate without profile is valid", "deactivate", "", false},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := PackageRequest{
				Command: tt.command,
				Version: "v3.0.1",
				OS:      "linux",
				Arch:    "x86_64",
				Auth:    PackageAuth{Mode: "userpass", Username: "u", Password: "p"},
				Profile: tt.profile,
			}

			// Gin binds with the "binding" tag, not validator's default "validate".
			validate := validator.New()
			validate.SetTagName("binding")

			err := validate.Struct(req)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPackageAuthModes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		mode    string
		wantErr bool
	}{
		{"none", false},
		{"token", false},
		{"userpass", false},
		{"", true},
		{"basic", true},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.mode, func(t *testing.T) {
			t.Parallel()

			validate := validator.New()
			validate.SetTagName("binding")

			err := validate.Struct(PackageAuth{Mode: tt.mode})
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
