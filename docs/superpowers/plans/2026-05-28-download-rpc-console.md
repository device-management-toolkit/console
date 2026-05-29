# Download RPC (Console Backend) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `GET /api/package/rpc-versions` and `POST /api/package` to Console so the Download RPC UI can list rpc-go releases and download a zip (extracted rpc-go binary + generated `config.yaml`) in one same-origin call.

**Architecture:** New `packaging` usecase (no DB, constructed from config + logger, like `export.FileExporter`) holding the GitHub listing, asset-name parsing, offline directory fallback, config.yaml rendering, JWT minting, and archive/zip handling. A thin `v1` controller exposes it under the protected `/api/package` group. Reuses the existing `github.Release` entity and the `login.go` HS256 JWT pattern.

**Tech Stack:** Go, Gin, `gopkg.in/yaml.v2`, `github.com/golang-jwt/jwt/v5`, stdlib `archive/zip`, `archive/tar`, `compress/gzip`. Table-driven tests with `go test ./...`.

**Conventions:** Go files have **no** license header (just `package <name>`). Wrap errors with `consoleerrors`/`ErrorResponse` as neighboring code does. Match the `dto`/usecase/controller layering. Reference the design at `docs/superpowers/specs/2026-05-28-download-rpc-console-design.md`.

**Companion:** The UI half is complete in `sample-web-ui` (branch `feat/download-rpc`); the request/response JSON shapes here MUST match that contract exactly.

---

## File Structure

- Create: `internal/entity/dto/v1/package.go` — `PackageRequest`, `RpcRelease`, `RpcAsset`.
- Modify: `internal/entity/github/release.go` — add `BrowserDownloadURL` to `Asset`.
- Modify: `config/config.go`, `config/config.yml` — add `Package` config section.
- Create: `internal/usecase/packaging/interface.go`, `packaging.go`, `github.go`, `config.go`, `archive.go`, `token.go` (+ `*_test.go`).
- Modify: `internal/usecase/usecase.go` — add `Packaging` to `Usecases`, construct it.
- Create: `internal/controller/httpapi/v1/package.go` (+ `package_test.go`).
- Modify: `internal/controller/httpapi/router.go` — register the package routes.

---

### Task 1: DTOs and GitHub asset field

**Files:**
- Create: `internal/entity/dto/v1/package.go`
- Modify: `internal/entity/github/release.go`

- [ ] **Step 1: Add the DTOs** in `internal/entity/dto/v1/package.go`:

```go
package dto

// PackageAuth selects how rpc-go authenticates to the server.
type PackageAuth struct {
	Mode     string `json:"mode" binding:"required,oneof=token userpass"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// PackageRequest is the body posted to POST /api/package.
type PackageRequest struct {
	Command string      `json:"command" binding:"required,oneof=activate deactivate"`
	Version string      `json:"version" binding:"required"`
	OS      string      `json:"os" binding:"required"`
	Arch    string      `json:"arch" binding:"required"`
	Auth    PackageAuth `json:"auth" binding:"required"`
	Profile string      `json:"profile"`
	Domain  string      `json:"domain"`
}

// RpcAsset is one downloadable build for a release.
type RpcAsset struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// RpcRelease is a single rpc-go release returned to the UI.
type RpcRelease struct {
	Version string     `json:"version"`
	Assets  []RpcAsset `json:"assets"`
}
```

- [ ] **Step 2: Add the download URL** to `Asset` in `internal/entity/github/release.go` (add the field to the existing struct):

```go
	BrowserDownloadURL string `json:"browser_download_url"`
```

- [ ] **Step 3: Verify it compiles**

Run: `go build ./...`
Expected: success.

- [ ] **Step 4: Commit**

```bash
git add internal/entity/dto/v1/package.go internal/entity/github/release.go
git commit -m "feat: add download-rpc package DTOs and asset download url"
```

---

### Task 2: Package config section

**Files:**
- Modify: `config/config.go`
- Modify: `config/config.yml`

- [ ] **Step 1: Add the `Package` struct** to the `Config` type group in `config/config.go` and a field on `Config`:

Add `Package` to the `Config` struct: `Package \`yaml:"package"\``. Add the struct definition near the others:

```go
	// Package -. Settings for the Download RPC packaging endpoints.
	Package struct {
		RPCRepo   string `yaml:"rpc_repo" env:"RPC_REPO"`
		LocalDir  string `yaml:"local_dir" env:"RPC_LOCAL_DIR"`
		PublicURL string `yaml:"public_url" env:"CONSOLE_PUBLIC_URL"`
	}
```

