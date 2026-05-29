package packaging

import "testing"

func TestParseAsset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
		wantOK   bool
		wantOS   string
		wantArch string
	}{
		{"linux amd64", "rpc-go_Linux_x86_64.tar.gz", true, "linux", "x86_64"},
		{"linux arm64", "rpc-go_Linux_arm64.tar.gz", true, "linux", "arm64"},
		{"windows", "rpc-go_Windows_x86_64.zip", true, "windows", "x86_64"},
		{"darwin", "rpc-go_Darwin_arm64.tar.gz", true, "darwin", "arm64"},
		{"checksums skipped", "rpc-go_checksums.txt", false, "", ""},
		{"source skipped", "Source code (zip)", false, "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			os, arch, ok := parseAsset(tc.filename)
			if ok != tc.wantOK || os != tc.wantOS || arch != tc.wantArch {
				t.Fatalf("parseAsset(%q) = (%q,%q,%v), want (%q,%q,%v)",
					tc.filename, os, arch, ok, tc.wantOS, tc.wantArch, tc.wantOK)
			}
		})
	}
}

func TestIsV3OrAbove(t *testing.T) {
	t.Parallel()

	cases := map[string]bool{
		"v3.0.1": true, "v3.1.0-beta": true, "v4.0.0": true,
		"v2.9.9": false, "v1.0.0": false, "not-a-tag": false,
	}
	for tag, want := range cases {
		if got := isV3OrAbove(tag); got != want {
			t.Fatalf("isV3OrAbove(%q) = %v, want %v", tag, got, want)
		}
	}
}
