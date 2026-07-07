// Package baseline emits and loads the machine-readable coverage summary JSON
// used for regression detection, and computes deltas between a current run and
// a previous baseline.
//
// The emitted JSON contains no timestamps or run-specific data, so identical
// coverage produces an identical baseline (clean diffs).
package baseline

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"sort"
)

// SchemaVersion is the version stamped into and expected from summary JSON.
const SchemaVersion = 1

// GeneratedFrom identifies the producer in the summary JSON.
const GeneratedFrom = "coverage"

// Totals is a set of line/branch counts.
type Totals struct {
	LinesValid      int `json:"lines_valid"`
	LinesCovered    int `json:"lines_covered"`
	BranchesValid   int `json:"branches_valid"`
	BranchesCovered int `json:"branches_covered"`
}

// WorkspaceSummary is a workspace's totals plus its folder breakdown.
type WorkspaceSummary struct {
	Totals
	Folders map[string]Totals `json:"folders"`
}

// Summary is the full machine-readable coverage summary.
type Summary struct {
	Schema        int                         `json:"schema"`
	GeneratedFrom string                      `json:"generated_from"`
	Total         Totals                      `json:"total"`
	Workspaces    map[string]WorkspaceSummary `json:"workspaces"`
}

// Emit writes s to path as indented JSON. Map keys are emitted in sorted order
// by encoding/json, so the output is reproducible.
func Emit(s *Summary, path string) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal summary json: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write summary json %q: %w", path, err)
	}
	return nil
}

// Load reads a baseline summary from path. found is false when the file does
// not exist (a normal first-run condition), in which case err is nil. A file
// that exists but cannot be parsed is an error.
func Load(path string) (s *Summary, found bool, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("read baseline %q: %w", path, err)
	}
	var parsed Summary
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, false, fmt.Errorf("parse baseline %q: %w", path, err)
	}
	return &parsed, true, nil
}

// PctDelta is a coverage change in percentage points.
type PctDelta struct {
	// IsNew marks an entity present now but absent from the baseline.
	IsNew bool
	// HasLine is true when a line-% delta could be computed.
	HasLine bool
	LinePP  float64
	// HasBranch is true when a branch-% delta could be computed.
	HasBranch bool
	BranchPP  float64
}

// WorkspaceDelta is a workspace-level delta plus per-folder deltas.
type WorkspaceDelta struct {
	PctDelta
	Folders map[string]PctDelta
}

// Regression records a workspace whose line coverage decreased.
type Regression struct {
	ID         string
	OldPercent float64
	NewPercent float64
	DropPP     float64
}

// Comparison is the result of comparing a current summary to a baseline.
type Comparison struct {
	Total       PctDelta
	Workspaces  map[string]WorkspaceDelta
	Removed     []string
	Regressions []Regression
	// TotalLineDropPP is the magnitude of a total line-% drop (>= 0). Zero when
	// coverage held or improved, or when a total delta was not computable.
	TotalLineDropPP float64
	HasTotalLine    bool
}

// Compare computes deltas of current versus base.
func Compare(current, base *Summary) *Comparison {
	c := &Comparison{
		Workspaces: map[string]WorkspaceDelta{},
	}

	c.Total = pctDelta(current.Total, base.Total, false)
	if c.Total.HasLine && c.Total.LinePP < 0 {
		c.TotalLineDropPP = -c.Total.LinePP
	}
	c.HasTotalLine = c.Total.HasLine

	for id, cur := range current.Workspaces {
		baseWS, ok := base.Workspaces[id]
		wd := WorkspaceDelta{Folders: map[string]PctDelta{}}
		if !ok {
			wd.PctDelta = PctDelta{IsNew: true}
		} else {
			wd.PctDelta = pctDelta(cur.Totals, baseWS.Totals, false)
			if wd.HasLine && roundPP(wd.LinePP) < 0 {
				c.Regressions = append(c.Regressions, Regression{
					ID:         id,
					OldPercent: pct(baseWS.LinesCovered, baseWS.LinesValid),
					NewPercent: pct(cur.LinesCovered, cur.LinesValid),
					DropPP:     -wd.LinePP,
				})
			}
		}
		for path, curF := range cur.Folders {
			baseF, ok := baseWS.Folders[path]
			if !ok {
				wd.Folders[path] = PctDelta{IsNew: true}
				continue
			}
			wd.Folders[path] = pctDelta(curF, baseF, false)
		}
		c.Workspaces[id] = wd
	}

	// Removed workspaces: present in baseline, absent now.
	for id := range base.Workspaces {
		if _, ok := current.Workspaces[id]; !ok {
			c.Removed = append(c.Removed, id)
		}
	}
	sort.Strings(c.Removed)
	sort.Slice(c.Regressions, func(i, j int) bool { return c.Regressions[i].ID < c.Regressions[j].ID })

	return c
}

// pctDelta computes a line/branch percentage-point delta between cur and base.
// isNew is threaded for callers that already know the entity is new.
func pctDelta(cur, base Totals, isNew bool) PctDelta {
	d := PctDelta{IsNew: isNew}
	if isNew {
		return d
	}
	if cur.LinesValid > 0 && base.LinesValid > 0 {
		d.HasLine = true
		d.LinePP = pct(cur.LinesCovered, cur.LinesValid) - pct(base.LinesCovered, base.LinesValid)
	}
	if cur.BranchesValid > 0 && base.BranchesValid > 0 {
		d.HasBranch = true
		d.BranchPP = pct(cur.BranchesCovered, cur.BranchesValid) - pct(base.BranchesCovered, base.BranchesValid)
	}
	return d
}

func pct(covered, valid int) float64 {
	if valid == 0 {
		return 0
	}
	return float64(covered) / float64(valid) * 100
}

func roundPP(v float64) float64 {
	return math.Round(v*10) / 10
}
