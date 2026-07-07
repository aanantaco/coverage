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

func TestGeneratedWorkflowIsValidYAML(t *testing.T) {
	langs := []Language{
		{Name: "Go", ID: "go", JobName: "test-go", Doc: "GO.md"},
		{Name: "TypeScript/JavaScript", ID: "web", JobName: "test-web", Doc: "TYPESCRIPT.md"},
		{Name: "Python", ID: "py", JobName: "test-py", Doc: "PYTHON.md"},
		{Name: "Rust", ID: "rust", JobName: "test-rust", Doc: "RUST.md"},
		{Name: "Java", ID: "java", JobName: "test-java", Doc: "JAVA.md"},
		{Name: "C#/.NET", ID: "dotnet", JobName: "test-dotnet", Doc: "CSHARP.md"},
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
	// Stub jobs must point at the docs and NOT bake in framework commands.
	if !strings.Contains(wf, "docs/GO.md") || !strings.Contains(wf, "docs/RUST.md") {
		t.Error("stub jobs should link to the per-language docs")
	}
	for _, leaked := range []string{"vitest", "gotestsum", "cargo llvm-cov", "pytest", "mvn ", "dotnet test"} {
		if strings.Contains(wf, leaked) {
			t.Errorf("generated workflow should not bake in framework command %q", leaked)
		}
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
