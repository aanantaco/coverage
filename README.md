# coverage-report

A single Go binary that aggregates **Cobertura** coverage XML and **JUnit**
test-result XML from every project in a repo into one **Markdown summary** for
the GitHub Actions "Summary" tab — with optional **regression detection**
against a baseline.

It works for TypeScript (Jest, Vitest) and Go workspaces, in any mix, and needs
only two files per workspace: `coverage-<id>.xml` and `tests-<id>.xml`.

```markdown
## Test Coverage

### Summary

| Workspace | Tests | Lines | % | Branches | % |
|---|---|---|---|---|---|
| compliance-api | 412 | 1842 / 2310 | 79.7% | 215/310 | 69.4% |
| compliance-worker | 175 | 3120 / 3680 | 84.8% | 480/620 | 77.4% |
| shared/awards | — | 320 / 400 | 80.0% | — | — |
| **Total** | **587** | 5282 / 6390 | **82.7%** | 695/930 | **74.7%** |
```

- **Tests** renders `—` when no JUnit artifact was uploaded for a workspace, so
  "missing artifact" is distinguishable from "0 tests".
- **Branches** renders `—` when a workspace reports no branch data (common for
  v8/Go output).
- Totals are always **recomputed from the leaf `<line>` elements** — the tool
  ignores the unreliable top-level totals emitters put on `<coverage>`.

## Install

```bash
go install github.com/aanantaco/coverage/cmd/coverage-report@latest
```

Or use the composite GitHub Action (`aanantaco/coverage@v1`) — see below.

## Usage

```bash
coverage-report --input ./coverage-artifacts --output "$GITHUB_STEP_SUMMARY"
```

| Flag | Default | Meaning |
|---|---|---|
| `--input` | *(required)* | directory containing `coverage-*.xml` and `tests-*.xml` |
| `--output` | `-` | output path; `-` is stdout. A file is **appended** (matching `$GITHUB_STEP_SUMMARY`). |
| `--ignore` | *(auto)* | path to a `.coverageignore` (gitignore syntax). Defaults to `./.coverageignore` if present. |
| `--config` | *(auto)* | path to `coverage.yaml`. Defaults to `./coverage.yaml` if present. |
| `--baseline` | — | path to a baseline `coverage-summary.json` to diff against. |
| `--fail-on-drop` | — | exit non-zero if **total** line coverage drops by more than this many percentage points. |
| `--emit-json` | — | also write a machine-readable `coverage-summary.json` to this path. |
| `--verbose` | `false` | log warnings for workspaces missing a config entry. |

CLI flags override `coverage.yaml`, which overrides built-in defaults.

## Filename conventions (load-bearing)

| Thing | Convention | Example |
|---|---|---|
| Coverage artifact | `coverage-<id>.xml` (Cobertura) | `coverage-compliance-api.xml` |
| Test-count artifact | `tests-<id>.xml` (JUnit) | `tests-compliance-api.xml` |
| Workspace id | the `<id>` in the filenames; may contain dashes | `shared-awards` |
| Input dir | all `coverage-*.xml` + `tests-*.xml` flattened together | `./coverage-artifacts` |

## `.coverageignore` (optional)

Standard gitignore syntax, matched against **repo-root-relative** paths. Start
from [`.coverageignore.example`](./.coverageignore.example). Typical exclusions:
generated code, test files, vendored deps, migrations, and non-unit-testable
glue (CLI entrypoints, test utilities).

When a workspace has a `prefix` configured, that prefix is prepended to each
file before matching, so one repo-root ignore file can target any workspace.

## `coverage.yaml` (optional)

Entirely optional — the tool runs zero-config. Add one only to set display
names, bridge coverage paths to your ignore patterns (common for Go module
paths, via `strip_prefix`/`prefix`), change the folder-group depth, or enable
regression detection. A present-but-malformed file is a hard error.

See [`coverage.yaml.example`](./coverage.yaml.example) for the annotated schema.

```yaml
folder_group_depth: 3
ignore_file: .coverageignore
display_from: id            # or "shared-slash"
baseline:
  path: .coverage-baseline.json
  # fail_on_drop: 0.5
workspaces:
  compliance-api:
    display_name: services/compliance-api
    prefix: services/compliance-api/
    # Go emitters write full module paths; strip that before prefixing:
    strip_prefix: github.com/acme/monorepo/services/compliance-api/
```

### Why `strip_prefix` + `prefix`?

