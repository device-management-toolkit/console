package dto

import (
	"regexp"
	"time"

	"github.com/go-playground/validator/v10"
)

type Domain struct {
	ProfileName                   string    `json:"profileName" binding:"required,alphanumhyphenunderscore" example:"my-profile_1"`
	DomainSuffix                  string    `json:"domainSuffix" binding:"required" example:"example.com"`
	ProvisioningCert              string    `json:"provisioningCert,omitempty" binding:"required" example:"-----BEGIN CERTIFICATE-----\n..."`
	ProvisioningCertStorageFormat string    `json:"provisioningCertStorageFormat" binding:"required,oneof=raw string" example:"string"`
	ProvisioningCertPassword      string    `json:"provisioningCertPassword,omitempty" binding:"required,lte=64" example:"my_password"`
	ExpirationDate                time.Time `json:"expirationDate,omitempty" example:"2022-01-01T00:00:00Z"`
	TenantID                      string    `json:"tenantId" example:"abc123"`
	Version                       string    `json:"version,omitempty" example:"1.0.0"`
}

// ValidateAlphaNumHyphenUnderscore validates that a field contains only alphanumeric characters, hyphens, and underscores.
func ValidateAlphaNumHyphenUnderscore(fl validator.FieldLevel) bool {
	// Pattern allows letters, numbers, hyphens (-), and underscores (_)
	pattern := `^[a-zA-Z0-9_-]+$`
	matched, _ := regexp.MatchString(pattern, fl.Field().String())

	return matched
}