- [ ] **Step 2: Add defaults** in the place `config.go` sets defaults (the literal near line 150 where `Repo: "device-management-toolkit/console"` is set). Set `RPCRepo` default:

```go
		Package: struct {
			RPCRepo   string `yaml:"rpc_repo" env:"RPC_REPO"`
			LocalDir  string `yaml:"local_dir" env:"RPC_LOCAL_DIR"`
			PublicURL string `yaml:"public_url" env:"CONSOLE_PUBLIC_URL"`
		}{RPCRepo: "device-management-toolkit/rpc-go"},
```

(If `config.go` uses a named type for the section instead of an inline struct, adapt accordingly — define `type PackageConfig struct {...}` and reference it; match whatever pattern the file already uses for sections like `EA`/`UI`.)

- [ ] **Step 3: Add yaml defaults** to `config/config.yml`:

```yaml
package:
  rpc_repo: device-management-toolkit/rpc-go
  local_dir: ""
  public_url: ""
```

- [ ] **Step 4: Verify**

Run: `go build ./... && go test ./config/...`
Expected: success.

- [ ] **Step 5: Commit**

```bash
git add config/config.go config/config.yml
git commit -m "feat: add package config section for download-rpc"
```

---

### Task 3: Asset-name parsing and version filtering

**Files:**
- Create: `internal/usecase/packaging/github.go`
- Test: `internal/usecase/packaging/github_test.go`

- [ ] **Step 1: Write failing tests** in `github_test.go`:

```go
package packaging

import "testing"

func TestParseAsset(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		wantOK     bool
		wantOS     string
		wantArch   string
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
			os, arch, ok := parseAsset(tc.filename)
			if ok != tc.wantOK || os != tc.wantOS || arch != tc.wantArch {
				t.Fatalf("parseAsset(%q) = (%q,%q,%v), want (%q,%q,%v)",
					tc.filename, os, arch, ok, tc.wantOS, tc.wantArch, tc.wantOK)
			}
		})
	}
}

func TestIsV3OrAbove(t *testing.T) {
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
```

- [ ] **Step 2: Run, confirm fail**

Run: `go test ./internal/usecase/packaging/...`
Expected: FAIL (undefined `parseAsset`, `isV3OrAbove`).

- [ ] **Step 3: Implement** in `github.go`:

```go
package packaging

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/device-management-toolkit/console/internal/entity/github"
)

var assetRe = regexp.MustCompile(`(?i)_(Linux|Windows|Darwin)_([A-Za-z0-9_]+?)\.(?:tar\.gz|zip)$`)

// parseAsset extracts a normalized os ("linux"/"windows"/"darwin") and arch
// token from an rpc-go release asset filename. ok is false for non-build assets.
func parseAsset(filename string) (os, arch string, ok bool) {
	m := assetRe.FindStringSubmatch(filename)
	if m == nil {
		return "", "", false
	}
	return strings.ToLower(m[1]), m[2], true
}

// isV3OrAbove reports whether a release tag is semver major >= 3 (betas count).
func isV3OrAbove(tag string) bool {
	t := strings.TrimPrefix(strings.TrimSpace(tag), "v")
	dot := strings.IndexByte(t, '.')
	if dot < 0 {
		return false
	}
	major, err := strconv.Atoi(t[:dot])
	if err != nil {
		return false
	}
	return major >= 3
}

// toRelease maps a github.Release to the UI DTO shape, keeping only parseable assets.
func toReleaseAssets(assets []github.Asset) []dtoAsset {
	out := make([]dtoAsset, 0, len(assets))
	for _, a := range assets {
		if os, arch, ok := parseAsset(a.Name); ok {
			out = append(out, dtoAsset{os: os, arch: arch, name: a.Name, url: a.BrowserDownloadURL})
		}
	}
	return out
}

// dtoAsset is the internal asset record (carries the download url/name for BuildPackage).
type dtoAsset struct {
	os, arch, name, url string
}
```

- [ ] **Step 4: Run, confirm pass**

Run: `go test ./internal/usecase/packaging/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/packaging/github.go internal/usecase/packaging/github_test.go
git commit -m "feat: add rpc-go asset parsing and version filtering"
```

---

### Task 4: Release listing with offline fallback

**Files:**
- Modify: `internal/usecase/packaging/github.go`
- Test: `internal/usecase/packaging/github_test.go`

