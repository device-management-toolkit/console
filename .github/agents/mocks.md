# Regenerating mocks

Read this whenever you add, remove, or change a method on any `Repository`,
`Feature`, or `WSMAN` interface. Stale mocks are the most common cause of
"tests pass on my machine but fail in CI".

## The command

```sh
make mock
```

Run it after the interface change, then commit the regenerated files together
with your change in the same commit.

## What it does

`internal/mocks/*` is generated from the `interfaces.go` file in each use-case
package. The `mock` target runs 13 separate `mockgen -source ... -mock_names ...`
invocations. Each one uses specific alias names that are easy to get wrong.

**Do not run `mockgen` by hand and do not edit files in `internal/mocks/`.**
Anything you write there is overwritten the next time someone runs `make mock`.

## Adding a new mocked interface

If you create a brand-new interface that needs a mock, add another `mockgen`
line to the `mock` target in the `Makefile`, following the pattern of the
existing lines. Then run `make mock` and commit both the `Makefile` change and
the generated file.

## Checking your work

```sh
make mock
git status --short internal/mocks/    # should show your expected changes
go test -race -count=1 ./...
```

If `git status` shows changes you did not expect, someone forgot to run
`make mock` in an earlier PR. Commit those too and say so in your PR
description.
