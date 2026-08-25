package app

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// coverageDoc builds a minimal Cobertura document with the given classes, where
// each class maps a filename to a slice of hit counts (one per line).
func coverageDoc(classes map[string][]int) string {
	var b strings.Builder
	b.WriteString(`<coverage><packages><package><classes>`)
	for name, hits := range classes {
		b.WriteString(`<class filename="` + name + `"><lines>`)
		for i, h := range hits {
			b.WriteString(`<line number="` + itoa(i+1) + `" hits="` + itoa(h) + `"/>`)
		}
		b.WriteString(`</lines></class>`)
	}
	b.WriteString(`</classes></package></packages></coverage>`)
	return b.String()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func run(t *testing.T, opts Options) (string, error) {
	t.Helper()
	var out bytes.Buffer
	var errBuf bytes.Buffer
	opts.Stdout = &out
	opts.Stderr = &errBuf
	opts.Output = "-"
	err := Run(opts)
	return out.String(), err
}

func TestRunBasicAggregation(t *testing.T) {
	dir := t.TempDir()
	// api: 3 lines, 2 covered.
	write(t, dir, "coverage-api.xml", coverageDoc(map[string][]int{"src/a.ts": {1, 1, 0}}))
	write(t, dir, "tests-api.xml", `<testsuites tests="9"/>`)
	// worker: 2 lines, 2 covered, no test file (Tests should render "—").
	write(t, dir, "coverage-worker.xml", coverageDoc(map[string][]int{"internal/x.go": {1, 1}}))

	out, err := run(t, Options{Input: dir})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out, "| api | 9 | 2 / 3 | 66.7% |") {
		t.Errorf("api row wrong:\n%s", out)
	}
	if !strings.Contains(out, "| worker | — | 2 / 2 | 100.0% |") {
		t.Errorf("worker row (missing test artifact -> dash) wrong:\n%s", out)
	}
	if !strings.Contains(out, "**Total**") {
		t.Errorf("missing total:\n%s", out)
	}
}

func TestRunNoArtifacts(t *testing.T) {
	out, err := run(t, Options{Input: t.TempDir()})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if strings.TrimSpace(out) != "_No coverage artifacts found._" {
		t.Errorf("got:\n%s", out)
	}
}

func TestRunStripPrefixAndIgnore(t *testing.T) {
	dir := t.TempDir()
	// Go emitter writes full module paths.
	write(t, dir, "coverage-worker.xml", coverageDoc(map[string][]int{
		"github.com/acme/repo/services/worker/internal/logic/calc.go": {1, 1, 1},
		"github.com/acme/repo/services/worker/internal/store/gen.go":  {0, 0},
	}))

	ignorePath := filepath.Join(dir, ".coverageignore")
	write(t, dir, ".coverageignore", "services/worker/internal/store/**\n")

	cfgPath := filepath.Join(dir, "coverage.yaml")
	write(t, dir, "coverage.yaml", `
workspaces:
  worker:
    prefix: services/worker/
    strip_prefix: github.com/acme/repo/services/worker/
`)

	out, err := run(t, Options{
		Input: dir, ConfigPath: cfgPath, ConfigSet: true,
		IgnorePath: ignorePath, IgnoreSet: true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// store/gen.go excluded -> only the 3 logic lines count, all covered.
	if !strings.Contains(out, "| worker | — | 3 / 3 | 100.0% |") {
		t.Errorf("strip+ignore aggregation wrong:\n%s", out)
	}
	if !strings.Contains(out, "└ internal/logic") {
		t.Errorf("expected stripped folder grouping:\n%s", out)
	}
	if !strings.Contains(out, "Excluded paths from `.coverageignore`: 1 files") {
		t.Errorf("expected excluded footer:\n%s", out)
	}
}

func TestRunFolderGroupDepth(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "coverage-api.xml", coverageDoc(map[string][]int{
		"src/api/thing/services/deep/foo.ts": {1, 1},
	}))
	cfgPath := filepath.Join(dir, "coverage.yaml")
	write(t, dir, "coverage.yaml", "folder_group_depth: 2\n")

	out, err := run(t, Options{Input: dir, ConfigPath: cfgPath, ConfigSet: true})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out, "└ src/api |") {
		t.Errorf("expected depth-2 folder 'src/api':\n%s", out)
	}
}

