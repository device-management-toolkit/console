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
	"slices"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

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
func newTestConfig(localDir string) *config.Config {
	return &config.Config{
		HTTP: config.HTTP{
			Host: "localhost",
			Port: "8181",
		},
		Auth: config.Auth{
			JWTKey: "test-jwt-key",
		},
		Package: config.Package{
			RPCRepo:  "device-management-toolkit/rpc-go",
			LocalDir: localDir,
		},
	}
}

// buildOfflineFixture writes a real tar.gz containing an "rpc" binary to
// <tmp>/v3.0.1/rpc_linux_x64.tar.gz and returns the tmp dir.
func buildOfflineFixture(t *testing.T) string {
	t.Helper()

	tmp := t.TempDir()
	verDir := filepath.Join(tmp, "v3.0.1")

	if err := os.MkdirAll(verDir, 0o750); err != nil {
		t.Fatal(err)
	}

	tarGzData := makeTarGz(t, "rpc_linux_x64", []byte("ELF-placeholder"))

	assetPath := filepath.Join(verDir, "rpc_linux_x64.tar.gz")
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

	cfg := newTestConfig(tmp)
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

	if releases[0].Assets[0].OS != "linux" || releases[0].Assets[0].Arch != "x64" {
		t.Errorf("asset = {OS:%q, Arch:%q}, want {OS:\"linux\", Arch:\"x64\"}",
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
		Arch:    "x64",
		Auth:    dto.PackageAuth{Mode: "token"},
	}

	reader, filename, err := svc.BuildPackage(context.Background(), req, "")
	if err != nil {
		t.Fatalf("BuildPackage returned error: %v", err)
	}

	const wantFilename = "rpc-deactivate-linux-x64.zip"
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

// "both" packages the Windows and Linux builds beside one shared config.
func TestBuildPackageBothOffline(t *testing.T) {
	t.Parallel()

	tmp := buildOfflineFixture(t)

	winPath := filepath.Join(tmp, "v3.0.1", "rpc_windows_x64.exe")
	if err := os.WriteFile(winPath, []byte("PE-placeholder"), 0o600); err != nil {
		t.Fatal(err)
	}

	svc := newOfflineService(t, tmp)

	reader, _, err := svc.BuildPackage(context.Background(), dto.PackageRequest{
		Command: "deactivate",
		Version: "v3.0.1",
		OS:      "both",
		Arch:    "x64",
		Auth:    dto.PackageAuth{Mode: "none"},
	}, "")
	if err != nil {
		t.Fatal(err)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}

	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}

	for _, want := range []string{"rpc", "rpc.exe", "config.yaml"} {
		if !names[want] {
			t.Errorf("zip missing %s, has %v", want, names)
		}
	}

	if len(zr.File) != 3 {
		t.Errorf("zip has %d entries, want 3", len(zr.File))
	}
}

func TestBuildConfigInputsBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		serverURL string
		wantBase  string
	}{
		{"request server url wins", "https://override.example:8181", "https://override.example:8181"},
		{"trailing slash trimmed", "https://override.example:8181/", "https://override.example:8181"},
		{"falls back to listen address", "", "http://localhost:8181"},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := New(newTestConfig(""), logger.New("error"))

			in, err := svc.buildConfigInputs(dto.PackageRequest{
				Command:   "activate",
				Auth:      dto.PackageAuth{Mode: "userpass"},
				ServerURL: tt.serverURL,
			}, "")
			if err != nil {
				t.Fatal(err)
			}

			if in.ExportBase != tt.wantBase {
				t.Errorf("ExportBase = %q, want %q", in.ExportBase, tt.wantBase)
			}

			if in.AuthEndpoint != tt.wantBase+"/api/v1/authorize" {
				t.Errorf("AuthEndpoint = %q, want base %q", in.AuthEndpoint, tt.wantBase)
			}

			if in.DevicesEndpoint != tt.wantBase+"/api/v1/devices" {
				t.Errorf("DevicesEndpoint = %q, want base %q", in.DevicesEndpoint, tt.wantBase)
			}
		})
	}
}

