package scaffold

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	yaml "github.com/goccy/go-yaml"
)

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func names(langs []Language) []string {
	out := make([]string, len(langs))
	for i, l := range langs {
		out[i] = l.Name
	}
	return out
}

func TestDetect(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "go.mod", "module x\n")
	write(t, dir, "pyproject.toml", "[project]\n")
	write(t, dir, "Cargo.toml", "[package]\n")
	write(t, dir, "pom.xml", "<project/>\n")
	write(t, dir, "svc.csproj", "<Project/>\n")
	write(t, dir, "package.json", `{"devDependencies":{"jest":"^29"}}`)

	got := strings.Join(names(Detect(dir)), ",")
	want := "C#/.NET,Go,Java,Python,Rust,TypeScript/JavaScript" // sorted by name
	if got != want {
		t.Errorf("Detect = %q, want %q", got, want)
	}
}

func TestDetectNone(t *testing.T) {
	if langs := Detect(t.TempDir()); len(langs) != 0 {
		t.Errorf("expected no languages, got %v", names(langs))
	}
}

func TestNodeVariant(t *testing.T) {
	jestDir := t.TempDir()
	write(t, jestDir, "package.json", `{"devDependencies":{"jest":"^29"}}`)
	if job := nodeJob(jestDir); !strings.Contains(job, "jest --coverage") {
		t.Error("expected jest job for a jest package.json")
	}

	vitestDir := t.TempDir()
	write(t, vitestDir, "package.json", `{"devDependencies":{"vitest":"^1"}}`)
	if job := nodeJob(vitestDir); !strings.Contains(job, "vitest run") {
		t.Error("expected vitest job for a vitest package.json")
	}
}

func TestGeneratedWorkflowIsValidYAML(t *testing.T) {
	// Every language present, to exercise all job snippets.
	langs := []Language{
		{Name: "Go", ID: "go", JobName: "test-go", job: goJob},
		{Name: "TypeScript/JavaScript", ID: "web", JobName: "test-web", job: nodeVitestJob},
		{Name: "Python", ID: "py", JobName: "test-py", job: pythonJob},
		{Name: "Rust", ID: "rust", JobName: "test-rust", job: rustJob},
		{Name: "Java", ID: "java", JobName: "test-java", job: javaJob},
		{Name: "C#/.NET", ID: "dotnet", JobName: "test-dotnet", job: dotnetJob},
	}
	wf := workflow(langs)

	var doc struct {
		Jobs map[string]any `yaml:"jobs"`
	}
	if err := yaml.Unmarshal([]byte(wf), &doc); err != nil {
		t.Fatalf("generated workflow is not valid YAML: %v\n%s", err, wf)
	}
	for _, want := range []string{"test-go", "test-web", "test-py", "test-rust", "test-java", "test-dotnet", "report"} {
		if _, ok := doc.Jobs[want]; !ok {
			t.Errorf("workflow missing job %q", want)
		}
	}
	if !strings.Contains(wf, "needs: [test-go, test-web, test-py, test-rust, test-java, test-dotnet]") {
		t.Error("report job needs list is wrong")
	}
}

func TestRunNonDestructive(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "go.mod", "module x\n")

	var out bytes.Buffer
	if err := Run(Options{Dir: dir, Stdout: &out, Stderr: &bytes.Buffer{}}); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{".github/workflows/coverage.yml", ".coverageignore", "coverage.yaml"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("expected %s to be created: %v", f, err)
		}
	}
	if !strings.Contains(out.String(), "3 created, 0 skipped") {
		t.Errorf("unexpected summary: %s", out.String())
	}

	// Pre-existing file must not be overwritten.
	sentinel := "# do not touch\n"
	write(t, dir, ".coverageignore", sentinel)
	out.Reset()
	if err := Run(Options{Dir: dir, Stdout: &out, Stderr: &bytes.Buffer{}}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, ".coverageignore"))
	if string(data) != sentinel {
		t.Error(".coverageignore was overwritten")
	}
	if !strings.Contains(out.String(), "0 created, 3 skipped") {
		t.Errorf("expected all skipped on re-run: %s", out.String())
	}
}

func TestRunDryRunWritesNothing(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "go.mod", "module x\n")
	var out bytes.Buffer
	if err := Run(Options{Dir: dir, DryRun: true, Stdout: &out, Stderr: &bytes.Buffer{}}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".coverageignore")); !os.IsNotExist(err) {
		t.Error("dry-run should not write files")
	}
	if !strings.Contains(out.String(), "dry-run") {
		t.Errorf("expected dry-run markers: %s", out.String())
	}
}

func TestRunNoLanguages(t *testing.T) {
	dir := t.TempDir()
	var out, errb bytes.Buffer
	if err := Run(Options{Dir: dir, Stdout: &out, Stderr: &errb}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".coverageignore")); !os.IsNotExist(err) {
		t.Error("nothing should be written when no languages are detected")
	}
	if !strings.Contains(errb.String(), "no supported languages") {
		t.Errorf("expected a no-languages message, got: %s", errb.String())
	}
}
