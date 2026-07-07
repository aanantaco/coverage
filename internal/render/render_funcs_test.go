package render

import (
	"strings"
	"testing"
)

func TestPercent(t *testing.T) {
	if got := percent(0, 0); got != "0.0%" {
		t.Errorf("percent(0,0) = %q", got)
	}
	if got := percent(1, 2); got != "50.0%" {
		t.Errorf("percent(1,2) = %q", got)
	}
}

func TestFormatDelta(t *testing.T) {
	cases := map[float64]string{
		1.24:  "▲ +1.2",
		-0.83: "▼ -0.8",
		0.0:   "▬ 0.0",
		0.04:  "▬ 0.0", // rounds to zero
	}
	for pp, want := range cases {
		if got := formatDelta(pp); got != want {
			t.Errorf("formatDelta(%v) = %q, want %q", pp, got, want)
		}
	}
}

func TestDeltaCells(t *testing.T) {
	if got := deltaLineCell(nil); got != "—" {
		t.Errorf("nil = %q", got)
	}
	if got := deltaLineCell(&Delta{IsNew: true}); got != "new" {
		t.Errorf("new = %q", got)
	}
	if got := deltaLineCell(&Delta{HasLine: false}); got != "—" {
		t.Errorf("no-line = %q", got)
	}
	if got := deltaLineCell(&Delta{HasLine: true, LinePP: 1.2}); got != "▲ +1.2" {
		t.Errorf("value = %q", got)
	}
	if got := deltaBranchCell(nil); got != "—" {
		t.Errorf("branch nil = %q", got)
	}
	if got := deltaBranchCell(&Delta{IsNew: true}); got != "new" {
		t.Errorf("branch new = %q", got)
	}
	if got := deltaBranchCell(&Delta{HasBranch: false}); got != "—" {
		t.Errorf("branch no-data = %q", got)
	}
	if got := deltaBranchCell(&Delta{HasBranch: true, BranchPP: -0.3}); got != "▼ -0.3" {
		t.Errorf("branch value = %q", got)
	}
}

func TestDeltaClass(t *testing.T) {
	cases := []struct {
		d    *Delta
		has  bool
		want string
	}{
		{nil, false, "na"},
		{&Delta{IsNew: true}, false, "new"},
		{&Delta{HasLine: true, LinePP: 1.0}, true, "up"},
		{&Delta{HasLine: true, LinePP: -1.0}, true, "down"},
		{&Delta{HasLine: true, LinePP: 0.0}, true, "flat"},
		{&Delta{HasLine: false}, false, "na"},
	}
	for _, c := range cases {
		if got := deltaClass(c.d, c.has, linePP); got != c.want {
			t.Errorf("deltaClass(%+v, %v) = %q, want %q", c.d, c.has, got, c.want)
		}
	}
	// branchPPv accessor path.
	if got := deltaClass(&Delta{HasBranch: true, BranchPP: 2.0}, true, branchPPv); got != "up" {
		t.Errorf("branch up = %q", got)
	}
}

func TestFmtDrop(t *testing.T) {
	if got := fmtDrop(1.7); got != "1.7" {
		t.Errorf("positive = %q", got)
	}
	if got := fmtDrop(-1.7); got != "1.7" {
		t.Errorf("negative should be magnitude = %q", got)
	}
}

func TestPluralize(t *testing.T) {
	if got := pluralize(1, "workspace", "workspaces"); got != "workspace" {
		t.Errorf("singular = %q", got)
	}
	if got := pluralize(2, "workspace", "workspaces"); got != "workspaces" {
		t.Errorf("plural = %q", got)
	}
}

func TestSortedFolders(t *testing.T) {
	in := []Folder{
		{Path: "z", LinesValid: 10},
		{Path: "empty", LinesValid: 0}, // omitted
		{Path: "a", LinesValid: 10},    // tie with z -> path asc
		{Path: "big", LinesValid: 50},
	}
	got := sortedFolders(in)
	want := []string{"big", "a", "z"}
	if len(got) != len(want) {
		t.Fatalf("got %d folders, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].Path != w {
			t.Errorf("position %d = %q, want %q", i, got[i].Path, w)
		}
	}
}

// TestMarkdownPluralRegressions exercises the "N workspaces" plural path.
func TestMarkdownPluralRegressions(t *testing.T) {
	s := exampleSummary()
	s.HasBaseline = true
	s.Regressions = []Regression{
		{DisplayName: "a", OldPercent: 80, NewPercent: 78, DropPP: 2},
		{DisplayName: "b", OldPercent: 90, NewPercent: 85, DropPP: 5},
	}
	if got := Markdown(s); !strings.Contains(got, "in 2 workspaces:") {
		t.Errorf("expected plural callout, got:\n%s", got)
	}
	if got := HTML(s); !strings.Contains(got, "in 2 workspaces:") {
		t.Errorf("expected plural callout in HTML")
	}
}
