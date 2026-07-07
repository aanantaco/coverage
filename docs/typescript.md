# TypeScript / JavaScript

## Supported test frameworks

| Framework | Coverage (Cobertura) | Tests (JUnit) |
|---|---|---|
| Jest | `--coverageReporters=cobertura` | `jest-junit` |
| Vitest | `--coverage.reporter=cobertura` | `--reporter=junit` |
| Mocha + nyc | `nyc --reporter=cobertura` | `mocha-junit-reporter` |

All three emit workspace-relative paths, so usually no `strip_prefix` is needed.

## Emit the artifacts

### Jest
Requires the `jest-junit` dev dependency.
```bash
JEST_JUNIT_OUTPUT_FILE=tests-<id>.xml \
  npx jest --coverage \
    --coverageReporters=cobertura \
    --reporters=default --reporters=jest-junit
cp coverage/cobertura-coverage.xml coverage-<id>.xml
```

### Vitest
Needs `@vitest/coverage-v8` (or `@vitest/coverage-istanbul`).
```bash
npx vitest run --coverage \
  --coverage.reporter=cobertura \
  --reporter=default --reporter=junit \
  --outputFile.junit=tests-<id>.xml
cp coverage/cobertura-coverage.xml coverage-<id>.xml
```

### Mocha + nyc
```bash
npx nyc --reporter=cobertura \
  npx mocha --reporter mocha-junit-reporter \
  --reporter-options mochaFile=tests-<id>.xml
cp coverage/cobertura-coverage.xml coverage-<id>.xml
```

## Example config

Coverage paths are relative to the workspace root (e.g. `src/thing/foo.ts`). In a
monorepo, add a `prefix` so a repo-root `.coverageignore` can target them:

```yaml
# coverage.yaml
workspaces:
  web:
    display_name: apps/web
    prefix: apps/web/
```

Single-package repos usually need no config at all.

## Upload

```yaml
- uses: actions/upload-artifact@v7
  if: always()
  with: { name: coverage-<id>, path: coverage-<id>.xml, if-no-files-found: warn }
- uses: actions/upload-artifact@v7
  if: always()
  with: { name: tests-<id>, path: tests-<id>.xml, if-no-files-found: warn }
```