func TestRunEmitJSONAndBaselineDelta(t *testing.T) {
	dir := t.TempDir()

	// First run at 100% -> emit baseline.
	write(t, dir, "coverage-api.xml", coverageDoc(map[string][]int{"a.ts": {1, 1, 1, 1}}))
	basePath := filepath.Join(dir, "baseline.json")
	if _, err := run(t, Options{Input: dir, EmitJSON: basePath}); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	if _, err := os.Stat(basePath); err != nil {
		t.Fatalf("baseline not emitted: %v", err)
	}

	// Second run at 50% -> should show a downward delta and a regression callout.
	write(t, dir, "coverage-api.xml", coverageDoc(map[string][]int{"a.ts": {1, 1, 0, 0}}))
	out, err := run(t, Options{Input: dir, BaselinePath: basePath, BaselineSet: true})
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}
	if !strings.Contains(out, "| % | Δ |") {
		t.Errorf("expected delta columns:\n%s", out)
	}
	if !strings.Contains(out, "▼ -50.0") {
		t.Errorf("expected downward delta:\n%s", out)
	}
	if !strings.Contains(out, "Coverage decreased") {
		t.Errorf("expected regression callout:\n%s", out)
	}
}

func TestRunFailOnDrop(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "coverage-api.xml", coverageDoc(map[string][]int{"a.ts": {1, 1, 1, 1}}))
	basePath := filepath.Join(dir, "baseline.json")
	if _, err := run(t, Options{Input: dir, EmitJSON: basePath}); err != nil {
		t.Fatalf("first Run: %v", err)
	}

	write(t, dir, "coverage-api.xml", coverageDoc(map[string][]int{"a.ts": {1, 1, 1, 0}}))
	threshold := 0.5
	_, err := run(t, Options{Input: dir, BaselinePath: basePath, BaselineSet: true, FailOnDrop: &threshold})
	if !errors.Is(err, ErrCoverageDropped) {
		t.Fatalf("expected ErrCoverageDropped, got %v", err)
	}
}

func TestRunNoBaselineFileIsNotError(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "coverage-api.xml", coverageDoc(map[string][]int{"a.ts": {1}}))
	out, err := run(t, Options{Input: dir, BaselinePath: filepath.Join(dir, "absent.json"), BaselineSet: true})
	if err != nil {
		t.Fatalf("missing baseline should not error: %v", err)
	}
	// No delta columns when baseline absent.
	if strings.Contains(out, "| % | Δ |") {
		t.Errorf("should not have delta columns without a baseline:\n%s", out)
	}
}

func TestRunInvalidInputDir(t *testing.T) {
	if err := Run(Options{Input: filepath.Join(t.TempDir(), "nope"), Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}); err == nil {
		t.Fatal("expected error for missing input dir")
	}
}

func TestRunHTMLFormat(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "coverage-api.xml", coverageDoc(map[string][]int{"src/a.ts": {1, 1, 0}}))

	// Explicit --format html to a file.
	htmlPath := filepath.Join(dir, "report.html")
	if err := Run(Options{Input: dir, Output: htmlPath, Format: "html", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}); err != nil {
		t.Fatalf("Run html: %v", err)
	}
	data, _ := os.ReadFile(htmlPath)
	if !strings.HasPrefix(string(data), "<!doctype html>") {
		t.Errorf("expected HTML document, got: %.40q", string(data))
	}

	// HTML output truncates on re-run (no concatenation).
	if err := Run(Options{Input: dir, Output: htmlPath, Format: "html", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}); err != nil {
		t.Fatal(err)
	}
	data2, _ := os.ReadFile(htmlPath)
	if strings.Count(string(data2), "<!doctype html>") != 1 {
		t.Error("HTML output should be truncated, not appended, on re-run")
	}
}

