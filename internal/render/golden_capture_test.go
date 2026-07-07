package render

import (
	"os"
	"path/filepath"
	"testing"
)

// baselineSummary builds a summary exercising deltas, a regression callout,
// a new workspace, folder-level deltas, and a removed-workspace note.
func baselineSummary() Summary {
	s := exampleSummary()
	s.HasBaseline = true
	s.TotalDelta = &Delta{HasLine: true, LinePP: -1.0, HasBranch: true, BranchPP: 0.0}
	s.Workspaces[0].Delta = &Delta{HasLine: true, LinePP: 1.2, HasBranch: true, BranchPP: 0.5}
	s.Workspaces[0].Folders[0].Delta = &Delta{HasLine: true, LinePP: 1.2, HasBranch: true, BranchPP: 0.5}
	s.Workspaces[1].Delta = &Delta{HasLine: true, LinePP: -1.7, HasBranch: true, BranchPP: -0.3}
	s.Workspaces[1].Folders[0].Delta = &Delta{HasLine: true, LinePP: -1.7, HasBranch: true, BranchPP: -0.3}
	s.Workspaces[2].Delta = &Delta{IsNew: true}
	s.Regressions = []Regression{{DisplayName: "thinger", OldPercent: 84.8, NewPercent: 83.1, DropPP: 1.7}}
	s.RemovedWorkspaces = []string{"old-thing"}
	return s
}

func goldenCases() map[string]Summary {
	return map[string]Summary{
		"basic.md":    exampleSummary(),
		"baseline.md": baselineSummary(),
		"empty.md":    {},
	}
}

// TestCaptureGolden regenerates golden files when CAPTURE_GOLDEN=1; skipped
// otherwise.
func TestCaptureGolden(t *testing.T) {
	if os.Getenv("CAPTURE_GOLDEN") != "1" {
		t.Skip("set CAPTURE_GOLDEN=1 to regenerate golden files")
	}
	if err := os.MkdirAll("testdata", 0o755); err != nil {
		t.Fatal(err)
	}
	for name, s := range goldenCases() {
		if err := os.WriteFile(filepath.Join("testdata", name), []byte(Markdown(s)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// TestMarkdownGolden asserts the template output is byte-identical to the
// captured golden files (the pre-template Go renderer's output).
func TestMarkdownGolden(t *testing.T) {
	for name, s := range goldenCases() {
		want, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatalf("read golden %s: %v", name, err)
		}
		if got := Markdown(s); got != string(want) {
			t.Errorf("%s mismatch:\n--- got ---\n%q\n--- want ---\n%q", name, got, string(want))
		}
	}
}
