# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Canonical guide for AI coding assistants working in this repository. The filename is historical; the content is tool-neutral and applies to any agent (Claude Code, Codex, Cursor, Aider, Continue, Gemini CLI, GitHub Copilot, etc.). `AGENTS.md` and `.github/copilot-instructions.md` are pointers to this file — keep edits here.

## Overview

Console is the **v3 successor to MPS + RPS** for the Intel® Device Management Toolkit. It collapses both services into a single Go binary that handles device activation **and** post-activation management (power control, KVM, SOL, IDE-R, audit/event logs, certificates, profiles, CIRA configuration, etc.). The same binary serves two deployment shapes:

- **Local / enterprise:** single instance with embedded SQLite, optional system-tray launcher. Release binaries bundle an embedded web UI; locally the UI is served separately on `:4200`. The default `go run` works with zero infra.
- **Cloud / multi-tenant:** Postgres or MongoDB backend, `noui` build flag, Vault-backed secrets, fronted by a gateway that terminates auth.

**API parity with MPS+RPS is a hard goal.** The `/api/v1/*` surface is intentionally 1:1 with the MPS/RPS REST contracts so existing integrators (Sample Web UI, scripts, partner tooling) migrate by swapping the base URL. New behavior that doesn't fit the legacy shape goes under `/api/v2/*`. **Never break `/api/v1/*`.** See [Implementation guidelines](#implementation-guidelines-non-negotiable).

Sibling reference repos in the workspace: [MPS](https://github.com/device-management-toolkit/mps) (Node, legacy management plane) and [RPS](https://github.com/device-management-toolkit/rps) (Node, legacy activation plane). Use them to verify request/response shapes when porting endpoints.

## Commands


**Go 1.25+ required** (per `go.mod`). **Module path is `github.com/device-management-toolkit/console`** — all intra-repo imports use that prefix. Copy `.env.example` to `.env` (the VS Code launch config and the few `make` targets that still apply read it via `envFile` / `-include`; raw `go run` reads env vars from your shell).

### Running the app

The recommended local workflow is **VS Code's Go debugger** — `.vscode/launch.json` ships a `Launch Package` config that points at `./cmd/app` with `.env` loaded as the env file. Hit F5 and you get breakpoints, variable inspection, and goroutine views for free. For the frontend, run `sample-web-ui` separately on `:4200` (`npm run enterprise` in that repo).

You don't need `-tags=noui` locally: `internal/controller/httpapi/ui/` is empty in this repo (the release workflow populates it from `sample-web-ui`'s `build-enterprise` output), so `//go:embed all:ui` just embeds an empty directory and UI routes return 404 / redirect to `ui.externalUrl`. Set `ui.externalUrl: "http://localhost:4200"` in `config/config.yml` if you want UI links to bounce to the dev server.

If you can't use the VS Code debugger, run via `go run` directly:

| Shell | Run |
|---|---|
| bash/zsh (Linux, macOS) | `GIN_MODE=debug go run ./cmd/app` |
| PowerShell (Windows) | `$env:GIN_MODE="debug"; go run ./cmd/app` |
| cmd.exe (Windows) | `set GIN_MODE=debug && go run ./cmd/app` |

App listens on `HTTP_PORT` (default `8181`); CIRA on `:4433` when `APP_DISABLE_CIRA=false`. Pass a custom config path with `--config /absolute/path/to/config.yml`. The `--tray` flag only works on binaries built with `-tags=tray`.

### Building

`CGO_ENABLED=0` produces statically-linked, cross-OS binaries from any host:

```sh
# bash/zsh
CGO_ENABLED=0 go build -o ./bin/console ./cmd/app
CGO_ENABLED=0 go build -tags=noui -o ./bin/console-noui ./cmd/app

# PowerShell
$env:CGO_ENABLED=0; go build -o ./bin/console ./cmd/app
$env:CGO_ENABLED=0; go build -tags=noui -o ./bin/console-noui ./cmd/app
```

Cross-compile by setting `GOOS=linux|windows|darwin` and `GOARCH=amd64|arm64`. Release binaries use `-ldflags "-s -w" -trimpath` to strip and reproducibly trim paths.

