# Security findings

Scan date: **2026-08-25**. Commit scanned: `5f759be` (branch `main`), plus the
working tree at the time of the scan.

This is a triage document. Nothing in it has been fixed — each finding is meant
to be picked up as its own change. No runnable exploit payloads are included.

## Summary

| Severity | Count |
|---|---|
| Critical | 0 |
| High | 1 |
| Medium | 4 |
| Low | 6 |
| **Total** | **11** |

### Stacks detected and scanned

| Stack | Present as | Scanned | How |
|---|---|---|---|
| Go | `go.mod`, 14 non-test `.go` files (2215 LOC), 1 direct dependency | Yes | govulncheck, gosec, staticcheck, `go vet`, trivy (gomod), manual review |
| GitHub Actions workflows | `.github/workflows/ci.yml`, `.github/dependabot.yml` | Yes | actionlint, manual review |
| GitHub composite Action | `action.yml` | Yes | Manual review only — see coverage gaps |
| Example / generated workflows | `examples/coverage.yml`, `internal/scaffold/templates.go` | Yes | Manual review |
| Go text/HTML templates | `internal/render/templates/report.md.tmpl`, `report.html.tmpl` | Yes | Manual review, govulncheck reachability |
| Release pipeline | `.goreleaser.yaml` | Yes | Manual review |
| Shell | Embedded in workflow `run:` blocks and `action.yml` | Partially | Manual review; shellcheck unavailable — see coverage gaps |
| Secrets in tree | All tracked files | Yes | trivy secret scanner, manual grep |

Stacks **not present** in this repository, and therefore not scanned: no
`Dockerfile` or container image build, no Terraform / CloudFormation / Helm /
Kubernetes manifests, no `package.json`, no Python, Java, .NET or Rust sources,
no SQL, and no HTTP server or client code. `git ls-files` matches zero files for
any of those markers. The repository is a single-module Go CLI plus its CI and
packaging.

### Tools run

Every command below was run from the repository root unless noted.

| Tool | Version | Command | Result |
|---|---|---|---|
| Go toolchain | `go1.26.0 linux/arm64` | `go version` | Toolchain auto-selected from the `go 1.26.0` directive in `go.mod` |
| go vet | bundled with go1.26.0 | `go vet ./...` | Exit 0, no diagnostics |
| govulncheck | `v1.5.0`, DB `https://vuln.go.dev` updated 2026-08-21 | `govulncheck ./...` | Exit 3 — 6 called stdlib advisories |
| gosec | `v2.28.0` | `gosec -fmt=text ./...` | Exit 1 — 9 issues across 14 files |
| staticcheck | `2026.2.1 (0.8.1)` | `staticcheck ./...` | Exit 0, no diagnostics |
| actionlint | `v1.7.12` | `actionlint -no-color -oneline` | Exit 0, no diagnostics |
| Trivy | `0.72.0`, image digest `sha256:cffe3f5161a47a6823fbd23d985795b3ed72a4c806da4c4df16266c02accdd6f` | see below | 0 vulnerabilities, 0 secrets, 0 misconfigurations |

The Trivy invocation mirrors the one CI already runs (`.github/workflows/ci.yml:85`),
widened to include `LOW`:

```text
trivy filesystem --exit-code 0 --severity LOW,MEDIUM,HIGH,CRITICAL \
  --scanners vuln,secret,misconfig --ignorefile .trivyignore \
  --skip-version-check --format table .
```

Reproducing this locally needs the repository copied *into* the container rather
than bind-mounted, if your Docker socket belongs to a different filesystem
namespace than your checkout:

```text
CID=$(docker create -w /workspace \
  aquasec/trivy:0.72.0@sha256:cffe3f5161a47a6823fbd23d985795b3ed72a4c806da4c4df16266c02accdd6f \
  filesystem --exit-code 0 --severity LOW,MEDIUM,HIGH,CRITICAL \
  --scanners vuln,secret,misconfig --ignorefile .trivyignore \
  --skip-version-check --format table .)
docker cp ./. "$CID":/workspace
docker start -a "$CID"
```

Tools installed for this scan (none were preinstalled):

