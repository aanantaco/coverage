# Dependency Risk Review

**Report date:** 2026-08-25
**Scope:** every manifest in this repository, all ecosystems, direct dependencies only.
Indirect dependencies appear only where a direct dependency pulls in one that is
itself flagged vulnerable; the responsible direct dependency is named.

This document reports. It changes no dependency and no code.

## Urgent findings

Three items warrant action ahead of the rest.

1. **The GoReleaser CLI is the only unpinned third-party executable in the repo, and it
   runs in the privileged release path.** `.github/workflows/ci.yml:179` and
   `.github/workflows/ci.yml:245` pass `version: "~> v2"` to `goreleaser-action`, which
   resolves the latest `v2.x` GoReleaser at run time — today `v2.18.0`, published
   2026-08-24, one day old. That job holds `contents: write` and publishes the binaries
   consumers download. Every other third-party artifact here is pinned by commit SHA,
   version tag, or image digest; this one floats. See
   [GoReleaser CLI](#goreleaser-cli-unpinned--v2).

2. **`markdownlint-cli2@0.18.1` hard-pins two transitively vulnerable packages.** It
   declares exact versions (not ranges) for `js-yaml` `4.1.0` and `markdown-it` `14.1.0`,
   so npm cannot resolve away six advisories — two of them High (CVSS 7.5). Bumping the
   direct dependency to `0.23.2` clears all six (`npm audit` on a resolved `0.23.2` tree
   returns zero findings). See [markdownlint-cli2](#markdownlint-cli20181).

3. **The `go 1.26.0` directive in `go.mod` permits a toolchain with six symbol-reachable
   stdlib advisories.** `govulncheck v1.7.0` run against go1.26.0 reports four
   `html/template` escaping/XSS advisories reachable from `render.HTML`
   (`internal/render/render.go:62`) and one `encoding/xml` recursion advisory reachable
   from both parsers. Two of the six are only fixed in go1.26.6.
   **This is not currently a CI or release exposure** — every automated build path pins
   `go-version: "1.26"`, which setup-go resolves to go1.26.7, where all six are fixed.
   The exposure is local builds on a machine whose toolchain satisfies the `go.mod` floor
   exactly. See [Go toolchain](#go-toolchain-gomod-go-directive).

No Critical-rated dependency was found. No dependency is archived, deprecated, or renamed.

## Summary

### Direct dependencies per ecosystem

| Ecosystem | Manifest(s) | Direct dependencies |
|---|---|---|
| Go modules | `go.mod` | 1 |
| Go toolchain | `go.mod` (`go` directive) | 1 |
| Go tools (build/CI, not in `go.mod`) | `.github/workflows/ci.yml`, `.goreleaser.yaml` | 4 |
| GitHub Actions | `.github/workflows/ci.yml`, `action.yml` | 4 |
| npm (invoked via `npx`, no manifest) | `.github/workflows/ci.yml` | 1 |
| Container images | `.github/workflows/ci.yml` | 1 |
| **Total** | | **12** |

### Counts per risk rating

| Rating | Count | Items |
|---|---|---|
| Critical | 0 | — |
| High | 1 | GoReleaser CLI (`~> v2`) |
| Medium | 3 | `markdownlint-cli2`, `aquasec/trivy`, Go toolchain directive |
| Low | 8 | `goccy/go-yaml`, `gotestsum`, `gocover-cobertura`, `govulncheck`, `actions/checkout`, `actions/setup-go`, `actions/upload-artifact`, `goreleaser/goreleaser-action` |

### Tools run

| Tool | Version | Used for |
|---|---|---|
| `govulncheck` | v1.7.0 (DB `vuln.go.dev`, last modified 2026-08-21) | Go symbol-level reachability analysis |
| Go toolchain | go1.25.0 host, go1.26.0 resolved from the `go.mod` directive | scan target |
| `npm audit` | npm 10.9.8 / Node v22.23.2 | npm advisory resolution over a `--package-lock-only` tree |
| OSV API | `api.osv.dev/v1` | advisory lookup for Go, npm, and container-image packages |
| GitHub REST API | `gh` 2.98.0 | repository health, release dates, tag→SHA verification |
| Go module proxy | `proxy.golang.org` | latest versions, release timestamps, version counts |
| npm registry | `registry.npmjs.org` | latest versions, publish times, declared dependency ranges |
| Docker Hub registry v2 | `registry-1.docker.io` | image digest verification |

Reachability was determined by `govulncheck` for Go only. `npm audit`, the OSV API, and
the registry queries do **not** perform reachability analysis; every non-Go advisory in
this report is therefore marked **unverified** for reachability.

`.github/workflows/ci.yml:40` pins `govulncheck@v1.5.0`; this review used v1.7.0. The
advisory database is fetched live in both cases, so the difference affects analysis
precision, not advisory coverage.

---

## Ecosystem: Go modules (`go.mod`)

| Package | Pinned | Latest | Age behind | Advisories | Maintenance signal | Risk |
|---|---|---|---|---|---|---|
| `github.com/goccy/go-yaml` | v1.19.2 | v1.19.2 | 0 releases / current | none (OSV: no advisory at any version; `govulncheck`: clean) | active, single-maintainer-dominant; last release 2026-01-08, last commit 2026-04-07 | Low |

The module graph is exactly two entries (`go list -m all`): this module and `go-yaml`.
There are no transitive Go module dependencies at all, so no indirect Go package can be
flagged.

## Ecosystem: Go toolchain (`go.mod` `go` directive)

| Package | Pinned | Latest | Age behind | Advisories | Maintenance signal | Risk |
|---|---|---|---|---|---|---|
| Go stdlib (`go 1.26.0`) | go1.26.0 (released 2026-02-10) | go1.26.7 (released 2026-08-18) | 7 patch releases / 6.3 months | 6 reachable, 5 imported-not-called, 22 required-not-imported (see subsection) | Google-maintained, current release train | Medium |

## Ecosystem: Go tools (installed in CI, not in `go.mod`)

These are `go install`ed at pinned versions in CI or downloaded by an action. They are
not compiled into the shipped binary, but they run with repository credentials.

| Package | Pinned | Latest | Age behind | Advisories | Maintenance signal | Risk |
|---|---|---|---|---|---|---|
| GoReleaser CLI | `~> v2` (**unpinned**) | v2.18.0 (2026-08-24) | n/a — floats to latest `v2.x` | none for any resolved version | very active (pushed 2026-08-25), 16.0k stars, MIT | **High** |
| `gotest.tools/gotestsum` | v1.13.0 (2025-09-11) | v1.13.0 | 0 releases / current | none (OSV) | active, last commit 2026-04-15, 2.7k stars, Apache-2.0 | Low |
| `github.com/boumenot/gocover-cobertura` | v1.5.0 (2026-05-15) | v1.5.0 | 0 releases / current | none (OSV) | last commit 2026-05-18, 156 stars, single maintainer, MIT | Low |
| `golang.org/x/vuln` (`govulncheck`) | v1.5.0 (2026-06-25) | v1.7.0 (2026-08-13) | 2 releases / 1.7 months | none (OSV) | active (pushed 2026-08-19), Google-maintained, BSD-3-Clause | Low |

## Ecosystem: GitHub Actions

All four are pinned by full 40-hex commit SHA with a version comment. **Every pin was
verified against the upstream tag object via the GitHub API — all four SHAs are the exact
commit the annotated version tag resolves to, and every one is the current latest
release.** No mismatched or stale comment was found.

| Package | Pinned | Latest | Age behind | Advisories | Maintenance signal | Risk |
|---|---|---|---|---|---|---|
| `actions/checkout` | v7.0.1 @ `3d3c42e5` (released 2026-07-20) | v7.0.1 | 0 releases / current | none | GitHub-maintained, pushed 2026-08-10, 8.7k stars, MIT | Low |
| `actions/setup-go` | v7.0.0 @ `b7ad1dad` (released 2026-07-16) | v7.0.0 | 0 releases / current | none | GitHub-maintained, pushed 2026-08-19, 1.8k stars, MIT | Low |
| `actions/upload-artifact` | v7.0.1 @ `043fb46d` (released 2026-04-10) | v7.0.1 | 0 releases / current | none | GitHub-maintained, pushed 2026-04-14, 4.2k stars, MIT | Low |
| `goreleaser/goreleaser-action` | v7.2.3 @ `f06c13b6` (released 2026-06-29) | v7.2.3 | 0 releases / current | none | active, pushed 2026-08-23, 1.0k stars, MIT | Low |

The action is current and SHA-pinned; the CLI it downloads at run time is not. That gap
is rated separately under Go tools.

## Ecosystem: npm (invoked via `npx`, no manifest or lockfile)

| Package | Pinned | Latest | Age behind | Advisories | Maintenance signal | Risk |
|---|---|---|---|---|---|---|
| `markdownlint-cli2` | 0.18.1 (published 2025-05-15) | 0.23.2 (published 2026-07-27) | 9 releases / 15.3 months | none directly; **6 transitive** (2 High, 4 Moderate) | very active (pushed 2026-08-24), 904 stars, sole maintainer `davidanson`, MIT | Medium |

### Indirect npm dependencies flagged vulnerable

Both are pulled in by the direct dependency **`markdownlint-cli2@0.18.1`**, which declares
them as *exact* versions rather than ranges — npm has no freedom to resolve a fixed
release, so these versions are deterministic on every `npx` run.

| Indirect package | Resolved | Advisory | Severity | Affected range |
|---|---|---|---|---|
| `js-yaml` | 4.1.0 | [GHSA-52cp-r559-cp3m](https://github.com/advisories/GHSA-52cp-r559-cp3m) | High (7.5) — quadratic CPU via merge-key chains | `>=4.0.0 <4.3.0` |
| `js-yaml` | 4.1.0 | [GHSA-5p4m-2wfm-xmqj](https://github.com/advisories/GHSA-5p4m-2wfm-xmqj) | High (7.5) — quadratic CPU in `!!omap` resolution | `>=4.0.0 <4.3.1` |
| `js-yaml` | 4.1.0 | [GHSA-mh29-5h37-fv8m](https://github.com/advisories/GHSA-mh29-5h37-fv8m) | Moderate (5.3) — prototype pollution in merge | `>=4.0.0 <4.1.1` |
| `js-yaml` | 4.1.0 | [GHSA-h67p-54hq-rp68](https://github.com/advisories/GHSA-h67p-54hq-rp68) | Moderate (5.3) — quadratic DoS via repeated aliases | `>=4.0.0 <=4.1.1` |
| `markdown-it` | 14.1.0 | [GHSA-38c4-r59v-3vqw](https://github.com/advisories/GHSA-38c4-r59v-3vqw) | Moderate (5.3) — ReDoS | `>=13.0.0 <14.1.1` |
| `markdown-it` | 14.1.0 | [GHSA-6v5v-wf23-fmfq](https://github.com/advisories/GHSA-6v5v-wf23-fmfq) | Moderate (5.3) — quadratic DoS in smartquotes | `<=14.1.1` |

Reachability: **unverified** — `npm audit` performs no reachability analysis.

## Ecosystem: Container images

| Package | Pinned | Latest | Age behind | Advisories | Maintenance signal | Risk |
|---|---|---|---|---|---|---|
| `aquasec/trivy` | `0.72.0@sha256:cffe3f51…accdd6f` (released 2026-06-30) | 0.74.0 (released 2026-08-14) | 2 releases / 1.8 months | [GO-2026-4919](https://pkg.go.dev/vuln/GO-2026-4919) / CVE-2026-33634 / GHSA-69fq-xp46-6x23 | very active (pushed 2026-08-21), 37.6k stars, Apache-2.0 | Medium |

---

## Dependencies rated Medium or above

### GoReleaser CLI (unpinned `~> v2`)

**Rating: High.** Dominant factor: **it is unpinned, in the one job that writes to the
repository and publishes artifacts.**

#### Reasoning

`goreleaser-action` does not bundle GoReleaser; it downloads the CLI at run time
according to its `version` input. `"~> v2"` is a floating constraint that resolves to the
newest `v2.x` on each run. Today that is v2.18.0, published 2026-08-24 — a release less
than a day old would already have been used by any run since.

The exposure is structural rather than a known vulnerability. OSV reports no advisory for
any GoReleaser v2 release, the project is exceptionally active (last push 2026-08-25,
16.0k stars), and no compromise is known. What makes this High is the combination of an
unreviewed floating version with the privilege of the job it runs in: the `release` job
holds `contents: write` (`.github/workflows/ci.yml`) and its output is published as
per-commit prereleases and version releases that `action.yml` then downloads and executes
on consumers' runners. A bad release — malicious or merely regressed — enters that chain
with no gate.

The inconsistency is the strongest argument. This repository SHA-pins all four GitHub
Actions, digest-pins its container image, and pins all three `go install` tools to exact
versions with an explicit "pinned for reproducibility" comment
(`.github/workflows/ci.yml:47`). The release toolchain is the single exception, and it is
the one with the most privilege.

#### API surface used

| Surface | Call site |
|---|---|
| `release --snapshot --clean` (snapshot build on merge to main) | `.github/workflows/ci.yml:180` |
| `release --clean` (version release on `v*` tag) | `.github/workflows/ci.yml:246` |
| `version: 2` config schema | `.goreleaser.yaml:7` |
| `builds` — `main`, `binary`, `env`, `goos`, `goarch`, `ldflags`, `mod_timestamp` | `.goreleaser.yaml:11-31` |
| `archives` — `formats`, `name_template`, `format_overrides` | `.goreleaser.yaml:33-40` |
| `checksum.name_template` | `.goreleaser.yaml:42-43` |
| `snapshot.version_template` | `.goreleaser.yaml:50-51` |
| Template vars `.Version`, `.FullCommit`, `.CommitTimestamp`, `.ProjectName`, `.Os`, `.Arch` | `.goreleaser.yaml:29-31`, `.goreleaser.yaml:37`, `.goreleaser.yaml:51` |

The surface is entirely declarative config plus one subcommand — load-bearing for
releases, but with no Go API coupling. A version pin costs nothing in flexibility.

#### Recommended action

**Bump — to an exact pin.** Replace `version: "~> v2"` with the current exact release
(`version: "v2.18.0"`) at both `.github/workflows/ci.yml:179` and
`.github/workflows/ci.yml:245`. Dependabot's existing `github-actions` group
(`.github/dependabot.yml`) will not track this value, so pair the pin with a note, or
accept a manual bump cadence — the same trade already accepted for the three `go install`
pins. Out of scope for this review; raise it as a separate change.

### `markdownlint-cli2@0.18.1`

**Rating: Medium.** Dominant factor: **version age** — 15.3 months and 9 releases behind,
which is what holds the six transitive advisories in place.

#### Reasoning

The direct package itself has no advisory and is very actively maintained (last push
2026-08-24). The risk is entirely what the stale pin drags along. `markdownlint-cli2`
0.18.1 declares its dependencies as exact versions rather than semver ranges:

```json
"js-yaml": "4.1.0",  "markdown-it": "14.1.0",  "markdownlint": "0.38.0"
```

so npm resolves `js-yaml` 4.1.0 and `markdown-it` 14.1.0 on every run and cannot pick up
`js-yaml` 4.3.1 (published 2026-07-31), which fixes both High advisories. The resolved
tree is 80 packages; `npm audit` reports 1 High and 2 Moderate advisory groups, six
advisories in total. The same audit against a resolved `0.23.2` tree returns zero.

It is rated Medium rather than High despite two CVSS 7.5 advisories because of blast
radius. All six are denial-of-service or prototype-pollution issues, the job runs with the
workflow's default `contents: read` permission (`.github/workflows/ci.yml:9-10`), and its
input is `**/*.md` from the repository itself — content that reaches the linter only via a
reviewed pull request. There is no untrusted-input path into this tool. Reachability is
**unverified**: no npm scanner run here does reachability analysis.

One secondary signal: `npx --yes markdownlint-cli2@0.18.1` fetches from the registry on
every CI run with no lockfile and no integrity hash committed. The version is pinned, so
the resolved tree is deterministic, but nothing in the repository records the expected
package integrity. The maintainer count is one (`davidanson`), typical for this project's
size and offset by its release cadence.

#### API surface used

Invoked as a CLI; the coupling is the config schema, not a code API.

| Surface | Call site |
|---|---|
| `npx --yes markdownlint-cli2@0.18.1` — bare invocation, no CLI flags | `.github/workflows/ci.yml:137` |
| `config:` — rule map (`default`, `MD013`, `MD024.siblings_only`) | `.markdownlint-cli2.yaml:2-9` |
| `globs:` — `**/*.md` | `.markdownlint-cli2.yaml:10-11` |
| `ignores:` — `node_modules`, `dist`, `internal/render/testdata` | `.markdownlint-cli2.yaml:12-17` |
| Exit-code contract (non-zero fails the job) | `.github/workflows/ci.yml:137` |

Three config keys and one invocation. This is about as replaceable as a dependency gets —
the estimate for swapping linters or bumping across the 0.18→0.23 range is bounded by the
rule-name compatibility of `.markdownlint-cli2.yaml`, not by any code change.

#### Recommended action

**Bump** to `markdownlint-cli2@0.23.2`. A one-token change at
`.github/workflows/ci.yml:137` clears all six transitive advisories, verified by resolving
and auditing a `0.23.2` tree. Expect to reconcile rule changes across five minor versions
in `.markdownlint-cli2.yaml`; the config surface is three keys, so that is a small job.
Out of scope for this review.

### `aquasec/trivy:0.72.0`

**Rating: Medium.** Dominant factor: **an open advisory with no recorded fixed version
matches the pinned release**, materially mitigated by a verified digest pin.

#### Reasoning

OSV returns GO-2026-4919 (CVE-2026-33634, GHSA-69fq-xp46-6x23) for
`github.com/aquasecurity/trivy` at 0.72.0. The advisory describes a credential compromise
on 2026-03-19 in which a threat actor published a malicious Trivy **v0.69.4** release. Its
affected range is `{introduced: 0.69.4}` with **no `fixed` event**, so every release from
0.69.4 onward matches — including 0.72.0, published 2026-06-30, three months after the
incident.

Two facts bound the actual risk:

- **The digest pin was verified against the registry.** `.github/workflows/ci.yml:93`
  pins `sha256:cffe3f5161a47a6823fbd23d985795b3ed72a4c806da4c4df16266c02accdd6f`.
  Querying `registry-1.docker.io` for the `0.72.0` manifest returns exactly that digest,
  and Docker Hub reports the tag last updated 2026-06-30T09:32:28Z. The pinned bytes are
  the official 0.72.0 image and have not been re-pushed. A digest pin is immune to tag
  replacement, which is the mechanism the advisory describes.
- **The compromised artifact is a different version.** The named malicious release is
  v0.69.4; nothing in the advisory implicates 0.72.0's contents.

What keeps it above Low is that the advisory remains open with no fixed version, so the
upstream project has not yet published a boundary past which releases are certified
clean, and the pin is 2 releases and 1.8 months behind (0.74.0, 2026-08-14). Reachability
is **unverified** — no reachability-capable scanner covers container images here.

Note that the scanner runs `filesystem` mode over the checkout, so the image is a
build-time tool, not a runtime dependency of the shipped binary.

#### API surface used

Invoked as a container entrypoint; the coupling is the CLI flag set.

| Surface | Call site |
|---|---|
| `docker run --rm -v <workspace>:/workspace -w /workspace <image>` | `.github/workflows/ci.yml:90-93` |
| `filesystem` subcommand | `.github/workflows/ci.yml:94` |
| `--exit-code 1` | `.github/workflows/ci.yml:95` |
| `--severity MEDIUM,HIGH,CRITICAL` | `.github/workflows/ci.yml:96` |
| `--scanners vuln,secret,misconfig` | `.github/workflows/ci.yml:97` |
| `--ignorefile .trivyignore` | `.github/workflows/ci.yml:98` |
| `--skip-version-check` | `.github/workflows/ci.yml:99` |
| `--format table` / `--output trivy-report/trivy-report.md` | `.github/workflows/ci.yml:100-101` |
| Ignore-file format (finding IDs, optional `exp:` dates) | `.trivyignore` (currently comments only — no accepted risks recorded) |

Eight flags and one subcommand, all stable across the 0.7x line.

#### Recommended action

**Bump**, at low urgency, to `0.74.0` with its digest, re-verifying the digest against the
registry at the time of the change. Keep the digest pin — it is the control that
neutralises the mechanism this advisory describes, and it is worth stating explicitly in
the workflow comment so a future bump does not silently drop it. The alternative,
**accept-with-reason**, is also defensible: the advisory has no fixed version, so bumping
does not clear it from OSV, and the digest pin already establishes artifact provenance.
If that path is taken, record `CVE-2026-33634` in `.trivyignore` with an expiry so the
acceptance is visible and time-boxed rather than implicit. Out of scope for this review.

### Go toolchain (`go.mod` `go` directive)

**Rating: Medium.** Dominant factor: **the declared toolchain floor (go1.26.0) has six
symbol-reachable stdlib advisories**, held to Medium because every automated build path in
the repository already floats past it.

#### Reasoning

`go.mod` declares `go 1.26.0`. go1.26.0 was published 2026-02-10; go1.26.7 was published
2026-08-18, seven patch releases and 6.3 months later. Running `govulncheck v1.7.0` with
the toolchain the directive selects produces:

| Level | Count |
|---|---|
| Symbol-reachable (your code calls the vulnerable symbol) | **6** |
| Package-imported, not called | 5 |
| Module-required, not imported | 22 |

The six reachable ones, with their traces as `govulncheck` reported them:

| Advisory | CVE | Package | Fixed in | Reachable via |
|---|---|---|---|---|
| GO-2026-4603 | CVE-2026-27142 | `html/template` — meta content URLs not escaped | go1.26.1 | `render.HTML` → `template.Template.Execute` (`internal/render/render.go:62`) |
| GO-2026-4865 | CVE-2026-32289 | `html/template` — JsBraceDepth context-tracking XSS | go1.26.2 | `render.init` → `template.Template.ParseFS` (`internal/render/render.go:39`); `render.HTML` (`internal/render/render.go:62`) |
| GO-2026-4980 | CVE-2026-39826 | `html/template` — escaper bypass → XSS | go1.26.3 | `render.HTML` → `template.Template.Execute` (`internal/render/render.go:62`) |
| GO-2026-4982 | CVE-2026-39823 | `html/template` — meta content URL escaping bypass → XSS | go1.26.3 | `render.HTML` → `template.Template.Execute` (`internal/render/render.go:62`) |
| GO-2026-6091 | CVE-2026-56858 | `html/template` — JavaScript regexp context tracking | **go1.26.6** | `render.HTML` → `template.Template.Execute` (`internal/render/render.go:62`) |
| GO-2026-6088 | CVE-2026-56859 | `encoding/xml` — missing recursion-depth guard on decode | **go1.26.6** | `cobertura.Parse` → `xml.Unmarshal` (`internal/cobertura/parser.go:84`); `junit.rootElement` → `xml.Token` (`internal/junit/parser.go:120`) |

The five `html/template` findings matter here in principle rather than in practice, but
the principle is not academic: `render.HTML` renders package names, file paths, and class
names taken from Cobertura and JUnit XML into an HTML page. A report generated from an XML
artifact produced by an untrusted build could carry attacker-influenced strings into that
template. Likewise `encoding/xml` decodes those artifacts directly, and GO-2026-6088 is a
recursion-depth issue on exactly that path.

**Why Medium and not High.** Nothing this repository builds automatically is affected:

- `.github/workflows/ci.yml:21`, `:173`, `:239` all set `go-version: "1.26"`, which
  setup-go resolves to the newest 1.26 patch — go1.26.7 today, past every fix above.
- `action.yml:139` uses `inputs.go-version`, defaulting to `"1.26"` (`action.yml:52`), so
  the composite Action's fallback build lands on the same patch.
- Released binaries are cross-compiled by GoReleaser inside that same job, so shipped
  artifacts are built with go1.26.7.

The residual exposure is a local build on a machine whose installed toolchain satisfies
`go 1.26.0` exactly — which is precisely what happened while producing this report: the
host had go1.25.0, `GOTOOLCHAIN=auto` fetched go1.26.0 verbatim from the directive, and
all six findings appeared. Nothing in the repository enforces a minimum patch level.

Reachability here is **verified**, not assumed — `govulncheck` ran at `scan_level: symbol`
in source mode, and the traces above are its own output.

#### API surface used

The stdlib surface that carries the findings:

| Symbol | Call site |
|---|---|
| `html/template.Must` / `.New` / `.Funcs` / `.ParseFS` | `internal/render/render.go:39` |
| `html/template.Template.Execute` | `internal/render/render.go:62` |
| `encoding/xml.Unmarshal` (Cobertura) | `internal/cobertura/parser.go:84` |
| `encoding/xml.Unmarshal` (JUnit `testsuites` / `testsuite`) | `internal/junit/parser.go:72`, `internal/junit/parser.go:78` |
| `encoding/xml.NewDecoder` / `Decoder.Token` / `xml.StartElement` | `internal/junit/parser.go:118`, `internal/junit/parser.go:120`, `internal/junit/parser.go:124` |
| `encoding/xml.Name` + `xml:"…"` struct tags | `internal/cobertura/parser.go:42`, `internal/junit/parser.go:25`, `internal/junit/parser.go:34` |

#### Recommended action

**Bump** the `go.mod` directive from `go 1.26.0` to `go 1.26.6` — the lowest floor that
excludes all six advisories, since GO-2026-6088 and GO-2026-6091 are only fixed there.
This changes nothing about what CI builds (already go1.26.7) and everything about what a
contributor's local `go build` and `go test` are permitted to use. It is a one-line
manifest edit and therefore out of scope for this review; raise it as a separate change.

Consider also bumping the pinned `govulncheck@v1.5.0` (`.github/workflows/ci.yml:40`) to
v1.7.0 in the same change — not because v1.5.0 misses advisories (the database is fetched
live) but to keep scanner and report reproducible against one another.

---

## Dependencies rated Low

Full evidence is recorded here for completeness; none requires action.

### `github.com/goccy/go-yaml` v1.19.2 — Low

The only module in `go.mod`, and the only third-party code compiled into the shipped
binary.

- **Version age:** pinned at v1.19.2, released 2026-01-08 — which is also the current
  latest. Zero releases behind. The 7.6 months since that release reflect upstream's
  cadence, not a stale pin here.
- **Advisories:** none. OSV returns no advisory for `github.com/goccy/go-yaml` at *any*
  version in its history, and `govulncheck` at symbol level reports nothing in this
  module. (The Go vulnerability database's `go-yaml` entries — GO-2020-0036, GO-2021-0061
  — belong to the unrelated `github.com/go-yaml/yaml` module, which this repository does
  not use.)
- **Project health:** not archived, not deprecated, not renamed or transferred. MIT.
  2.2k stars. Last release 2026-01-08; last commit on `master` 2026-04-07 (`edee2f91`),
  with 6 commits since the release tag. **Single-maintainer-dominant**: `goccy` has 605
  contributions against 22 for the next-highest of 77 contributors.
- **Reachability:** verified clean by `govulncheck` (no findings to reach).
- **Transitive dependencies:** none. `go list -m all` returns exactly this module and
  `go-yaml`.

**Why Low, not Medium:** the bus-factor and the 4.6-month commit gap are real signals, and
they are the reason this entry is documented rather than waved through. They are
outweighed by the two strongest facts available — the pin is at the latest release, and
the package has never had an advisory — combined with an API surface of three exported
symbols. If the next CVE arrives with no maintainer to fix it, the cost of moving is a
day, not a quarter. Re-evaluate to Medium if a release does not appear by roughly
2026-12, or immediately on any advisory.

**API surface used** — three exported symbols plus the struct-tag contract:

| Symbol | Call site |
|---|---|
| `yaml.UnmarshalWithOptions(data, &cfg, …)` | `internal/config/config.go:88` |
| `yaml.Strict()` (`DecodeOption`, enables `DisallowUnknownField`) | `internal/config/config.go:88` |
| `yaml.Unmarshal` | `internal/scaffold/scaffold_test.go:64` (test only) |
| Import alias `yaml "github.com/goccy/go-yaml"` | `internal/config/config.go:15`, `internal/scaffold/scaffold_test.go:10` |
| `yaml:"…"` struct tags — `display_name`, `prefix`, `strip_prefix`, `path`, `fail_on_drop`, `folder_group_depth`, `ignore_file`, `baseline`, `workspaces`, `display_from` | `internal/config/config.go:32`, `:35`, `:38`, `:45`, `:49`, `:54`-`:58` |
| `yaml:"jobs"` struct tag | `internal/scaffold/scaffold_test.go:62` |

One production call site. The dependency is load-bearing for `coverage.yaml` parsing but
narrow: the only feature used beyond plain unmarshalling is `yaml.Strict()`, and the
struct-tag spelling is shared with `gopkg.in/yaml.v3` and `sigs.k8s.io/yaml`, so a
migration would be a same-day change. **Recommended action: accept** — current, clean, and
cheap to leave alone.

### Go tools — Low

| Package | Evidence | Action |
|---|---|---|
| `gotest.tools/gotestsum` v1.13.0 | Pinned at latest (0 releases behind). Released 2025-09-11 — 11.5 months, but nothing newer exists. No OSV advisory. Not archived; last push 2026-04-15; 2.7k stars; Apache-2.0; `gotestyourself` org (multi-maintainer). Reachability: unverified (not scanned by `govulncheck`, which analyses this module's own code, not CI tools). | Accept |
| `github.com/boumenot/gocover-cobertura` v1.5.0 | Pinned at latest (0 releases behind), released 2026-05-15. No OSV advisory. Not archived; last push 2026-05-18; 156 stars; MIT; effectively single-maintainer. Reachability: unverified. Small project — worth periodic re-check, but current and clean. | Accept |
| `golang.org/x/vuln` v1.5.0 (`govulncheck`) | Released 2026-06-25; latest v1.7.0 (2026-08-13) — 2 releases / 1.7 months behind. No OSV advisory. Google-maintained, BSD-3-Clause, last push 2026-08-19. Being behind is a detection-precision gap, not a vulnerability: the advisory database is fetched live at run time. Reachability: n/a. | Bump opportunistically (see Go toolchain section) |

**API surface used:**

| Package | Surface | Call site |
|---|---|---|
| `gotestsum` | `go install gotest.tools/gotestsum@v1.13.0` | `.github/workflows/ci.yml:50` |
| `gotestsum` | `--junitfile tests-coverage.xml`, `--format pkgname`, `--` passthrough of `-coverprofile`/`-covermode`/`-count` | `.github/workflows/ci.yml:57-58` |
| `gocover-cobertura` | `go install github.com/boumenot/gocover-cobertura@v1.5.0` | `.github/workflows/ci.yml:51` |
| `gocover-cobertura` | stdin→stdout filter, no flags | `.github/workflows/ci.yml:59` |
| `govulncheck` | `go install golang.org/x/vuln/cmd/govulncheck@v1.5.0` | `.github/workflows/ci.yml:40` |
| `govulncheck` | `govulncheck ./...`, exit-code contract | `.github/workflows/ci.yml:41` |

All three are CLI-only with no Go API coupling; each is two lines of workflow.

### GitHub Actions — Low

All four are at the latest release, SHA-pinned, and SHA-verified against the upstream tag
object. None is archived or deprecated; all four are MIT; three are GitHub-maintained.
Advisories: none for any of the four. Reachability: unverified (no reachability-capable
scanner covers GitHub Actions). Dependabot's `github-actions` group
(`.github/dependabot.yml`) tracks all four weekly and updates both the SHA and the version
comment. **Recommended action: accept** for all four.

**API surface used** — the `with:` inputs actually set:

| Action | Inputs used | Call sites |
|---|---|---|
| `actions/checkout` | none at three sites; `fetch-depth: 0` at two | `.github/workflows/ci.yml:17`, `:83`, `:132`, `:167` (+`:169`), `:233` (+`:235`) |
| `actions/setup-go` | `go-version`, `cache` | `.github/workflows/ci.yml:19-22`, `:171-174`, `:237-240`; `action.yml:137-140` |
| `actions/upload-artifact` | `name`, `path`, `if-no-files-found`, `retention-days` | `.github/workflows/ci.yml:118-123`, `:183-190` |
| `goreleaser/goreleaser-action` | `version`, `args` | `.github/workflows/ci.yml:177-180`, `:243-246` |

Two inputs each at most. Every one of these is a thin wrapper around a shell command; none
is load-bearing in the sense of being hard to replace.

---

## Method and limitations

- **Manifest discovery.** The repository was swept for `go.mod`, `package.json`,
  `requirements*.txt`, `pyproject.toml`, `Cargo.toml`, `Gemfile`, `*.gemspec`, `pom.xml`,
  and `composer.json`. Only `go.mod` was found. The other four ecosystems in this report
  have **no manifest**: they are dependencies declared inline in
  `.github/workflows/ci.yml`, `action.yml`, and `.goreleaser.yaml`. They are included
  because they are real third-party code executing with repository credentials, and
  because a reader asking "what are we standing on" is not served by a report that stops
  at `go.mod`.
- **Transitive scope.** The full transitive tree is deliberately not enumerated
  package by package. The Go tree has no transitive entries at all. The npm tree has 80,
  of which only the two carrying advisories are listed, with the direct dependency
  responsible for each named. The container image's internal package tree is not
  enumerated; it is a pinned build-time tool, not a runtime dependency.
- **Reachability.** Verified only for Go, by `govulncheck` at symbol level in source mode.
  Marked **unverified** everywhere else, because no scanner run for this report performs
  reachability analysis on npm packages, GitHub Actions, or container images.
- **`docs/README.md` and the language guides** (`docs/GO.md`, `docs/PYTHON.md`,
  `docs/TYPESCRIPT.md`, `docs/RUST.md`, `docs/JAVA.md`, `docs/CSHARP.md`) document coverage
  *formats this tool consumes*, not dependencies of this repository. They introduce no
  ecosystem and are out of scope.
- **`THIRD_PARTY_NOTICES.md`** records that `internal/ignore/gitignore.go` is a vendored
  port of `github.com/sabhiram/go-gitignore` (MIT). It is inlined source, not a
  dependency: it appears in no manifest, is not fetched at build time, and cannot be
  bumped. It carries no advisory. It is noted here so the inventory is honest about it,
  but it is not rated — there is no version to be behind on. If upstream ever ships a
  security fix, nothing in this repository would surface it; that is the standing cost of
  the vendoring decision, and it was taken deliberately.
- Version and date facts were gathered on 2026-08-25 and will drift.
