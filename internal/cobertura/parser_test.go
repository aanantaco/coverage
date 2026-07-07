package cobertura

import (
	"strings"
	"testing"
)

func TestParseRecomputesFromLeafLines(t *testing.T) {
	// The top-level totals here are deliberately wrong; the parser must ignore
	// them and expose only the leaf lines.
	const doc = `<?xml version="1.0"?>
<coverage lines-valid="999" lines-covered="999">
  <packages>
    <package>
      <classes>
        <class filename="src/a.ts">
          <lines>
            <line number="1" hits="3"/>
            <line number="2" hits="0"/>
          </lines>
        </class>
      </classes>
    </package>
  </packages>
</coverage>`

	report, err := Parse(strings.NewReader(doc))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(report.Classes) != 1 {
		t.Fatalf("got %d classes, want 1", len(report.Classes))
	}
	c := report.Classes[0]
	if c.Filename != "src/a.ts" {
		t.Errorf("filename = %q, want src/a.ts", c.Filename)
	}
	if len(c.Lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(c.Lines))
	}
	if c.Lines[0].Hits != 3 || c.Lines[1].Hits != 0 {
		t.Errorf("unexpected hits: %+v", c.Lines)
	}
}

func TestParseBranchConditionCoverage(t *testing.T) {
	const doc = `<coverage><packages><package><classes>
      <class filename="b.go"><lines>
        <line number="1" hits="1" branch="true" condition-coverage="50% (1/2)"/>
        <line number="2" hits="1" branch="1" condition-coverage="100% (4/4)"/>
        <line number="3" hits="1" branch="false"/>
        <line number="4" hits="1" branch="true" condition-coverage="garbage"/>
      </lines></class>
    </classes></package></packages></coverage>`

	report, err := Parse(strings.NewReader(doc))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	lines := report.Classes[0].Lines

	if !lines[0].Branch || lines[0].BranchesCovered != 1 || lines[0].BranchesTotal != 2 {
		t.Errorf("line1 = %+v", lines[0])
	}
	if !lines[1].Branch || lines[1].BranchesCovered != 4 || lines[1].BranchesTotal != 4 {
		t.Errorf("line2 = %+v", lines[1])
	}
	if lines[2].Branch {
		t.Errorf("line3 should not be a branch: %+v", lines[2])
	}
	// Malformed condition-coverage => no branch contribution.
	if lines[3].BranchesTotal != 0 || lines[3].BranchesCovered != 0 {
		t.Errorf("line4 malformed should yield zero branches: %+v", lines[3])
	}
}

func TestParseEmptyInput(t *testing.T) {
	report, err := Parse(strings.NewReader("   \n  "))
	if err != nil {
		t.Fatalf("Parse empty: %v", err)
	}
	if len(report.Classes) != 0 {
		t.Errorf("expected no classes, got %d", len(report.Classes))
	}
}

func TestParseMalformedXML(t *testing.T) {
	_, err := Parse(strings.NewReader("<coverage><not-closed>"))
	if err == nil {
		t.Fatal("expected error for malformed xml")
	}
}
