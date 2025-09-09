![Build](https://img.shields.io/github/actions/workflow/status/device-management-toolkit/console/ci.yml?style=for-the-badge&label=Build&logo=github)
![Codecov](https://img.shields.io/codecov/c/github/device-management-toolkit/console?style=for-the-badge&logo=codecov)
[![OSSF-Scorecard Score](https://img.shields.io/ossf-scorecard/github.com/device-management-toolkit/console?style=for-the-badge&label=OSSF%20Score)](https://api.securityscorecards.dev/projects/github.com/device-management-toolkit/console)
[![Discord](https://img.shields.io/discord/1063200098680582154?style=for-the-badge&label=Discord&logo=discord&logoColor=white&labelColor=%235865F2&link=https%3A%2F%2Fdiscord.gg%2FDKHeUNEWVH)](https://discord.gg/DKHeUNEWVH)

# Console

> Disclaimer: Production viable releases are tagged and listed under 'Releases'. Console is under development. **The current available tags for download are Alpha version code and should not be used in production.** For these Alpha tags, certain features may not function yet, visual look and feel may change, or bugs/errors may occur. Follow along our [Feature Backlog for future releases and feature updates](https://github.com/orgs/device-management-toolkit/projects/10).

## Overview

Console is an application that provides a 1:1, direct connection for AMT devices for use in an enterprise environment. Users can add activated AMT devices to access device information and device management functionality such as power control, remote keyboard-video-mouse (KVM) control, and more.

<br>

## Quick start 

### For Users

1. Find the latest release of Console under [Github Releases](https://github.com/device-management-toolkit/console/releases/latest).

2. Download the appropriate binary assets for your OS and Architecture under the *Assets* dropdown section.

3. Make sure you have enough permission to run the application. For example, 

```sh
# Extract the archive
tar -xzf console_linux_x64.tar.gz

# Make executable
chmod +x console_linux_x64
```

**Important**: You'll see `"Warning: Key Not Found, Generate new key? Y/N"` on first run - this is normal. Simply type `Y` and press Enter.

**Linux Users**: If you see  `"Object does not exist at path '/' " after answering 'Y'` indicates lack of a built-in keychain. Manually install any keychain and restart the application to use the system keychain.

4. Run Console.

### For Developers

Local development (in Linux or WSL):

#### Environment Setup:

1. Clone the repository:

```sh
git clone https://github.com/device-management-toolkit/console.git
cd console
```

2. Copy the environment template

```sh
cp .env.example .env
```

3. Change the GIN_MODE to debug in the environment template

```sh
DISABLE_SWAGGER_HTTP_HANDLER=true
GIN_MODE=debug
# DB_URL=postgres://postgresadmin:admin123@localhost:5432/rpsdb  # Commented out for SQLite
# OAUTH CONFIGURATION
AUTH_CLIENT_ID=""
AUTH_ISSUER=""
```

#### Running Options

1. Running with SQLite (Default - Recommended for Development)

```sh
# Install dependencies and run
go mod tidy && go mod download

# Run the application directly
go run ./cmd/app/main.go
```

**Important**: When prompted with `"Generate new key? Y/N"`, type `Y` and press Enter.
The SQLite database will be automatically created at `~/.config/device-management-toolkit/console.db`.

2. Running with PostgreSQL

```sh
# Start PostgreSQL with Docker
$ make compose-up
# Run app with migrations
$ make run
```

3. Sample Web UI

Download and check out the sample-web-ui:
```
git clone https://github.com/device-management-toolkit/sample-web-ui
```

Ensure that the environment file has cloud set to `false` and that the URLs for RPS and MPS are pointing to where you have `Console` running. The default is `http://localhost:8181`. Follow the instructions for launching and running the UI in the sample-web-ui readme.

**Note**: To contribute to code base, make sure you go through the [Console Architecture](https://github.com/device-management-toolkit/console/wiki/Architecture-Overview).
For detailed information about database schema and data storage, see [Console Data Storage](https://github.com/device-management-toolkit/console/wiki/Console-Data-Storage)

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
