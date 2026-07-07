package ignore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEmptyPathPassthrough(t *testing.T) {
	m, exists, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if exists {
		t.Error("exists should be false for empty path")
	}
	if m.Match("anything/at/all.go") {
		t.Error("passthrough matcher should match nothing")
	}
}

func TestLoadMissingFilePassthrough(t *testing.T) {
	m, exists, err := Load(filepath.Join(t.TempDir(), "nope.coverageignore"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if exists {
		t.Error("exists should be false for missing file")
	}
	if m.Match("x.go") {
		t.Error("passthrough matcher should match nothing")
	}
}

func TestLoadExistingFileMatches(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".coverageignore")
	content := "**/*_test.go\nservices/worker/internal/store/**\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	m, exists, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !exists {
		t.Error("exists should be true")
	}
	cases := map[string]bool{
		"services/api/foo_test.go":               true,
		"services/worker/internal/store/gen.go":  true,
		"services/api/foo.go":                    false,
		"services/worker/internal/logic/calc.go": false,
	}
	for path, want := range cases {
		if got := m.Match(path); got != want {
			t.Errorf("Match(%q) = %v, want %v", path, got, want)
		}
	}
}
