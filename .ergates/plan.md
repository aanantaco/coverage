# Plan — task 06G3EKQRDRG0F41287BY7KC6EG [Y7KC6EG]
<!-- ergates:plan v1 -->

## Sections
- [ ] S1: Build evidence — staticcheck U1000 + exhaustive git grep census  ← next
- [ ] S2: Delete demonstrably unreferenced symbols (+ freed imports/branches)
- [ ] S3: Verify with `go build ./...`, `go test ./...`, `go vet`, staticcheck
- [ ] S4: PR description listing deletions + human-review candidates

## Criteria map
- AC1 "`go build ./...` passes" → S3
- AC2 "`go test ./...` passes" → S3
- AC3 "each deleted symbol verified unreferenced via git grep" → S1, S2
- AC4 "generated files untouched" → S1 (census found none in this repo)
- AC5 "cobertura Merge/Report/Class/Line preserved" → S1, S2
- AC6 "documented exported symbols preserved; ambiguous ones listed" → S1, S4
- AC7 "freed imports and stranded branches removed" → S2
- AC8 "PR lists deletions + exported candidates" → S4

## Notes for successor
- Repo is ~2.1k non-test Go lines, module `github.com/aanantaco/coverage`,
  zero generated files (no `DO NOT EDIT` header anywhere, no sqlc/protobuf/mocks).
- staticcheck 2025.1.1 is installed at `~/go/bin/staticcheck` and DOES work on
  this toolchain (verified by planting a dummy unused func). `-checks U1000`
  over `./...` reports NOTHING, with or without `-tests=false`. So there are no
  wholly-unreferenced top-level symbols; `unused` also does not flag
  write-only struct fields, which is where the real dead code is.
- A per-symbol census (`ergates-symbols outline` over every non-test .go file,
  then `git grep -c` per name) confirmed every top-level func/type/const/var
  has at least one reference outside its declaration, and none are
  test-only-referenced. Do not go hunting for dead functions; there are none.
- Confirmed dead, and the whole of the deletion:
  - `internal/render/view.go` `viewRow.HasTests` — assigned twice in
    `buildView`, never read. Neither template mentions `.HasTests`
    (`report.md.tmpl` / `report.html.tmpl` read `.Tests`, the preformatted
    string). Distinct from `render.Workspace.HasTests` (types.go), which IS
    read at view.go:51 — do not confuse the two same-named fields.
  - `internal/baseline/baseline.go` `Comparison.HasTotalLine` — assigned in
    `Compare`, never read. `app.checkFailOnDrop` uses
    `summary.TotalDelta.HasLine` instead.
- Deliberately LEFT IN PLACE (say why in the PR, do not "fix" them):
  - `baseline.Comparison.TotalLineDropPP` — read only by
    `baseline_test.go:100`, so it is referenced; production reads
    `render.Summary.TotalDelta.LinePP` instead.
  - `junit.Report.Failures/Errors/Skipped` — the app only reads `.Tests`, but
    `junit/parser_test.go` asserts all four, so they are referenced.
  - `baseline.pctDelta`'s `isNew bool` param: all three call sites pass the
    literal `false`, so `if isNew { return d }` is unreachable. Removing it is
    a signature change, which the task puts out of scope.
  - Everything in `internal/cobertura` (`Merge`/`Report`/`Class`/`Line` are all
    reachable from `app.aggregateCoverage`).
- Nothing in the deletion frees an import or strands a branch; `anyTests` in
  `buildView` stays, it still feeds the Total row's `Tests` cell.
