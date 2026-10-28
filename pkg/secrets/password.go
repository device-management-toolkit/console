package secrets

import "unicode/utf8"

const (
	minAdminPasswordLength        = 8
	minAdminPasswordDistinctChars = 4
)

// IsAdminPasswordStrong reports whether a password meets the standalone
// administrator password length and character-diversity requirements.
func IsAdminPasswordStrong(password string) bool {
	if utf8.RuneCountInString(password) < minAdminPasswordLength {
		return false
	}

	seen := make(map[rune]struct{})
	for _, character := range password {
		seen[character] = struct{}{}
	}

	return len(seen) >= minAdminPasswordDistinctChars
}
