package packaging

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/device-management-toolkit/console/internal/entity/github"
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
		{"linux", "rpc_linux_x64.tar.gz", true, "linux", "x64"},
		{"windows", "rpc_windows_x86.exe", true, "windows", "x86"},
		{"shared library skipped", "rpc_so_x64.tar.gz", false, "", ""},
		{"signature skipped", "rpc_windows_x64.exe.sigstore.json", false, "", ""},
		{"licenses skipped", "licenses.zip", false, "", ""},
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

func TestGetReleasesOnline(t *testing.T) {
	t.Parallel()

	body := `[
	  {"tag_name":"v3.0.1","prerelease":false,"assets":[
	     {"name":"rpc_linux_x64.tar.gz","browser_download_url":"http://x/l"},
	     {"name":"rpc_windows_x64.exe","browser_download_url":"http://x/w"}]},
	  {"tag_name":"v2.9.0","prerelease":false,"assets":[
	     {"name":"rpc_linux_x64.tar.gz","browser_download_url":"http://x/old"}]}
	]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	fetched, err := getReleases(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	rels := filterReleases(fetched)

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

	if err := os.WriteFile(filepath.Join(verDir, "rpc_linux_x64.tar.gz"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	rels, err := listLocalReleases(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(rels) != 1 || rels[0].Version != "v3.0.1" || len(rels[0].Assets) != 1 ||
		rels[0].Assets[0].OS != "linux" || rels[0].Assets[0].Arch != "x64" {
		t.Fatalf("unexpected local releases: %+v", rels)
	}
}

func TestGetReleasesHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := getReleases(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error on non-200 response, got nil")
	}

	if !errors.Is(err, ErrFetchReleases) {
		t.Fatalf("expected error to wrap ErrFetchReleases, got %v", err)
	}
}

// Only the newest few releases are offered, so the releases call stays a single
// unpaginated request.
func TestFilterReleasesCapsAtMaxReleases(t *testing.T) {
	t.Parallel()

	releases := make([]github.Release, 0, 12)
	for i := range 12 {
		releases = append(releases, github.Release{
			TagName: fmt.Sprintf("v3.0.%d", i),
			Assets:  []github.Asset{{Name: "rpc_linux_x64.tar.gz"}},
		})
	}

	got := filterReleases(releases)
	if len(got) != maxReleases {
		t.Fatalf("filterReleases() returned %d releases, want %d", len(got), maxReleases)
	}

	// The cap must keep the newest, which GitHub returns first.
	if got[0].Version != "v3.0.0" {
		t.Errorf("first release = %q, want the first returned by the API", got[0].Version)
	}
}

// Pre-v3 tags must not consume cap slots that a supported release could fill.
func TestFilterReleasesSkipsPreV3WithinCap(t *testing.T) {
	t.Parallel()

	releases := []github.Release{
		{TagName: "v2.9.0", Assets: []github.Asset{{Name: "rpc_linux_x64.tar.gz"}}},
		{TagName: "v3.1.0", Assets: []github.Asset{{Name: "rpc_linux_x64.tar.gz"}}},
		{TagName: "v3.0.0", Assets: []github.Asset{{Name: "rpc_linux_x64.tar.gz"}}},
	}

	got := filterReleases(releases)
	if len(got) != 2 {
		t.Fatalf("filterReleases() returned %d releases, want 2", len(got))
	}

	if got[0].Version != "v3.1.0" || got[1].Version != "v3.0.0" {
		t.Errorf("unexpected releases: %+v", got)
	}
}

func TestReleasesURLRequestsOnePage(t *testing.T) {
	t.Parallel()

	got := releasesURL("https://api.github.com", "owner/repo")
	want := "https://api.github.com/repos/owner/repo/releases?per_page=5"

	if got != want {
		t.Errorf("releasesURL() = %q, want %q", got, want)
	}
}

func TestListLocalReleasesNewestFirstV3Only(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	for _, v := range []string{"v2.9.0", "v3.0.0", "v3.10.0", "v3.2.0", "v3.1.0", "v3.3.0-beta.1", "v3.3.0", "v3.9.0"} {
		verDir := filepath.Join(dir, v)

		if err := os.MkdirAll(verDir, 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(filepath.Join(verDir, "rpc_linux_x64.tar.gz"), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	rels, err := listLocalReleases(dir)
	if err != nil {
		t.Fatal(err)
	}

	got := make([]string, 0, len(rels))
	for _, r := range rels {
		got = append(got, r.Version)
	}

	if want := []string{"v3.10.0", "v3.9.0", "v3.3.0", "v3.3.0-beta.1", "v3.2.0"}; !slices.Equal(got, want) {
		t.Fatalf("versions = %v, want %v", got, want)
	}
}