- [ ] **Step 1: Write failing tests** — add to `github_test.go`: spin up an `httptest.Server` returning a fixed `/releases` JSON (one v3 release with linux/windows assets, one v2 release), point the lister at it, assert only the v3 release is returned with the parsed assets. Add a second test where the lister URL is unreachable AND a temp `LocalDir` contains `rpc-go_Linux_x86_64.tar.gz` — assert the fallback returns `[{version from dir, [{linux,x86_64}]}]`. (Use `t.TempDir()`; for the offline dir, derive the version from a `version.txt` file or the directory name — define the convention in the implementation and assert it.)

```go
func TestListReleasesOnline(t *testing.T) {
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
```

- [ ] **Step 2: Run, confirm fail.** `go test ./internal/usecase/packaging/...` → FAIL (undefined `listReleasesFrom`).

- [ ] **Step 3: Implement** in `github.go`: a `listReleasesFrom(ctx, baseURL string) ([]dto.RpcRelease, error)` that GETs `baseURL` (the releases endpoint), decodes `[]github.Release`, filters with `isV3OrAbove`, maps assets via `toReleaseAssets`, and returns the DTOs. Add `listLocalReleases(dir string) ([]dto.RpcRelease, error)` that walks `dir`, parses filenames via `parseAsset`, groups by a version read from an adjacent `version.txt` (or the parent dir name — pick one and document it), and returns the DTOs. Add the public `releasesURL(repo string)` helper = `https://api.github.com/repos/<repo>/releases`. Import the dto package as `dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"`.

- [ ] **Step 4: Run, confirm pass.** `go test ./internal/usecase/packaging/...` → PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/packaging/github.go internal/usecase/packaging/github_test.go
git commit -m "feat: list rpc-go releases with offline directory fallback"
```

---

### Task 5: config.yaml rendering

**Files:**
- Create: `internal/usecase/packaging/config.go`
- Test: `internal/usecase/packaging/config_test.go`

- [ ] **Step 1: Write failing tests** in `config_test.go`: call `renderConfig(req, endpoints)` for (a) token+activate+domain, (b) userpass+activate no-domain, (c) token+deactivate. Unmarshal the produced bytes back into a `map[string]interface{}` (assert valid YAML) and assert: token case sets `auth-token` non-empty and `activate.url` contains the profile + `domainName`; userpass case sets `auth-username`/`auth-password` and an `activate.url` without `domainName`; deactivate case leaves `activate.url` empty and sets the deactivate/auth fields. `endpoints` is a small struct `{AuthEndpoint, DevicesEndpoint, ExportBase, AuthToken string}` passed in (token already minted by the caller — keeps this function pure/testable).

- [ ] **Step 2: Run, confirm fail.**

- [ ] **Step 3: Implement** `renderConfig` in `config.go`: define a Go struct mirroring `docs/rpc-config.sample.yml` (in the UI repo) with `yaml` tags, populate the shared auth block + the command-specific section per the design's field-mapping table, and `yaml.Marshal` it. Build `activate.url` as `<ExportBase>/api/v1/admin/profiles/export/<profile>` adding `?domainName=<domain>` only when `req.Domain != ""`. Use `gopkg.in/yaml.v2`.

- [ ] **Step 4: Run, confirm pass.**

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/packaging/config.go internal/usecase/packaging/config_test.go
git commit -m "feat: render rpc-go config.yaml from package request"
```

---

### Task 6: JWT minting helper

**Files:**
- Create: `internal/usecase/packaging/token.go`
- Test: `internal/usecase/packaging/token_test.go`

- [ ] **Step 1: Write failing test** in `token_test.go`: `mintToken("test-key")` returns a string; parse it with `jwt.Parse` using the same key + HS256 and assert it's valid and has an `exp` in the future.

- [ ] **Step 2: Run, confirm fail.**

- [ ] **Step 3: Implement** `mintToken(jwtKey string) (string, error)` in `token.go` mirroring `LoginRoute.handleBasicAuth`: `claims := jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}`, `jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(jwtKey))`. (If practical, refactor `login.go` to call this shared helper to avoid duplication.)