```text
go install golang.org/x/vuln/cmd/govulncheck@v1.5.0
go install github.com/securego/gosec/v2/cmd/gosec@latest      # resolved to v2.28.0
go install honnef.co/go/tools/cmd/staticcheck@latest          # resolved to v0.8.1
go install github.com/rhysd/actionlint/cmd/actionlint@latest  # resolved to v1.7.12
```

`gosec` self-reports its version as `dev` because `go install` does not stamp
its ldflags; the `v2.28.0` above comes from
`go version -m "$(command -v gosec)"`.

### Coverage gaps

These are stated so the report is not mistaken for a clean bill of health on
ground it never covered.

- **`action.yml` is not linted by any tool.** actionlint only reads
  `.github/workflows/`; it does not parse composite-action definitions. Trivy's
  misconfiguration scanner has no GitHub Actions policies and reported
  `Detected config files num=0` for this tree. Everything found in `action.yml`
  below came from manual review. This gap is itself reported as SEC-005.
- **shellcheck is not installed** in the scan environment, so actionlint's
  shellcheck integration for `run:` blocks was inert, and the shell embedded in
  `action.yml` and `ci.yml` was reviewed by hand only. CI does run shellcheck,
  but only over `git ls-files '*.sh' '*.bash'` (`.github/workflows/ci.yml`), and
  this repository tracks no such files — so that step is a no-op today.
- **zizmor was not run.** It is the tool that specialises in GitHub Actions
  template injection and would likely have flagged SEC-001 automatically. It
  installs via `pip`, and no `pip` is present in this environment
  (`pip: command not found`); no alternative install path was available.
- **hadolint, tfsec, checkov, semgrep, osv-scanner and grype were not run.** The
  first three have no applicable input (no Dockerfile, no IaC). semgrep,
  osv-scanner and grype were not installed and were not fetched, so their
  coverage of the Go tree is simply absent; govulncheck plus gosec plus Trivy's
  gomod scanner is the dependency and SAST coverage that was actually achieved.
- **`.trivyignore` is empty of entries** (comments only), so no Trivy finding is
  currently being suppressed. If entries are added later, each is an accepted
  risk that should be re-reviewed alongside this document.

### Manual-review classes considered

| Class | Conclusion |
|---|---|
| Committed secrets / credentials | **None found.** Trivy's secret scanner reported 0. A manual `git grep` for key-, token-, password- and PEM-shaped strings across all tracked files returned only documentation prose, `--scanners secret` flags in `ci.yml`, and `GH_TOKEN: ${{ github.token }}` / `GITHUB_TOKEN: ${{ github.token }}` in the release jobs — both are the ephemeral per-run token, not a stored secret. The repository declares no `secrets.*` reference anywhere. |
| Authentication / authorisation bypass | **Not applicable.** The tool has no users, sessions, roles or auth checks. It is a single-shot CLI that reads files and writes a report. |
| SQL injection | **Not applicable.** No SQL, no database driver, no query construction anywhere in the tree. |
| Command injection (Go) | **None found.** The Go code imports neither `os/exec` nor `syscall`; it spawns no subprocess. Verified by `git grep -nE 'exec\.Command\|os/exec' -- '*.go'` returning nothing. |
| Command injection (CI / shell) | **Found — SEC-001.** `action.yml` interpolates action inputs into a `bash` `run:` block. |
| SSRF / unvalidated outbound requests | **None found in Go.** The Go code imports no `net/http` and parses no URLs. The only outbound requests in the repository are two `curl` calls in `action.yml:104` and `action.yml:112`, both to a fixed `https://github.com/aanantaco/coverage/releases/download/` base with a path segment derived from `github.action_ref`, which is validated against `^[0-9a-fA-F]{40}$` or `^v[0-9]+\.[0-9]+\.[0-9]+$` before use (`action.yml:76-84`). The host is not attacker-influenceable. |
| Open redirect | **Not applicable.** No HTTP request handling and no redirect logic. |
| Permissive CORS | **Not applicable.** No HTTP server. The HTML report (`internal/render/templates/report.html.tmpl`) is a static local file with no headers, no fetch, and no script. |
| Missing / disabled TLS verification | **None found.** No `InsecureSkipVerify`, no custom `tls.Config`, no `GODEBUG` TLS relaxations. The two `curl` calls use `-fsSL` without `-k`/`--insecure`, so certificate verification is on. |
| Overly broad workflow / IAM permissions | **None found, and the configuration is good.** `.github/workflows/ci.yml:9` sets a repository-default `permissions: contents: read`; only the two release jobs escalate, each to `contents: write` scoped at the job level (`ci.yml:164`, `ci.yml:230`) with an inline comment justifying it. The workflow triggers on `pull_request`, not `pull_request_target`, so fork PRs get no secrets and a read-only token. All third-party actions in this repository's own workflows are pinned to full commit SHAs with `# vX.Y.Z` comments, and Dependabot is configured to bump them. There is no cloud IAM in this repository. |
| XXE / XML external entities | **None found.** Both parsers use Go's `encoding/xml` (`internal/cobertura/parser.go:84`, `internal/junit/parser.go:72`), which does not resolve external entities and does not expand DTD-declared internal entities — so neither classic XXE nor a billion-laughs expansion applies. The related depth-recursion advisory *is* reachable and is reported as SEC-002. |
| Template injection / XSS | **None found in the HTML path.** Both templates are compile-time constants embedded with `//go:embed` (`internal/render/render.go:16`); no template text is ever built from input. The HTML path uses `html/template` (`render.go:39`), which contextually auto-escapes. The Markdown path uses `text/template` (`render.go:38`), which does not escape — reported as SEC-003. |
| Path traversal | **Present but not a boundary.** gosec flags five variable-path file opens; all five paths originate from the operator's own CLI flags or their own `coverage.yaml`. Reported and downgraded as SEC-008. |
| Supply chain / artifact integrity | **Found — SEC-004 and SEC-006.** |

