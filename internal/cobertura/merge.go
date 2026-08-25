package cobertura

import "sort"

// Merge combines multiple Reports into one by unioning classes that share the
// same Filename. Nil entries in reports are skipped; Merge(nil) returns a
// non-nil empty *Report.
//
// Class ordering follows first-seen order across the input slice: a filename is
// appended to the merged report the first time it appears, and subsequent
// occurrences merge into the existing class.
//
// Within a class, lines are unioned by line number:
//
//   - Hits is the max across inputs — a line is "hit" if any suite executed it.
//   - Branch is the logical OR — the merged line is a branch if any input marks
//     it as one.
//   - BranchesTotal is the max. Cobertura's branch total is a per-line constant
//     (the number of possible branches at that source point), so max recovers
//     the true total when one input didn't run the file (total=0).
//   - BranchesCovered is the max. This is an approximation of the true union:
//     it is always >= any single input and <= the true union, but Cobertura
//     does not carry the per-branch coverage mask needed to reconstruct the
//     exact covered set, so exact union is not possible from parsed reports.
//
// The merged class's Lines slice is sorted by line number ascending for
// deterministic, diff-friendly output.
func Merge(reports []*Report) *Report {
	merged := &Report{}
	// index maps filename -> position in merged.Classes so we can look up
	// existing classes without a linear scan.
	index := map[string]int{}
	// lineIndex maps (class position, line number) -> position in that class's
	// Lines slice, again to avoid quadratic behavior.
	lineIndex := map[int]map[int]int{}

	for _, r := range reports {
		if r == nil {
			continue
		}
		for _, cls := range r.Classes {
			pos, seen := index[cls.Filename]
			if !seen {
				pos = len(merged.Classes)
				index[cls.Filename] = pos
				merged.Classes = append(merged.Classes, Class{Filename: cls.Filename})
				lineIndex[pos] = map[int]int{}
			}
			target := &merged.Classes[pos]
			lines := lineIndex[pos]
			for _, ln := range cls.Lines {
				lpos, ok := lines[ln.Number]
				if !ok {
					lines[ln.Number] = len(target.Lines)
					target.Lines = append(target.Lines, ln)
					continue
				}
				existing := &target.Lines[lpos]
				if ln.Hits > existing.Hits {
					existing.Hits = ln.Hits
				}
				existing.Branch = existing.Branch || ln.Branch
				if ln.BranchesTotal > existing.BranchesTotal {
					existing.BranchesTotal = ln.BranchesTotal
				}
				if ln.BranchesCovered > existing.BranchesCovered {
					existing.BranchesCovered = ln.BranchesCovered
				}
			}
		}
	}

	for i := range merged.Classes {
		sort.Slice(merged.Classes[i].Lines, func(a, b int) bool {
			return merged.Classes[i].Lines[a].Number < merged.Classes[i].Lines[b].Number
		})
	}
	return merged
}
