package dto

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestValidateAlphaNumHyphenUnderscore(t *testing.T) {
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
			name:    "valid with letters and numbers",
			input:   "test123",
			wantErr: false,
		},
		{
			name:    "valid with hyphen",
			input:   "test-domain",
			wantErr: false,
		},
		{
			name:    "valid with underscore",
			input:   "test_domain",
			wantErr: false,
		},
		{
			name:    "valid with mixed",
			input:   "test-123_domain",
			wantErr: false,
		},
		{
			name:    "invalid with special chars",
			input:   "test!@#",
			wantErr: true,
		},
		{
			name:    "invalid with spaces",
			input:   "test domain",
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
				Field string `validate:"alphanumhyphenunderscore"`
			}

			s := testStruct{Field: tt.input}
			err := validate.Struct(s)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
