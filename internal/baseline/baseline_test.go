package baseline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sampleSummary() *Summary {
	return &Summary{
		Schema:        SchemaVersion,
		GeneratedFrom: GeneratedFrom,
		Total:         Totals{LinesValid: 100, LinesCovered: 80, BranchesValid: 20, BranchesCovered: 10},
		Workspaces: map[string]WorkspaceSummary{
			"api": {
				Totals:  Totals{LinesValid: 100, LinesCovered: 80, BranchesValid: 20, BranchesCovered: 10},
				Folders: map[string]Totals{"src": {LinesValid: 100, LinesCovered: 80, BranchesValid: 20, BranchesCovered: 10}},
			},
		},
	}
}

func TestEmitIsReproducible(t *testing.T) {
	dir := t.TempDir()
	p1 := filepath.Join(dir, "a.json")
	p2 := filepath.Join(dir, "b.json")
	if err := Emit(sampleSummary(), p1); err != nil {
		t.Fatal(err)
	}
	if err := Emit(sampleSummary(), p2); err != nil {
		t.Fatal(err)
	}
	d1, _ := os.ReadFile(p1)
	d2, _ := os.ReadFile(p2)
	if string(d1) != string(d2) {
		t.Error("emit is not reproducible")
	}
	if strings.Contains(string(d1), "time") || strings.Contains(string(d1), "date") {
		t.Error("summary JSON should not contain timestamps")
	}
	if !strings.Contains(string(d1), `"schema": 1`) {
		t.Errorf("expected schema field, got:\n%s", d1)
	}
}

func TestEmitLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.json")
	if err := Emit(sampleSummary(), path); err != nil {
		t.Fatal(err)
	}
	got, found, err := Load(path)
	if err != nil || !found {
		t.Fatalf("Load: found=%v err=%v", found, err)
	}
	if got.Total.LinesCovered != 80 {
		t.Errorf("round trip mismatch: %+v", got.Total)
	}
}

func TestLoadMissingIsNotError(t *testing.T) {
	_, found, err := Load(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatalf("expected no error for missing baseline, got %v", err)
	}
	if found {
		t.Error("found should be false")
	}
}

func TestLoadMalformedErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	os.WriteFile(path, []byte("{not json"), 0o644)
	if _, _, err := Load(path); err == nil {
		t.Fatal("expected error for malformed baseline")
	}
}

func TestCompareDeltasAndRegressions(t *testing.T) {
	base := &Summary{
		Total: Totals{LinesValid: 100, LinesCovered: 90},
		Workspaces: map[string]WorkspaceSummary{
			"down":    {Totals: Totals{LinesValid: 100, LinesCovered: 90}},
			"removed": {Totals: Totals{LinesValid: 10, LinesCovered: 10}},
		},
	}
	current := &Summary{
		Total: Totals{LinesValid: 100, LinesCovered: 80},
		Workspaces: map[string]WorkspaceSummary{
			"down": {Totals: Totals{LinesValid: 100, LinesCovered: 80}},
			"new":  {Totals: Totals{LinesValid: 50, LinesCovered: 25}},
		},
	}

	c := Compare(current, base)

	if !c.Total.HasLine || c.Total.LinePP != -10 {
		t.Errorf("total delta = %+v", c.Total)
	}
	if c.TotalLineDropPP != 10 {
		t.Errorf("total drop = %v", c.TotalLineDropPP)
	}
	if !c.Workspaces["new"].IsNew {
		t.Error("new workspace should be marked new")
	}
	down := c.Workspaces["down"]
	if !down.HasLine || down.LinePP != -10 {
		t.Errorf("down delta = %+v", down)
	}
	if len(c.Regressions) != 1 || c.Regressions[0].ID != "down" {
		t.Errorf("regressions = %+v", c.Regressions)
	}
	if c.Regressions[0].OldPercent != 90 || c.Regressions[0].NewPercent != 80 || c.Regressions[0].DropPP != 10 {
		t.Errorf("regression detail = %+v", c.Regressions[0])
	}
	if len(c.Removed) != 1 || c.Removed[0] != "removed" {
		t.Errorf("removed = %+v", c.Removed)
	}
}

func TestEmitWriteError(t *testing.T) {
	// A path under a directory that does not exist -> WriteFile fails.
	bad := filepath.Join(t.TempDir(), "missing-dir", "s.json")
	if err := Emit(sampleSummary(), bad); err == nil {
		t.Fatal("expected a write error for a path in a nonexistent directory")
	}
}

func TestLoadReadErrorNotNotExist(t *testing.T) {
	// A directory is not ErrNotExist, so Load must surface it as an error.
	if _, _, err := Load(t.TempDir()); err == nil {
		t.Fatal("expected a read error when the path is a directory")
	}
}

