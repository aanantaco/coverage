// Package cobertura parses the subset of the Cobertura XML schema needed to
// aggregate line and branch coverage.
//
// The parser deliberately ignores the top-level lines-valid/lines-covered
// totals emitted on the <coverage> element: different emitters (Jest,
// gocover-cobertura, vitest/v8) disagree on how those are computed, so all
// totals are recomputed from the leaf <line> elements instead.
package cobertura

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Line is a single source line's coverage record.
type Line struct {
	Number          int
	Hits            int
	Branch          bool
	BranchesCovered int
	BranchesTotal   int
}

// Class is one source file's worth of coverage.
type Class struct {
	Filename string
	Lines    []Line
}

// Report is the parsed, emitter-agnostic view of a Cobertura document.
type Report struct {
	Classes []Class
}

// xmlCoverage mirrors the nested Cobertura structure we care about. Attributes
// on <coverage> (lines-valid etc.) are intentionally not decoded.
type xmlCoverage struct {
	XMLName  xml.Name     `xml:"coverage"`
	Packages []xmlPackage `xml:"packages>package"`
}

type xmlPackage struct {
	Classes []xmlClass `xml:"classes>class"`
}

type xmlClass struct {
	Filename string    `xml:"filename,attr"`
	Lines    []xmlLine `xml:"lines>line"`
}

type xmlLine struct {
	Number            int    `xml:"number,attr"`
	Hits              int    `xml:"hits,attr"`
	Branch            string `xml:"branch,attr"`
	ConditionCoverage string `xml:"condition-coverage,attr"`
}

// ParseFile reads and parses a Cobertura XML file at path.
func ParseFile(path string) (*Report, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

// Parse parses Cobertura XML from r. An empty input yields an empty Report with
// a nil error.
func Parse(r io.Reader) (*Report, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return &Report{}, nil
	}

	var cov xmlCoverage
	if err := xml.Unmarshal(data, &cov); err != nil {
		return nil, fmt.Errorf("parse cobertura xml: %w", err)
	}

	report := &Report{}
	for _, pkg := range cov.Packages {
		for _, cls := range pkg.Classes {
			c := Class{Filename: cls.Filename}
			for _, ln := range cls.Lines {
				line := Line{
					Number: ln.Number,
					Hits:   ln.Hits,
					Branch: parseBool(ln.Branch),
				}
				if line.Branch {
					if covered, total, ok := parseConditionCoverage(ln.ConditionCoverage); ok {
						line.BranchesCovered = covered
						line.BranchesTotal = total
					}
				}
				c.Lines = append(c.Lines, line)
			}
			report.Classes = append(report.Classes, c)
		}
	}
	return report, nil
}

// parseBool treats "true" and "1" as true; anything else is false.
func parseBool(s string) bool {
	switch strings.TrimSpace(s) {
	case "true", "1":
		return true
	default:
		return false
	}
}

// parseConditionCoverage extracts the (covered/total) pair from a
// condition-coverage attribute formatted like "50% (1/2)". Returns ok=false if
// the pair is missing or malformed, in which case the line contributes no
// branch data.
func parseConditionCoverage(s string) (covered, total int, ok bool) {
	open := strings.IndexByte(s, '(')
	close := strings.IndexByte(s, ')')
	if open < 0 || close < 0 || close < open {
		return 0, 0, false
	}
	inner := s[open+1 : close]
	slash := strings.IndexByte(inner, '/')
	if slash < 0 {
		return 0, 0, false
	}
	c, err1 := strconv.Atoi(strings.TrimSpace(inner[:slash]))
	t, err2 := strconv.Atoi(strings.TrimSpace(inner[slash+1:]))
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return c, t, true
}
