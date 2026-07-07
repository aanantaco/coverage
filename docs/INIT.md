# `coverage init` — scaffold a repo

`coverage init` inspects the current repository, detects its languages, and
**non-destructively** scaffolds a tailored setup:

- `.github/workflows/coverage.yml` — the **report/aggregation job is complete**;
  each per-language test job is a framework-agnostic **stub** with a `TODO` that
  links to the matching per-language doc for the actual test command.
- `.coverageignore` — a starter ignore file
- `coverage.yaml` — a commented starter config (optional; the tool works with
  zero config)

It **never overwrites** existing files — anything already present is skipped.

Why stubs rather than full test commands? Per-framework commands (Jest/Vitest
flags, `gocover-cobertura`, JaCoCo→Cobertura conversion, action versions) change
over time and depend on your project's setup — so they live in the docs (easy for
you, or an AI assistant via [`llms.txt`](../llms.txt), to fill in) instead of
being baked into the tool where they would rot. The tool generates the durable
part: the wiring, artifact names, and the report job.

## Usage

```bash
# In your repo root:
coverage init

# Preview without writing anything:
coverage init --dry-run

# Scaffold a different directory:
coverage init --dir path/to/repo
```

Example:

```text
Detected: Go, Rust, TypeScript/JavaScript

  create  .coverageignore
  create  .github/workflows/coverage.yml
  create  coverage.yaml

3 created, 0 skipped.
```

## Detection

| Language | Detected by |
|---|---|
| Go | `go.mod` |
| TypeScript/JavaScript | `package.json` |
| Python | `pyproject.toml`, `setup.py`, `setup.cfg`, `tox.ini`, `pytest.ini`, or `requirements.txt` |
| Rust | `Cargo.toml` |
| Java | `pom.xml`, `build.gradle`, or `build.gradle.kts` |
| C#/.NET | any `*.csproj` or `*.sln` |

## After running

1. Fill in each test job's `TODO` with your test command, following the linked
   [per-language guide](./README.md), so it emits `coverage-<id>.xml` and
   `tests-<id>.xml`.
2. Replace `aanantaco/coverage@<commit-sha>` in the report job with a real commit
   SHA to pin the tool.
3. Adjust `.coverageignore` and `coverage.yaml` as needed.
4. Commit the files.
