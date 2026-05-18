//go:build tray

package main

import (
	"testing"

	"github.com/device-management-toolkit/console/config"
)

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

func TestListenURLsBracketedIPv6(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		HTTP: config.HTTP{
			Host: "[::1]",
			Port: "8181",
			TLS:  config.TLS{Enabled: true},
		},
	}

	urls := listenURLs(cfg)
	if len(urls) != 1 {
		t.Fatalf("listenURLs returned %d URLs, want 1: %v", len(urls), urls)
	}

	if want := "https://[::1]:8181"; urls[0] != want {
		t.Errorf("listenURLs[0] = %q, want %q", urls[0], want)
	}
}

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

func TestIsVirtualInterfaceName(t *testing.T) {
	t.Parallel()

	virtual := []string{
		"docker0", "br-1234abcd", "veth0a1b2c",
		"tun0", "tap0", "utun3",
		"virbr0", "vmnet1", "vboxnet0",
		"awdl0", "llw0",
		"zt0", "wg0",
	}
	for _, name := range virtual {
		name := name
		t.Run("virtual/"+name, func(t *testing.T) {
			t.Parallel()

			if !isVirtualInterfaceName(name) {
				t.Errorf("isVirtualInterfaceName(%q) = false, want true", name)
			}
		})
	}

	physical := []string{
		"eth0", "eth1", "en0", "en1", "wlan0", "wlp3s0", "enp0s31f6",
	}
	for _, name := range physical {
		name := name
		t.Run("physical/"+name, func(t *testing.T) {
			t.Parallel()

			if isVirtualInterfaceName(name) {
				t.Errorf("isVirtualInterfaceName(%q) = true, want false", name)
			}
		})
	}
}
