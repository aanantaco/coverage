# Python

## Supported test frameworks

`pytest` with [`pytest-cov`](https://pytest-cov.readthedocs.io/) (which wraps
[`coverage.py`](https://coverage.readthedocs.io/)). `coverage.py`'s XML report
**is** Cobertura, and pytest writes JUnit directly — so no conversion is needed.

`unittest` works too, run under `coverage` with `coverage xml`.

## Emit the artifacts

```bash
pip install pytest pytest-cov

pytest \
  --cov=your_package \
  --cov-report=xml:coverage-<id>.xml \
  --junitxml=tests-<id>.xml
```

Plain `unittest`:

```bash
pip install coverage
coverage run -m pytest --junitxml=tests-<id>.xml   # or -m unittest
coverage xml -o coverage-<id>.xml
```

## Example config

`coverage.py` writes paths relative to the project root (e.g.
`your_package/module.py`). In a monorepo, add a `prefix`:

```yaml
# coverage.yaml
workspaces:
  api:
    display_name: services/api
    prefix: services/api/
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
