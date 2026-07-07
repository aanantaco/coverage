package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "coverage.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDefault(t *testing.T) {
	c := Default()
	if c.FolderGroupDepth != DefaultFolderGroupDepth {
		t.Errorf("depth = %d", c.FolderGroupDepth)
	}
	if c.DisplayFrom != "id" {
		t.Errorf("display_from = %q", c.DisplayFrom)
	}
}

func TestLoadFullConfig(t *testing.T) {
	path := writeConfig(t, `
folder_group_depth: 2
ignore_file: .coverageignore
baseline:
  path: .coverage-baseline.json
  fail_on_drop: 0.5
display_from: shared-slash
workspaces:
  thingy:
    display_name: services/thingy
    prefix: services/thingy/
    strip_prefix: github.com/acme/monorepo/services/thingy/
  shared-widget:
    prefix: shared/widget/
`)
	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.FolderGroupDepth != 2 {
		t.Errorf("depth = %d", c.FolderGroupDepth)
	}
	if c.Baseline.Path != ".coverage-baseline.json" {
		t.Errorf("baseline path = %q", c.Baseline.Path)
	}
	if c.Baseline.FailOnDrop == nil || *c.Baseline.FailOnDrop != 0.5 {
		t.Errorf("fail_on_drop = %v", c.Baseline.FailOnDrop)
	}
	ws := c.Workspaces["thingy"]
	if ws.StripPrefix != "github.com/acme/monorepo/services/thingy/" {
		t.Errorf("strip_prefix = %q", ws.StripPrefix)
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	path := writeConfig(t, "workspaces:\n  a:\n    prefix: a/\n")
	c, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.FolderGroupDepth != DefaultFolderGroupDepth {
		t.Errorf("depth default not applied: %d", c.FolderGroupDepth)
	}
	if c.DisplayFrom != "id" {
		t.Errorf("display_from default not applied: %q", c.DisplayFrom)
	}
}

func TestLoadUnknownFieldErrors(t *testing.T) {
	path := writeConfig(t, "folder_group_depth: 3\nbogus_field: 1\n")
	if _, err := Load(path); err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestLoadInvalidDepthErrors(t *testing.T) {
	path := writeConfig(t, "folder_group_depth: -1\n")
	if _, err := Load(path); err == nil {
		t.Fatal("expected error for negative depth")
	}
}

func TestLoadInvalidDisplayFromErrors(t *testing.T) {
	path := writeConfig(t, "display_from: bananas\n")
	if _, err := Load(path); err == nil {
		t.Fatal("expected error for bad display_from")
	}
}

func TestLoadMalformedYAMLErrors(t *testing.T) {
	path := writeConfig(t, "folder_group_depth: [not, an, int]\n")
	if _, err := Load(path); err == nil {
		t.Fatal("expected error for malformed yaml")
	}
}

func TestDisplayName(t *testing.T) {
	c := Default()
	c.DisplayFrom = "shared-slash"
	c.Workspaces = map[string]Workspace{
		"thingy": {DisplayName: "services/thingy"},
	}
	if got := c.DisplayName("thingy"); got != "services/thingy" {
		t.Errorf("explicit display_name = %q", got)
	}
	if got := c.DisplayName("shared-widget"); got != "shared/widget" {
		t.Errorf("shared-slash = %q", got)
	}
	if got := c.DisplayName("plain"); got != "plain" {
		t.Errorf("plain = %q", got)
	}

	c.DisplayFrom = "id"
	if got := c.DisplayName("shared-widget"); got != "shared-widget" {
		t.Errorf("id rule = %q", got)
	}
}