- [ ] **Step 4: Run, confirm pass.**

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/packaging/token.go internal/usecase/packaging/token_test.go
git commit -m "feat: add jwt minting helper for download-rpc token auth"
```

---

### Task 7: Archive download, extraction, and output zip

**Files:**
- Create: `internal/usecase/packaging/archive.go`
- Test: `internal/usecase/packaging/archive_test.go`

- [ ] **Step 1: Write failing tests** in `archive_test.go`: build an in-memory `.tar.gz` containing a file named `rpc` with known bytes, call `extractBinary(data, "rpc-go_Linux_x86_64.tar.gz")`, assert the returned name/bytes. Do the same for a `.zip` containing `rpc.exe`. Then call `buildZip(binaryName, binaryBytes, []byte("config-yaml"))` and read it back with `archive/zip`, asserting both `rpc`/`rpc.exe` and `config.yaml` entries exist with the right contents.

- [ ] **Step 2: Run, confirm fail.**

- [ ] **Step 3: Implement** in `archive.go`:
  - `extractBinary(data []byte, assetName string) (name string, content []byte, err error)` — pick tar.gz vs zip by suffix; walk entries; return the entry whose base name is `rpc` or `rpc.exe`.
  - `buildZip(binaryName string, binary, configYAML []byte) ([]byte, error)` — `archive/zip` writer; add the binary entry (set `Mode 0o755` for non-windows via `zip.FileHeader`) and a `config.yaml` entry.
  - `downloadAsset(ctx, url string) ([]byte, error)` — HTTP GET, read body (used online; offline path reads from disk in Task 8).

- [ ] **Step 4: Run, confirm pass.**

- [ ] **Step 5: Commit**

```bash
git add internal/usecase/packaging/archive.go internal/usecase/packaging/archive_test.go
git commit -m "feat: extract rpc-go binary and assemble download zip"
```

---

### Task 8: Packaging usecase (Feature) tying it together

**Files:**
- Create: `internal/usecase/packaging/interface.go`, `internal/usecase/packaging/packaging.go`
- Test: `internal/usecase/packaging/packaging_test.go`

- [ ] **Step 1: Define the interface** in `interface.go`:

```go
package packaging

import (
	"context"
	"io"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
)

type Feature interface {
	ListVersions(ctx context.Context) ([]dto.RpcRelease, error)
	BuildPackage(ctx context.Context, req dto.PackageRequest) (io.Reader, string, error)
}
```

- [ ] **Step 2: Write failing test** in `packaging_test.go`: construct a `Service` whose GitHub base points at an `httptest.Server` and whose `LocalDir` holds a fixture archive; call `BuildPackage` with a token+deactivate request; assert no error, a filename `rpc-deactivate-linux-x86_64.zip`, and that reading the returned zip yields `config.yaml` + binary. Use the offline path (point GitHub base at an unreachable URL) so the test is deterministic and offline.

- [ ] **Step 3: Run, confirm fail.**

- [ ] **Step 4: Implement** `Service` in `packaging.go`:
  - `New(cfg *config.Config, l logger.Interface) *Service`.
  - `ListVersions`: try `listReleasesFrom(ctx, releasesURL(cfg.Package.RPCRepo))`; on error and when `cfg.Package.LocalDir != ""`, return `listLocalReleases(cfg.Package.LocalDir)`.
  - `BuildPackage`: resolve the asset for `req.Version/OS/Arch` (online asset list or local dir); obtain bytes (`downloadAsset` or read file); `extractBinary`; compute endpoints from `cfg.Package.PublicURL` (or host/port); if `req.Auth.Mode=="token"` set `AuthToken = mintToken(cfg.JWTKey)`; `renderConfig`; `buildZip`; return `bytes.NewReader(zip), "rpc-<cmd>-<os>-<arch>.zip", nil`.
  - Wrap errors with `consoleerrors` as neighbors do.

- [ ] **Step 5: Run, confirm pass.** `go test ./internal/usecase/packaging/...`

- [ ] **Step 6: Commit**

```bash
git add internal/usecase/packaging/interface.go internal/usecase/packaging/packaging.go internal/usecase/packaging/packaging_test.go
git commit -m "feat: add packaging usecase for download-rpc"
```

---

### Task 9: Controller + wiring

**Files:**
- Create: `internal/controller/httpapi/v1/package.go`
- Test: `internal/controller/httpapi/v1/package_test.go`
- Modify: `internal/usecase/usecase.go`, `internal/controller/httpapi/router.go`

- [ ] **Step 1: Write failing handler tests** in `package_test.go` (follow `domains_test.go` / `ciraconfigs_test.go` setup with a mocked `packaging.Feature` and `gin` test context): GET `/api/package/rpc-versions` returns 200 + the mock's releases JSON; POST `/api/package` with an invalid body (missing command) returns 400; POST with a valid body returns 200, `Content-Type: application/zip`, and the bytes from the mock. Generate or hand-write a mock for `packaging.Feature` consistent with how other usecases are mocked in this repo (`internal/mocks`).

- [ ] **Step 2: Run, confirm fail.**

- [ ] **Step 3: Implement** `internal/controller/httpapi/v1/package.go`:

```go
package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	dto "github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/internal/usecase/packaging"
	"github.com/device-management-toolkit/console/pkg/logger"
)