**`make build` / `make build-noui` / `make build-tray` / `make build-all-platforms`** are useful convenience wrappers for the canonical flag combinations (the all-platforms target emits the full release matrix to `dist/`). `build-tray` requires `CGO_ENABLED=1` and only builds for the host OS — system tray pulls in native deps.

### Testing

```sh
go test -race -count=1 ./...                                    # whole suite
go test -race ./internal/usecase/devices/...                    # one package
go test -race -run '^TestSpecificName$' ./internal/usecase/...  # one test
go test -race -v -coverprofile=coverage.out ./...               # with coverage
```

`-count=1` defeats the test cache. `-race` is mandatory locally — CI runs with it.

### Fuzz tests

Fuzz targets live next to their packages as `*_fuzz_test.go`. Go can only fuzz one target per `go test` invocation, so the `make fuzz-*` helpers exist to enumerate + iterate:

```sh
make fuzz-list                                                  # discover targets
make fuzz-one PKG=./internal/usecase/devices TARGET=FuzzParseInterval FUZZTIME=30s
make fuzz-smoke                                                 # every target, 1x execution (CI smoke)
make fuzz-all FUZZTIME=2m                                       # every target, 2m budget each
```

The raw equivalent for ad-hoc fuzzing is `go test -run=^$ -fuzz='^FuzzName$' -fuzztime=30s ./path`.

### Mocks

`internal/mocks/*` is generated from the `interfaces.go` files. Regenerate after editing any `Repository` / `Feature` / `WSMAN` interface and commit alongside the change:

```sh
make mock
```

The target encodes 13 `mockgen -source ... -mock_names ...` invocations with non-obvious aliases; reproducing them by hand is error-prone.

### OpenAPI spec

```sh
go run ./cmd/openapi-gen      # writes doc/openapi.json
```

The live spec is also served at `GET /api/openapi.json` on the running server. CI publishes `doc/openapi.json` to SwaggerHub — commit changes whenever you touch routes.

### Database (optional, for Postgres / Mongo development)

Default `DB_PROVIDER=sqlite` needs nothing — the DB file is created at `~/.config/device-management-toolkit/console.db` (Linux/macOS) or `%APPDATA%\device-management-toolkit\console.db` (Windows). For Postgres or Mongo dev:

```sh
docker compose up -d postgres                # Postgres-backed local dev
docker compose --profile mongo up -d mongo   # Mongo-backed local dev (profile-gated)
docker compose down --remove-orphans         # stop everything
```

