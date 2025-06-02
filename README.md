![Build](https://img.shields.io/github/actions/workflow/status/device-management-toolkit/console/ci.yml?style=for-the-badge&label=Build&logo=github)
![Codecov](https://img.shields.io/codecov/c/github/device-management-toolkit/console?style=for-the-badge&logo=codecov)
[![OSSF-Scorecard Score](https://img.shields.io/ossf-scorecard/github.com/device-management-toolkit/console?style=for-the-badge&label=OSSF%20Score)](https://api.securityscorecards.dev/projects/github.com/device-management-toolkit/console)
[![Discord](https://img.shields.io/discord/1063200098680582154?style=for-the-badge&label=Discord&logo=discord&logoColor=white&labelColor=%235865F2&link=https%3A%2F%2Fdiscord.gg%2FDKHeUNEWVH)](https://discord.gg/DKHeUNEWVH)
# Console


> Disclaimer: Production viable releases are tagged and listed under 'Releases'. Console is under development. **The current available tags for download are Alpha version code and should not be used in production.** For these Alpha tags, certain features may not function yet, visual look and feel may change, or bugs/errors may occur. Follow along our [Feature Backlog for future releases and feature updates](https://github.com/orgs/open-amt-cloud-toolkit/projects/10).

## Overview

Console is an application that provides a 1:1, direct connection for AMT devices for use in an enterprise environment. Users can add activated AMT devices to access device information and device management functionality such as power control, remote keyboard-video-mouse (KVM) control, and more.

<!-- **For detailed documentation** about Getting Started or other features of the Open AMT Cloud Toolkit, see the [docs](https://open-amt-cloud-toolkit.github.io/docs). -->

<br>

## Quick start 

### For Users

1. Find the latest release of Console under [Github Releases](https://github.com/device-management-toolkit/console/releases/latest).

2. Download the appropriate binary assets for your OS and Architecture under the *Assets* dropdown section.

3. Run Console.

### For Developers

Local development (in Linux or WSL):

To start the service with Postgres: 

```sh
# Postgres
$ make compose-up
# Run app with migrations
$ make run
```

Download and check out the sample-web-ui:
```
git clone https://github.com/open-amt-cloud-toolkit/sample-web-ui
```

Ensure that the environment file has cloud set to `false` and that the URLs for RPS and MPS are pointing to where you have `Console` running. The default is `http://localhost:8181`. Follow the instructions for launching and running the UI in the sample-web-ui readme.




## Dev tips for passing CI Checks

- Install gofumpt `go install mvdan.cc/gofumpt@latest` (replaces gofmt)
- Install gci `go install github.com/daixiang0/gci@latest` (organizes imports)
- Ensure code is formatted correctly with `gofumpt -l -w -extra ./`
- Ensure all unit tests pass with `go test ./...`
- Ensure code has been linted with:
  - Windows: `docker run --rm -v ${pwd}:/app -w /app golangci/golangci-lint:latest golangci-lint run -v`
  - Unix: `docker run --rm -v .:/app -w /app golangci/golangci-lint:latest golangci-lint run -v`


## Additional Resources

- For detailed documentation and Getting Started, [visit the docs site](https://open-amt-cloud-toolkit.github.io/docs).

<!-- - Looking to contribute? [Find more information here about contribution guidelines and practices](.\CONTRIBUTING.md). -->

- Find a bug? Or have ideas for new features? [Open a new Issue](https://github.com/device-management-toolkit/console/issues).

- Need additional support or want to get the latest news and events about Open AMT? Connect with the team directly through Discord.

    [![Discord Banner 1](https://discordapp.com/api/guilds/1063200098680582154/widget.png?style=banner2)](https://discord.gg/DKHeUNEWVH)
