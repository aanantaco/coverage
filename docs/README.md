# coverage — documentation

`coverage` aggregates **Cobertura** coverage XML and **JUnit** test-result XML
from every project in a repo into one Markdown summary for the CI run, with
optional config and regression detection.

It is **language-agnostic**: any toolchain that can emit Cobertura + JUnit works.
Each project produces two files in a shared directory:

| File | Format | Purpose |
|---|---|---|
| `coverage-<id>.xml` | Cobertura | line & branch coverage |
| `tests-<id>.xml` | JUnit | test count (renders `—` if absent) |

`<id>` is any workspace id you choose (may contain dashes).

## Per-language how-to guides

Each guide lists the supported test frameworks, the exact commands to emit the
two artifacts, and an example `coverage.yaml` block.

- [Go](./GO.md)
- [TypeScript / JavaScript](./TYPESCRIPT.md)
- [Rust](./RUST.md)
- [Python](./PYTHON.md)
- [Java](./JAVA.md)
- [C# / .NET](./CSHARP.md)

## Guides

- [Coverage deltas across runs (regression detection)](./REGRESSION.md)

## Reference

- Output formats — Markdown (default) or a self-contained HTML page
  (`--format html`, or an `.html`/`.htm` `--output`). Both render from templates
  in [`internal/render/templates/`](../internal/render/templates/).
- Config schema — [`coverage.yaml.example`](../coverage.yaml.example)
- Ignore file — [`.coverageignore.example`](../.coverageignore.example)
- Example CI workflow — [`examples/coverage.yml`](../examples/coverage.yml)
- CLI flags and conventions — [README](../README.md)

## The shared pattern

Every language follows the same three steps:

1. **Emit** `coverage-<id>.xml` (Cobertura) and `tests-<id>.xml` (JUnit) in each
   test job.
2. **Upload** both as artifacts named `coverage-<id>` / `tests-<id>` (use
   `if: always()` so failed runs still report).
3. **Aggregate** in one downstream job that downloads all artifacts into a
   single directory and runs `coverage --input <dir> --output "$GITHUB_STEP_SUMMARY"`.

### "No tests yet" placeholder

To keep a workspace in the report before it has real tests, emit minimal XML:

```xml
<!-- coverage-<id>.xml -->
<?xml version="1.0"?>
<coverage lines-valid="0" lines-covered="0"><packages/></coverage>
```
```xml
<!-- tests-<id>.xml -->
<?xml version="1.0" encoding="UTF-8"?>
<testsuites tests="0" failures="0" errors="0" skipped="0"/>
```
