# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **`coverage init` subcommand.** Detects a repo's languages (Go, TypeScript/
  JavaScript, Rust, Python, Java, C#) and non-destructively scaffolds
  `.github/workflows/coverage.yml` (a complete report job plus framework-agnostic
  test-job stubs that link to the per-language docs), a starter `.coverageignore`,
  and a commented `coverage.yaml`. Never overwrites existing files; `--dry-run`
  previews. Volatile per-framework commands are deliberately left to the docs
  rather than baked into the tool.
- **`llms.txt`** at the repo root — a concise, LLM-oriented map (the stable
  two-file contract plus links to the per-language recipes) so AI assistants can
  wire coverage into a repo from durable docs.
- **Lint job (markdown & shell).** A `lint` CI job runs `markdownlint-cli2`
  (config in `.markdownlint-cli2.yaml`) over the docs and `shellcheck` over any
  tracked shell scripts. Existing Markdown lint findings were fixed.
- **Per-commit release binaries (GoReleaser).** A `release` job runs on every
  merge to `main`, after build/test and the security scan pass, cross-compiling
  `coverage` (linux/macOS/Windows × amd64/arm64) in snapshot mode and uploading
  the archives + `checksums.txt` as a workflow artifact (`coverage-binaries-<sha>`).
  No git tags and no GitHub Release — archives are keyed to the commit SHA and
  consumers pin by SHA. Config in `.goreleaser.yaml`.
- **HTML report output.** `--format html` (or an `.html`/`.htm` `--output`)
  renders a self-contained, theme-aware HTML page. Both Markdown and HTML are
  now rendered from embedded templates (`internal/render/templates/report.md.tmpl`
  and `report.html.tmpl`); the Markdown output is unchanged (locked by golden
  tests).
- **Documentation site** under `docs/` with an index and per-language how-to
  guides (Go, TypeScript/JS, Rust, Python, Java, C#) — each listing supported
  test frameworks, artifact commands, and an example config — plus a
  [regression guide](docs/REGRESSION.md) for coverage deltas across runs. The
  README is trimmed to an overview with a supported-languages table linking into
  `docs/`.
- **Example consumer workflow** at `examples/coverage.yml`.

### Changed

- **Docs filenames uppercased; `docs/index.md` → `docs/README.md`** so the
  `docs/` folder auto-renders its index on GitHub. Per-language guides are now
  `docs/GO.md`, `docs/TYPESCRIPT.md`, etc.
- **Composite Action now builds from its own source at the pinned ref** instead
  of `go install …@latest`, so pinning the Action by commit SHA
  (`uses: aanantaco/coverage@<sha>`) selects the exact tool version. The
  `version` input was removed.
- **Renamed the binary from `coverage-report` to `coverage`.** Install and run
  paths are now `go install github.com/aanantaco/coverage/cmd/coverage@latest`
  and `coverage …`. The `generated_from` field in the summary JSON is now
  `"coverage"`.
- **Dropped the `github.com/sabhiram/go-gitignore` dependency.** `.coverageignore`
  matching is now implemented in-repo (`internal/ignore/gitignore.go`) as a
  dependency-free port of that library, preserving its exact matching semantics.
  Added `THIRD_PARTY_NOTICES.md` with the ported code's MIT attribution.
- **Switched YAML parsing from `gopkg.in/yaml.v3` to `github.com/goccy/go-yaml`.**
  The upstream `go-yaml/yaml` was archived and marked unmaintained in April 2025;
  `goccy/go-yaml` is an actively maintained, dependency-free replacement. Strict
  decoding (unknown-key rejection) is preserved via `yaml.Strict()`. The binary
  now has a single external dependency with no transitive module dependencies.

## [0.1.0] - 2026-07-07

Initial release: the report-only coverage aggregator plus configuration and
regression detection.

### Added

- **Coverage aggregation.** Collects `coverage-<id>.xml` (Cobertura) and
  `tests-<id>.xml` (JUnit) artifacts and renders a single Markdown summary
  (per-workspace table + per-folder breakdown) for `$GITHUB_STEP_SUMMARY`.
  Line and branch totals are recomputed from the leaf `<line>` elements.
- **`.coverageignore`** support (gitignore syntax) for excluding generated code,
  test files, vendored deps, migrations, and non-unit-testable glue. Optional;
  a passthrough matcher makes it a no-op when absent.
- **`coverage.yaml`** configuration (fully optional, strict when present):
  `folder_group_depth`, `ignore_file`, `display_from` (`id` / `shared-slash`),
  and per-workspace `display_name`, `prefix`, and `strip_prefix` (for Go module
  import paths). Missing file = defaults; malformed file = hard error.
- **Regression detection.** `--emit-json` writes a reproducible
  `coverage-summary.json` (no timestamps). `--baseline` (or `baseline.path`)
  diffs against it, adding Δ columns, a "coverage decreased" callout, `new`
  markers, and a removed-workspace note. `--fail-on-drop` (or
  `baseline.fail_on_drop`) exits non-zero on a total line-coverage regression,
  after the report is written.
- **Distribution.** `go install github.com/aanantaco/coverage/cmd/coverage@latest`
  and a composite GitHub Action (`action.yml`).
- **Examples.** `.coverageignore.example` and `coverage.yaml.example`.

[0.1.0]: https://github.com/aanantaco/coverage/releases/tag/v0.1.0
