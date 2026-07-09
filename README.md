# coverage

[![CI](https://github.com/aanantaco/coverage/actions/workflows/ci.yml/badge.svg)](https://github.com/aanantaco/coverage/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/aanantaco/coverage.svg)](https://pkg.go.dev/github.com/aanantaco/coverage)
[![Go 1.26](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A single Go binary (`coverage`) that aggregates **Cobertura** coverage XML and
**JUnit** test-result XML from every project in a repo into one **Markdown
summary** for the CI run — with optional **regression detection** against a
baseline.

It is **language-agnostic**: any toolchain that emits Cobertura + JUnit works
(Go, TypeScript/JavaScript, Rust, Python, Java, C#), in any mix. Each project
produces two files: `coverage-<id>.xml` and `tests-<id>.xml`.

```markdown
## Test Coverage

### Summary

| Workspace | Tests | Lines | % | Branches | % |
|---|---|---|---|---|---|
| thingy | 412 | 1842 / 2310 | 79.7% | 215/310 | 69.4% |
| thinger | 175 | 3120 / 3680 | 84.8% | 480/620 | 77.4% |
| shared/widget | — | 320 / 400 | 80.0% | — | — |
| **Total** | **587** | 5282 / 6390 | **82.7%** | 695/930 | **74.7%** |
```

- **Tests** renders `—` when no JUnit artifact was uploaded (distinct from `0`).
- **Branches** renders `—` when a workspace reports no branch data.
- Totals are **recomputed from the leaf `<line>` elements** — the tool ignores
  the unreliable top-level totals emitters put on `<coverage>`.

## Supported languages & test frameworks

| Language | Coverage → Cobertura | Tests → JUnit | Guide |
|---|---|---|---|
| Go | `gocover-cobertura` | `gotestsum` | [GO.md](./docs/GO.md) |
| TypeScript / JS | Jest · Vitest · nyc | jest-junit · vitest · mocha-junit-reporter | [TYPESCRIPT.md](./docs/TYPESCRIPT.md) |
| Rust | `cargo-llvm-cov --cobertura` | `cargo-nextest` · `cargo2junit` | [RUST.md](./docs/RUST.md) |
| Python | `pytest --cov ... --cov-report=xml` | `pytest --junitxml` | [PYTHON.md](./docs/PYTHON.md) |
| Java | JaCoCo (→ Cobertura) | Surefire / Gradle | [JAVA.md](./docs/JAVA.md) |
| C# / .NET | coverlet (cobertura) | JunitXml.TestLogger | [CSHARP.md](./docs/CSHARP.md) |

Any other tool that produces Cobertura + JUnit works too — see [`docs/`](./docs/README.md).

New to it? **`coverage init`** detects your languages and scaffolds the workflow
wiring + config non-destructively — see [docs/INIT.md](./docs/INIT.md). Setting
this up with an AI assistant? Point it at [`llms.txt`](./llms.txt).

## Install

```bash
go install github.com/aanantaco/coverage/cmd/coverage@latest
```

Or use the composite GitHub Action, **pinned to a commit SHA** (recommended):

```yaml
# Pin to a reviewed commit SHA (latest main shown — check for a newer one).
- uses: aanantaco/coverage@4dde4d44807a0b7d29358bb9417768b4a7cc1960
```

The Action builds the tool from its own source at that SHA, so the pin selects
the exact tool version. (`go install …@<sha>` works the same way for the CLI.)

### Binaries

There are no version tags. Every merge to `main` runs the `release` job in
[`.github/workflows/ci.yml`](./.github/workflows/ci.yml), which cross-compiles
`coverage` (linux/macOS/Windows × amd64/arm64) with GoReleaser and uploads the
archives + `checksums.txt` as a workflow artifact named
`coverage-binaries-<sha>`. Non-Go projects can download that artifact instead of
installing a Go toolchain. Archives are versioned by the commit SHA
(`coverage_0.0.0-<shortsha>_<os>_<arch>`).

## Usage

```bash
coverage --input ./coverage-artifacts --output "$GITHUB_STEP_SUMMARY"
```

| Flag | Default | Meaning |
|---|---|---|
| `--input` | *(required)* | directory containing `coverage-*.xml` and `tests-*.xml` |
| `--output` | `-` | output path; `-` is stdout. A file is **appended**. |
| `--ignore` | *(auto)* | path to a `.coverageignore` (gitignore syntax). Defaults to `./.coverageignore` if present. |
| `--config` | *(auto)* | path to `coverage.yaml`. Defaults to `./coverage.yaml` if present. |
| `--baseline` | — | baseline `coverage-summary.json` to diff against. |
| `--fail-on-drop` | — | exit non-zero if **total** line coverage drops by more than this many percentage points. |
| `--emit-json` | — | also write a machine-readable `coverage-summary.json` to this path. |
| `--format` | *(auto)* | output format: `markdown` or `html`. Auto-detects `html` from an `.html`/`.htm` `--output`. |
| `--verbose` | `false` | log warnings for workspaces missing a config entry. |

CLI flags override `coverage.yaml`, which overrides built-in defaults.

`coverage version` prints the build version and the commit SHA it was built
from — handy for confirming which SHA-pinned build you're running.

### Output formats

- **Markdown** (default) — for `$GITHUB_STEP_SUMMARY`. Written in append mode.
- **HTML** — a self-contained, theme-aware page (`--format html`, or an
  `.html`/`.htm` output path). Written in truncate mode.

Both formats are rendered from templates in
[`internal/render/templates/`](./internal/render/templates/) (`report.md.tmpl`,
`report.html.tmpl`).

## Filename conventions (load-bearing)

| Thing | Convention | Example |
|---|---|---|
| Coverage artifact | `coverage-<id>.xml` (Cobertura) | `coverage-thingy.xml` |
| Test-count artifact | `tests-<id>.xml` (JUnit) | `tests-thingy.xml` |
| Workspace id | the `<id>` in the filenames; may contain dashes | `shared-widget` |
| Input dir | all `coverage-*.xml` + `tests-*.xml` flattened together | `./coverage-artifacts` |

## CI in one job

Each test job uploads two artifacts; one downstream job runs the tool once. Full
copy-paste workflow: [`examples/coverage.yml`](./examples/coverage.yml).

```yaml
report:
  needs: [test-web, test-api]
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v6
    - uses: actions/download-artifact@v8
      with: { pattern: coverage-*, path: ./cov, merge-multiple: true }
    - uses: actions/download-artifact@v8
      with: { pattern: tests-*, path: ./cov, merge-multiple: true }
    - uses: aanantaco/coverage@4dde4d44807a0b7d29358bb9417768b4a7cc1960 # pin to a reviewed SHA
      with:
        input: ./cov
        output: $GITHUB_STEP_SUMMARY
```

## Optional config

- **`.coverageignore`** — gitignore syntax, matched against repo-root-relative
  paths, for excluding generated code, test files, vendored deps, etc. Start
  from [`.coverageignore.example`](./.coverageignore.example).
- **`coverage.yaml`** — optional and zero-config by default. Sets display names,
  bridges coverage paths to ignore patterns (`prefix` / `strip_prefix`), folder
  depth, and the regression baseline. Annotated schema:
  [`coverage.yaml.example`](./coverage.yaml.example). A present-but-malformed
  file is a hard error.

`prefix`/`strip_prefix` bridge emitter paths to a repo-root ignore file: the tool
computes `rel = strip_prefix removed from filename` (used for folder grouping)
and `full = prefix + rel` (matched against `.coverageignore`). Go module import
paths are the usual reason to set `strip_prefix` — see [docs/GO.md](./docs/GO.md).

## Coverage deltas across runs

Emit a baseline with `--emit-json coverage-summary.json`, then pass it back on a
later run with `--baseline` to get Δ columns, a "coverage decreased" callout, and
`new`/removed markers; add `--fail-on-drop 0.5` to fail on a total drop. The full
recipe (default-branch cache, artifact, or committed baseline) is in
[docs/REGRESSION.md](./docs/REGRESSION.md).

## Documentation

Full docs — per-language guides, the regression guide, and references — live in
[`docs/`](./docs/README.md).

## Development

```bash
go build ./...
go test ./...
```

A single runtime dependency: [`github.com/goccy/go-yaml`](https://github.com/goccy/go-yaml)
(config parsing) — actively maintained and dependency-free. Everything else —
Cobertura/JUnit parsing, `.coverageignore` gitignore matching, folder grouping,
delta computation, baseline JSON — is implemented in-repo on the standard
library. Tests use the standard library only. The `.coverageignore` matcher is a
dependency-free port of [`github.com/sabhiram/go-gitignore`](https://github.com/sabhiram/go-gitignore)
(MIT) — see [`THIRD_PARTY_NOTICES.md`](./THIRD_PARTY_NOTICES.md).

## License

MIT — see [LICENSE](./LICENSE).
