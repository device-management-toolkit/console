package secrets

import (
	"regexp"
	"unicode/utf8"
)

// OS keyring entries that hold the standalone administrator credentials.
const (
	KeyringServiceName      = "device-management-toolkit"
	KeyringAdminUsernameKey = "admin-username"
	KeyringAdminPasswordKey = "admin-password-hash"
)

// MinAdminPasswordLength is the floor, in runes, for administrator passwords.
// No ceiling: this password is only compared against the login request in httpapi/v1.
const MinAdminPasswordLength = 8

// RE2 has no lookahead, so complexity is one regex per required class. The last is
// "not a letter or digit", broader than the !@#$%^&* AMT profiles require.
var adminPasswordComplexity = []*regexp.Regexp{
	regexp.MustCompile(`[a-z]`),
	regexp.MustCompile(`[A-Z]`),
	regexp.MustCompile(`\d`),
	regexp.MustCompile(`[^a-zA-Z0-9]`),
}

// IsStrongAdminPassword reports whether a password meets the administrator
// password policy shared by the CLI, first-run setup, and password change.
func IsStrongAdminPassword(password string) bool {
	if utf8.RuneCountInString(password) < MinAdminPasswordLength {
		return false
	}

	for _, rule := range adminPasswordComplexity {
		if !rule.MatchString(password) {
			return false
		}
	}

	return true
}
