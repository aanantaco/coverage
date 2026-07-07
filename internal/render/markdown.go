// Package render turns an aggregated coverage Summary into a Markdown report
// suitable for $GITHUB_STEP_SUMMARY.
package render

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// Delta describes a coverage change versus a baseline, in percentage points.
type Delta struct {
	// IsNew marks an entity present now but absent from the baseline.
	IsNew bool
	// HasLine is true when a line-% delta could be computed (both sides had
	// valid lines).
	HasLine bool
	LinePP  float64
	// HasBranch is true when a branch-% delta could be computed.
	HasBranch bool
	BranchPP  float64
}

// Folder is one folder group within a workspace.
type Folder struct {
	Path            string
	LinesValid      int
	LinesCovered    int
	BranchesValid   int
	BranchesCovered int
	// Delta is optional; non-nil only when a baseline was supplied.
	Delta *Delta
}

// Workspace is a single reporting unit (e.g. a service or app).
type Workspace struct {
	ID              string
	DisplayName     string
	LinesValid      int
	LinesCovered    int
	BranchesValid   int
	BranchesCovered int
	Tests           int
	HasTests        bool
	Folders         []Folder
	// Delta is optional; non-nil only when a baseline was supplied.
	Delta *Delta
}

// Regression names a workspace whose line coverage decreased versus baseline.
type Regression struct {
	DisplayName string
	OldPercent  float64
	NewPercent  float64
	DropPP      float64
}

// Summary is the complete input to Markdown.
type Summary struct {
	Workspaces       []Workspace
	ExcludedFiles    int
	IgnoreFileLoaded bool

	// Baseline-related fields; populated only when a baseline was supplied.
	HasBaseline       bool
	TotalDelta        *Delta
	RemovedWorkspaces []string
	Regressions       []Regression
}

// Markdown renders the summary. It is a pure function of its input.
func Markdown(s Summary) string {
	if len(s.Workspaces) == 0 {
		return "_No coverage artifacts found._\n"
	}

	workspaces := make([]Workspace, len(s.Workspaces))
	copy(workspaces, s.Workspaces)
	sort.Slice(workspaces, func(i, j int) bool {
		return workspaces[i].DisplayName < workspaces[j].DisplayName
	})

	var b strings.Builder
	b.WriteString("## Test Coverage\n\n")

	writeRegressionCallout(&b, s)
	writeSummaryTable(&b, workspaces, s)
	writeBreakdownTable(&b, workspaces, s)
	writeRemoved(&b, s)
	writeFooter(&b, s)

	return b.String()
}

func writeRegressionCallout(b *strings.Builder, s Summary) {
	if len(s.Regressions) == 0 {
		return
	}
	regs := make([]Regression, len(s.Regressions))
	copy(regs, s.Regressions)
	sort.Slice(regs, func(i, j int) bool { return regs[i].DisplayName < regs[j].DisplayName })

	plural := "workspace"
	if len(regs) != 1 {
		plural = "workspaces"
	}
	fmt.Fprintf(b, "> ⚠️ **Coverage decreased** in %d %s:\n", len(regs), plural)
	for _, r := range regs {
		fmt.Fprintf(b, "> - `%s`: %s → %s (▼ %spp)\n",
			r.DisplayName, percentValue(r.OldPercent), percentValue(r.NewPercent), percentValueDrop(r.DropPP))
	}
	b.WriteString("\n")
}

// percentValue formats an already-computed percentage value like "79.7%".
func percentValue(v float64) string {
	return fmt.Sprintf("%.1f%%", v)
}

// percentValueDrop formats a drop magnitude (always positive) with one decimal.
func percentValueDrop(dropPP float64) string {
	if dropPP < 0 {
		dropPP = -dropPP
	}
	return fmt.Sprintf("%.1f", dropPP)
}

