package dto

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestCIRAConfig_ConfigName_AlphaNumHyphenUnderscore(t *testing.T) {
	t.Parallel()

	validate := validator.New()
	err := validate.RegisterValidation("alphanumhyphenunderscore", ValidateAlphaNumHyphenUnderscore)
	assert.NoError(t, err)

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid alphanumeric",
			input:   "ciraconfig1",
			wantErr: false,
		},
		{
			name:    "valid with hyphen",
			input:   "cira-config",
			wantErr: false,
		},
		{
			name:    "valid with underscore",
			input:   "cira_config",
			wantErr: false,
		},
		{
			name:    "valid mixed",
			input:   "my-cira-config_1",
			wantErr: false,
		},
		{
			name:    "invalid with spaces",
			input:   "cira config",
			wantErr: true,
		},
		{
			name:    "invalid with special chars",
			input:   "cira@config!",
			wantErr: true,
		},
		{
			name:    "invalid with dots",
			input:   "cira.config",
			wantErr: true,
		},
		{
			name:    "invalid empty",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			type testStruct struct {
				ConfigName string `validate:"alphanumhyphenunderscore"`
			}

			s := testStruct{ConfigName: tt.input}
			err := validate.Struct(s)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
