// Package app holds the orchestration that was historically main.go: it globs
// coverage and test artifacts, aggregates them under optional config-driven
// prefixes, computes regression deltas against a baseline, and writes the
// Markdown report.
package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aanantaco/coverage/internal/baseline"
	"github.com/aanantaco/coverage/internal/cobertura"
	"github.com/aanantaco/coverage/internal/config"
	"github.com/aanantaco/coverage/internal/ignore"
	"github.com/aanantaco/coverage/internal/junit"
	"github.com/aanantaco/coverage/internal/render"
)

// ErrCoverageDropped is returned (after the report is written) when total line
// coverage dropped by more than the configured fail-on-drop threshold.
var ErrCoverageDropped = errors.New("total line coverage dropped beyond threshold")

// Options are the resolved inputs to Run. The *Set booleans record whether a
// flag was explicitly provided, so config values can fill the gaps.
type Options struct {
	Input string

	Output string // "-" for stdout

	IgnorePath string
	IgnoreSet  bool

	ConfigPath string
	ConfigSet  bool

	BaselinePath string
	BaselineSet  bool

	// FailOnDrop is non-nil only when --fail-on-drop was passed.
	FailOnDrop *float64

	EmitJSON string

	// Format is the output format: "markdown", "html", or "" to auto-detect
	// from the --output extension (.html/.htm => html, else markdown).
	Format string

	Verbose bool

	Stdout io.Writer
	Stderr io.Writer
}

// aggregate structures accumulate leaf-line counts as artifacts are parsed.
type folderAgg struct {
	path string
	totals
}

type totals struct {
	linesValid      int
	linesCovered    int
	branchesValid   int
	branchesCovered int
}

func (t *totals) add(o totals) {
	t.linesValid += o.linesValid
	t.linesCovered += o.linesCovered
	t.branchesValid += o.branchesValid
	t.branchesCovered += o.branchesCovered
}

type workspaceAgg struct {
	id          string
	displayName string
	totals
	tests    int
	hasTests bool
	folders  map[string]*folderAgg
	order    []string // folder insertion order, for stable output
}

