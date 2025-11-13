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

### 2. Running the Backend

#### Option A: SQLite (default, easiest)

```sh
# Install dependencies
go mod tidy && go mod download

# Run Console
go run ./cmd/app/main.go
```

**First run**: When prompted with `Warning: Key Not Found, Generate new key? Y/N`, type `Y` and press Enter.

> **Custom Config**: You can specify a custom configuration file:
> ```sh
> go run ./cmd/app/main.go --config "/absolute/path/to/config.yml"
> ```

> **Database Location**: SQLite database is automatically created at:
> - Linux/macOS: `~/.config/device-management-toolkit/console.db`
> - Windows: `%APPDATA%\device-management-toolkit\console.db`

#### Option B: PostgreSQL

```sh
# Start Postgres via Docker
make compose-up

# Run Console with database migrations
make run
```

This will use the `DB_URL` you configured in `.env`.

### 4. Running the Frontend

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

Console automatically generates OpenAPI documentation when running in debug mode:

1. **Enable Debug Mode**: Set `GIN_MODE=debug` in your `.env` file
2. **Run Console**: Start the application with `go run ./cmd/app/main.go`
3. **Access OpenAPI Spec**: The OpenAPI specification is automatically generated and available at:
   - JSON format: `http://localhost:8181/openapi.json`
   - The spec is also written to `doc/openapi.json` in your project directory
4. **To add API Documentation**: Check wiki `https://github.com/device-management-toolkit/console/wiki/API-Documentation-to-Console`

> **Note**: OpenAPI generation only occurs in debug mode. Production builds will not expose these endpoints. 

## Dev tips for passing CI Checks

- Install gofumpt `go install mvdan.cc/gofumpt@latest` (replaces gofmt)
- Install gci `go install github.com/daixiang0/gci@latest` (organizes imports)
- Ensure code is formatted correctly with `gofumpt -l -w -extra ./`
- Ensure all unit tests pass with `go test ./...`
- Ensure code has been linted with:
  - Windows: `docker run --rm -v ${pwd}:/app -w /app golangci/golangci-lint:latest golangci-lint run -v`
  - Unix: `docker run --rm -v .:/app -w /app golangci/golangci-lint:latest golangci-lint run -v`


## Additional Resources

- For detailed documentation and Getting Started, [visit the docs site](https://device-management-toolkit.github.io/docs).

<!-- - Looking to contribute? [Find more information here about contribution guidelines and practices](.\CONTRIBUTING.md). -->

- Find a bug? Or have ideas for new features? [Open a new Issue](https://github.com/device-management-toolkit/console/issues).

- Need additional support or want to get the latest news and events about Device Management Toolkit? Connect with the team directly through Discord.

    [![Discord Banner 1](https://discordapp.com/api/guilds/1063200098680582154/widget.png?style=banner2)](https://discord.gg/DKHeUNEWVH)