// TestCompareBranchAndFolderDeltas exercises the branch-delta path in pctDelta,
// both folder branches (existing folder delta + new folder), and a brand-new
// workspace that carries folders.
func TestCompareBranchAndFolderDeltas(t *testing.T) {
	base := &Summary{
		Total: Totals{LinesValid: 100, LinesCovered: 80, BranchesValid: 20, BranchesCovered: 10},
		Workspaces: map[string]WorkspaceSummary{
			"api": {
				Totals: Totals{LinesValid: 100, LinesCovered: 80, BranchesValid: 20, BranchesCovered: 10},
				Folders: map[string]Totals{
					"src": {LinesValid: 50, LinesCovered: 40, BranchesValid: 10, BranchesCovered: 5},
				},
			},
		},
	}
	current := &Summary{
		Total: Totals{LinesValid: 100, LinesCovered: 90, BranchesValid: 20, BranchesCovered: 16},
		Workspaces: map[string]WorkspaceSummary{
			"api": {
				Totals: Totals{LinesValid: 100, LinesCovered: 90, BranchesValid: 20, BranchesCovered: 16},
				Folders: map[string]Totals{
					"src": {LinesValid: 50, LinesCovered: 45, BranchesValid: 10, BranchesCovered: 8},
					"cmd": {LinesValid: 10, LinesCovered: 5}, // new folder
				},
			},
			"web": { // new workspace, with folders
				Totals:  Totals{LinesValid: 40, LinesCovered: 20},
				Folders: map[string]Totals{"ui": {LinesValid: 40, LinesCovered: 20}},
			},
		},
	}

	c := Compare(current, base)

	if !c.Total.HasBranch || roundPP(c.Total.BranchPP) != 30 {
		t.Errorf("total branch delta = %+v (want +30pp)", c.Total)
	}
	api := c.Workspaces["api"]
	if !api.HasBranch || roundPP(api.BranchPP) != 30 {
		t.Errorf("api branch delta = %+v", api)
	}
	if src := api.Folders["src"]; !src.HasLine || roundPP(src.LinePP) != 10 {
		t.Errorf("src folder delta = %+v (want +10pp line)", src)
	}
	if !api.Folders["cmd"].IsNew {
		t.Errorf("cmd folder should be new: %+v", api.Folders["cmd"])
	}
	if !c.Workspaces["web"].IsNew {
		t.Error("web workspace should be marked new")
	}
	if len(c.Regressions) != 0 {
		t.Errorf("no regressions expected (coverage improved), got %+v", c.Regressions)
	}
}

func TestPctZeroValidIsZero(t *testing.T) {
	if got := pct(5, 0); got != 0 {
		t.Errorf("pct(5, 0) = %v, want 0", got)
	}
}

// TestCompareRegressionsSortedByID feeds two regressing workspaces in
// non-sorted order and asserts the output is deterministically ID-sorted.
func TestCompareRegressionsSortedByID(t *testing.T) {
	base := &Summary{Workspaces: map[string]WorkspaceSummary{
		"zebra": {Totals: Totals{LinesValid: 100, LinesCovered: 90}},
		"alpha": {Totals: Totals{LinesValid: 100, LinesCovered: 90}},
	}}
	current := &Summary{Workspaces: map[string]WorkspaceSummary{
		"zebra": {Totals: Totals{LinesValid: 100, LinesCovered: 70}},
		"alpha": {Totals: Totals{LinesValid: 100, LinesCovered: 80}},
	}}
	c := Compare(current, base)
	if len(c.Regressions) != 2 {
		t.Fatalf("expected 2 regressions, got %+v", c.Regressions)
	}
	if c.Regressions[0].ID != "alpha" || c.Regressions[1].ID != "zebra" {
		t.Errorf("regressions not ID-sorted: %+v", c.Regressions)
	}
}

func TestCompareZeroValidLinesNoDivideByZero(t *testing.T) {
	base := &Summary{Workspaces: map[string]WorkspaceSummary{
		"w": {Totals: Totals{LinesValid: 0, LinesCovered: 0}},
	}}
	current := &Summary{Workspaces: map[string]WorkspaceSummary{
		"w": {Totals: Totals{LinesValid: 0, LinesCovered: 0}},
	}}
	c := Compare(current, base)
	if c.Workspaces["w"].HasLine {
		t.Error("should not compute line delta with zero valid lines")
	}
	if len(c.Regressions) != 0 {
		t.Errorf("no regressions expected, got %+v", c.Regressions)
	}
}
