package main

import "testing"

func TestUnbracketHost(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"[::1]":     "::1",
		"[fe80::1]": "fe80::1",
		"[]":        "",
		"::1":       "::1",
		"localhost": "localhost",
		"":          "",
		"[":         "[",
		"]":         "]",
	}

	for in, want := range tests {
		in, want := in, want
		t.Run(in, func(t *testing.T) {
			t.Parallel()

			if got := unbracketHost(in); got != want {
				t.Errorf("unbracketHost(%q) = %q, want %q", in, got, want)
			}
		})
	}
}

func TestIsWildcardListenHost(t *testing.T) {
	t.Parallel()

	tests := map[string]bool{
		"":          true,
		"0.0.0.0":   true,
		"::":        true,
		"[::]":      true,
		"localhost": false,
		"127.0.0.1": false,
		"10.0.0.1":  false,
	}

	for host, want := range tests {
		host, want := host, want
		t.Run(host, func(t *testing.T) {
			t.Parallel()

			if got := isWildcardListenHost(host); got != want {
				t.Errorf("isWildcardListenHost(%q) = %v, want %v", host, got, want)
			}
		})
	}
}

func TestNavigableHost(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		// Wildcard addresses map to localhost.
		"":        "localhost",
		"0.0.0.0": "localhost",
		"::":      "localhost",
		"[::]":    "localhost",
		// Specific addresses are returned as-is (brackets stripped for IPv6).
		"::1":         "::1",
		"[::1]":       "::1",
		"127.0.0.1":   "127.0.0.1",
		"192.168.1.1": "192.168.1.1",
		"myserver":    "myserver",
	}

	for in, want := range tests {
		in, want := in, want
		t.Run(in, func(t *testing.T) {
			t.Parallel()

			if got := navigableHost(in); got != want {
				t.Errorf("navigableHost(%q) = %q, want %q", in, got, want)
			}
		})
	}
}
