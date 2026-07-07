# Go

## Supported test frameworks

The standard `go test` toolchain. Test counts come from
[`gotestsum`](https://github.com/gotestyourself/gotestsum); Cobertura coverage
comes from [`gocover-cobertura`](https://github.com/boumenot/gocover-cobertura).

## Emit the artifacts

```bash
go install gotest.tools/gotestsum@latest
go install github.com/boumenot/gocover-cobertura@latest

gotestsum --junitfile tests-<id>.xml --format pkgname \
  -- -coverprofile=coverage.out -covermode=atomic ./...
gocover-cobertura < coverage.out > coverage-<id>.xml
```

## Example config

`gocover-cobertura` may write `<class filename>` as a **full module import
path** (e.g. `github.com/acme/monorepo/services/thingy/foo.go`). If your
`.coverageignore` patterns are repo-root-relative, strip the module prefix and
re-add a repo-root prefix so ignore matching lines up:

```yaml
# coverage.yaml
workspaces:
  thingy:
    display_name: services/thingy
    strip_prefix: github.com/acme/monorepo/services/thingy/
    prefix: services/thingy/
```

With `strip_prefix`, folder grouping also uses the cleaned path, so folders
render as `internal/worker` rather than the full import path.

## Upload

```yaml
- uses: actions/upload-artifact@v7
  if: always()
  with: { name: coverage-<id>, path: coverage-<id>.xml, if-no-files-found: warn }
- uses: actions/upload-artifact@v7
  if: always()
  with: { name: tests-<id>, path: tests-<id>.xml, if-no-files-found: warn }
```
