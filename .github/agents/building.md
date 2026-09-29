# Building release binaries

Read this when you need to produce a binary for release, test a build tag
(`noui`, `tray`), or cross-compile for another OS. For normal local
development you do not need this file — use `go run ./cmd/app` instead
(see the "Running the app" section of `CLAUDE.md`).

## Standard builds

`CGO_ENABLED=0` produces statically-linked, cross-OS binaries from any host:

```sh
# bash/zsh
CGO_ENABLED=0 go build -o ./bin/console ./cmd/app
CGO_ENABLED=0 go build -tags=noui -o ./bin/console-noui ./cmd/app

# PowerShell
$env:CGO_ENABLED=0; go build -o ./bin/console ./cmd/app
$env:CGO_ENABLED=0; go build -tags=noui -o ./bin/console-noui ./cmd/app
```

## Cross-compiling

Set `GOOS` and `GOARCH` before the build command:

- `GOOS`: `linux`, `windows`, or `darwin`
- `GOARCH`: `amd64` or `arm64`

Release binaries add `-ldflags "-s -w" -trimpath`. `-s -w` removes debug
symbols to make the file smaller. `-trimpath` removes local directory names
from the binary, so the same source always produces the same output.

## Make targets

These wrap the flag combinations above. Use them instead of typing the flags
by hand:

| Target | What it does |
|---|---|
| `make build` | Normal binary, UI embedded |
| `make build-noui` | Binary with `-tags=noui` (cloud deployments) |
| `make build-tray` | Binary with the system-tray launcher |
| `make build-all-platforms` | Full release matrix, written to `dist/` |

`make build-tray` needs `CGO_ENABLED=1` and only builds for the machine you
run it on. The system tray uses native OS libraries, so it cannot be
cross-compiled.

## Build tags

| Tag | Effect |
|---|---|
| (none) | Embeds `internal/controller/httpapi/ui/` and serves the UI |
| `noui` | No embedded UI. Used for cloud/multi-tenant deployments |
| `tray` | Adds the system-tray launcher and enables the `--tray` flag |

`internal/controller/httpapi/ui/` is empty in a fresh clone — everything in it
except `.gitkeep` is ignored by `.gitignore`. The release workflow fills it
from `sample-web-ui`'s `build-enterprise` output. This means `//go:embed
all:ui` embeds an empty directory locally, so you do not need `-tags=noui`
during development.
