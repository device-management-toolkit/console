![Build](https://img.shields.io/github/actions/workflow/status/device-management-toolkit/console/ci.yml?style=for-the-badge&label=Build&logo=github)
![Codecov](https://img.shields.io/codecov/c/github/device-management-toolkit/console?style=for-the-badge&logo=codecov)
[![OSSF-Scorecard Score](https://img.shields.io/ossf-scorecard/github.com/device-management-toolkit/console?style=for-the-badge&label=OSSF%20Score)](https://api.securityscorecards.dev/projects/github.com/device-management-toolkit/console)
[![Discord](https://img.shields.io/discord/1063200098680582154?style=for-the-badge&label=Discord&logo=discord&logoColor=white&labelColor=%235865F2&link=https%3A%2F%2Fdiscord.gg%2FDKHeUNEWVH)](https://discord.gg/DKHeUNEWVH)

# Console

> Disclaimer: Production viable releases are tagged and listed under 'Releases'. Console is under development. **The current available tags for download are Alpha version code and should not be used in production.** For these Alpha tags, certain features may not function yet, visual look and feel may change, or bugs/errors may occur. Follow along our [Feature Backlog for future releases and feature updates](https://github.com/orgs/device-management-toolkit/projects/10).

## Overview

Console is an application that provides a 1:1, direct connection for AMT devices for use in an enterprise environment. Users can add activated AMT devices to access device information and device management functionality such as power control, remote keyboard-video-mouse (KVM) control, and more.

## Quick Start for Users

### 1. Download Console

1. Visit the [latest release page](https://github.com/device-management-toolkit/console/releases/latest)
2. Download the appropriate binary for your operating system and architecture from the **Assets** section

### 2. Run Console

#### Linux/macOS
```sh
# Navigate to your download directory
cd <path-to-your-download>

# Extract the archive (example for Linux x64)
tar -xzf console_linux_x64.tar.gz

# Make the binary executable
chmod +x console_linux_x64

# Run Console
./console_linux_x64
```

#### Windows
```cmd
# Simply double-click the downloaded executable to run
console_windows_x64.exe
```

Or run from Command Prompt:
```cmd
console_windows_x64.exe
```


### 3. First Run Setup

On first startup, you'll see:
```
Warning: Key Not Found, Generate new key? Y/N
```

**Simply type `Y` and press Enter** - this generates the necessary encryption keys for secure operation.


> **Linux Users**: If you encounter `"Object does not exist at path '/'"` after answering 'Y', this indicates your system lacks a keychain service. Install a keychain manager (like `seahorse`) and restart Console binary.

---

## Docker Compose Deployment

Run Console end-to-end (app + database + vault) in containers. Two database backends are supported via Docker Compose profiles: **PostgreSQL** (default) and **MongoDB**.

### Prerequisites

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (Windows/macOS) or Docker Engine + Compose plugin (Linux)
- [Node.js 22.x](https://nodejs.org/) (only required if you want to embed the Web UI in the image)
- A clone of [`device-management-toolkit/sample-web-ui`](https://github.com/device-management-toolkit/sample-web-ui) next to this repo (only required for the embedded UI)
- A `console.local` hostname mapped to `127.0.0.1` on the machine that opens the browser. The compose `extra_hosts` entry handles the in-container side; the host-side mapping is your responsibility.

#### Add the `console.local` host entry

Why this is needed: the Go binary uses a single `HTTP_HOST` value both to bind the HTTP server inside the container *and* to inject the API base URL into `main.js`. The browser therefore needs to reach the same hostname the binary was told about. `console.local` is a hostname that resolves to `0.0.0.0` inside the container (binds on all interfaces, reachable through Docker port forwarding) and to `127.0.0.1` on the host (browser-friendly).

**Windows** (run PowerShell as **Administrator**):
```powershell
Add-Content -Path C:\Windows\System32\drivers\etc\hosts -Value "`n127.0.0.1`tconsole.local"
```

**Linux/macOS**:
```sh
echo "127.0.0.1 console.local" | sudo tee -a /etc/hosts
```

### 1. Build the embedded Web UI (optional but recommended)

Skip this step if you want a headless build (API only, no SPA). For an embedded UI, mirror what CI does in `.github/workflows/release.yml`:

```sh
# From a sibling directory of the console repo
git clone https://github.com/device-management-toolkit/sample-web-ui
cd sample-web-ui

npm ci
npm run build-enterprise        # produces ./ui/browser/

# Copy the built assets into the console repo's embed directory
# (the contents of ui/browser/, not the folder itself)
```

**Windows PowerShell:**
```powershell
Copy-Item -Recurse -Force ..\sample-web-ui\ui\browser\* `
  ..\console\internal\controller\httpapi\ui\
```

**Linux/macOS:**
```sh
cp -r ../sample-web-ui/ui/browser/. ../console/internal/controller/httpapi/ui/
```

Verify `internal/controller/httpapi/ui/` contains at minimum `index.html`, `main.js`, `polyfills.js`, `styles.css`, and the `assets/` and `media/` subfolders.

> **Headless build instead?** Build with the `noui` tag (`docker build --build-arg BUILD_TAGS=noui ...`) and skip this step entirely.

### 2. Choose your database backend

The compose file has two database services. PostgreSQL runs by default; MongoDB is gated behind the `mongo` Compose profile.

#### Option A — PostgreSQL (default)

```sh
docker compose up -d --build
```

This starts `postgres`, `vault`, and `app`. The app uses `DB_PROVIDER=postgres` and `DB_URL=postgres://postgresadmin:admin123@postgres:5432/rpsdb` by default.

#### Option B — MongoDB

```sh
# Bash / Linux / macOS
DB_PROVIDER=mongo \
DB_URL='mongodb://mongoadmin:admin123@mongo:27017/consoledb?authSource=admin' \
  docker compose --profile mongo up -d --build
```

```powershell
# Windows PowerShell
$env:DB_PROVIDER = "mongo"
$env:DB_URL = "mongodb://mongoadmin:admin123@mongo:27017/consoledb?authSource=admin"
docker compose --profile mongo up -d --build
```

The `--profile mongo` flag activates the `mongo` service (it has `profiles: [mongo]` in `docker-compose.yml`); without it the mongo container is not started. The app waits for `mongo` to become healthy before starting.

### 3. Open the UI

```
http://console.local:8181
```

> **Don't use `http://localhost:8181`** — the page loads, but the API URL embedded in `main.js` is `http://console.local:8181`, so the browser will block XHRs as cross-origin. Use the same hostname both places.

### 4. Stop and clean up

```sh
docker compose down              # stop containers, keep volumes (pg-data, mongo-data)
docker compose down -v           # also drop volumes (wipes the database)
docker compose --profile mongo down -v   # include mongo profile when tearing it down
```

### Troubleshooting

#### `UI assets not embedded; skipping browser launch` in startup logs

`internal/controller/httpapi/ui/` is empty (only `.gitkeep`). Re-run step 1 — make sure you copied the **contents** of `sample-web-ui/ui/browser/` (note the trailing `*` or `/.`), not the `browser` folder itself. The file `internal/controller/httpapi/ui/index.html` must exist directly under `ui/`.

#### Browser console: `Failed to execute 'open' on XMLHttpRequest: Invalid URL` or `net::ERR_ADDRESS_INVALID`

The `##CONSOLE_SERVER_API##` placeholder was injected with an empty or `0.0.0.0` host. Confirm `docker-compose.yml` sets `HTTP_HOST: "console.local"` (not `""` or `"0.0.0.0"`) for the `app` service, and that you opened the UI at `http://console.local:8181`. If `console.local` doesn't resolve, redo the hosts-file step in the prerequisites.

#### `Dirty database version <ts>. Fix and force version.`

A migration ran partially, then was retried and failed (e.g., `column "..." already exists`). Clear the dirty flag without losing data:

```sh
docker exec -it postgres psql -U postgresadmin -d rpsdb -c \
  "UPDATE schema_migrations SET dirty = false WHERE version = <ts>;"
docker compose restart app
```

Replace `<ts>` with the timestamp shown in the error. If data is disposable, `docker compose down -v && docker compose up -d --build` is faster.

---

## For Developers

### Development Environment

You’ll run two components during development:

- **Console** – the backend service (Go Service)  
- **Web UI** – the frontend (for the Angular Web UI)

Local development can be done on **Linux**, **macOS**, and **Windows**. On Windows, WSL is recommended if you plan to use `make`, but it’s not required for running Console directly.

### Prerequisites

Before you begin, ensure you have:
- [Go 1.24+](https://go.dev/dl/)
- [Git](https://git-scm.com/downloads)
- [Node.js & npm](https://nodejs.org/) (for the Web UI)
- [Docker & Docker Compose](https://docs.docker.com/get-docker/) (optional, for PostgreSQL)

### 1. Clone and Configure Console

```sh
# Clone Console
git clone https://github.com/device-management-toolkit/console.git
cd console

# Copy env template
cp .env.example .env
```

Edit `.env` as needed. For local dev, set:

```sh
GIN_MODE=debug
# DB_URL=postgres://postgresadmin:admin123@localhost:5432/rpsdb  # uncomment for Postgres
```

### 2. Configure Backend for Development

For local development with a separate UI, configure `config/config.yml` to point to your frontend:

```yaml
ui:
  externalUrl: "http://localhost:4200"
```

This tells the backend to redirect UI requests to the separately running frontend application.

### 3. Running the Backend

> **Important**: For development, use the `console-noui` binary or build with the `noui` tag. This allows the backend to work with a separately running frontend without embedding UI files.

#### Option A: SQLite (default, easiest)

```sh
# Install dependencies
go mod tidy && go mod download

# Run Console with noui tag
go run -tags=noui ./cmd/app/main.go

# OR use the pre-built noui binary
./bin/console-noui -config ./config/config.yml
```

**First run**: When prompted with `Warning: Key Not Found, Generate new key? Y/N`, type `Y` and press Enter.

> **Custom Config**: You can specify a custom configuration file:
> ```sh
> go run -tags=noui ./cmd/app/main.go --config "/absolute/path/to/config.yml"
> ```

> **Database Location**: SQLite database is automatically created at:
> - Linux/macOS: `~/.config/device-management-toolkit/console.db`
> - Windows: `%APPDATA%\device-management-toolkit\console.db`

#### Option B: PostgreSQL

```sh
# Start Postgres via Docker
make compose-up

# Run Console with database migrations (noui)
make run-noui
```

This will use the `DB_URL` you configured in `.env`.

### 4. Build Options

Console supports multiple build configurations optimized for different use cases:

#### Default Build
- Full build with embedded web UI
- Ideal for development and single-instance deployments

#### `noui` Build (Headless/API-only)
- Excludes embedded web UI from binary
- **Reduces binary size** by ~30MB
- All API endpoints remain fully functional
- Health checks, metrics, and API docs still available
- Optional: Configure `ui.externalUrl` in `config.yml` to redirect UI requests to separately hosted frontend
- Ideal for microservice architectures or when using a separate UI deployment

**Configuration for headless builds:**

Edit `config.yml`:
```yaml
ui:
  # Redirect UI requests to external frontend (optional)
  # If empty: UI requests return 404
  # If set: UI requests redirect to this URL
  externalUrl: "https://your-ui-domain.com"
```

Or use environment variable:
```sh
UI_EXTERNAL_URL=https://your-ui-domain.com ./console-noui
```

**Build commands:**
```sh
# Default build (with UI) for current platform
make build

# Headless build (no UI) for current platform
make build-noui

# Cross-compile for all platforms (Linux, Windows, macOS)
# Produces binaries in dist/ directory for distribution
make build-all-platforms
```

**Manual build examples:**
```sh
# Build for current platform
go build -o console ./cmd/app
go build -tags=noui -o console-noui ./cmd/app

# Cross-compilation examples (CGO_ENABLED=0 produces static binaries)
# Linux
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o console-linux ./cmd/app
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags=noui -o console-linux-headless ./cmd/app

# Windows
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o console.exe ./cmd/app
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -tags=noui -o console-headless.exe ./cmd/app

# macOS
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o console-macos ./cmd/app
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -tags=noui -o console-macos-headless ./cmd/app
```

> **Note**: With `CGO_ENABLED=0`, Go produces statically-linked binaries that are cross-platform compatible. You can build binaries for any target platform from any development machine.

### 5. Running the Frontend

```sh
# Clone Sample Web UI
git clone https://github.com/device-management-toolkit/sample-web-ui
cd sample-web-ui

# Install dependencies
npm install

# Build and run
npm run enterprise
```

Check the output of `npm run enterprise` to verify the port (default is 4200), then open the UI in your browser at:

```
http://localhost:4200
```

## Architecture and Documentation

Before contributing code changes, familiarize yourself with:

- [Console Architecture Overview](https://github.com/device-management-toolkit/console/wiki/Architecture-Overview)
- [Console Data Storage Documentation](https://github.com/device-management-toolkit/console/wiki/Console-Data-Storage)

### OpenAPI Documentation

Console exposes the OpenAPI specification two ways:

1. **Live HTTP endpoint** (always available when the server runs):
   - `http://localhost:8181/api/openapi.json`
2. **Regenerate `doc/openapi.json` on disk** (used by CI to publish to SwaggerHub):
   ```sh
   make openapi
   ```
   Equivalent to `go run ./cmd/openapi-gen` — a standalone binary that writes `doc/openapi.json` and exits without starting the server.
3. **To add API Documentation**: Check wiki `https://github.com/device-management-toolkit/console/wiki/API-Documentation-to-Console`

## Dev tips for passing CI Checks

- Install gofumpt `go install mvdan.cc/gofumpt@latest` (replaces gofmt)
- Install gci `go install github.com/daixiang0/gci@latest` (organizes imports)
- Ensure code is formatted correctly with `gofumpt -l -w -extra ./`
- Ensure all unit tests pass with `go test ./...`
- Ensure code has been linted with:
  - Windows: `docker run --rm -v ${pwd}:/app -w /app golangci/golangci-lint:latest golangci-lint run --config=./.golangci.yml -v`
  - Unix: `docker run --rm -v .:/app -w /app golangci/golangci-lint:latest golangci-lint run --config=./.golangci.yml -v`

## Fuzz Testing

Console includes Go fuzz tests across multiple packages and targets. For full coverage details, seed corpus strategy, and CI guidance, see wiki: [Fuzz Testing](https://github.com/device-management-toolkit/console/wiki/Fuzz-Testing)

### Prerequisite

The Makefile includes `.env`, so ensure it exists before running make commands:

```sh
cp .env.example .env
```

### List all fuzz targets

```sh
make fuzz-list
```

Output format:

```text
./internal/usecase/devices FuzzParseInterval
./internal/entity/dto/v1 FuzzDeviceJSONProcessing
...
```

### Run a single fuzz target (recommended local workflow)

Go fuzzing runs one fuzz target at a time. Use `fuzz-one` for focused debugging and development:

```sh
make fuzz-one PKG=./internal/usecase/devices TARGET=FuzzParseInterval FUZZTIME=30s
```

### Run a quick all-target smoke pass

Runs each fuzz target once (`-fuzztime=1x`):

```sh
make fuzz-smoke
```

### Run all fuzz targets with time budget per target

```sh
make fuzz-all FUZZTIME=2m
```

This executes all discovered fuzz targets sequentially and is suitable for scheduled CI jobs.


## Additional Resources

- For detailed documentation and Getting Started, [visit the docs site](https://device-management-toolkit.github.io/docs).

<!-- - Looking to contribute? [Find more information here about contribution guidelines and practices](.\CONTRIBUTING.md). -->

- Find a bug? Or have ideas for new features? [Open a new Issue](https://github.com/device-management-toolkit/console/issues).

- Need additional support or want to get the latest news and events about Device Management Toolkit? Connect with the team directly through Discord.

    [![Discord Banner 1](https://discordapp.com/api/guilds/1063200098680582154/widget.png?style=banner2)](https://discord.gg/DKHeUNEWVH)
