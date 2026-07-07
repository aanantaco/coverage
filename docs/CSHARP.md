# C# / .NET

## Supported test frameworks

`dotnet test` (xUnit, NUnit, or MSTest), with:

| Concern | Tool |
|---|---|
| Coverage (Cobertura) | [`coverlet`](https://github.com/coverlet-coverage/coverlet) (`--collect:"XPlat Code Coverage"`) |
| Tests (JUnit) | [`JunitXml.TestLogger`](https://github.com/spekt/junit.testlogger) |

## Emit the artifacts

Add the logger package to your test project:

```bash
dotnet add package JunitXml.TestLogger
```

Run tests with Cobertura coverage and JUnit logging:

```bash
dotnet test \
  --collect:"XPlat Code Coverage" \
  --logger "junit;LogFilePath=tests-<id>.xml" \
  -- DataCollectionRunSettings.DataCollectors.DataCollector.Configuration.Format=cobertura

# coverlet writes to a GUID subfolder; copy it to the stable name:
cp $(find . -name coverage.cobertura.xml | head -1) coverage-<id>.xml
```

## Example config

coverlet writes source-relative paths. In a solution with multiple projects,
add a `prefix` (and `strip_prefix` if the paths carry a leading absolute or
repo prefix you want removed):

```yaml
# coverage.yaml
workspaces:
  service:
    display_name: src/Service
    prefix: src/Service/
    # strip_prefix: /home/runner/work/repo/repo/src/Service/
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