## Findings

Ordered Critical first. Severities are judged by exploitability in *this*
codebase; where that differs from a scanner's own rating, the scanner's rating
and the reason for the move are both stated.

### SEC-001 — Composite Action interpolates action inputs into a shell command

- **Severity:** High
- **Source:** manual review (no tool covers `action.yml`; see Coverage gaps)
- **Location:** `action.yml:151`–`action.yml:156`

The final step of the composite Action builds its argument array by pasting
`${{ inputs.* }}` expressions directly into a `bash` `run:` block. GitHub
substitutes those expressions into the script text *before* bash ever sees it,
so the surrounding double quotes provide no protection: any shell metacharacter
in an input value is interpreted as shell syntax, not as data. All seven inputs
consumed by that script are affected — `input`, `output`, `ignore`, `config`,
`baseline`, `fail-on-drop`, and `emit-json`.

**Attacker impact.** This Action is published to the Marketplace and runs inside
other people's repositories. A consumer who wires any externally-influenced
value into one of these inputs — a pull request title, a branch or tag name, an
issue body, a matrix value derived from repository content — hands an outside
contributor arbitrary command execution on the consumer's runner, with that
job's token and any secrets exposed to the step. The blast radius is the
consumer's CI, not this repository. Note that the earlier `Resolve coverage
binary` step already does this correctly: it passes `github.action_ref` through
an `env:` block (`action.yml:66-67`) and reads it as `$ACTION_REF`, so the
pattern to copy is already present in the file.

**Remediation.** Move every input into the step's `env:` block and reference the
resulting environment variables from the script (`"$INPUT_INPUT"`,
`"$INPUT_OUTPUT"`, …), matching what `action.yml:66-67` already does; the
values then reach bash as data rather than as script text.

### SEC-002 — Unbounded XML recursion depth reachable from both artifact parsers

- **Severity:** Medium
- **Scanner's rating:** govulncheck reports Go advisory `GO-2026-6088` without a
  severity score
- **Source:** `govulncheck ./...`
- **Location:** `internal/cobertura/parser.go:84`, `internal/junit/parser.go:120`

govulncheck confirms both call paths are reachable:

```text
Vulnerability #2: GO-2026-6088
    Add recursion depth guard during decode in encoding/xml
  Standard library
    Found in: encoding/xml@go1.26
    Fixed in: encoding/xml@go1.26.6
    Example traces found:
      #1: internal/junit/parser.go:120:24: junit.rootElement calls xml.Decoder.Token
      #2: internal/cobertura/parser.go:84:25: cobertura.Parse calls xml.Unmarshal
