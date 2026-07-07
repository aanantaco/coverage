// Package scaffold implements `coverage init`: it detects the languages in a
// repository and non-destructively scaffolds a tailored GitHub Actions workflow
// plus starter config files. It never overwrites existing files.
//
// The generated workflow's report/aggregation job is complete; each per-language
// test job is a framework-agnostic stub with a TODO pointing at the docs for the
// actual test command. That split keeps `init` durable — the volatile,
// per-framework commands live in the docs (and are easy for a human or an AI
// assistant to fill in), not baked into the tool.
package scaffold

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Language is a detected language/toolchain.
type Language struct {
	Name    string // human name, e.g. "Go"
	ID      string // workspace id used in artifact filenames, e.g. "go"
	JobName string // workflow job name, e.g. "test-go"
	Doc     string // per-language doc filename, e.g. "GO.md"
}

// Options configures Run.
type Options struct {
	Dir    string // target repo directory
	DryRun bool   // print planned actions without writing
	Stdout io.Writer
	Stderr io.Writer
}

// Run detects languages under opts.Dir and writes the scaffold files that don't
// already exist. It returns an error only on I/O failures.
func Run(opts Options) error {
	if opts.Dir == "" {
		opts.Dir = "."
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}

	langs := Detect(opts.Dir)
	if len(langs) == 0 {
		fmt.Fprintln(opts.Stderr, "coverage init: no supported languages detected under", opts.Dir)
		fmt.Fprintln(opts.Stderr, "  looked for: go.mod, package.json, pyproject.toml/setup.py, Cargo.toml, pom.xml/build.gradle, *.csproj")
		return nil
	}

	names := make([]string, len(langs))
	for i, l := range langs {
		names[i] = l.Name
	}
	fmt.Fprintf(opts.Stdout, "Detected: %s\n\n", strings.Join(names, ", "))

	files := Files(langs)
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	created, skipped := 0, 0
	for _, rel := range paths {
		full := filepath.Join(opts.Dir, rel)
		if _, err := os.Stat(full); err == nil {
			fmt.Fprintf(opts.Stdout, "  skip    %s (already exists)\n", rel)
			skipped++
			continue
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("stat %q: %w", full, err)
		}
		if opts.DryRun {
			fmt.Fprintf(opts.Stdout, "  create  %s (dry-run)\n", rel)
			created++
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return fmt.Errorf("mkdir for %q: %w", rel, err)
		}
		if err := os.WriteFile(full, []byte(files[rel]), 0o644); err != nil {
			return fmt.Errorf("write %q: %w", rel, err)
		}
		fmt.Fprintf(opts.Stdout, "  create  %s\n", rel)
		created++
	}

	fmt.Fprintf(opts.Stdout, "\n%d created, %d skipped.\n", created, skipped)
	fmt.Fprintln(opts.Stdout, "\nNext: fill in each test job's TODO with your test command (see the linked")
	fmt.Fprintln(opts.Stdout, "docs), pin the coverage action to a commit SHA, then commit.")
	return nil
}

// Files returns the scaffold files (repo-relative path -> content) for the
// detected languages.
func Files(langs []Language) map[string]string {
	return map[string]string{
		".github/workflows/coverage.yml": workflow(langs),
		".coverageignore":                coverageIgnore,
		"coverage.yaml":                  coverageYAML,
	}
}

// Detect inspects marker files under dir and returns the languages found,
// sorted by name.
func Detect(dir string) []Language {
	var out []Language
	has := func(name string) bool {
		_, err := os.Stat(filepath.Join(dir, name))
		return err == nil
	}
	glob := func(pattern string) bool {
		m, _ := filepath.Glob(filepath.Join(dir, pattern))
		return len(m) > 0
	}

	if has("go.mod") {
		out = append(out, Language{Name: "Go", ID: "go", JobName: "test-go", Doc: "GO.md"})
	}
	if has("package.json") {
		out = append(out, Language{Name: "TypeScript/JavaScript", ID: "web", JobName: "test-web", Doc: "TYPESCRIPT.md"})
	}
	if has("pyproject.toml") || has("setup.py") || has("setup.cfg") || has("tox.ini") || has("pytest.ini") || has("requirements.txt") {
		out = append(out, Language{Name: "Python", ID: "py", JobName: "test-py", Doc: "PYTHON.md"})
	}
	if has("Cargo.toml") {
		out = append(out, Language{Name: "Rust", ID: "rust", JobName: "test-rust", Doc: "RUST.md"})
	}
	if has("pom.xml") || has("build.gradle") || has("build.gradle.kts") {
		out = append(out, Language{Name: "Java", ID: "java", JobName: "test-java", Doc: "JAVA.md"})
	}
	if glob("*.csproj") || glob("*.sln") {
		out = append(out, Language{Name: "C#/.NET", ID: "dotnet", JobName: "test-dotnet", Doc: "CSHARP.md"})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// workflow composes the full coverage workflow from the detected languages.
func workflow(langs []Language) string {
	var b strings.Builder
	b.WriteString(workflowHeader)
	for _, l := range langs {
		b.WriteString(stubJob(l))
		b.WriteString("\n")
	}
	needs := make([]string, len(langs))
	for i, l := range langs {
		needs[i] = l.JobName
	}
	fmt.Fprintf(&b, reportJob, "["+strings.Join(needs, ", ")+"]")
	return b.String()
}