type packageRoutes struct {
	t packaging.Feature
	l logger.Interface
}

func NewPackageRoutes(handler *gin.RouterGroup, t packaging.Feature, l logger.Interface) {
	r := &packageRoutes{t, l}
	h := handler.Group("/package")
	{
		h.GET("/rpc-versions", r.versions)
		h.POST("", r.build)
	}
}

func (r *packageRoutes) versions(c *gin.Context) {
	rels, err := r.t.ListVersions(c.Request.Context())
	if err != nil {
		r.l.Error(err, "http - v1 - rpc-versions")
		ErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, rels)
}

func (r *packageRoutes) build(c *gin.Context) {
	var req dto.PackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorResponse(c, err)
		return
	}
	reader, filename, err := r.t.BuildPackage(c.Request.Context(), req)
	if err != nil {
		r.l.Error(err, "http - v1 - build package")
		ErrorResponse(c, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.DataFromReader(c.Writer.Status(), -1, "application/zip", reader, nil)
}
```

(If `c.DataFromReader` needs a known length, buffer the zip into `[]byte` in the usecase and return its length; adjust `BuildPackage` to also return size, or use `c.Data(200, "application/zip", bytes)`.)

- [ ] **Step 4: Wire it up.** In `internal/usecase/usecase.go`: add `Packaging packaging.Feature` to `Usecases` and set `Packaging: packaging.New(cfg, log)` in `NewUseCases` (thread `cfg` through if `NewUseCases` doesn't already receive it — check the signature; `config` may need to be passed in). In `internal/controller/httpapi/router.go`: after the protected group is created, add `v1.NewPackageRoutes(protected, t.Packaging, l)`.

- [ ] **Step 5: Run, confirm pass.** `go test ./internal/controller/... ./internal/usecase/...`

- [ ] **Step 6: Commit**

```bash
git add internal/controller/httpapi/v1/package.go internal/controller/httpapi/v1/package_test.go internal/usecase/usecase.go internal/controller/httpapi/router.go
git commit -m "feat: expose download-rpc package endpoints"
```

---

### Task 10: Full verification

- [ ] **Step 1: Build** — `go build ./...` → success.
- [ ] **Step 2: Test** — `go test ./...` → all pass.
- [ ] **Step 3: Lint/vet** — run the repo's configured linter (check `Makefile`: e.g. `make lint` or `golangci-lint run`) → clean.
- [ ] **Step 4: Manual smoke (optional)** — start Console with `RPC_LOCAL_DIR` pointing at a folder with one rpc-go archive; `curl` `GET /api/package/rpc-versions` (with auth) and a `POST /api/package`; confirm a valid zip downloads.
- [ ] **Step 5: Final commit** if anything was adjusted.

---

## Self-Review Notes

- **Spec coverage:** versions endpoint + v3 filter + offline fallback (Tasks 3,4,8) ✓; POST package → zip (Tasks 5–9) ✓; token vs userpass (Tasks 5,6,8) ✓; activate url + domain / deactivate (Task 5) ✓; binary extraction for tar.gz/zip (Task 7) ✓; DTO contract matches UI (Task 1) ✓; config + wiring (Tasks 2,9) ✓; tests at every layer (each task) ✓.
- **Type consistency:** `dto.PackageRequest/RpcRelease/RpcAsset`, `packaging.Feature`, `Service`, `parseAsset`, `isV3OrAbove`, `listReleasesFrom`, `listLocalReleases`, `renderConfig`, `mintToken`, `extractBinary`, `buildZip`, `downloadAsset`, `NewPackageRoutes` are referenced consistently across tasks.
- **Open items carried from the design** (resolve during implementation): exact rpc-go v3 `auth-endpoint`/`devices-endpoint` paths; whether `NewUseCases` already receives `*config.Config` (thread it if not); `c.DataFromReader` length handling; offline-dir version convention (`version.txt` vs dir name — Task 4 picks and documents one).
