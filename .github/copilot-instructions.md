# GitHub Copilot Instructions

See **[../CLAUDE.md](../CLAUDE.md)** — the canonical guide for AI coding assistants in this repository. The content is tool-neutral and applies to GitHub Copilot Chat / Copilot Workspace just as it does to any other agent. Edit `CLAUDE.md`; this file is a pointer.

Key non-negotiables (full detail in `CLAUDE.md`):

- Console is the v3 successor to **MPS + RPS**; `/api/v1/*` is the 1:1 migration contract and must stay backwards compatible. New shapes that don't fit v1 go under `/api/v2/*`.
- Never hand-author WSMAN XML — use `github.com/device-management-toolkit/go-wsman-messages/v2` via the `devices.WSMAN` interface in `internal/usecase/devices/wsman`.
- Controllers under `internal/controller/httpapi/v{1,2}/*.go` depend on the matching `<feature>.Feature` interface (e.g. `devices.Feature`) — they do not reach into repos or build WSMAN themselves.
- API changes must update the Gin handler (`internal/controller/httpapi/v{1,2}/*.go`), the Fuego/OpenAPI declaration (`internal/controller/openapi/*.go`), **and** the Postman collections under `integration-test/collections/` (`console_mps_apis.postman_collection.json`, `console_rps_apis.postman_collection.json`, plus `console_environment.postman_environment.json` if new variables are needed) in the same PR. Run `go run ./cmd/openapi-gen` and commit `doc/openapi.json` if affected.
- Storage backends must stay in lock-step: three databases (Postgres, SQLite, MongoDB) across two implementation packages. Any new `Repository` method needs implementations in **both** `internal/usecase/sqldb` (Postgres + SQLite share code) **and** `internal/usecase/nosqldb/mongo`, plus matching SQL migrations under `internal/app/migrations/`.
- Run `make mock` after editing any `Repository`/`Feature`/`WSMAN` interface — `internal/mocks/` is generated. The `mock` target encodes non-obvious `mockgen` flags; don't reproduce them by hand.
- Device credentials (`Password`, `MPSPassword`, `MEBXPassword`) are always encrypted via `safeRequirements.Encrypt` in `dtoToEntity`. Never write plaintext to `entity.Device`.
- Touching `internal/controller/tcp/cira/`, `internal/usecase/devices/redirection*.go`, or the WebSocket relay (`internal/controller/ws/v1/redirect.go`)? Trace the byte flow end to end and update the sibling `*_test.go` — APF state-machine races have caused real outages in the predecessor.
- Module path is `github.com/device-management-toolkit/console`; Go 1.25+ required. Commands run natively on Linux, macOS, and Windows — WSL is not required.
- Small, focused PRs only. No scope creep. Stack prerequisite refactors as `refactor:` / `test:` / `build:` ahead of the `feat:` that triggers a release.
- Commits follow Conventional Commits (commitlint-enforced). `feat:` cuts a minor release, `fix:`/`perf:`/`chore:` cuts a patch, `BREAKING CHANGE:` cuts a major — avoid breaking `/api/v1/*`. See `CONTRIBUTING.md` for the full scope list.
- Before declaring done: `go test -race -count=1 ./...`, `gofumpt -l -w -extra ./` (no diff), `go vet ./...`, and `docker run --rm -v .:/app -w /app golangci/golangci-lint:latest golangci-lint run -v --fix` (use `-v ${pwd}:/app` on Windows PowerShell — no remaining diagnostics) all green.
