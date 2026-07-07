# Rust

## Supported test frameworks

Rust's built-in test harness, with:

| Concern | Tool |
|---|---|
| Coverage (Cobertura) | [`cargo-llvm-cov`](https://github.com/taiki-e/cargo-llvm-cov) |
| Tests (JUnit) | [`cargo-nextest`](https://nexte.st/) (JUnit output), or `cargo2junit` |

## Emit the artifacts

`cargo-llvm-cov` produces Cobertura directly:

```bash
cargo install cargo-llvm-cov cargo-nextest

# Coverage → Cobertura
cargo llvm-cov --cobertura --output-path coverage-<id>.xml

# Tests → JUnit (via nextest)
cargo llvm-cov nextest --no-report --profile ci
# nextest writes target/nextest/ci/junit.xml when the profile enables JUnit:
cp target/nextest/ci/junit.xml tests-<id>.xml
```

Enable JUnit for the nextest `ci` profile in `.config/nextest.toml`:

```toml
[profile.ci.junit]
path = "junit.xml"
```

Alternatively, without nextest:

```bash
cargo install cargo2junit
cargo test -- -Z unstable-options --format json --report-time \
  | cargo2junit > tests-<id>.xml
```

## Example config

`cargo-llvm-cov` writes paths relative to the crate root (e.g. `src/lib.rs`). In
a workspace/monorepo, add a `prefix`:

```yaml
# coverage.yaml
workspaces:
  engine:
    display_name: crates/engine
    prefix: crates/engine/
```

## Upload

```yaml
- uses: actions/upload-artifact@v7
  if: always()
  with: { name: coverage-<id>, path: coverage-<id>.xml, if-no-files-found: warn }
- uses: actions/upload-artifact@v7
  if: always()
  with: { name: tests-<id>, path: tests-<id>.xml, if-no-files-found: warn }
```