func writeSummaryTable(b *strings.Builder, workspaces []Workspace, s Summary) {
	b.WriteString("### Summary\n\n")

	if s.HasBaseline {
		b.WriteString("| Workspace | Tests | Lines | % | Δ | Branches | % | Δ |\n")
		b.WriteString("|---|---|---|---|---|---|---|---|\n")
	} else {
		b.WriteString("| Workspace | Tests | Lines | % | Branches | % |\n")
		b.WriteString("|---|---|---|---|---|---|\n")
	}

	var tLV, tLC, tBV, tBC, tTests int
	anyTests := false
	for _, w := range workspaces {
		writeSummaryRow(b, w, s.HasBaseline)
		tLV += w.LinesValid
		tLC += w.LinesCovered
		tBV += w.BranchesValid
		tBC += w.BranchesCovered
		if w.HasTests {
			tTests += w.Tests
			anyTests = true
		}
	}

	// Total row.
	testsCell := "—"
	if anyTests {
		testsCell = fmt.Sprintf("%d", tTests)
	}
	if s.HasBaseline {
		fmt.Fprintf(b, "| **Total** | **%s** | %s | **%s** | %s | %s | %s | %s |\n",
			testsCell,
			linesCell(tLC, tLV), percent(tLC, tLV), deltaLineCell(s.TotalDelta),
			branchesCell(tBC, tBV), percentOrDash(tBC, tBV, true), deltaBranchCell(s.TotalDelta))
	} else {
		fmt.Fprintf(b, "| **Total** | **%s** | %s | **%s** | %s | %s |\n",
			testsCell,
			linesCell(tLC, tLV), percent(tLC, tLV),
			branchesCell(tBC, tBV), percentOrDash(tBC, tBV, true))
	}
	b.WriteString("\n")
}

func writeSummaryRow(b *strings.Builder, w Workspace, hasBaseline bool) {
	testsCell := "—"
	if w.HasTests {
		testsCell = fmt.Sprintf("%d", w.Tests)
	}
	if hasBaseline {
		fmt.Fprintf(b, "| %s | %s | %s | %s | %s | %s | %s | %s |\n",
			w.DisplayName, testsCell,
			linesCell(w.LinesCovered, w.LinesValid), percent(w.LinesCovered, w.LinesValid), deltaLineCell(w.Delta),
			branchesCell(w.BranchesCovered, w.BranchesValid), percentOrDash(w.BranchesCovered, w.BranchesValid, false), deltaBranchCell(w.Delta))
	} else {
		fmt.Fprintf(b, "| %s | %s | %s | %s | %s | %s |\n",
			w.DisplayName, testsCell,
			linesCell(w.LinesCovered, w.LinesValid), percent(w.LinesCovered, w.LinesValid),
			branchesCell(w.BranchesCovered, w.BranchesValid), percentOrDash(w.BranchesCovered, w.BranchesValid, false))
	}
}