Migrations (Postgres only — the SQLite path runs them automatically, Mongo doesn't use them):

```sh
make migrate-create        # scaffold a new migration pair
make migrate-up            # apply pending migrations
```

Both targets shell out to `golang-migrate/migrate` — install once via `make bin-deps` (also installs `mockgen` into `./bin/`).

### Linting and formatting

CI rejects unformatted or unlinted code. Recommended local checks:

```sh
go install mvdan.cc/gofumpt@latest                              # one-time install
gofumpt -l -w -extra ./                                         # auto-format
go vet ./...                                                    # vet

# Dockerized golangci-lint — applies the same .golangci.yml that CI uses
# bash/zsh:
docker run --rm -v .:/app -w /app golangci/golangci-lint:latest golangci-lint run -v --fix
# PowerShell:
docker run --rm -v ${pwd}:/app -w /app golangci/golangci-lint:latest golangci-lint run -v --fix
```

`--fix` auto-applies safe corrections (including import grouping via `gci`). What CI actually does: the `formatting` job runs `gofmt -s -l . | wc -l` and fails on any output, then `go vet ./...`; the `golangci-lint` job runs `reviewdog/action-golangci-lint` against the same `.golangci.yml`. `gofumpt` is a strict superset of `gofmt -s`, so anything passing `gofumpt` also passes CI's formatter; the same `.golangci.yml` drives both lint invocations, but **binary versions are not pinned identically** between the Docker image and `reviewdog`'s action — occasional skew on bleeding-edge linters is possible.

`.golangci.yml` is strict — `cyclop` 15, `funlen` 100, `mnd`, `paralleltest`, `wsl_v5`, `err113`, and friends. CI runs the same; fix locally first.

## Architecture

Console follows a Clean Architecture layering. Each layer depends only inward; entities are the innermost ring.

```
cmd/app/main.go        // composition root: config, secrets, certs, encryption key, tray vs server
  └─ internal/app/     // wiring: buildRepos() + Run() (HTTP server + CIRA server)
       ├─ internal/controller/      // delivery layer (Gin HTTP, WebSocket, TCP/TLS)
       │    ├─ httpapi/v1, v2       // REST routes
       │    ├─ ws/v1                // /relay/webrelay.ashx — KVM/SOL/IDER over WebSocket
       │    ├─ tcp/cira             // CIRA listener on :4433 (APF protocol)
       │    └─ openapi              // Fuego adapter that emits the spec
       ├─ internal/usecase/         // business logic (devices, profiles, domains, …)
       │    ├─ devices/wsman        // WSMAN client wrapper (only place that builds WSMAN)
       │    ├─ sqldb                // shared Postgres+SQLite repo implementations
       │    └─ nosqldb/mongo        // Mongo repo implementations
       ├─ internal/entity/          // domain types (entity.Device, etc.) and DTOs (dto/v1, dto/v2)
       ├─ internal/repoerrors/      // typed repo errors that controllers map to HTTP
       └─ internal/mocks/           // generated by `make mock` — do not hand-edit
pkg/                   // reusable, no-internal-deps utilities (db, logger, httpserver, secrets/vault, tray, consoleerrors)
config/                // config.yml, generated certs, package-level config.Config
```

**The composition root is `cmd/app/main.go` plus `internal/app/`.** They are the only places that import storage drivers, Vault, Gin, the TLS listener, etc. Use cases and entities stay free of those — they accept interfaces (`devices.Repository`, `devices.WSMAN`, `security.Cryptor`).

### Wiring (`cmd/app/main.go` → `internal/app/app.go`)

1. `config.NewConfig()` layers defaults (`defaultConfig`) → `config/config.yml` (auto-created on first run) → env vars via `cleanenv`. The resulting `*config.Config` is also stashed in the package-level `config.ConsoleConfig` singleton — that's how middleware/handlers read auth settings without constructor-passing.
2. `handleSecretsConfig` dials Vault if `SECRETS_ADDR` and `SECRETS_TOKEN` are set. Vault stores the AMT encryption key (`default-security-key`) and domain certificates.
3. `handleEncryptionKey` tries Vault → OS keyring (`zalando/go-keyring`) → prompts the user to generate. The key is required to encrypt device passwords; losing it makes existing rows unreadable.
4. `setupCIRACertificates` (only if `APP_DISABLE_CIRA=false`) loads/generates the root + web-server certs that the CIRA listener will present to AMT.
5. `app.Run` then constructs:
   - **Repos** via `buildRepos` — switches on `cfg.Provider` (`postgres` | `sqlite` (default) | `mongo`).
   - **Usecases** via `usecase.NewUseCases(repos, log, certStore)` — wires every feature.
   - **HTTP server** (Gin) with Prometheus middleware, the Fuego OpenAPI adapter, `/api/v1` and `/api/v2` router groups, `/relay/webrelay.ashx` WebSocket, and (when UI is embedded) the embedded UI handler.
   - **CIRA server** (`internal/controller/tcp/cira`) on `:4433`, terminating TLS for inbound AMT devices.
6. `waitForShutdown` blocks on SIGINT/SIGTERM or either server's notify channel; `shutdownServers` then closes them.

### Database providers (pluggable)

`internal/app/repos.go` is the *only* place that imports concrete drivers:

- **Postgres / SQLite** share the `internal/usecase/sqldb` package and the `pkg/db` pool. SQLite is the default (`Provider: ""` or `"sqlite"`) and stores at `~/.config/device-management-toolkit/console.db` (Linux/macOS) or `%APPDATA%\device-management-toolkit\console.db` (Windows). Postgres requires `DB_URL`.
- **MongoDB** lives in `internal/usecase/nosqldb/mongo`. `buildMongoRepos` uses a 30s startup timeout and a 5s shutdown timeout so an unreachable Mongo fails fast.

When adding a new repo method:

1. Define it on the `Repository` interface in `internal/usecase/<feature>/interfaces.go`.
2. Implement it in **both** `internal/usecase/sqldb/<feature>.go` and `internal/usecase/nosqldb/mongo/<feature>.go`. They must behave identically — the use case can't know which is mounted.
3. Run `make mock` to regenerate `internal/mocks/<feature>_mocks.go`.

### Device flow — CIRA + APF (CRITICAL)

The device-side path is **not** request/response. AMT firmware opens a TLS connection to `:4433`; on top of that runs the **APF protocol** (Intel's SSH-variant channel multiplexer). The CIRA listener lives in `internal/controller/tcp/cira/tunnel.go` and uses APF + WSMAN helpers from `github.com/device-management-toolkit/go-wsman-messages/v2/pkg/apf` and `.../pkg/wsman`. Understand the layering before changing anything in `tcp/cira/` or `internal/usecase/devices/redirection.go`:

```
TLS socket (tunnel.go)
  └─ APF (go-wsman-messages /pkg/apf)
       ├─ Auth handshake (USERAUTH_REQUEST) — verified against the devices repo + encrypted MPSPassword
       └─ Channel(s) per AMT target port
            ├─ port 16992 → WSMAN-over-HTTP (digest), driven by usecase/devices via wsman.Management
            └─ port 16994/16995 → raw byte forwarding to the browser WebSocket for KVM/SOL/IDER
```

Long-lived CIRA sockets are tracked in-process and surfaced through `devices.Feature` (`UpdateConnectionStatus`, `UpdateLastSeen`). **Multi-instance cloud deployments require the same socket-affinity model MPS uses:** REST callers must hit the Console instance that owns the device's CIRA socket. Preserve `entity.Device.MPSInstance` semantics on inserts/updates — the field name is historical (matches the MPS DB schema for migration) and reads as "which Console instance currently holds this device's connection."

### REST API (`internal/controller/httpapi/`)

`NewRouter` (`router.go`) mounts:

- **`POST /api/v1/authorize`** — public, JWT issuance via `v1.LoginRoute`. Signed with `cfg.Auth.JWTKey` (HS256), `exp = cfg.Auth.JWTExpiration` (default 24h). When `cfg.Auth.Disabled` is true, the JWT middleware is bypassed — only for local/single-user deployments.
- **`/api/v1/*`** (protected): `/devices`, `/amt/*` (every operation that talks to a live device — power, boot, hwinfo, audit/event log, alarms, certs, KVM screen, link preference, consent…), `/ciracert`.
- **`/api/v1/admin/*`** (protected): `/domains`, `/ciraconfigs`, `/profiles`, `/wirelessconfigs`, `/ieee8021xconfigs`. These are the **former RPS surface** — configuration objects consumed during activation.
- **`/api/v2/*`** (protected): currently `/amt/version` and `/amt/features`. v2 is where new shapes go; do not retrofit v1.
- **`GET /healthz`**, **`GET /metrics`** (Prometheus), **`GET /version`**, **`GET /api/openapi.json`** — operational.
- **`GET /relay/webrelay.ashx`** — WebSocket upgrade for KVM/SOL/IDER. The JWT travels in the `Sec-Websocket-Protocol` header (matching the MPS contract).

Custom validators (`alphanumhyphenunderscore`, `wifistate`) are registered once on the Gin binding engine — see `router.go`.

### OpenAPI generation

Two interlocked routes describe the API:

- **Routes** (Gin handlers) — `internal/controller/httpapi/v1/*.go` and `v2/*.go`. These serve traffic.
- **OpenAPI definitions** — `internal/controller/openapi/*.go` use the Fuego adapter to declare the same endpoints with full request/response schemas. `make openapi` (or `go run ./cmd/openapi-gen`) writes `doc/openapi.json` from these.

**When you add or change a route, update both.** The Fuego declaration is what integrators and SwaggerHub see; the Gin handler is what executes. CI publishes `doc/openapi.json`, so a stale spec is treated as a bug.

### WSMAN access (never hand-author XML)

All WSMAN construction goes through `github.com/device-management-toolkit/go-wsman-messages/v2` — the `AMT`, `CIM`, and `IPS` namespaces under `/pkg/wsman/`. The pattern is:

1. Controllers (`httpapi/v1/*.go`) call methods on `usecases.Devices` (the `devices.Feature` interface).
2. The use case (`internal/usecase/devices/`) holds a `WSMAN` interface (see `interfaces.go`) — concretely implemented by `internal/usecase/devices/wsman` using `go-wsman-messages`.
3. Per-device clients are created lazily by `SetupWsmanClient` (which the use case calls), cached, and torn down via `DestroyWsmanClient`. A background `Worker()` goroutine prunes idle clients.

If you need a WSMAN call that isn't in `go-wsman-messages` yet, **fix it upstream** in that repo (it's a sibling working directory in this workspace) rather than crafting raw XML here.

### Secrets and encryption

- `pkg/secrets/vault` wraps the Vault HTTP API to read/write the encryption key and certificates. It's optional; absence is non-fatal.
- `security.Crypto` (from `go-wsman-messages/v2/pkg/security`) does AES-GCM with the encryption key. Every device password (`Password`, `MPSPassword`, `MEBXPassword`) is encrypted in `dtoToEntity` before the DB write. Never store plaintext in `entity.Device`.
- The OS keyring (via `zalando/go-keyring`) is the local fallback for the encryption key. On Linux this needs a Secret Service implementation (GNOME Keyring / `seahorse`); the README has the troubleshooting note.

### Testing notes

- Framework: standard `testing` + `github.com/stretchr/testify/require` + `go.uber.org/mock/gomock`. Mocks live in `internal/mocks/` and are **generated** — do not hand-edit them.
- `paralleltest` is on in `.golangci.yml`. **New tests should call `t.Parallel()`** at the top of every `Test*` (and in each subtest's closure for table-driven tests). Existing files that don't are technical debt; don't add new violations.
- Table-driven tests with `gomock`: capture the loop variable into a local before the subtest closure (`tt := tt`) — `paralleltest` flags this.
- Fuzz tests live next to their package as `*_fuzz_test.go`. They share the same `t.Parallel` rule (when not using `*testing.F`).
- Integration tests live in `./integration-test/...` and expect a running Compose stack (`docker compose up -d postgres`). Run them with `go test -v ./integration-test/...` when you need them; they're slow and not part of the default loop.
- `internal/mocks/` and `internal/usecase/devices/wsman/*` are excluded from coverage (`codecov.yml`) — the latter is a thin adapter over the WSMAN library.

## Implementation guidelines (non-negotiable)

- **Never hand-author WSMAN XML.** All WSMAN goes through `go-wsman-messages` via the `devices.WSMAN` interface in `internal/usecase/devices/wsman`. Add new device operations as methods on `devices.Feature` and call them from controllers; if a needed message is missing, fix it upstream in `go-wsman-messages` rather than crafting raw XML here.
- **Controllers go through `usecases.Devices` (or the matching feature).** Don't reach into the repo from a controller. Don't hold WSMAN clients in the HTTP layer. The flow is always controller → feature → (repo and/or WSMAN).
- **REST API changes must be backwards compatible.** `/api/v1/*` is the MPS+RPS migration contract — Sample Web UI, partner tooling, and existing scripts depend on field names, status codes, and query params staying stable. Prefer additive changes (new optional field, new endpoint, new query param) over renaming, removing, or tightening existing ones. Behaviour that genuinely doesn't fit the legacy shape goes under `/api/v2/*`. The same rule applies to DB schema and DTO changes: existing rows must keep working.
- **API changes must update the Gin handler, the Fuego/OpenAPI declaration, AND the Postman collections in the same PR.** `internal/controller/httpapi/v{1,2}/*.go` serves traffic; `internal/controller/openapi/*.go` describes it; `integration-test/collections/console_mps_apis.postman_collection.json` (device/management surface) and `console_rps_apis.postman_collection.json` (activation/config surface — domains, profiles, wireless, CIRA configs) are what integrators and QA test against. Add new request entries, update changed shapes, and bump `console_environment.postman_environment.json` if your change introduces new variables. Run `go run ./cmd/openapi-gen` and commit the regenerated `doc/openapi.json` if your change is reachable from the spec. A drifted Postman collection or Swagger spec is treated as a bug.
- **Storage backends must stay in lock-step.** Console supports three databases (Postgres, SQLite, MongoDB) across two implementation packages: `internal/usecase/sqldb` (Postgres + SQLite share code) and `internal/usecase/nosqldb/mongo`. Any new method on a `Repository` interface needs implementations in **both** packages, plus matching SQL migrations under `internal/app/migrations/`. A method that only works on one backend is a bug — the use case has no way to know which it got.
- **Run `make mock` after any interface change.** Stale mocks in `internal/mocks/` are a frequent source of "tests pass locally but fail in CI" reports. The target enumerates every mocked interface with the right `mockgen` flags; keep new interfaces in sync there.
- **Encryption is mandatory for credential fields.** Device passwords (`Password`, `MPSPassword`, `MEBXPassword`) are always written through `safeRequirements.Encrypt` and read through `safeRequirements.Decrypt`. Don't add a new credential field to `entity.Device` without wiring both sides in `dtoToEntity` / `entityToDTO`.
- **Keep PRs small and scoped to one concern.** Touch only the files relevant to the issue. **Do not scope-creep** — unrelated bug, dead code, lint nit, or formatting drift you notice while working belongs in a separate PR/issue. A focused 50-line diff gets reviewed and merged; a 500-line "while I was in there" diff stalls and risks regressions in unrelated CIRA/redirection paths.
- **Work in incremental phases — this is an agile team.** Plan features as a sequence of small, independently-reviewable PRs rather than one big bang. If a PR grows past the point where a reviewer can hold it in their head (rough rule of thumb: a few hundred meaningful lines, or more than one logical concern), stop and break it into smaller PRs that stack. Each PR should leave `main` in a working state.
- **Order PRs around the semver release impact.** Releases are automated by semantic-release from conventional commits: `feat:` cuts a **minor**, `fix:`/`perf:` cuts a **patch**, `BREAKING CHANGE:` cuts a **major**, `chore:` is configured to cut a **patch** in `.releaserc.json`, and `refactor:`/`docs:`/`test:`/`style:`/`build:`/`ci:` do **not** cut a release. When a feature needs prerequisite plumbing (extracted helpers, internal API reshaping, test scaffolding, schema groundwork that's a no-op without the feature), land those prerequisites first as `refactor:` (or `test:`/`build:`) so they ship invisibly. The final user-visible PR is the `feat:` that flips the switch and triggers the release. Never bundle prerequisites into a `feat:` commit just to save PRs — that ties the release to scaffolding that wasn't ready for users.
- **Touching CIRA / APF / redirection?** Trace the byte flow from `cira.Server.ListenAndServe` → APF channel handler → `usecase/devices.Redirect` / WSMAN consumer, and confirm every state transition you touch is covered by the sibling `*_test.go`. Multi-instance deployments rely on `entity.Device.MPSInstance` to know which Console owns a given CIRA socket — old/new connection races have caused real outages in the predecessor (MPS).
- **Before declaring work done, all of these must be green:** `go test -race -count=1 ./...`, `gofumpt -l -w -extra ./` (no diff), `go vet ./...`, and `docker run --rm -v .:/app -w /app golangci/golangci-lint:latest golangci-lint run -v --fix` (no remaining diagnostics — use `-v ${pwd}:/app` on Windows PowerShell). CI runs the same; fix locally first. Also run `go run ./cmd/openapi-gen` if you changed any handler under `httpapi/` or any declaration under `openapi/`, and `make mock` if you touched any mocked interface.
- **Errors over panics.** Wrap with `fmt.Errorf("layer.fn: %w", err)`; the `errorlint` linter enforces `%w` on wraps. Sentinel errors get `var Err... = errors.New(...)` at package scope. `consoleerrors` provides typed wrappers for use-case errors that map cleanly to HTTP responses in `httpapi/v1/error.go`.
- **Config keys are lowercase YAML with matching `env` tags.** New tunables go in `config/config.go` (with a `defaultConfig` value), `.env.example` for env-var docs, and `config.yml` is auto-rewritten on first run. The package-level `config.ConsoleConfig` is the singleton — don't pass `*Config` through every constructor; reach for it in the few places that need it (auth middleware, websocket validator, encryption key flow).

## Commit conventions (see CONTRIBUTING.md)

Format: `<type>(<scope>): <subject>` with body and optional footer. Types: `feat | fix | docs | style | refactor | perf | test | build | ci | chore | revert`. Common scopes in this repo: `api`, `cira`, `apf`, `config`, `db`, `deps`, `deps-dev`, `docker`, `events`, `gh-actions`, `health`, `redir`, `secrets`, `tray`, `ui`, `utils`, `wsman`. Subject + body lines ≤72 chars (commitlint enforces `body-max-line-length: 200`). Footer references a GitHub issue (`Resolves: #1234` or `Fixes: #1234`). Linear history is preferred; PR authors merge via Rebase or Squash.
