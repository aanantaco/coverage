package render

import (
	"strings"
	"testing"
)

func exampleSummary() Summary {
	return Summary{
		Workspaces: []Workspace{
			{
				DisplayName: "thingy", LinesValid: 2310, LinesCovered: 1842,
				BranchesValid: 310, BranchesCovered: 215, Tests: 412, HasTests: true,
				Folders: []Folder{{Path: "src/api/thing", LinesValid: 2000, LinesCovered: 1500, BranchesValid: 250, BranchesCovered: 170}},
			},
			{
				DisplayName: "thinger", LinesValid: 3680, LinesCovered: 3120,
				BranchesValid: 620, BranchesCovered: 480, Tests: 175, HasTests: true,
				Folders: []Folder{{Path: "internal/worker", LinesValid: 3680, LinesCovered: 3120, BranchesValid: 620, BranchesCovered: 480}},
			},
			{
				DisplayName: "shared/widget", LinesValid: 400, LinesCovered: 320,
				HasTests: false,
			},
		},
		ExcludedFiles:    12,
		IgnoreFileLoaded: true,
	}
}

func TestMarkdownNoArtifacts(t *testing.T) {
	got := Markdown(Summary{})
	if got != "_No coverage artifacts found._\n" {
		t.Errorf("got %q", got)
	}
}

func TestMarkdownBasics(t *testing.T) {
	got := Markdown(exampleSummary())

	wants := []string{
		"## Test Coverage",
		"| thingy | 412 | 1842 / 2310 | 79.7% | 215/310 | 69.4% |",
		"| thinger | 175 | 3120 / 3680 | 84.8% | 480/620 | 77.4% |",
		"| shared/widget | — | 320 / 400 | 80.0% | — | — |",
		"| **Total** | **587** | 5282 / 6390 | **82.7%** | 695/930 | **74.7%** |",
		"| └ src/api/thing | 1500 / 2000 | 75.0% | 170/250 | 68.0% |",
		"_Excluded paths from `.coverageignore`: 12 files_",
	}
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Errorf("output missing line:\n%s\n---full---\n%s", w, got)
		}
	}
}

func TestMarkdownNoFooterWhenIgnoreNotLoaded(t *testing.T) {
	s := exampleSummary()
	s.IgnoreFileLoaded = false
	got := Markdown(s)
	if strings.Contains(got, "Excluded paths") {
		t.Error("footer should be omitted when ignore file not loaded")
	}
}

func TestMarkdownOmitsZeroLineFolders(t *testing.T) {
	s := Summary{Workspaces: []Workspace{{
		DisplayName: "w", LinesValid: 10, LinesCovered: 5,
		Folders: []Folder{
			{Path: "keep", LinesValid: 10, LinesCovered: 5},
			{Path: "drop", LinesValid: 0, LinesCovered: 0},
		},
	}}}
	got := Markdown(s)
	if !strings.Contains(got, "└ keep") {
		t.Error("expected non-empty folder to render")
	}
	if strings.Contains(got, "└ drop") {
		t.Error("zero-line folder should be omitted")
	}
}

func TestMarkdownDeltaColumnsAndCallout(t *testing.T) {
	s := exampleSummary()
	s.HasBaseline = true
	s.TotalDelta = &Delta{HasLine: true, LinePP: -1.0, HasBranch: true, BranchPP: 0.0}
	s.Workspaces[0].Delta = &Delta{HasLine: true, LinePP: 1.2, HasBranch: true, BranchPP: 0.5}
	s.Workspaces[1].Delta = &Delta{HasLine: true, LinePP: -1.7, HasBranch: true, BranchPP: -0.3}
	s.Workspaces[2].Delta = &Delta{IsNew: true}
	s.Regressions = []Regression{{DisplayName: "thinger", OldPercent: 84.8, NewPercent: 83.1, DropPP: 1.7}}

	got := Markdown(s)
	for _, w := range []string{
		"| Workspace | Tests | Lines | % | Δ | Branches | % | Δ |",
		"▲ +1.2",
		"▼ -1.7",
		"▬ 0.0",
		"new",
		"> ⚠️ **Coverage decreased** in 1 workspace:",
		"> - `thinger`: 84.8% → 83.1% (▼ 1.7pp)",
	} {
		if !strings.Contains(got, w) {
			t.Errorf("delta output missing:\n%s\n---full---\n%s", w, got)
		}
	}
}

func TestMarkdownRemovedNote(t *testing.T) {
	s := exampleSummary()
	s.HasBaseline = true
	s.RemovedWorkspaces = []string{"old-service"}
	got := Markdown(s)
	if !strings.Contains(got, "_Removed since baseline: old-service_") {
		t.Errorf("expected removed note, got:\n%s", got)
	}
}