`gocover-cobertura` often emits `<class filename>` as a **package import path**
(`github.com/acme/monorepo/services/api/foo.go`), while your `.coverageignore`
lives at the repo root and is written relative to it
(`services/api/internal/store/**`). The tool computes:

```
rel  = strip_prefix removed from class.filename   # -> internal/store/foo.go, used for folder grouping
full = prefix + rel                               # -> services/api/internal/store/foo.go, matched against .coverageignore
```

Jest/Vitest already emit workspace-relative paths, so those workspaces usually
need only `prefix` (or nothing at all).

## Producing the artifacts

**Jest**
```bash
JEST_JUNIT_OUTPUT_FILE=tests-<id>.xml \
  npx jest --coverage --coverageReporters=cobertura \
    --reporters=default --reporters=jest-junit
cp coverage/cobertura-coverage.xml coverage-<id>.xml
```

**Vitest**
```bash
npx vitest run --coverage --coverage.reporter=cobertura \
  --reporter=default --reporter=junit --outputFile.junit=tests-<id>.xml
cp coverage/cobertura-coverage.xml coverage-<id>.xml
```

**Go**
```bash
gotestsum --junitfile tests-<id>.xml -- \
  -coverprofile=coverage.out -covermode=atomic ./...
gocover-cobertura < coverage.out > coverage-<id>.xml
```

## CI wiring

Each test job uploads two artifacts; one downstream job downloads them all and
runs the tool once.

```yaml
# in each test job
- uses: actions/upload-artifact@v7
  if: always()
  with:
    name: coverage-compliance-api
    path: services/compliance-api/coverage-compliance-api.xml
- uses: actions/upload-artifact@v7
  if: always()
  with:
    name: tests-compliance-api
    path: services/compliance-api/tests-compliance-api.xml
```

```yaml
report:
  needs: [test-ts, test-go]
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v6
    - uses: actions/download-artifact@v8
      with: { pattern: coverage-*, path: ./cov, merge-multiple: true }
    - uses: actions/download-artifact@v8
      with: { pattern: tests-*, path: ./cov, merge-multiple: true }

    # Option A — the composite Action
    - uses: aanantaco/coverage@v1
      with:
        input: ./cov
        output: $GITHUB_STEP_SUMMARY

    # Option B — go install
    # - uses: actions/setup-go@v6
    #   with: { go-version: '1.24' }
    # - run: go install github.com/aanantaco/coverage/cmd/coverage-report@latest
    # - run: coverage-report --input ./cov --output "$GITHUB_STEP_SUMMARY"
```

## Regression detection

The tool can diff the current run against a baseline and add **Δ columns** plus a
callout for any workspace whose line coverage dropped:

```markdown
> ⚠️ **Coverage decreased** in 1 workspace:
> - `compliance-worker`: 84.8% → 83.1% (▼ 1.7pp)
```

The baseline is a `coverage-summary.json` the tool emits with `--emit-json`. It
contains no timestamps, so identical coverage produces an identical file (clean
diffs). Deltas render as `▲ +1.2` / `▼ -0.8` / `▬ 0.0`; a workspace absent from
the baseline is marked `new`; removed workspaces are listed in a note.

Recommended flow:

1. **On merges to your default branch**, run with `--emit-json coverage-summary.json`
   and upload/cache it as the baseline (no repo commits, always fresh).
2. **On PRs**, download that baseline and pass `--baseline coverage-summary.json`
   (or set `baseline.path` in `coverage.yaml`).
3. Optionally set `--fail-on-drop 0.5` to fail a PR that drops total coverage by
   more than 0.5 percentage points. Left unset, it only annotates. The report is
   always written **before** a fail-on-drop exit, so the summary still renders.

Edge cases: a missing baseline file (normal on a new branch) is not an error —
regression detection is simply skipped. A baseline whose `schema` doesn't match
is skipped with a warning rather than crashing.

## Development

```bash
go build ./...
go test ./...
```

A single runtime dependency: `gopkg.in/yaml.v3` (config parsing). Everything
else — Cobertura/JUnit parsing, `.coverageignore` gitignore matching, folder
grouping, delta computation, baseline JSON — is implemented in-repo on the
standard library. Tests use the standard library only.

The `.coverageignore` matcher (`internal/ignore/gitignore.go`) is a
dependency-free port of [`github.com/sabhiram/go-gitignore`](https://github.com/sabhiram/go-gitignore)
(MIT), preserving its exact matching semantics.

## License

MIT — see [LICENSE](./LICENSE).