// Run executes the full pipeline. It writes the report before returning
// ErrCoverageDropped, so a regression still surfaces the summary.
func Run(opts Options) error {
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Output == "" {
		opts.Output = "-"
	}

	cfg, err := loadConfig(opts)
	if err != nil {
		return err
	}

	info, err := os.Stat(opts.Input)
	if err != nil {
		return fmt.Errorf("input: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("input %q is not a directory", opts.Input)
	}

	ignorePath := resolveIgnorePath(opts, cfg)
	matcher, ignoreExists, err := ignore.Load(ignorePath)
	if err != nil {
		return err
	}

	workspaces, excluded, err := aggregateCoverage(opts, cfg, matcher)
	if err != nil {
		return err
	}
	attachTests(opts, workspaces)

	current := buildCurrentSummary(workspaces)

	summary := render.Summary{
		ExcludedFiles:    excluded,
		IgnoreFileLoaded: ignoreExists,
	}
	for _, w := range workspaces {
		summary.Workspaces = append(summary.Workspaces, toRenderWorkspace(w))
	}

	// Regression detection.
	if err := applyBaseline(opts, cfg, current, &summary); err != nil {
		return err
	}

	// Render and write.
	format, err := resolveFormat(opts)
	if err != nil {
		return err
	}
	var out string
	switch format {
	case "html":
		out = render.HTML(summary)
	default:
		out = render.Markdown(summary)
	}
	if err := writeOutput(opts, out, format); err != nil {
		return err
	}

	// Emit machine-readable summary if requested.
	if opts.EmitJSON != "" {
		if err := baseline.Emit(current, opts.EmitJSON); err != nil {
			return err
		}
	}

	// Fail-on-drop is evaluated after everything is written.
	if failed := checkFailOnDrop(opts, cfg, current, &summary); failed {
		return ErrCoverageDropped
	}

	return nil
}

func loadConfig(opts Options) (*config.Config, error) {
	if opts.ConfigSet {
		return config.Load(opts.ConfigPath)
	}
	const defaultPath = "coverage.yaml"
	if _, err := os.Stat(defaultPath); err == nil {
		return config.Load(defaultPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("stat %q: %w", defaultPath, err)
	}
	return config.Default(), nil
}

// resolveIgnorePath applies precedence: explicit flag > config > default file.
func resolveIgnorePath(opts Options, cfg *config.Config) string {
	if opts.IgnoreSet {
		return opts.IgnorePath
	}
	if cfg.IgnoreFile != "" {
		return cfg.IgnoreFile
	}
	if _, err := os.Stat(config.DefaultIgnoreFile); err == nil {
		return config.DefaultIgnoreFile
	}
	return ""
}

func aggregateCoverage(opts Options, cfg *config.Config, matcher ignore.Matcher) ([]*workspaceAgg, int, error) {
	files, err := filepath.Glob(filepath.Join(opts.Input, "coverage-*.xml"))
	if err != nil {
		return nil, 0, fmt.Errorf("glob coverage artifacts: %w", err)
	}
	sort.Strings(files)

	var workspaces []*workspaceAgg
	excluded := 0

	for _, file := range files {
		id := artifactID(file, "coverage-")
		if id == "" {
			continue
		}
		wsCfg, configured := cfg.Workspaces[id]
		if !configured && opts.Verbose {
			fmt.Fprintf(opts.Stderr, "warning: workspace %q has no config entry; matching with raw filenames\n", id)
		}

		report, err := cobertura.ParseFile(file)
		if err != nil {
			fmt.Fprintf(opts.Stderr, "warning: skipping %q: %v\n", file, err)
			continue
		}

		ws := &workspaceAgg{
			id:          id,
			displayName: cfg.DisplayName(id),
			folders:     map[string]*folderAgg{},
		}
		for _, class := range report.Classes {
			rel := stripPrefix(class.Filename, wsCfg.StripPrefix)
			full := wsCfg.Prefix + rel
			if matcher.Match(full) {
				excluded++
				continue
			}
			ct := classTotals(class)
			ws.totals.add(ct)
			group := folderGroup(rel, cfg.FolderGroupDepth)
			fa, ok := ws.folders[group]
			if !ok {
				fa = &folderAgg{path: group}
				ws.folders[group] = fa
				ws.order = append(ws.order, group)
			}
			fa.totals.add(ct)
		}
		workspaces = append(workspaces, ws)
	}

	return workspaces, excluded, nil
}

// classTotals sums the leaf lines of a class.
func classTotals(class cobertura.Class) totals {
	var t totals
	for _, line := range class.Lines {
		t.linesValid++
		if line.Hits > 0 {
			t.linesCovered++
		}
		if line.Branch && line.BranchesTotal > 0 {
			t.branchesValid += line.BranchesTotal
			t.branchesCovered += line.BranchesCovered
		}
	}
	return t
}

func attachTests(opts Options, workspaces []*workspaceAgg) {
	files, err := filepath.Glob(filepath.Join(opts.Input, "tests-*.xml"))
	if err != nil {
		fmt.Fprintf(opts.Stderr, "warning: glob test artifacts: %v\n", err)
		return
	}
	sort.Strings(files)

	byID := map[string]*workspaceAgg{}
	for _, w := range workspaces {
		byID[w.id] = w
	}

	for _, file := range files {
		id := artifactID(file, "tests-")
		if id == "" {
			continue
		}
		report, err := junit.ParseFile(file)
		if err != nil {
			fmt.Fprintf(opts.Stderr, "warning: skipping %q: %v\n", file, err)
			continue
		}
		if w, ok := byID[id]; ok {
			w.tests = report.Tests
			w.hasTests = true
		}
	}
}

func buildCurrentSummary(workspaces []*workspaceAgg) *baseline.Summary {
	s := &baseline.Summary{
		Schema:        baseline.SchemaVersion,
		GeneratedFrom: baseline.GeneratedFrom,
		Workspaces:    map[string]baseline.WorkspaceSummary{},
	}
	for _, w := range workspaces {
		ws := baseline.WorkspaceSummary{
			Totals:  toBaselineTotals(w.totals),
			Folders: map[string]baseline.Totals{},
		}
		for path, fa := range w.folders {
			ws.Folders[path] = toBaselineTotals(fa.totals)
		}
		s.Workspaces[w.id] = ws
		s.Total.LinesValid += w.linesValid
		s.Total.LinesCovered += w.linesCovered
		s.Total.BranchesValid += w.branchesValid
		s.Total.BranchesCovered += w.branchesCovered
	}
	return s
}

func applyBaseline(opts Options, cfg *config.Config, current *baseline.Summary, summary *render.Summary) error {
	basePath := resolveBaselinePath(opts, cfg)
	if basePath == "" {
		return nil
	}
	base, found, err := baseline.Load(basePath)
	if err != nil {
		return err
	}
	if !found {
		fmt.Fprintf(opts.Stderr, "note: no baseline available at %q; skipping regression detection\n", basePath)
		return nil
	}
	if base.Schema != baseline.SchemaVersion {
		fmt.Fprintf(opts.Stderr, "warning: baseline schema %d != %d; skipping regression detection\n", base.Schema, baseline.SchemaVersion)
		return nil
	}

	cmp := baseline.Compare(current, base)
	summary.HasBaseline = true
	summary.TotalDelta = toRenderDelta(cmp.Total)

	deltaByID := map[string]baseline.WorkspaceDelta{}
	for id, wd := range cmp.Workspaces {
		deltaByID[id] = wd
	}
	for i := range summary.Workspaces {
		w := &summary.Workspaces[i]
		wd, ok := deltaByID[w.ID]
		if !ok {
			continue
		}
		w.Delta = toRenderDelta(wd.PctDelta)
		for j := range w.Folders {
			if fd, ok := wd.Folders[w.Folders[j].Path]; ok {
				w.Folders[j].Delta = toRenderDelta(fd)
			}
		}
	}

	for _, id := range cmp.Removed {
		summary.RemovedWorkspaces = append(summary.RemovedWorkspaces, cfg.DisplayName(id))
	}
	for _, r := range cmp.Regressions {
		summary.Regressions = append(summary.Regressions, render.Regression{
			DisplayName: cfg.DisplayName(r.ID),
			OldPercent:  r.OldPercent,
			NewPercent:  r.NewPercent,
			DropPP:      r.DropPP,
		})
	}
	return nil
}

func checkFailOnDrop(opts Options, cfg *config.Config, current *baseline.Summary, summary *render.Summary) bool {
	threshold := resolveFailOnDrop(opts, cfg)
	if threshold == nil {
		return false
	}
	if !summary.HasBaseline || summary.TotalDelta == nil || !summary.TotalDelta.HasLine {
		return false
	}
	drop := -summary.TotalDelta.LinePP
	if drop > *threshold {
		fmt.Fprintf(opts.Stderr, "error: total line coverage dropped %.1fpp (threshold %.1fpp)\n", drop, *threshold)
		return true
	}
	return false
}

func resolveBaselinePath(opts Options, cfg *config.Config) string {
	if opts.BaselineSet {
		return opts.BaselinePath
	}
	return cfg.Baseline.Path
}

func resolveFailOnDrop(opts Options, cfg *config.Config) *float64 {
	if opts.FailOnDrop != nil {
		return opts.FailOnDrop
	}
	return cfg.Baseline.FailOnDrop
}

func writeOutput(opts Options, content, format string) error {
	if opts.Output == "-" {
		_, err := io.WriteString(opts.Stdout, content)
		return err
	}
	// Markdown appends (so $GITHUB_STEP_SUMMARY accumulates); a standalone HTML
	// document is truncated so re-runs don't concatenate multiple pages.
	flags := os.O_APPEND | os.O_CREATE | os.O_WRONLY
	if format == "html" {
		flags = os.O_TRUNC | os.O_CREATE | os.O_WRONLY
	}
	f, err := os.OpenFile(opts.Output, flags, 0o644)
	if err != nil {
		return fmt.Errorf("open output %q: %w", opts.Output, err)
	}
	defer f.Close()
	if _, err := io.WriteString(f, content); err != nil {
		return fmt.Errorf("write output %q: %w", opts.Output, err)
	}
	return nil
}

// resolveFormat determines the output format from the flag, or auto-detects
// from the --output extension. Unknown formats are an error.
func resolveFormat(opts Options) (string, error) {
	f := strings.ToLower(strings.TrimSpace(opts.Format))
	if f == "" {
		lower := strings.ToLower(opts.Output)
		if strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".htm") {
			return "html", nil
		}
		return "markdown", nil
	}
	switch f {
	case "markdown", "md":
		return "markdown", nil
	case "html":
		return "html", nil
	default:
		return "", fmt.Errorf("unknown --format %q (want markdown or html)", opts.Format)
	}
}

// --- helpers ---

// artifactID strips a leading prefix and a trailing ".xml" from a path's base
// name. The id itself may contain dashes.
func artifactID(path, prefix string) string {
	base := filepath.Base(path)
	if !strings.HasPrefix(base, prefix) || !strings.HasSuffix(base, ".xml") {
		return ""
	}
	return base[len(prefix) : len(base)-len(".xml")]
}

func stripPrefix(filename, prefix string) string {
	if prefix != "" && strings.HasPrefix(filename, prefix) {
		return filename[len(prefix):]
	}
	return filename
}

// folderGroup derives a folder bucket from a workspace-relative filename,
// truncated to at most depth leading path components. Root files bucket to
// "(root)".
func folderGroup(filename string, depth int) string {
	idx := strings.LastIndex(filename, "/")
	if idx < 0 {
		return "(root)"
	}
	dir := filename[:idx]
	parts := strings.Split(dir, "/")
	if len(parts) > depth {
		parts = parts[:depth]
	}
	return strings.Join(parts, "/")
}

func toBaselineTotals(t totals) baseline.Totals {
	return baseline.Totals{
		LinesValid:      t.linesValid,
		LinesCovered:    t.linesCovered,
		BranchesValid:   t.branchesValid,
		BranchesCovered: t.branchesCovered,
	}
}

func toRenderWorkspace(w *workspaceAgg) render.Workspace {
	rw := render.Workspace{
		ID:              w.id,
		DisplayName:     w.displayName,
		LinesValid:      w.linesValid,
		LinesCovered:    w.linesCovered,
		BranchesValid:   w.branchesValid,
		BranchesCovered: w.branchesCovered,
		Tests:           w.tests,
		HasTests:        w.hasTests,
	}
	for _, path := range w.order {
		fa := w.folders[path]
		rw.Folders = append(rw.Folders, render.Folder{
			Path:            fa.path,
			LinesValid:      fa.linesValid,
			LinesCovered:    fa.linesCovered,
			BranchesValid:   fa.branchesValid,
			BranchesCovered: fa.branchesCovered,
		})
	}
	return rw
}

func toRenderDelta(d baseline.PctDelta) *render.Delta {
	return &render.Delta{
		IsNew:     d.IsNew,
		HasLine:   d.HasLine,
		LinePP:    d.LinePP,
		HasBranch: d.HasBranch,
		BranchPP:  d.BranchPP,
	}
}