func TestRunFormatAutoDetect(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "coverage-api.xml", coverageDoc(map[string][]int{"src/a.ts": {1, 1}}))
	out := filepath.Join(dir, "cov.htm")
	if err := Run(Options{Input: dir, Output: out, Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(out)
	if !strings.HasPrefix(string(data), "<!doctype html>") {
		t.Error(".htm output should auto-detect html")
	}
}

func TestRunUnknownFormatErrors(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "coverage-api.xml", coverageDoc(map[string][]int{"src/a.ts": {1}}))
	err := Run(Options{Input: dir, Format: "pdf", Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}})
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
}

// Same workspace id split across two coverage artifacts (unit + integration
// suites) merges into one row with per-line hits unioned.
func TestRunMergesSplitSuitesUnderOneWorkspace(t *testing.T) {
	dir := t.TempDir()
	// Same file, three lines. Unit hits lines 1 and 2; integration hits lines 2
	// and 3. Merged: 3/3 covered.
	write(t, dir, "coverage-web.unit.xml", coverageDoc(map[string][]int{"src/a.ts": {1, 1, 0}}))
	write(t, dir, "coverage-web.integration.xml", coverageDoc(map[string][]int{"src/a.ts": {0, 1, 1}}))
	// And the corresponding split test artifacts — counts should sum.
	write(t, dir, "tests-web.unit.xml", `<testsuites tests="3"/>`)
	write(t, dir, "tests-web.integration.xml", `<testsuites tests="5"/>`)

	out, err := run(t, Options{Input: dir})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out, "| web | 8 | 3 / 3 | 100.0% |") {
		t.Errorf("expected merged web row (8 tests, 3/3 lines), got:\n%s", out)
	}
	if strings.Count(out, "| web ") != 1 {
		t.Errorf("expected exactly one web row, got:\n%s", out)
	}
}

// A dashed id (no dot) is untouched — split-suite parsing only kicks in when a
// dot appears in the id remainder.
func TestRunPreservesDashedIDs(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "coverage-shared-widget.xml", coverageDoc(map[string][]int{"src/w.ts": {1, 1}}))
	write(t, dir, "tests-shared-widget.xml", `<testsuites tests="4"/>`)

	out, err := run(t, Options{Input: dir})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out, "| shared-widget | 4 | 2 / 2 | 100.0% |") {
		t.Errorf("dashed id should not be split at the dash, got:\n%s", out)
	}
}

// A workspaces config entry keyed on the base id applies to every same-id
// artifact regardless of suffix.
func TestRunSplitSuitesShareConfig(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "coverage-worker.unit.xml", coverageDoc(map[string][]int{
		"github.com/acme/repo/services/worker/logic/a.go": {1, 1},
	}))
	write(t, dir, "coverage-worker.integration.xml", coverageDoc(map[string][]int{
		"github.com/acme/repo/services/worker/logic/b.go": {1},
	}))
	cfgPath := filepath.Join(dir, "coverage.yaml")
	write(t, dir, "coverage.yaml", `
workspaces:
  worker:
    strip_prefix: github.com/acme/repo/services/worker/
`)

	out, err := run(t, Options{
		Input: dir, ConfigPath: cfgPath, ConfigSet: true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// 2 lines from unit + 1 line from integration, all covered.
	if !strings.Contains(out, "| worker | — | 3 / 3 | 100.0% |") {
		t.Errorf("split-suite config bridging failed, got:\n%s", out)
	}
	// Folder grouping should reflect the stripped rel path, proving strip_prefix
	// was applied to both merged files.
	if !strings.Contains(out, "└ logic") {
		t.Errorf("expected stripped folder grouping for merged files:\n%s", out)
	}
}