```

Sufficiently deeply nested XML exhausts the goroutine stack. In Go, stack
exhaustion is a fatal runtime error that `recover` cannot intercept, so the
process dies rather than returning an error — which matters here because
`aggregateCoverage` is otherwise carefully written to survive a bad artifact
(`internal/app/app.go:219` logs a warning and skips the file).

**Attacker impact.** Anyone who can place a file matching `coverage-*.xml` or
`tests-*.xml` into the tool's `--input` directory can kill the report step. In
the realistic consumer setup this includes a fork pull request author, whose
test job produces the artifacts that the aggregating job later downloads and
parses. The result is a denial of service on a CI job — no code execution, no
data disclosure, and no effect on merge decisions beyond the failed step.
Medium, not High, because the impact ceiling is a failed build.

**Whether you are affected depends on the toolchain**, which is worth stating
because CI and a local build can disagree. `go.mod` declares `go 1.26.0`, so
`GOTOOLCHAIN=auto` selects exactly go1.26.0 — the stdlib this scan used, and the
one a consumer following the documented `go install …@latest` path
(`examples/coverage.yml:101`) may get. CI's `actions/setup-go` with
`go-version: "1.26"` resolves to the latest 1.26.x patch instead, so CI's own
govulncheck step can pass while this scan reports six advisories.

**Remediation.** Raise the `go` directive in `go.mod` to `1.26.6` so every build
path — including `go install` by consumers — picks up the patched `encoding/xml`.

### SEC-003 — Coverage-XML filenames are rendered unescaped into the job summary

- **Severity:** Medium
- **Source:** manual review
- **Location:** `internal/render/templates/report.md.tmpl:12`,
  `report.md.tmpl:20`, `report.md.tmpl:21`; value flows from
  `internal/app/app.go:245` and `internal/app/app.go:253` via
  `internal/render/view.go:64` and `internal/render/view.go:71`

The Markdown report is rendered with `text/template`
(`internal/render/render.go:38`), which performs no escaping. The `Label` cell
in both tables is the workspace display name or a folder path, and folder paths
are derived straight from the `filename` attribute of `<class>` elements in the
parsed Cobertura XML: `stripPrefix(class.Filename, …)` at `app.go:245`, then
`folderGroup(rel, …)` at `app.go:253`, then `Label: f.Path` at `view.go:71`. No
sanitisation happens anywhere along that path. A `filename` containing pipe
characters, newlines or Markdown syntax is emitted verbatim into a Markdown
table.

**Attacker impact.** The Action's default `output` is `$GITHUB_STEP_SUMMARY`
(`action.yml:24`), so this content lands in the run's Summary tab, which is
exactly what reviewers read to decide whether coverage is acceptable. A
contributor who controls their own test output can break out of the table cell
and append arbitrary Markdown — a forged "all checks passed" block, a fabricated
coverage total, or a link pointing somewhere it should not. This is content
spoofing and reviewer deception, not script execution: GitHub sanitises step
summaries to a Markdown subset, so no XSS is available, which is why this is
Medium rather than High.

**Remediation.** Sanitise `Label` before rendering — strip or escape `|`, CR/LF
and backticks — or add a Markdown-escaping template function and apply it to
every data-derived cell in `report.md.tmpl`.

### SEC-004 — Generated and example workflows ship mutable action tags and `@latest` installs

- **Severity:** Medium
- **Source:** manual review
- **Location:** `internal/scaffold/templates.go:29`, `:33`, `:36`, `:47`, `:48`,
  `:50`; `examples/coverage.yml:27`, `:28`, `:41`, `:44`, `:52`, `:53`, `:58`,
  `:59`, `:64`, `:67`, `:76`, `:79`, `:82`, `:101`

`coverage init` writes a `.github/workflows/coverage.yml` into the user's
repository that references `actions/checkout@v6`, `actions/upload-artifact@v7`
and `actions/download-artifact@v8` — mutable tags, not commit SHAs. The shipped
example does the same and additionally installs build tooling from moving
targets: `go install gotest.tools/gotestsum@latest` and
`go install github.com/boumenot/gocover-cobertura@latest`
(`examples/coverage.yml:58-59`), plus
`go install github.com/aanantaco/coverage/cmd/coverage@latest` in the commented
Option B (`examples/coverage.yml:101`).

This is notable precisely because the repository holds itself to a higher
standard: its own `ci.yml` pins every third-party action to a full commit SHA
with a version comment, and pins `govulncheck@v1.5.0`, `gotestsum@v1.13.0` and
`gocover-cobertura@v1.5.0` exactly. The guidance the tool hands to its users
does not match the practice its maintainers follow.

**Attacker impact.** A tag can be repointed, and a compromised or
tag-hijacked action or module executes with the consumer's job token and
secrets. Every repository that ran `coverage init` inherits that exposure, so
one upstream compromise propagates to all of them. Medium: it requires a
third-party compromise to trigger, but it silently multiplies the consequences
when one happens.

**Remediation.** Emit SHA-pinned refs with `# vX.Y.Z` comments from
`internal/scaffold/templates.go` and in `examples/coverage.yml`, and pin the
`go install` tool versions, mirroring what `.github/workflows/ci.yml` already
does. Point users at Dependabot's `github-actions` ecosystem to keep the pins
fresh.

