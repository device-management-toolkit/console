package packaging

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/device-management-toolkit/console/config"
	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// newFailingServer returns an httptest.Server that always responds with 500 and
// registers t.Cleanup to close it.
func newFailingServer(t *testing.T) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	t.Cleanup(srv.Close)

	return srv
}

// newTestConfig returns a minimal *config.Config suitable for packaging tests.
func newTestConfig(localDir, publicURL, jwtKey string) *config.Config {
	return &config.Config{
		HTTP: config.HTTP{
			Host: "localhost",
			Port: "8181",
		},
		Auth: config.Auth{
			JWTKey: jwtKey,
		},
		Package: config.Package{
			RPCRepo:   "device-management-toolkit/rpc-go",
			LocalDir:  localDir,
			PublicURL: publicURL,
		},
	}
}

// buildOfflineFixture writes a real tar.gz containing an "rpc" binary to
// <tmp>/v3.0.1/rpc-go_Linux_x86_64.tar.gz and returns the tmp dir.
func buildOfflineFixture(t *testing.T) string {
	t.Helper()

	tmp := t.TempDir()
	verDir := filepath.Join(tmp, "v3.0.1")

	if err := os.MkdirAll(verDir, 0o750); err != nil {
		t.Fatal(err)
	}

	tarGzData := makeTarGz(t, "rpc", []byte("ELF-placeholder"))

	assetPath := filepath.Join(verDir, "rpc-go_Linux_x86_64.tar.gz")
	if err := os.WriteFile(assetPath, tarGzData, 0o600); err != nil {
		t.Fatal(err)
	}

	return tmp
}

// newOfflineService constructs a Service backed by an httptest server that always
// returns 500 (forcing the online path to fail) and a local fixture directory.
func newOfflineService(t *testing.T, tmp string) *Service {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	t.Cleanup(srv.Close)

	cfg := newTestConfig(tmp, "http://console.example", "k")
	svc := New(cfg, logger.New("error"))
	svc.githubBase = srv.URL

	return svc
}

func TestListVersionsLocalFallback(t *testing.T) {
	t.Parallel()

	tmp := buildOfflineFixture(t)
	svc := newOfflineService(t, tmp)

	releases, err := svc.ListVersions(context.Background())
	if err != nil {
		t.Fatalf("ListVersions returned error: %v", err)
	}

	if len(releases) != 1 {
		t.Fatalf("expected 1 release, got %d: %+v", len(releases), releases)
	}

	if releases[0].Version != "v3.0.1" {
		t.Errorf("release version = %q, want %q", releases[0].Version, "v3.0.1")
	}

	if len(releases[0].Assets) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(releases[0].Assets))
	}

	if releases[0].Assets[0].OS != "linux" || releases[0].Assets[0].Arch != "x86_64" {
		t.Errorf("asset = {OS:%q, Arch:%q}, want {OS:\"linux\", Arch:\"x86_64\"}",
			releases[0].Assets[0].OS, releases[0].Assets[0].Arch)
	}
}

func TestBuildPackageDeactivateOffline(t *testing.T) {
	t.Parallel()

	tmp := buildOfflineFixture(t)
	svc := newOfflineService(t, tmp)

	req := dto.PackageRequest{
		Command: "deactivate",
		Version: "v3.0.1",
		OS:      "linux",
		Arch:    "x86_64",
		Auth:    dto.PackageAuth{Mode: "token"},
	}

	reader, filename, err := svc.BuildPackage(context.Background(), req)
	if err != nil {
		t.Fatalf("BuildPackage returned error: %v", err)
	}

	const wantFilename = "rpc-deactivate-linux-x86_64.zip"
	if filename != wantFilename {
		t.Errorf("filename = %q, want %q", filename, wantFilename)
	}

	zipBytes, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading zip bytes: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		t.Fatalf("opening zip: %v", err)
	}

	names := make(map[string]bool, len(zr.File))
	for _, f := range zr.File {
		names[f.Name] = true
	}

	if !names["rpc"] {
		t.Errorf("zip does not contain 'rpc'; entries: %v", names)
	}

	if !names["config.yaml"] {
		t.Errorf("zip does not contain 'config.yaml'; entries: %v", names)
	}
}

