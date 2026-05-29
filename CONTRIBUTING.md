# Contributing to Console

Thanks for helping out! This guide covers the conventions our automated tooling and reviewers expect. For deeper architectural and implementation guidance, see [CLAUDE.md](./CLAUDE.md) — it's the canonical reference for both humans and AI coding assistants.

## <a name="commit"></a> Commit Message Guidelines

We have precise rules over how our git commit messages should be formatted. This leads to more readable messages that are easy to follow when looking through the project history, and it drives our **semantic-release** automation (`.releaserc.json`) — your commit type determines whether a release is cut and at what level.

### Commit Message Format

Each commit message consists of a **header**, a **body** and a **footer**. The header has a special format that includes a **type**, a **scope** and a **subject**:

```
<type>(<scope>): <subject>
<BLANK LINE>
<body>
<BLANK LINE>
<footer>
```

The **header** with **type** is mandatory. The **scope** of the header is optional as far as the automated PR checks are concerned (the `scope-enum` rule in `.github/commitlint.config.cjs` is unrestricted), but reviewers **may request** you provide an applicable scope.

Subject + body lines ≤72 characters where practical; commitlint enforces `body-max-line-length: 200`. The shorter limit keeps messages readable on GitHub and in `git log`.

The footer should contain a reference to a GitHub issue (e.g. `#1234`) using the [GitHub closing-keyword syntax](https://docs.github.com/en/issues/tracking-your-work-with-issues/linking-a-pull-request-to-an-issue) so the issue auto-closes when the PR merges.

Example 1:

```
feat(api): add /api/v2/amt/features endpoint

Mirrors the v1 shape but returns the structured features payload the new
UI consumes. v1 remains untouched.

Closes: #1234
```

Example 2:

```
fix(db): retry mongo connect on transient server-selection errors

The 30s startup budget was being consumed by a single 27s server
selection attempt, leaving no headroom for index ensure. Backoff
retries are now bounded by the same 30s ceiling.

Fixes: #4567
```

### Revert

If the commit reverts a previous commit, it should begin with `revert: `, followed by the header of the reverted commit. In the body it should say: `This reverts commit <hash>.`, where the hash is the SHA of the commit being reverted.

### Type

Must be one of the following. The semantic-release impact is shown for each:

- **feat**: A new feature — cuts a **minor** release.
- **fix**: A bug fix — cuts a **patch** release.
- **perf**: A code change that improves performance — cuts a **patch** release.
- **chore**: Maintenance work — cuts a **patch** release (configured in `.releaserc.json`).
- **docs**: Documentation only changes — no release.
- **style**: Changes that do not affect the meaning of the code (white-space, formatting, etc) — no release.
- **refactor**: A code change that neither fixes a bug nor adds a feature — no release.
- **test**: Adding missing tests or correcting existing tests — no release.
- **build**: Changes that affect the CI/CD pipeline or build system or external dependencies — no release.
- **ci**: Changes to CI configuration — no release.
- **revert**: Reverts a previous commit.

Any commit body containing `BREAKING CHANGE:` cuts a **major** release regardless of type. Avoid this on `/api/v1/*` — see [CLAUDE.md](./CLAUDE.md#implementation-guidelines-non-negotiable) for why.

### Scope

Common scopes in this repo (not exhaustive; the linter does not enforce a fixed list):

- **api**: A change or addition to REST functionality (`internal/controller/httpapi/`)
- **apf**: A change or addition to AMT Port Forwarding handling
- **cira**: A change or addition to Client Initiated Remote Access functionality (`internal/controller/tcp/cira/`)
- **config**: A change or addition to service configuration (`config/`)
- **db**: A change or addition to repository / migration / driver code
- **deps**: A change or addition to dependencies (primarily used by dependabot)
- **deps-dev**: A change or addition to developer dependencies
- **docker**: A change or addition to Dockerfile or compose
- **events**: A change or addition to eventing from the service
- **gh-actions**: A change or addition to GitHub Actions
- **health**: A change or addition to health checks
- **redir**: A change or addition to KVM/SOL/IDER redirection
- **secrets**: A change or addition to Vault / keyring / encryption flows
- **tray**: A change or addition to the system-tray launcher (`pkg/tray`, `cmd/app/tray*.go`)
- **ui**: A change or addition to the embedded UI handler or `noui` plumbing
- **utils**: A change or addition to utility helpers
- **wsman**: A change or addition to the WSMAN client wrapper (`internal/usecase/devices/wsman`)
- _no scope_: If no scope is provided, it is assumed the PR does not apply to the above scopes

### Body

Just as in the **subject**, use the imperative, present tense: "change" not "changed" nor "changes".
Here is detailed guideline on how to write the body of the commit message ([Reference](https://chris.beams.io/posts/git-commit/)):

```
More detailed explanatory text, if necessary. Wrap it to about 72
characters or so. In some contexts, the first line is treated as the
subject of the commit and the rest of the text as the body. The
blank line separating the summary from the body is critical (unless
you omit the body entirely); various tools like `log`, `shortlog`
and `rebase` can get confused if you run the two together.

Explain the problem that this commit is solving. Focus on why you
are making this change as opposed to how (the code explains that).
Are there side effects or other unintuitive consequences of this
change? Here's the place to explain them.

Further paragraphs come after blank lines.

 - Bullet points are okay, too

 - Typically a hyphen or asterisk is used for the bullet, preceded
   by a single space, with blank lines in between, but conventions
   vary here
```

### Footer

The footer should contain a reference to the GitHub issue this commit **Closes**, **Fixes**, or **Resolves** (e.g. `Closes: #1234`). See [GitHub's closing-keyword syntax](https://docs.github.com/en/issues/tracking-your-work-with-issues/linking-a-pull-request-to-an-issue) for the keywords that trigger auto-close on merge.

The footer should also contain any information about **Breaking Changes**.

**Breaking Changes** should start with the word `BREAKING CHANGE:` with a space or two newlines. **Don't introduce breaking changes to `/api/v1/*`** — that surface is the MPS+RPS migration contract. Land additive changes under `/api/v2/*` instead.

## Pull Request practices

- **Keep PRs small and scoped to one concern.** A focused 50-line diff gets reviewed and merged quickly; a 500-line "while I was in there" diff stalls. Unrelated cleanups belong in their own PRs.
- **Work in incremental phases.** When a feature needs prerequisite plumbing, land that first as `refactor:` / `test:` / `build:` so it ships invisibly, then ship the user-visible behaviour as a single `feat:` PR that triggers the release. Each PR should leave `main` in a working state.
- **PR title follows the same guidelines as the commit header** — `commitlint` validates it via the `pull-request-name-linter-action`.
- **PR author is responsible for merging** after review and CI pass.
- **Preserve linear history.** PR authors choose `Rebase and merge` or `Squash and merge`, whichever fits the change.
- **Update the OpenAPI spec when you touch the REST API.** Run `go run ./cmd/openapi-gen` and commit `doc/openapi.json` alongside your handler changes.
- **Update the Postman collections when you touch the REST API.** `integration-test/collections/console_mps_apis.postman_collection.json` (device/management surface) and `console_rps_apis.postman_collection.json` (activation/config surface) are what integrators and QA test against; bump `console_environment.postman_environment.json` if your change introduces new variables. The OpenAPI spec and the Postman collections are the API contract — drift in either is treated as a bug.
- **Regenerate mocks when you touch an interface.** `make mock` rewrites `internal/mocks/`; commit the result.

## Before pushing

Recommended local checks before pushing. CI enforces equivalent behaviour (formatting via `gofmt -s`, linting via `reviewdog/action-golangci-lint` against the same `.golangci.yml`), but binary versions aren't pinned identically — `gofumpt` is a strict superset of `gofmt -s`, and the Dockerized `golangci-lint` may differ in revision from `reviewdog`'s. Everything below runs natively on Linux, macOS, and Windows (WSL is **not** required).

```sh
# bash/zsh (Linux, macOS)
gofumpt -l -w -extra ./                                          # no diff
go vet ./...                                                     # clean
docker run --rm -v .:/app -w /app golangci/golangci-lint:latest golangci-lint run -v --fix
                                                                 # no remaining diagnostics
go test -race -count=1 ./...                                     # all pass
go run ./cmd/openapi-gen                                         # only if you touched routes; commit doc/openapi.json
make mock                                                        # only if you touched a mocked interface; commit internal/mocks/
```

On Windows PowerShell, swap `-v .:/app` for `-v ${pwd}:/app` and replace any inline `VAR=value cmd` with `$env:VAR="value"; cmd`. Everything else (Go, Docker, mockgen) runs natively — WSL is **not** required.

Install `gofumpt` once:

```sh
go install mvdan.cc/gofumpt@latest
```

For deeper architectural and implementation guidance — pluggable storage backends, the CIRA/APF byte flow, WSMAN constraints, encryption rules, the v1/v2 API split — see [CLAUDE.md](./CLAUDE.md).
