# `coverage init` — scaffold a repo

`coverage init` inspects the current repository, detects its languages, and
**non-destructively** scaffolds a tailored setup:

- `.github/workflows/coverage.yml` — a test job per detected language plus the
  aggregation job
- `.coverageignore` — a starter ignore file
- `coverage.yaml` — a commented starter config (optional; the tool works with
  zero config)

It **never overwrites** existing files — anything already present is skipped.

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
| TypeScript/JavaScript | `package.json` (Jest vs Vitest inferred from its deps) |
| Python | `pyproject.toml`, `setup.py`, `setup.cfg`, `tox.ini`, `pytest.ini`, or `requirements.txt` |
| Rust | `Cargo.toml` |
| Java | `pom.xml`, `build.gradle`, or `build.gradle.kts` |
| C#/.NET | any `*.csproj` or `*.sln` |

## After running

The generated test commands are a **best-effort starting point** — review them
for your project (some, like Java's JaCoCo→Cobertura conversion, are left as
`TODO` comments). Then:

1. Replace `aanantaco/coverage@<commit-sha>` in the report job with a real
   commit SHA to pin the tool.
2. Adjust `.coverageignore` and `coverage.yaml` as needed (see the
   [per-language guides](./README.md)).
3. Commit the files.
