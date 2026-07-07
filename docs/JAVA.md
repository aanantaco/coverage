# Java

## Supported test frameworks

JUnit tests via **Maven Surefire** or **Gradle**, with
[**JaCoCo**](https://www.jacoco.org/jacoco/) for coverage.

> **Note on formats.** JaCoCo's native XML is *not* Cobertura. Emit Cobertura
> either with a JaCoCo report format that supports it, or by converting
> JaCoCo XML (e.g. [`cover2cover`](https://github.com/rix0rrr/cover2cover) or
> [`jacoco-to-cobertura`](https://github.com/rene-strauss/jacoco-to-cobertura)).
> Surefire/Gradle already produce JUnit XML.

## Emit the artifacts

### Maven (Surefire + JaCoCo)
```bash
mvn test    # JaCoCo bound to the test phase produces target/site/jacoco/jacoco.xml
python cover2cover.py target/site/jacoco/jacoco.xml > coverage-<id>.xml

# JUnit: merge the per-class Surefire reports, or point at one module's file
cp target/surefire-reports/TEST-*.xml tests-<id>.xml   # single module
```

### Gradle
```bash
./gradlew test jacocoTestReport
python cover2cover.py build/reports/jacoco/test/jacocoTestReport.xml > coverage-<id>.xml
cp build/test-results/test/TEST-*.xml tests-<id>.xml
```

## Example config

Cobertura output from JaCoCo uses source-relative paths (e.g.
`com/acme/thing/Service.java` or `src/main/java/...`). Add a `prefix` in a
multi-module build so a repo-root `.coverageignore` lines up:

```yaml
# coverage.yaml
workspaces:
  service:
    display_name: services/service
    prefix: services/service/
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
