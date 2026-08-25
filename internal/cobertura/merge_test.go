package cobertura

import (
	"reflect"
	"testing"
)

func TestMergeNil(t *testing.T) {
	got := Merge(nil)
	if got == nil {
		t.Fatal("Merge(nil) returned nil, want non-nil *Report")
	}
	if len(got.Classes) != 0 {
		t.Errorf("Merge(nil) classes = %d, want 0", len(got.Classes))
	}
}

func TestMergeDisjointClassesPreserveFirstSeenOrder(t *testing.T) {
	a := &Report{Classes: []Class{{Filename: "a.go", Lines: []Line{{Number: 1, Hits: 1}}}}}
	b := &Report{Classes: []Class{{Filename: "b.go", Lines: []Line{{Number: 1, Hits: 2}}}}}

	got := Merge([]*Report{a, b})
	if len(got.Classes) != 2 {
		t.Fatalf("got %d classes, want 2", len(got.Classes))
	}
	if got.Classes[0].Filename != "a.go" || got.Classes[1].Filename != "b.go" {
		t.Errorf("filenames = [%q, %q], want [a.go, b.go]", got.Classes[0].Filename, got.Classes[1].Filename)
	}
	// And the reverse order — first-seen still wins.
	got = Merge([]*Report{b, a})
	if got.Classes[0].Filename != "b.go" || got.Classes[1].Filename != "a.go" {
		t.Errorf("filenames = [%q, %q], want [b.go, a.go]", got.Classes[0].Filename, got.Classes[1].Filename)
	}
}

func TestMergeUnionLineHits(t *testing.T) {
	// Four lines: hit in A only, hit in B only, hit in both (max wins), hit in neither.
	a := &Report{Classes: []Class{{Filename: "f.go", Lines: []Line{
		{Number: 1, Hits: 3},
		{Number: 2, Hits: 0},
		{Number: 3, Hits: 1},
		{Number: 4, Hits: 0},
	}}}}
	b := &Report{Classes: []Class{{Filename: "f.go", Lines: []Line{
		{Number: 1, Hits: 0},
		{Number: 2, Hits: 5},
		{Number: 3, Hits: 7},
		{Number: 4, Hits: 0},
	}}}}

	got := Merge([]*Report{a, b})
	if len(got.Classes) != 1 {
		t.Fatalf("got %d classes, want 1", len(got.Classes))
	}
	want := []Line{
		{Number: 1, Hits: 3},
		{Number: 2, Hits: 5},
		{Number: 3, Hits: 7},
		{Number: 4, Hits: 0},
	}
	if !reflect.DeepEqual(got.Classes[0].Lines, want) {
		t.Errorf("lines = %+v, want %+v", got.Classes[0].Lines, want)
	}
}

func TestMergeAddsLinesFromLaterReportSorted(t *testing.T) {
	a := &Report{Classes: []Class{{Filename: "f.go", Lines: []Line{
		{Number: 10, Hits: 1},
		{Number: 30, Hits: 1},
	}}}}
	b := &Report{Classes: []Class{{Filename: "f.go", Lines: []Line{
		{Number: 20, Hits: 2}, // new line, in between existing numbers
	}}}}

	got := Merge([]*Report{a, b})
	nums := []int{}
	for _, l := range got.Classes[0].Lines {
		nums = append(nums, l.Number)
	}
	if !reflect.DeepEqual(nums, []int{10, 20, 30}) {
		t.Errorf("line numbers = %v, want [10 20 30]", nums)
	}
}

func TestMergeUnionBranchCoverage(t *testing.T) {
	// Same branch line in both: A covers 1/2, B covers 2/2. Max approx => 2/2.
	a := &Report{Classes: []Class{{Filename: "f.go", Lines: []Line{
		{Number: 1, Hits: 1, Branch: true, BranchesCovered: 1, BranchesTotal: 2},
	}}}}
	b := &Report{Classes: []Class{{Filename: "f.go", Lines: []Line{
		{Number: 1, Hits: 1, Branch: true, BranchesCovered: 2, BranchesTotal: 2},
	}}}}

	got := Merge([]*Report{a, b})
	line := got.Classes[0].Lines[0]
	if !line.Branch || line.BranchesCovered != 2 || line.BranchesTotal != 2 {
		t.Errorf("line = %+v, want branch=true covered=2 total=2", line)
	}
}

func TestMergePromotesBranchFlag(t *testing.T) {
	// A: not a branch. B: branch with 3/4 covered and more hits.
	a := &Report{Classes: []Class{{Filename: "f.go", Lines: []Line{
		{Number: 1, Hits: 1, Branch: false},
	}}}}
	b := &Report{Classes: []Class{{Filename: "f.go", Lines: []Line{
		{Number: 1, Hits: 5, Branch: true, BranchesCovered: 3, BranchesTotal: 4},
	}}}}

	got := Merge([]*Report{a, b})
	line := got.Classes[0].Lines[0]
	if !line.Branch {
		t.Errorf("expected branch=true after promotion, got %+v", line)
	}
	if line.Hits != 5 {
		t.Errorf("hits = %d, want 5 (max across inputs)", line.Hits)
	}
	if line.BranchesCovered != 3 || line.BranchesTotal != 4 {
		t.Errorf("branches = %d/%d, want 3/4", line.BranchesCovered, line.BranchesTotal)
	}
}

func TestMergeSkipsNilEntries(t *testing.T) {
	a := &Report{Classes: []Class{
		{Filename: "a.go", Lines: []Line{{Number: 1, Hits: 1}}},
		{Filename: "b.go", Lines: []Line{{Number: 2, Hits: 2}}},
	}}
	got := Merge([]*Report{nil, a, nil})
	if !reflect.DeepEqual(got.Classes, a.Classes) {
		t.Errorf("classes = %+v, want %+v", got.Classes, a.Classes)
	}
}
