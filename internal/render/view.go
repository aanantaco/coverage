package render

import "sort"

// viewRow is one rendered table row (workspace, total, or folder). It carries
// the raw numbers; formatting is done by the template funcs so Markdown and
// HTML share identical cell values.
type viewRow struct {
	Label    string // display name, or folder path for folder rows
	IsTotal  bool
	IsFolder bool

	Tests    string // summary rows only ("—" or a number)
	HasTests bool

	LinesCovered, LinesValid       int
	BranchesCovered, BranchesValid int
	Delta                          *Delta
}

// view is the format-agnostic model both templates render.
type view struct {
	HasBaseline      bool
	Summary          []viewRow // workspace rows followed by the total row
	Breakdown        []viewRow // workspace rows with their folders interleaved
	Regressions      []Regression
	Removed          []string
	IgnoreFileLoaded bool
	ExcludedFiles    int
}

// buildView derives the render model from a Summary, applying the same sorting
// and aggregation rules the report has always used.
func buildView(s Summary) view {
	workspaces := make([]Workspace, len(s.Workspaces))
	copy(workspaces, s.Workspaces)
	sort.Slice(workspaces, func(i, j int) bool {
		return workspaces[i].DisplayName < workspaces[j].DisplayName
	})

	v := view{
		HasBaseline:      s.HasBaseline,
		IgnoreFileLoaded: s.IgnoreFileLoaded,
		ExcludedFiles:    s.ExcludedFiles,
	}

	var tLV, tLC, tBV, tBC, tTests int
	anyTests := false
	for _, w := range workspaces {
		testsCell := "—"
		if w.HasTests {
			testsCell = itoa(w.Tests)
			tTests += w.Tests
			anyTests = true
		}
		v.Summary = append(v.Summary, viewRow{
			Label: w.DisplayName, Tests: testsCell, HasTests: w.HasTests,
			LinesCovered: w.LinesCovered, LinesValid: w.LinesValid,
			BranchesCovered: w.BranchesCovered, BranchesValid: w.BranchesValid,
			Delta: w.Delta,
		})

		v.Breakdown = append(v.Breakdown, viewRow{
			Label:        w.DisplayName,
			LinesCovered: w.LinesCovered, LinesValid: w.LinesValid,
			BranchesCovered: w.BranchesCovered, BranchesValid: w.BranchesValid,
			Delta: w.Delta,
		})
		for _, f := range sortedFolders(w.Folders) {
			v.Breakdown = append(v.Breakdown, viewRow{
				Label: f.Path, IsFolder: true,
				LinesCovered: f.LinesCovered, LinesValid: f.LinesValid,
				BranchesCovered: f.BranchesCovered, BranchesValid: f.BranchesValid,
				Delta: f.Delta,
			})
		}

		tLV += w.LinesValid
		tLC += w.LinesCovered
		tBV += w.BranchesValid
		tBC += w.BranchesCovered
	}

	totalTests := "—"
	if anyTests {
		totalTests = itoa(tTests)
	}
	v.Summary = append(v.Summary, viewRow{
		Label: "Total", IsTotal: true, Tests: totalTests, HasTests: anyTests,
		LinesCovered: tLC, LinesValid: tLV,
		BranchesCovered: tBC, BranchesValid: tBV,
		Delta: s.TotalDelta,
	})

	regs := make([]Regression, len(s.Regressions))
	copy(regs, s.Regressions)
	sort.Slice(regs, func(i, j int) bool { return regs[i].DisplayName < regs[j].DisplayName })
	v.Regressions = regs

	removed := make([]string, len(s.RemovedWorkspaces))
	copy(removed, s.RemovedWorkspaces)
	sort.Strings(removed)
	v.Removed = removed

	return v
}

// sortedFolders returns folders with LinesValid > 0, sorted by LinesValid
// descending then Path ascending.
func sortedFolders(in []Folder) []Folder {
	out := make([]Folder, 0, len(in))
	for _, f := range in {
		if f.LinesValid > 0 {
			out = append(out, f)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].LinesValid != out[j].LinesValid {
			return out[i].LinesValid > out[j].LinesValid
		}
		return out[i].Path < out[j].Path
	})
	return out
}
