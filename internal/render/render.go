// Package render turns an aggregated coverage Summary into a report. Two output
// formats are supported — Markdown (for $GITHUB_STEP_SUMMARY) and a standalone
// HTML page — both driven by embedded templates in templates/.
package render

import (
	"embed"
	"fmt"
	"html/template"
	"math"
	"strconv"
	"strings"
	tmpltext "text/template"
)

//go:embed templates/report.md.tmpl templates/report.html.tmpl
var templatesFS embed.FS

// funcs are the shared cell-formatting helpers. They return plain text so the
// Markdown and HTML templates render identical values, differing only in markup.
var funcs = map[string]any{
	"linesCell":     linesCell,
	"branchesCell":  branchesCell,
	"linePct":       linePct,
	"branchPct":     branchPct,
	"branchPctBold": branchPctBold,
	"deltaLine":     func(r viewRow) string { return deltaLineCell(r.Delta) },
	"deltaBranch":   func(r viewRow) string { return deltaBranchCell(r.Delta) },
	"deltaClass":    func(r viewRow) string { return deltaClass(r.Delta, r.Delta != nil && r.Delta.HasLine, linePP) },
	"branchClass":   func(r viewRow) string { return deltaClass(r.Delta, r.Delta != nil && r.Delta.HasBranch, branchPPv) },
	"fmtPct":        func(f float64) string { return fmt.Sprintf("%.1f%%", f) },
	"fmtDrop":       fmtDrop,
	"pluralize":     pluralize,
	"join":          strings.Join,
}

var (
	mdTmpl   = tmpltext.Must(tmpltext.New("report.md.tmpl").Funcs(funcs).ParseFS(templatesFS, "templates/report.md.tmpl"))
	htmlTmpl = template.Must(template.New("report.html.tmpl").Funcs(funcs).ParseFS(templatesFS, "templates/report.html.tmpl"))
)

// Markdown renders the summary as Markdown. It is a pure function of its input.
func Markdown(s Summary) string {
	if len(s.Workspaces) == 0 {
		return "_No coverage artifacts found._\n"
	}
	var b strings.Builder
	// Templates are embedded and validated at init (template.Must), so execution
	// cannot fail for valid input; ignore the unreachable error rather than panic.
	_ = mdTmpl.Execute(&b, buildView(s))
	return b.String()
}

// HTML renders the summary as a standalone HTML page.
func HTML(s Summary) string {
	var b strings.Builder
	data := struct {
		view
		Empty bool
	}{view: buildView(s), Empty: len(s.Workspaces) == 0}
	// See Markdown: the embedded template cannot fail to execute for valid input.
	_ = htmlTmpl.Execute(&b, data)
	return b.String()
}

// --- cell formatting (shared by both templates) ---

func linesCell(r viewRow) string {
	return fmt.Sprintf("%d / %d", r.LinesCovered, r.LinesValid)
}

func branchesCell(r viewRow) string {
	if r.BranchesValid == 0 {
		return "—"
	}
	return fmt.Sprintf("%d/%d", r.BranchesCovered, r.BranchesValid)
}

func linePct(r viewRow) string {
	return percent(r.LinesCovered, r.LinesValid)
}

func branchPct(r viewRow) string {
	if r.BranchesValid == 0 {
		return "—"
	}
	return percent(r.BranchesCovered, r.BranchesValid)
}

// branchPctBold bolds the branch percentage for the Total row, but leaves "—"
// un-bolded when there is no branch data.
func branchPctBold(r viewRow) string {
	if r.BranchesValid == 0 {
		return "—"
	}
	return "**" + percent(r.BranchesCovered, r.BranchesValid) + "**"
}

// percent returns a one-decimal percentage, or "0.0%" when total is 0.
func percent(covered, total int) string {
	if total == 0 {
		return "0.0%"
	}
	return fmt.Sprintf("%.1f%%", float64(covered)/float64(total)*100)
}

func deltaLineCell(d *Delta) string {
	switch {
	case d == nil:
		return "—"
	case d.IsNew:
		return "new"
	case !d.HasLine:
		return "—"
	default:
		return formatDelta(d.LinePP)
	}
}

func deltaBranchCell(d *Delta) string {
	switch {
	case d == nil:
		return "—"
	case d.IsNew:
		return "new"
	case !d.HasBranch:
		return "—"
	default:
		return formatDelta(d.BranchPP)
	}
}

// formatDelta renders a percentage-point change with a direction marker.
func formatDelta(pp float64) string {
	rounded := math.Round(pp*10) / 10
	switch {
	case rounded > 0:
		return fmt.Sprintf("▲ +%.1f", rounded)
	case rounded < 0:
		return fmt.Sprintf("▼ %.1f", rounded)
	default:
		return "▬ 0.0"
	}
}

func linePP(d *Delta) float64    { return d.LinePP }
func branchPPv(d *Delta) float64 { return d.BranchPP }

// deltaClass returns a CSS class for an HTML delta cell.
func deltaClass(d *Delta, has bool, pp func(*Delta) float64) string {
	if d == nil {
		return "na"
	}
	if d.IsNew {
		return "new"
	}
	if !has {
		return "na"
	}
	switch r := math.Round(pp(d)*10) / 10; {
	case r > 0:
		return "up"
	case r < 0:
		return "down"
	default:
		return "flat"
	}
}

func fmtDrop(f float64) string {
	if f < 0 {
		f = -f
	}
	return fmt.Sprintf("%.1f", f)
}

func pluralize(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func itoa(n int) string { return strconv.Itoa(n) }
