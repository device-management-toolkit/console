package packaging

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

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

			goos, arch, ok := parseAsset(tc.filename)
			if ok != tc.wantOK || goos != tc.wantOS || arch != tc.wantArch {
				t.Fatalf("parseAsset(%q) = (%q,%q,%v), want (%q,%q,%v)",
					tc.filename, goos, arch, ok, tc.wantOS, tc.wantArch, tc.wantOK)
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

func TestListReleasesFromOnline(t *testing.T) {
	t.Parallel()

	body := `[
	  {"tag_name":"v3.0.1","prerelease":false,"assets":[
	     {"name":"rpc-go_Linux_x86_64.tar.gz","browser_download_url":"http://x/l"},
	     {"name":"rpc-go_Windows_x86_64.zip","browser_download_url":"http://x/w"}]},
	  {"tag_name":"v2.9.0","prerelease":false,"assets":[
	     {"name":"rpc-go_Linux_x86_64.tar.gz","browser_download_url":"http://x/old"}]}
	]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	rels, err := listReleasesFrom(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	if len(rels) != 1 || rels[0].Version != "v3.0.1" || len(rels[0].Assets) != 2 {
		t.Fatalf("unexpected releases: %+v", rels)
	}
}

func TestListLocalReleases(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	verDir := filepath.Join(dir, "v3.0.1")

	if err := os.MkdirAll(verDir, 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(verDir, "rpc-go_Linux_x86_64.tar.gz"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	rels, err := listLocalReleases(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(rels) != 1 || rels[0].Version != "v3.0.1" || len(rels[0].Assets) != 1 ||
		rels[0].Assets[0].OS != "linux" || rels[0].Assets[0].Arch != "x86_64" {
		t.Fatalf("unexpected local releases: %+v", rels)
	}
}

func TestListReleasesFromHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := listReleasesFrom(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error on non-200 response, got nil")
	}

	if !errors.Is(err, ErrFetchReleases) {
		t.Fatalf("expected error to wrap ErrFetchReleases, got %v", err)
	}
}