func TestBuildConfigInputsTokenTTL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		requested string
		maxTTL    time.Duration
		wantTTL   time.Duration
		wantErr   error
	}{
		{"defaults to an hour", "", 0, defaultTokenTTL, nil},
		{"honors the requested lifetime", "15m", 24 * time.Hour, 15 * time.Minute, nil},
		{"rejects a lifetime above the configured cap", "24h", time.Hour, 0, ErrTokenTTLTooLong},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := newTestConfig("")
			cfg.MaxTokenTTL = tt.maxTTL

			svc := New(cfg, logger.New("error"))

			in, err := svc.buildConfigInputs(dto.PackageRequest{
				Command:  "activate",
				Auth:     dto.PackageAuth{Mode: "token"},
				TokenTTL: tt.requested,
			}, "")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				return
			}

			claims := &jwt.RegisteredClaims{}
			if _, perr := jwt.ParseWithClaims(in.AuthToken, claims, func(_ *jwt.Token) (interface{}, error) {
				return []byte(cfg.JWTKey), nil
			}); perr != nil {
				t.Fatal(perr)
			}

			if got := time.Until(claims.ExpiresAt.Time); got < tt.wantTTL-time.Minute || got > tt.wantTTL+time.Minute {
				t.Errorf("token expires in %v, want roughly %v", got, tt.wantTTL)
			}
		})
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
		Arch:    "x64",
		Auth:    dto.PackageAuth{Mode: "token"},
	}

	_, _, err := svc.BuildPackage(context.Background(), req, "")
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

	cfg := newTestConfig("")
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

	tgz := makeTarGz(t, "rpc_linux_x64", []byte("ELF"))

	mux := http.NewServeMux()

	mux.HandleFunc("/dl/rpc.tar.gz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(tgz)
	})

	// srvURL is set after the server is created; the closure captures the pointer.
	var srvURL string

	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		body := `[{"tag_name":"v3.0.1","assets":[{"name":"rpc_linux_x64.tar.gz","browser_download_url":"` + srvURL + `/dl/rpc.tar.gz"}]}]`
		_, _ = w.Write([]byte(body))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	srvURL = srv.URL

	cfg := newTestConfig("")
	cfg.RPCRepo = "owner/repo"
	svc := New(cfg, logger.New("error"))
	svc.githubBase = srv.URL

	reader, filename, err := svc.BuildPackage(context.Background(), dto.PackageRequest{
		Command: "activate",
		Version: "v3.0.1",
		OS:      "linux",
		Arch:    "x64",
		Auth:    dto.PackageAuth{Mode: "token"},
		Profile: "p1",
	}, "")
	if err != nil {
		t.Fatal(err)
	}

	const wantFilename = "rpc-activate-linux-x64.zip"
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

func TestBuildConfigInputsTokenOmittedWhenAuthDisabled(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig("")
	cfg.Disabled = true

	svc := New(cfg, logger.New("error"))

	in, err := svc.buildConfigInputs(dto.PackageRequest{
		Command: "activate",
		Auth:    dto.PackageAuth{Mode: "token"},
	}, "")
	if err != nil {
		t.Fatal(err)
	}

	if in.AuthToken != "" {
		t.Errorf("AuthToken = %q, want empty when auth is disabled", in.AuthToken)
	}
}

func TestBuildConfigInputsTokenRejectedUnderOIDC(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig("")
	cfg.ClientID = "console-client"

	svc := New(cfg, logger.New("error"))

	_, err := svc.buildConfigInputs(dto.PackageRequest{
		Command: "activate",
		Auth:    dto.PackageAuth{Mode: "token"},
	}, "")
	if !errors.Is(err, ErrTokenModeUnsupported) {
		t.Fatalf("err = %v, want ErrTokenModeUnsupported", err)
	}
}

// newReleasesServer serves body as the GitHub releases list and fails the test
// if called when fetching is disabled.
func newReleasesServer(t *testing.T, body string, allowed bool) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if !allowed {
			t.Errorf("github contacted with fetching disabled")
		}

		_, _ = w.Write([]byte(body))
	}))

	t.Cleanup(srv.Close)

	return srv
}

func TestListVersionsFetchedThenLocal(t *testing.T) {
	t.Parallel()

	tmp := buildOfflineFixture(t) // v3.0.1 locally

	body := `[
	  {"tag_name":"v3.1.0","assets":[{"name":"rpc_linux_x64.tar.gz","browser_download_url":"http://x/a"}]},
	  {"tag_name":"v3.0.1","assets":[{"name":"rpc_windows_x64.exe","browser_download_url":"http://x/b"}]}
	]`

	if err := os.MkdirAll(filepath.Join(tmp, "v3.0.0"), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(tmp, "v3.0.0", "rpc_linux_x64.tar.gz"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	svc := New(newTestConfig(tmp), logger.New("error"))
	svc.githubBase = newReleasesServer(t, body, true).URL

	releases, err := svc.ListVersions(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	got := make([]string, 0, len(releases))
	for _, r := range releases {
		got = append(got, r.Version)
	}

	if want := []string{"v3.1.0", "v3.0.1", "v3.0.0"}; !slices.Equal(got, want) {
		t.Fatalf("versions = %v, want %v", got, want)
	}

	// v3.0.1 exists in both; the fetched entry (Windows asset) wins.
	if releases[1].Assets[0].OS != "windows" {
		t.Errorf("v3.0.1 assets = %+v, want the fetched windows asset", releases[1].Assets)
	}
}

func TestListVersionsFetchDisabled(t *testing.T) {
	t.Parallel()

	tmp := buildOfflineFixture(t)

	cfg := newTestConfig(tmp)
	cfg.DisableFetch = true

	svc := New(cfg, logger.New("error"))
	svc.githubBase = newReleasesServer(t, `[]`, false).URL

	releases, err := svc.ListVersions(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(releases) != 1 || releases[0].Version != "v3.0.1" {
		t.Fatalf("releases = %+v, want only local v3.0.1", releases)
	}
}

func TestBuildPackageFetchDisabled(t *testing.T) {
	t.Parallel()

	tmp := buildOfflineFixture(t)

	cfg := newTestConfig(tmp)
	cfg.DisableFetch = true

	svc := New(cfg, logger.New("error"))
	svc.githubBase = newReleasesServer(t, `[]`, false).URL

	_, _, err := svc.BuildPackage(context.Background(), dto.PackageRequest{
		Command: "deactivate",
		Version: "v3.0.1",
		OS:      "linux",
		Arch:    "x64",
		Auth:    dto.PackageAuth{Mode: "none"},
	}, "")
	if err != nil {
		t.Fatal(err)
	}
}
