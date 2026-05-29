# Download RPC — Console Backend Design

**Date:** 2026-05-28
**Status:** Approved contract (from the UI brainstorming); pending implementation plan review
**Repo:** `console` (Go API gateway)
**Companion:** `sample-web-ui` design `docs/superpowers/specs/2026-05-28-download-rpc-design.md` (the UI half, already implemented on branch `feat/download-rpc`).

## Summary

Add two Console endpoints that back the UI's "Download RPC" page. Console does all GitHub
network access, archive handling, config templating, and token minting server-side so the browser
makes a single same-origin call and avoids CORS:

- `GET /api/package/rpc-versions` — list rpc-go releases (v3+, betas included), each with the
  OS/arch assets available. Falls back to scanning a local directory when there's no internet.
- `POST /api/package` — given a command (activate/deactivate), version, os, arch, and auth choice,
  return a zip containing the extracted rpc-go binary plus a generated `config.yaml`.

This is the Console-only half of the feature; the cloud/standalone (MPS+RPS) flavor does not have
Console and intentionally does not get this feature.

## Endpoints

Both are registered under the existing protected `/api` group (so they require the same JWT/OAuth
gate as the rest of the API, and the UI reaches them at `${CONSOLE_SERVER_API}/api/package...`).
Register via a new `v1.NewPackageRoutes(protected, t.Packaging, l, cfg)` in
`internal/controller/httpapi/router.go`, creating `protected.Group("/package")`:

```
GET  /api/package/rpc-versions
POST /api/package
```

### `GET /api/package/rpc-versions`

Response `200` — array of releases:

```json
[
  { "version": "v3.0.1", "assets": [ { "os": "linux", "arch": "x86_64" }, { "os": "windows", "arch": "x86_64" }, { "os": "linux", "arch": "arm64" } ] },
  { "version": "v3.1.0-beta", "assets": [ ... ] }
]
```

- Source: `GET https://api.github.com/repos/<rpc-go repo>/releases` (paginated; the repo defaults to
  `device-management-toolkit/rpc-go`, configurable).
- Filter: keep releases whose semver **major >= 3**. Include prereleases (betas).
- Map each GitHub asset name (e.g. `rpc-go_Linux_x86_64.tar.gz`, `rpc-go_Windows_x86_64.zip`,
  `rpc-go_Linux_arm64.tar.gz`) to normalized `{ os, arch }` where `os ∈ {linux, windows, darwin}`
  (lowercased) and `arch` is the raw arch token (`x86_64`, `arm64`, …). Skip assets that don't parse
  as an OS/arch build (checksums, sources).
- **Offline fallback:** if the GitHub request fails (network error / non-200), scan the configured
  local directory (`Package.LocalDir`) for rpc-go release archives and synthesize the same response
  from the filenames found. The `os`/`arch` returned must match exactly what `POST /api/package`
  can later resolve from that directory.

### `POST /api/package`