func writeBreakdownTable(b *strings.Builder, workspaces []Workspace, s Summary) {
	b.WriteString("### Breakdown by folder\n\n")

	if s.HasBaseline {
		b.WriteString("| Workspace / Folder | Lines | % | Δ | Branches | % | Δ |\n")
		b.WriteString("|---|---|---|---|---|---|---|\n")
	} else {
		b.WriteString("| Workspace / Folder | Lines | % | Branches | % |\n")
		b.WriteString("|---|---|---|---|---|\n")
	}

	for _, w := range workspaces {
		if s.HasBaseline {
			fmt.Fprintf(b, "| **%s** | %s | %s | %s | %s | %s | %s |\n",
				w.DisplayName,
				linesCell(w.LinesCovered, w.LinesValid), percent(w.LinesCovered, w.LinesValid), deltaLineCell(w.Delta),
				branchesCell(w.BranchesCovered, w.BranchesValid), percentOrDash(w.BranchesCovered, w.BranchesValid, false), deltaBranchCell(w.Delta))
		} else {
			fmt.Fprintf(b, "| **%s** | %s | %s | %s | %s |\n",
				w.DisplayName,
				linesCell(w.LinesCovered, w.LinesValid), percent(w.LinesCovered, w.LinesValid),
				branchesCell(w.BranchesCovered, w.BranchesValid), percentOrDash(w.BranchesCovered, w.BranchesValid, false))
		}

		folders := sortedFolders(w.Folders)
		for _, f := range folders {
			if s.HasBaseline {
				fmt.Fprintf(b, "| └ %s | %s | %s | %s | %s | %s | %s |\n",
					f.Path,
					linesCell(f.LinesCovered, f.LinesValid), percent(f.LinesCovered, f.LinesValid), deltaLineCell(f.Delta),
					branchesCell(f.BranchesCovered, f.BranchesValid), percentOrDash(f.BranchesCovered, f.BranchesValid, false), deltaBranchCell(f.Delta))
			} else {
				fmt.Fprintf(b, "| └ %s | %s | %s | %s | %s |\n",
					f.Path,
					linesCell(f.LinesCovered, f.LinesValid), percent(f.LinesCovered, f.LinesValid),
					branchesCell(f.BranchesCovered, f.BranchesValid), percentOrDash(f.BranchesCovered, f.BranchesValid, false))
			}
		}
	}
	b.WriteString("\n")
}

func writeRemoved(b *strings.Builder, s Summary) {
	if len(s.RemovedWorkspaces) == 0 {
		return
	}
	removed := make([]string, len(s.RemovedWorkspaces))
	copy(removed, s.RemovedWorkspaces)
	sort.Strings(removed)
	fmt.Fprintf(b, "_Removed since baseline: %s_\n\n", strings.Join(removed, ", "))
}

func writeFooter(b *strings.Builder, s Summary) {
	if s.IgnoreFileLoaded {
		fmt.Fprintf(b, "_Excluded paths from `.coverageignore`: %d files_\n", s.ExcludedFiles)
	}
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

// linesCell renders the "covered / total" cell used in the Lines column.
func linesCell(covered, total int) string {
	return fmt.Sprintf("%d / %d", covered, total)
}

// branchesCell renders the "covered/total" cell (no spaces), or "—" when there
// is no branch data.
func branchesCell(covered, total int) string {
	if total == 0 {
		return "—"
	}
	return fmt.Sprintf("%d/%d", covered, total)
}

// percent returns a one-decimal percentage, or "0.0%" when total is 0.
func percent(covered, total int) string {
	if total == 0 {
		return "0.0%"
	}
	return fmt.Sprintf("%.1f%%", float64(covered)/float64(total)*100)
}

// percentOrDash renders a branch percentage, "—" when there is no branch data.
// The bold flag wraps the value for the Total row.
func percentOrDash(covered, total int, bold bool) string {
	if total == 0 {
		return "—"
	}
	v := fmt.Sprintf("%.1f%%", float64(covered)/float64(total)*100)
	if bold {
		return "**" + v + "**"
	}
	return v
}

// deltaLineCell renders the line-% delta cell.
func deltaLineCell(d *Delta) string {
	if d == nil {
		return "—"
	}
	if d.IsNew {
		return "new"
	}
	if !d.HasLine {
		return "—"
	}
	return formatDelta(d.LinePP)
}

// deltaBranchCell renders the branch-% delta cell.
func deltaBranchCell(d *Delta) string {
	if d == nil {
		return "—"
	}
	if d.IsNew {
		return "new"
	}
	if !d.HasBranch {
		return "—"
	}
	return formatDelta(d.BranchPP)
}

// formatDelta renders a percentage-point change with a direction marker.
func formatDelta(pp float64) string {
	rounded := math.Round(pp*10) / 10
	switch {
	case rounded > 0:
		return fmt.Sprintf("▲ +%.1f", rounded)
	case rounded < 0:
		return fmt.Sprintf("▼ %.1f", rounded)
	default:
		return "▬ 0.0"
	}
}