### SEC-005 — The automated security gate does not cover the CI or Action stack

- **Severity:** Medium
- **Source:** manual review, corroborated by Trivy output
- **Location:** `.github/workflows/ci.yml:78` (the `security-scan` job),
  `.github/workflows/ci.yml:96-97`

The `security-scan` job is the repository's only automated security control, and
it runs Trivy with `--scanners vuln,secret,misconfig`. On this tree Trivy
reports:

```text
┌────────┬───────┬─────────────────┬─────────┬───────────────────┐
│ Target │ Type  │ Vulnerabilities │ Secrets │ Misconfigurations │
├────────┼───────┼─────────────────┼─────────┼───────────────────┤
│ go.mod │ gomod │        0        │    -    │         -         │
└────────┴───────┴─────────────────┴─────────┴───────────────────┘
```

with `Detected config files num=0` on stderr. The `misconfig` scanner has
policies for Dockerfiles, Kubernetes, Terraform, CloudFormation and Helm — none
of which exist here — and none for GitHub Actions. So the job's real coverage of
this repository is: dependency CVEs in `go.mod`, plus secret scanning. It
inspects neither `.github/workflows/` nor `action.yml`.

Two further gaps compound it. The `Lint shell scripts` step runs shellcheck over
`git ls-files '*.sh' '*.bash'`, and this repository tracks no such files, so it
is a no-op — the shell that actually ships lives inside `run:` blocks and
`action.yml`, which it never sees. And CI's `govulncheck` step runs against the
`setup-go`-resolved patch release rather than the `go.mod` floor, so it can pass
while a consumer building at the declared minimum is exposed (see SEC-002).

**Attacker impact.** No direct impact — this is a control gap, not a
vulnerability. Its consequence is SEC-001: a High-severity injection sink sits
in the repository's most externally-exposed file and no gate would have caught
it, nor would catch the next one. Medium reflects the likelihood of future
findings shipping undetected in the highest-exposure part of the tree.

**Remediation.** Out of scope for this report to change, but for triage: adding
zizmor (or actionlint with `-shellcheck`) over `.github/workflows/` and
`action.yml` would close the largest part of the gap.

### SEC-006 — Release-binary checksum verification fails open

- **Severity:** Low
- **Source:** manual review
- **Location:** `action.yml:112`–`action.yml:123`, executed at `action.yml:130`

The Action downloads a prebuilt binary, then attempts to verify it. Every arm of
that verification is skippable: if the `checksums.txt` fetch fails the whole
block is skipped (`action.yml:112`); if the asset's line is absent from the file
the `|| true` at `action.yml:113` swallows it and the `if [ -n "$line" ]` guard
skips verification; and if neither `sha256sum` nor `shasum` is on PATH it prints
a notice and continues (`action.yml:119-121`). In all three cases execution
falls through to `mv`/`chmod +x` at `action.yml:130-131` and the unverified
binary runs. The file's own comment describes this as intentional
("best-effort on the tool's presence"), so the fail-open is a known design
choice — but two of the three fall-through paths are about the *download*, not
the tool's presence.

**Attacker impact.** Low, and deliberately rated below gosec-style defaults for
integrity checks, because the control was weak to begin with: `checksums.txt` is
fetched from the same `releases/download/${tag}` origin as the asset it
validates, so any attacker able to replace the asset can replace the checksum
alongside it. Verification here only ever detected accidental corruption and
partial transfers, and the fail-open paths remove even that. Transport is
`https` with verification enabled, so a network attacker is not in scope.