func TestBuildPackagePathTraversalRejected(t *testing.T) {
	t.Parallel()

	tmp := buildOfflineFixture(t)
	svc := newOfflineService(t, tmp)

	req := dto.PackageRequest{
		Command: "deactivate",
		Version: "../evil",
		OS:      "linux",
		Arch:    "x86_64",
		Auth:    dto.PackageAuth{Mode: "token"},
	}

	_, _, err := svc.BuildPackage(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for path-traversal version, got nil")
	}

	if !errors.Is(err, ErrUnsafeVersion) {
		t.Errorf("expected ErrUnsafeVersion, got: %v", err)
	}
}

func TestValidateVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		version string
		wantErr bool
	}{
		{"v3.0.1", false},
		{".", true},
		{"..", true},
		{"../x", true},
		{"a/b", true},
		{"a\\b", true},
		{"", true},
	}

	for _, tc := range tests {
		t.Run(tc.version, func(t *testing.T) {
			t.Parallel()

			err := validateVersion(tc.version)
			if tc.wantErr && err == nil {
				t.Fatalf("validateVersion(%q) = nil, want non-nil error", tc.version)
			}

			if !tc.wantErr && err != nil {
				t.Fatalf("validateVersion(%q) = %v, want nil", tc.version, err)
			}
		})
	}
}

func TestListVersionsGitHubFailNoLocalDir(t *testing.T) {
	t.Parallel()

	srv := newFailingServer(t)

	cfg := newTestConfig("", "http://console.example", "k")
	svc := New(cfg, logger.New("error"))
	svc.githubBase = srv.URL

	_, err := svc.ListVersions(context.Background())
	if err == nil {
		t.Fatal("expected error when GitHub returns 500 and no LocalDir, got nil")
	}

	if !errors.Is(err, ErrFetchReleases) {
		t.Errorf("expected error to wrap ErrFetchReleases, got: %v", err)
	}
}

func TestSafeFilenamePart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{`linux"evil`, "linux-evil"},
		{"linux/etc/passwd", "linux-etc-passwd"},
		{"win\\path", "win-path"},
		{"v3.0.1", "v3.0.1"},
		{"x86_64", "x86_64"},
		{"activate", "activate"},
		{"hello world", "hello-world"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			got := safeFilenamePart(tc.input)
			if got != tc.want {
				t.Errorf("safeFilenamePart(%q) = %q, want %q", tc.input, got, tc.want)
			}

			for _, ch := range got {
				if ch == '"' {
					t.Errorf("safeFilenamePart(%q) = %q still contains double-quote", tc.input, got)
				}
			}
		})
	}
}

func TestBuildPackageOnline(t *testing.T) {
	t.Parallel()

	tgz := makeTarGz(t, "rpc", []byte("ELF"))

	mux := http.NewServeMux()

	mux.HandleFunc("/dl/rpc.tar.gz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(tgz)
	})

	// srvURL is set after the server is created; the closure captures the pointer.
	var srvURL string

	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		body := `[{"tag_name":"v3.0.1","assets":[{"name":"rpc-go_Linux_x86_64.tar.gz","browser_download_url":"` + srvURL + `/dl/rpc.tar.gz"}]}]`
		_, _ = w.Write([]byte(body))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	srvURL = srv.URL

	cfg := newTestConfig("", "http://console.example", "k")
	cfg.RPCRepo = "owner/repo"
	svc := New(cfg, logger.New("error"))
	svc.githubBase = srv.URL

	reader, filename, err := svc.BuildPackage(context.Background(), dto.PackageRequest{
		Command: "activate",
		Version: "v3.0.1",
		OS:      "linux",
		Arch:    "x86_64",
		Auth:    dto.PackageAuth{Mode: "token"},
		Profile: "p1",
	})
	if err != nil {
		t.Fatal(err)
	}

	const wantFilename = "rpc-activate-linux-x86_64.zip"
	if filename != wantFilename {
		t.Fatalf("filename = %q, want %q", filename, wantFilename)
	}

	zipBytes, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading zip bytes: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		t.Fatalf("opening zip: %v", err)
	}

	names := make(map[string]bool, len(zr.File))
	for _, f := range zr.File {
		names[f.Name] = true
	}

	if !names["rpc"] {
		t.Errorf("zip does not contain 'rpc'; entries: %v", names)
	}

	if !names["config.yaml"] {
		t.Errorf("zip does not contain 'config.yaml'; entries: %v", names)
	}
}