Request body (matches the UI's `PackageRequest`):

```json
{
  "command": "activate",            // "activate" | "deactivate"
  "version": "v3.0.1",
  "os": "linux",
  "arch": "x86_64",
  "auth": { "mode": "token" },      // or { "mode": "userpass", "username": "...", "password": "..." }
  "profile": "myProfile",            // present for activate
  "domain": "my.domain.com"          // present for activate + ACM only
}
```

Response `200`: `Content-Type: application/zip`, `Content-Disposition: attachment; filename="rpc-<command>-<os>-<arch>.zip"`, body = zip containing:
- the extracted `rpc` (or `rpc.exe` for windows) binary, and
- `config.yaml`.

Validation errors → `400`; upstream/internal failures → `500`, using the existing `ErrorResponse`
helper and `consoleerrors` wrapping idiom.

## Config.yaml generation

Render the **full** config (same shape as the UI's `docs/rpc-config.sample.yml`); only the rows
below are populated, the rest keep their sample defaults. Use `gopkg.in/yaml.v2` (already a dep),
driven by a struct or a template — prefer a typed struct mirroring the sample for safety.

| Input | config.yaml field(s) |
|---|---|
| always | `auth-endpoint`, `devices-endpoint` (derived from Console config — see below) |
| `auth.mode = token` | `auth-token` = freshly minted JWT (see below) |
| `auth.mode = userpass` | `auth-username`, `auth-password` (written verbatim) |
| `command = activate` | `activate.url` = `<endpoint base>/api/v1/admin/profiles/export/<profile>` plus `?domainName=<domain>` when domain present |
| `command = deactivate` | remote deactivate fields + the shared server-auth block (token or user/pass) |

**Server endpoints.** `auth-endpoint` / `devices-endpoint` and the `activate.url` host are derived
from a configured public base URL for this Console (new config `Package.PublicURL`, env
`CONSOLE_PUBLIC_URL`; if unset, construct from `HTTP.Host`/`HTTP.Port`). Concrete paths to wire and
confirm against rpc-go v3 expectations: `devices-endpoint = <base>/api/v1/devices`,
`auth-endpoint = <base>/api/v1/authorize`. **Open item for the implementer:** verify these exact
paths against the rpc-go v3 client before shipping.

**Token minting.** For `auth.mode = token`, mint an HS256 JWT exactly as `login.go` does
(`jwt.NewWithClaims(jwt.SigningMethodHS256, claims)` signed with `Config.JWTKey`), with a registered
expiry. Reuse/extract the minting logic from `LoginRoute.handleBasicAuth` rather than duplicating
the signing details.

## Binary download & extraction

- **Online:** download the resolved asset from its `browser_download_url`. The existing
  `github.Asset` entity must gain a `BrowserDownloadURL string \`json:"browser_download_url"\`` field
  (currently only the API `URL` is modeled).
- **Offline:** read the matching archive from `Package.LocalDir`.
- Extract the `rpc`/`rpc.exe` binary from the archive: `.tar.gz` via `compress/gzip` + `archive/tar`;
  `.zip` via `archive/zip`. Choose the extractor by file extension.
- Assemble the output zip in-memory with `archive/zip` (binary entry preserves the executable mode
  bits for non-windows).

## Architecture & files

Follow the existing controller → usecase → entity layering. The packaging usecase needs no DB
(like `export.FileExporter`), so it is constructed with just config + logger.

- **Entity / DTO** — `internal/entity/dto/v1/package.go`: `PackageRequest`, `RpcRelease`, `RpcAsset`
  with binding/validation tags consistent with other DTOs.
- **Entity** — extend `internal/entity/github/release.go` `Asset` with `BrowserDownloadURL`.
- **Usecase** — `internal/usecase/packaging/`:
  - `interface.go`: `Feature` interface — `ListVersions(ctx) ([]dto.RpcRelease, error)` and
    `BuildPackage(ctx, req dto.PackageRequest) (io.Reader, string /*filename*/, error)`.
  - `packaging.go`: implementation (`Service`) holding `cfg *config.Config` and `logger`.
  - `github.go`: release listing + asset-name parsing + offline directory scan.
  - `config.go`: config.yaml struct + render.
  - `archive.go`: download, extract binary, build output zip.
  - `token.go`: JWT minting helper (shared with / extracted from login).
- **Controller** — `internal/controller/httpapi/v1/package.go`: `packageRoutes{ t packaging.Feature, l logger.Interface }`, `NewPackageRoutes(...)`, handlers `versions` (GET) and `build` (POST).
- **Wiring** — add `Packaging packaging.Feature` to `usecase.Usecases` and construct it in
  `NewUseCases`; register the route in `router.go`.
- **Config** — add a `Package` section to `config/config.go` (+ `config/config.yml` defaults):
  `RPCRepo` (default `device-management-toolkit/rpc-go`, env `RPC_REPO`), `LocalDir`
  (env `RPC_LOCAL_DIR`), `PublicURL` (env `CONSOLE_PUBLIC_URL`).

## Testing

Go table-driven unit tests alongside each file (`*_test.go`), run with `go test ./...`:
- Asset-name parsing: various rpc-go filenames → expected os/arch; non-asset files skipped.
- Version filtering: v2/v1 excluded, v3+ and v3-beta included.
- Offline fallback: GitHub error → directory scan path produces the same shape.
- config.yaml generation: token vs userpass; activate (with/without domain) vs deactivate — assert
  the populated fields and that the file is valid YAML.
- Token minting: produced JWT verifies against `Config.JWTKey` with HS256 and has an expiry.
- Archive extraction: a small `.tar.gz` and `.zip` fixture each yield the expected binary; output
  zip contains binary + config.yaml.
- Controller handlers: 400 on bad body, 200 with `application/zip` on success (usecase mocked).

## Out of scope (matches UI v1)

- `configure` command tree.
- Local (on-box) deactivate — remote only.
- Multi-asset bundling — exactly one os/arch binary per package.

## Open items to confirm during implementation

1. Exact rpc-go v3 `auth-endpoint` / `devices-endpoint` paths and whether `activate.url` should point
   at Console's profile export route or RPS's.
2. Whether GitHub release listing needs auth (rate limits) in air-gapped/CI; the offline directory
   path is the mitigation.
3. Final `Package.PublicURL` derivation when unset (host/port vs. request host header).
