# OpenAPI spec

Read this whenever you add, remove, or change a route under
`internal/controller/httpapi/v1/` or `v2/`.

## Two files describe every endpoint

| File | Role |
|---|---|
| `internal/controller/httpapi/v{1,2}/*.go` | The Gin handler. This serves real traffic. |
| `internal/controller/openapi/*.go` | The Fuego declaration. This is the schema integrators and SwaggerHub see. |

**When you change a route, you must change both.** The Gin handler is what
runs; the Fuego declaration is what people read. If they disagree, integrators
build against a contract the server does not honour.

## Do NOT commit doc/openapi.json

`doc/openapi.json` is **ignored by git** (see `.gitignore`, the
`**/doc/openapi.json` line). It is build output, not source. `git add
doc/openapi.json` will fail with "paths are ignored by .gitignore".

The release workflow generates and publishes it for you. In
`.github/workflows/release.yml`, on a release build, CI:

1. Checks whether any file under `internal/controller/openapi/` changed.
2. If yes, runs `go run ./cmd/openapi-gen` to write `doc/openapi.json`.
3. Uploads that file to SwaggerHub.

CI keys off changes to **`internal/controller/openapi/`** — not off a
committed JSON file. Editing the Fuego declaration is what triggers
publication. That is the only thing you need to commit.

## Generating the spec locally

Useful for checking your Fuego declaration produces the schema you expect:

```sh
go run ./cmd/openapi-gen      # writes doc/openapi.json locally (git ignores it)
make openapi                  # same thing
```

You can also read the live spec from a running server:

```
GET /api/openapi.json
```

## Checklist for a route change

1. Edit the Gin handler in `internal/controller/httpapi/v{1,2}/`.
2. Edit the matching Fuego declaration in `internal/controller/openapi/`.
3. Update the Postman collections in `integration-test/collections/`.
4. Run `go run ./cmd/openapi-gen` and open `doc/openapi.json` to confirm the
   schema looks right. Do not commit this file.
5. Commit steps 1, 2, and 3.
