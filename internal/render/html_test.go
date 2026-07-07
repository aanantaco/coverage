package render

import (
	"strings"
	"testing"
)

func TestHTMLBasic(t *testing.T) {
	got := HTML(exampleSummary())
	for _, want := range []string{
		"<!doctype html>",
		"<title>Test Coverage</title>",
		"<h2>Summary</h2>",
		"<h2>Breakdown by folder</h2>",
		">thingy<",
		"1842 / 2310",
		"79.7%",
		"<tr class=\"total\">",
		"Excluded paths from",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("HTML missing %q", want)
		}
	}
	// No baseline => no Δ header column.
	if strings.Contains(got, "<th class=\"num\">Δ</th>") {
		t.Error("did not expect Δ column without a baseline")
	}
}

func TestHTMLBaselineDeltaClasses(t *testing.T) {
	got := HTML(baselineSummary())
	for _, want := range []string{
		"<th class=\"num\">Δ</th>", // delta column present
		"class=\"num down\"",       // thinger dropped
		"class=\"num up\"",         // thingy improved
		"class=\"num new\"",        // shared/widget is new
		"class=\"callout\"",        // regression callout
		"Removed since baseline",   // removed note
	} {
		if !strings.Contains(got, want) {
			t.Errorf("HTML(baseline) missing %q", want)
		}
	}
}

func TestHTMLEmpty(t *testing.T) {
	got := HTML(Summary{})
	if !strings.Contains(got, "No coverage artifacts found.") {
		t.Errorf("empty HTML should note no artifacts, got:\n%s", got)
	}
	if strings.Contains(got, "<table>") {
		t.Error("empty HTML should not render a table")
	}
}

func TestHTMLEscaping(t *testing.T) {
	// A display name with HTML metacharacters must be escaped.
	s := Summary{Workspaces: []Workspace{{DisplayName: "a<b>&c", LinesValid: 2, LinesCovered: 1}}}
	got := HTML(s)
	if strings.Contains(got, "a<b>&c") {
		t.Error("display name was not HTML-escaped")
	}
	if !strings.Contains(got, "a&lt;b&gt;&amp;c") {
		t.Error("expected escaped display name")
	}
}
