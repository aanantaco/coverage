# Coverage deltas across runs (regression detection)

`coverage` can compare the current run against a **baseline** from an earlier run
and show what changed — per-workspace and per-folder Δ columns, a "coverage
decreased" callout, and `new` / removed markers. It can also fail the build when
total coverage drops beyond a threshold.

The tool itself is stateless. "Across runs" works by **persisting one small JSON
file** (`coverage-summary.json`) and feeding it back on the next run. How you
persist it is a CI choice; three options are below.

## 1. Emit the baseline

Add `--emit-json` to write a machine-readable summary next to the Markdown:

```bash
coverage --input ./cov --output "$GITHUB_STEP_SUMMARY" \
  --emit-json coverage-summary.json
```

The JSON is **reproducible** — no timestamps or run-specific data — so identical
coverage produces an identical file (clean diffs).

## 2. Diff against it

On a later run, pass the previous summary as the baseline:

```bash
coverage --input ./cov --output "$GITHUB_STEP_SUMMARY" \
  --baseline coverage-summary.json \
  --emit-json coverage-summary.json      # emit the new one too
```

This adds Δ columns to both tables:

- `▲ +1.2` / `▼ -0.8` / `▬ 0.0` — percentage-point change (one decimal)
- `new` — a workspace/folder present now but absent from the baseline
- a `_Removed since baseline: …_` note for workspaces gone since the baseline
- a callout listing any workspace whose line coverage dropped

A missing baseline file is **not** an error (normal on a new branch) — regression
detection is simply skipped. A schema mismatch is skipped with a warning.

## 3. Optionally fail on a drop

```bash
coverage --input ./cov --baseline coverage-summary.json --fail-on-drop 0.5
```

Exits non-zero if **total** line coverage drops by more than 0.5 percentage
points. The report is always written **before** the failing exit, so the summary
still renders. Unset (the default) = annotate only, never fail.

You can also set these in `coverage.yaml`:

```yaml
baseline:
  path: coverage-summary.json
  fail_on_drop: 0.5      # omit to annotate without failing
```

CLI flags override the config block.

---

## Where the baseline comes from

The tool only needs a path to a baseline JSON. Pick one persistence strategy.

### Recommended — default-branch cache (no repo commits)

Save the baseline to the Actions cache on `main`; restore it on PRs. GitHub lets
a PR read caches created on its base branch, so the prefix restore picks up the
latest `main` baseline.

```yaml
# On the report job:
- name: Restore coverage baseline
  uses: actions/cache/restore@v4
  with:
    path: coverage-summary.json
    key: coverage-baseline-${{ github.sha }}
    restore-keys: coverage-baseline-        # newest matching prefix wins

- name: Report
  run: |
    coverage --input ./cov --output "$GITHUB_STEP_SUMMARY" \
      --baseline coverage-summary.json \
      --emit-json coverage-summary.json \
      --fail-on-drop 0.5

# Refresh the baseline only on main:
- name: Save coverage baseline
  if: github.ref == 'refs/heads/main'
  uses: actions/cache/save@v4
  with:
    path: coverage-summary.json
    key: coverage-baseline-${{ github.sha }}
```

### Alternative — main-branch artifact

On pushes to `main`, upload `coverage-summary.json` as a versioned artifact; on
PRs, download the latest one (via the GitHub API or a download-artifact action)
and pass it with `--baseline`. Similar to the cache approach, no repo commits.

### Alternative — committed baseline

Commit `.coverage-baseline.json` to the repo and refresh it on merges to `main`
via a bot commit. Simple and diffable in PRs, but noisier history. Point
`baseline.path` at the committed file.

---

## Summary JSON shape

```json
{
  "schema": 1,
  "generated_from": "coverage",
  "total": { "lines_valid": 6390, "lines_covered": 5282, "branches_valid": 930, "branches_covered": 695 },
  "workspaces": {
    "thingy": {
      "lines_valid": 2310, "lines_covered": 1842,
      "branches_valid": 310, "branches_covered": 215,
      "folders": {
        "src/api/thing": { "lines_valid": 2000, "lines_covered": 1500, "branches_valid": 250, "branches_covered": 170 }
      }
    }
  }
}
```

Deltas are matched by workspace id, and by `workspace + folder path` for folders.
