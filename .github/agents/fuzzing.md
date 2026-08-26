# Fuzz testing

Read this when you are adding a fuzz target, or when a maintainer asks you to
fuzz a parser or decoder. Fuzz tests are not part of the normal test loop —
`go test ./...` runs them only as ordinary unit tests, using their seed corpus.

**Fuzzing does not run in CI today.** No workflow under `.github/workflows/`
invokes any fuzz target. Run these locally when you touch parsing or decoding
code; nothing will catch a regression for you automatically.

## Where fuzz targets live

Next to the package they test, in a file named `*_fuzz_test.go`.

## Why the make targets exist

`go test` can only fuzz **one** target per invocation. If a package has three
fuzz targets, you must run `go test` three times. The `make fuzz-*` targets
find every target for you and loop over them.

```sh
make fuzz-list       # list every fuzz target in the repo
make fuzz-smoke      # run every target once (quick check before pushing)
make fuzz-all FUZZTIME=2m    # run every target for 2 minutes each
```

To fuzz one specific target:

```sh
make fuzz-one PKG=./internal/usecase/devices TARGET=FuzzParseInterval FUZZTIME=30s
```

The plain `go test` equivalent, if you need it:

```sh
go test -run=^$ -fuzz='^FuzzName$' -fuzztime=30s ./path/to/package
```

`-run=^$` means "run no normal tests" — without it, Go runs the whole unit
test suite before it starts fuzzing.

## Rules for new fuzz targets

- Name the file `<thing>_fuzz_test.go`.
- Call `t.Parallel()` in any `Test*` function in that file. Do **not** call it
  in the `Fuzz*` function itself — that one takes `*testing.F`, not `*testing.T`.
- Add seed inputs with `f.Add(...)` so the target still tests something useful
  during `make fuzz-smoke`.
- If fuzzing finds a crash, Go writes the failing input to
  `testdata/fuzz/<FuzzName>/`. Commit that file — it becomes a regression test.
