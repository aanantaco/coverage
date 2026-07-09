# Contributing

Thanks for your interest in improving `coverage`. This is a small,
single-purpose tool; the bar is "does one thing well, with no surprises."

## Prerequisites

- Go **1.26** or newer (`go.mod` pins `go 1.26.0`).
- No other toolchain is required to build or test. Markdown/shell linting in CI
  use `markdownlint-cli2` (via `npx`) and `shellcheck`.

## Build, test, lint

```bash
go build ./...                # build
go test ./...                 # run the test suite
go test ./... -cover          # with coverage
gofmt -l .                    # must print nothing
go vet ./...                  # must pass

npx --yes markdownlint-cli2@0.18.1   # lint Markdown (same version as CI)
```

Try the tool against its own coverage:

```bash
go run ./cmd/coverage --input ./some-dir-of-artifacts --output -
go run ./cmd/coverage version
```

## Conventions

A few things this repo holds to — please match them:

- **Standard library only in tests.** No `testify` or other assertion
  libraries; table tests and plain `if` checks are the norm.
- **Minimal dependencies.** The binary has a single external dependency
  (`github.com/goccy/go-yaml`) with no transitive modules. New dependencies are
  a hard sell — prefer the standard library, and if you must vendor behavior,
  port it in-repo with attribution in `THIRD_PARTY_NOTICES.md`.
- **No panics in library code.** Return errors; `cmd/coverage` decides exit
  codes.
- **Golden tests are load-bearing.** Markdown output is locked by golden files
  under `internal/render/testdata/`. If a change intentionally alters output,
  regenerate them with `CAPTURE_GOLDEN=1 go test ./internal/render/...` and
  review the diff.
- **Keep formatting clean.** `gofmt` and `go vet` must pass; CI enforces both.
- **Update `CHANGELOG.md`** for any user-facing change (skip it for
  test-only or purely internal changes).

## Pull requests

- Keep PRs focused; one logical change per PR.
- Make sure `go test ./...`, `gofmt -l .`, `go vet ./...`, and the Markdown lint
  pass locally before opening.
- Describe the change and how you verified it. If it changes behavior, say what
  the old and new behavior are.

## Reporting bugs & security issues

- **Bugs / features:** open a GitHub issue with a minimal repro (a small
  Cobertura/JUnit snippet is ideal).
- **Security vulnerabilities:** please do **not** open a public issue — see
  [SECURITY.md](SECURITY.md).
