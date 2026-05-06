package mongo

import "testing"

func TestIdentifierRegex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"alphanumeric", "cira1", true},
		{"underscore", "my_config", true},
		{"hyphen", "my-config", true},
		{"uuid-shape", "550e8400-e29b-41d4-a716-446655440000", true},
		{"empty", "", false},
		{"dot rejected", "acme.example.com", false},
		{"slash rejected", "foo/bar", false},
		{"space rejected", "foo bar", false},
		{"mongo operator rejected", "$ne", false},
		{"newline rejected", "foo\nbar", false},
		{"null byte rejected", "foo\x00bar", false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := identifierRegex.MatchString(tc.in); got != tc.want {
				t.Fatalf("identifierRegex.MatchString(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestDomainSuffixRegex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"fqdn", "acme.example.com", true},
		{"hyphenated", "my-domain.example.com", true},
		{"empty", "", false},
		{"slash rejected", "example.com/path", false},
		{"space rejected", "example .com", false},
		{"mongo operator rejected", "$where", false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := domainSuffixRegex.MatchString(tc.in); got != tc.want {
				t.Fatalf("domainSuffixRegex.MatchString(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