**Remediation.** Treat a missing `checksums.txt`, a missing asset line, or a
missing checksum tool as fatal rather than as a skip, so an unverified binary is
never executed. For real integrity, publish and verify a signature (for example
Sigstore/cosign via GoReleaser) rather than a same-origin checksum.

### SEC-007 — `html/template` escaper-bypass advisories reachable from the HTML renderer

- **Severity:** Low
- **Scanner's rating:** govulncheck reports these as called vulnerabilities;
  the upstream Go advisories are XSS-class
- **Source:** `govulncheck ./...`
- **Location:** `internal/render/render.go:62` (all five traces),
  additionally `internal/render/render.go:39` for `GO-2026-4865`

Five of the six advisories govulncheck reports are `html/template` escaper
bypasses, all reached through `render.HTML` calling `Template.Execute`:

| Advisory | Title | Fixed in |
|---|---|---|
| `GO-2026-6091` | Fix Javascript regexp context tracking in html/template | go1.26.6 |
| `GO-2026-4982` | Bypass of meta content URL escaping causes XSS in html/template | go1.26.3 |
| `GO-2026-4980` | Escaper bypass leads to XSS in html/template | go1.26.3 |
| `GO-2026-4865` | JsBraceDepth Context Tracking Bugs (XSS) in html/template | go1.26.2 |
| `GO-2026-4603` | URLs in meta content attribute actions are not escaped in html/template | go1.26.1 |

**Downgraded from XSS-class to Low, on exploitability here.** Each of these bugs
requires attacker-influenced data to land in a specific escaping context that
the escaper mishandles: a JavaScript regexp literal, a JavaScript brace context,
or a `<meta …content=…>` URL. `report.html.tmpl` contains none of them. It has
no `<script>` element, no event-handler attribute, no `href`/`src`/`srcset`
bound to data, and its only two `<meta>` tags are `charset` and `viewport` with
static literal content (`report.html.tmpl:4-5`). Every interpolation site is
either element text or a `class` attribute fed by `deltaClass`/`branchClass`,
which return one of five hardcoded strings (`render.go:150-168`). The templates
themselves are `//go:embed` compile-time constants (`render.go:16`), so no
attacker can introduce a vulnerable context. The output is also a local file,
not a served page. Reachability is real; exploitability in this application is
not.

**Remediation.** Same single change as SEC-002: raise the `go` directive in
`go.mod` to `1.26.6`, which clears all six advisories at once.

### SEC-008 — File opens from variable paths (gosec G304, five sites)

- **Severity:** Low
- **Scanner's rating:** gosec MEDIUM (CWE-22, Confidence HIGH)
- **Source:** `gosec -fmt=text ./...`
- **Location:** `internal/junit/parser.go:43`, `internal/ignore/ignore.go:47`,
  `internal/config/config.go:75`, `internal/cobertura/parser.go:64`,
  `internal/baseline/baseline.go:65`

gosec flags each `os.Open`/`os.ReadFile` whose path comes from a variable.

**Downgraded from MEDIUM to Low, on exploitability here.** There is no trust
boundary for these paths to cross. Four of the five come from the operator's own
CLI flags — `--config`, `--baseline`, `--ignore` (`cmd/coverage/main.go:60-64`)
— or from `ignore_file` / `baseline.path` in the operator's own `coverage.yaml`;
the fifth is a member of the `filepath.Glob` result over the operator-supplied
`--input` directory (`internal/app/app.go:201`, `:285`). The process runs with
the invoker's own privileges and reads nothing they could not read directly with
`cat`. Traversal implies an attacker supplying a path, and no such attacker
exists in this design. Reported for completeness, not as an exposure.

**Remediation.** No change required. If the sites are to be silenced, prefer
`os.Root`-scoped access under the resolved `--input` directory for the two
artifact parsers over a blanket `#nosec` comment.

### SEC-009 — World-readable output permissions (gosec G301/G302/G306, four sites)

