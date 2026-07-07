# Producing coverage & test artifacts per language / framework

`coverage` is language-agnostic: it only consumes two files per project, placed
in a single input directory:

| File | Format | Purpose |
|---|---|---|
| `coverage-<id>.xml` | **Cobertura** XML | line & branch coverage |
| `tests-<id>.xml` | **JUnit** XML | test count (renders `—` if absent) |

`<id>` is any workspace id you choose (may contain dashes, e.g. `shared-awards`).
The **test-count file is optional** — omit it and the Tests column shows `—`.

So "supporting a new language" just means: make your test runner emit Cobertura
coverage and (optionally) JUnit results, then rename/copy them to the
`coverage-<id>.xml` / `tests-<id>.xml` convention and upload them as artifacts.

Below are the common toolchains. Replace `<id>` with your workspace id.

---

## JavaScript / TypeScript

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
```bash
npx vitest run --coverage \
  --coverage.reporter=cobertura \
  --reporter=default --reporter=junit \
  --outputFile.junit=tests-<id>.xml
cp coverage/cobertura-coverage.xml coverage-<id>.xml
```
(Vitest coverage needs `@vitest/coverage-v8` or `@vitest/coverage-istanbul`.)

### Mocha + nyc (Istanbul)
```bash
npx nyc --reporter=cobertura \
  npx mocha --reporter mocha-junit-reporter \
  --reporter-options mochaFile=tests-<id>.xml
cp coverage/cobertura-coverage.xml coverage-<id>.xml
```

---

## Go
```bash
go install gotest.tools/gotestsum@latest
go install github.com/boumenot/gocover-cobertura@latest

gotestsum --junitfile tests-<id>.xml --format pkgname \
  -- -coverprofile=coverage.out -covermode=atomic ./...
gocover-cobertura < coverage.out > coverage-<id>.xml
```
**Path note:** `gocover-cobertura` may emit `<class filename>` as a full module
import path (e.g. `github.com/acme/repo/internal/foo.go`). If your
`.coverageignore` patterns are repo-root-relative, set `strip_prefix` (and
optionally `prefix`) for that workspace in `coverage.yaml`. See the main README.

---

## Python (pytest)
```bash
pip install pytest pytest-cov
pytest \
  --cov=your_package \
  --cov-report=xml:coverage-<id>.xml \   # coverage.py emits Cobertura XML
  --junitxml=tests-<id>.xml
```
`coverage.py`'s `xml` report is Cobertura, so no conversion step is needed.

---

## Java

### Maven (JaCoCo + Surefire)
JaCoCo's native XML is **not** Cobertura — enable its Cobertura-format output
(JaCoCo 0.8.7+) or convert. Surefire already writes JUnit XML.
```bash
mvn test    # JaCoCo bound to the test phase
# Cobertura report:  target/site/jacoco/cobertura.xml  (with the cobertura format enabled)
# JUnit reports:     target/surefire-reports/*.xml  (merge or point one file per module)
cp target/site/jacoco/cobertura.xml coverage-<id>.xml
```
If you cannot emit Cobertura directly, convert JaCoCo XML with a tool such as
`cover2cover` before renaming.

---

## Ruby (RSpec)
```ruby
# spec_helper.rb
require 'simplecov'
require 'simplecov-cobertura'
SimpleCov.start
SimpleCov.formatter = SimpleCov::Formatter::CoberturaFormatter
```
```bash
# Gemfile: gem 'rspec_junit_formatter'
bundle exec rspec --format RspecJunitFormatter --out tests-<id>.xml
cp coverage/coverage.xml coverage-<id>.xml
```

---

## .NET (coverlet)
```bash
dotnet test \
  --collect:"XPlat Code Coverage" \
  --logger "junit;LogFilePath=tests-<id>.xml" \
  -- DataCollectionRunSettings.DataCollectors.DataCollector.Configuration.Format=cobertura
cp **/TestResults/**/coverage.cobertura.xml coverage-<id>.xml
```
(`coverlet` emits Cobertura; the `JunitXml.TestLogger` package provides the
`junit` logger.)

---

## Uploading the artifacts

In each test job, upload both files named to match the file (minus `.xml`), with
`if: always()` so failed runs still report:

```yaml
- uses: actions/upload-artifact@v7
  if: always()
  with: { name: coverage-<id>, path: coverage-<id>.xml, if-no-files-found: warn }
- uses: actions/upload-artifact@v7
  if: always()
  with: { name: tests-<id>, path: tests-<id>.xml, if-no-files-found: warn }
```

Then one `report` job downloads them all (`pattern: coverage-*` / `tests-*`,
`merge-multiple: true`) into a single directory and runs `coverage` once. A full
example is in [`examples/coverage.yml`](../examples/coverage.yml).

## Placeholder for "no tests yet"

To keep a workspace in the report before it has real tests, emit minimal valid
XML by hand:

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