- **Severity:** Low
- **Scanner's rating:** gosec MEDIUM (CWE-276, Confidence HIGH)
- **Source:** `gosec -fmt=text ./...`
- **Location:** `internal/app/app.go:434` (G302, `0o644`),
  `internal/scaffold/scaffold.go:85` (G301, `0o755`),
  `internal/scaffold/scaffold.go:88` (G306, `0o644`),
  `internal/baseline/baseline.go:55` (G306, `0o644`)

gosec wants `0600`/`0750` on created files and directories.

**Downgraded from MEDIUM to Low, on exploitability here.** Every file written at
these sites is non-sensitive by construction and public by intent: the rendered
coverage report (`app.go:434`, typically `$GITHUB_STEP_SUMMARY`, which is
published to the run's Summary tab), the machine-readable
`coverage-summary.json` (`baseline.go:55`), and the scaffold files that
`coverage init` writes for the user to commit (`scaffold.go:85`, `:88`).
Tightening those to `0600` would actively break the scaffold's purpose, since
the generated workflow and config are meant to be readable and committed. No
secret, token or credential reaches any of these writers. `0644`/`0755` is the
correct permission for this content.

**Remediation.** No change required; the modes match the intended visibility of
the content.

### SEC-010 — Artifact files are read fully into memory with no size limit

- **Severity:** Low
- **Source:** manual review
- **Location:** `internal/cobertura/parser.go:75`, `internal/junit/parser.go:54`

Both parsers call `io.ReadAll(r)` on the artifact stream before decoding, with
no cap. `internal/junit/parser.go` then materialises a second full copy as a
string (`parser.go:58`) and hands that to a second decoder (`parser.go:64`,
`:118`), so a JUnit artifact is resident at roughly twice its size during
parsing. There is no streaming path and no `io.LimitReader`.

**Attacker impact.** An oversized `coverage-*.xml` or `tests-*.xml` in the
`--input` directory can drive the process to consume memory proportional to the
file, ending in an OOM kill of the report step. Same reachability as SEC-002 —
anyone who can write into the artifact directory, including a fork PR author in
the usual consumer setup — and a strictly lesser impact, since the runner's own
disk and memory limits bound how large the input can be in the first place.
Denial of service on a CI job only.

**Remediation.** Wrap the reader in an `io.LimitReader` with a documented ceiling
and return a clear error past it, or decode with `xml.NewDecoder(r)` directly so
neither parser needs the whole document in memory.

### SEC-011 — Uncompilable `.coverageignore` patterns are silently discarded

- **Severity:** Low
- **Source:** manual review
- **Location:** `internal/ignore/gitignore.go:141`–`:144`, reached from
  `internal/ignore/gitignore.go:33`–`:35`

`compilePattern` translates each `.coverageignore` line into a regexp; if
`regexp.Compile` fails it returns `ok == false`, and `compileGitIgnoreLines`
responds with a bare `continue`. Nothing is logged, no error is surfaced, and
the run proceeds as though the line were a comment. A pattern containing an
unbalanced construct that survives the translation — most plausibly a stray
bracket or parenthesis in a path — is therefore dropped with no signal. This is
fail-open in the wrong direction: the safe default for a translation failure in
an *exclusion* list is to keep the path out, not to let it through.

**Attacker impact.** Information disclosure through a maintainer's mistaken
belief. `.coverageignore` is the mechanism for keeping paths out of a coverage
report that is published to a run Summary tab, and a silently-dropped pattern
means paths a maintainer intended to exclude are enumerated in that public
output instead. There is no attacker action involved and the exposure is limited
to file paths and their line counts, never file contents — hence Low.

**Remediation.** Log a warning to stderr naming the line number and the offending
pattern when `regexp.Compile` fails, so a broken exclusion is visible rather than
silent.

## Notes for the next scan

- Re-run with `zizmor` once a `pip`-capable environment is available; it is the
  tool most likely to catch SEC-001-shaped issues automatically, and its absence
  is the widest hole in this scan.
- SEC-002 and SEC-007 (six advisories, one root cause) are cleared by a single
  `go.mod` change and should be re-checked with `govulncheck ./...` afterwards.
  Confirm the toolchain govulncheck resolves — the answer differs between this
  environment and CI, which is what makes the finding easy to miss.
- `.trivyignore` was empty of entries at the time of this scan. If it has grown
  since, review each entry before trusting a clean Trivy result.
