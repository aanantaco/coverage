// Package config loads the optional coverage.yaml file that drives workspace
// display names, ignore-path prefixes, folder grouping, and regression
// detection.
//
// The file is entirely optional: with no file present the tool runs with
// sensible defaults. A file that is present but malformed is a hard error —
// there are no silent fallbacks.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultFolderGroupDepth is the number of leading path components a folder
	// group is truncated to.
	DefaultFolderGroupDepth = 3
	// DefaultDisplayFrom is the fallback display rule.
	DefaultDisplayFrom = "id"
	// DefaultIgnoreFile is the conventional ignore-file name used when the
	// config does not specify one.
	DefaultIgnoreFile = ".coverageignore"
)

// Workspace holds per-workspace overrides. Every field is optional.
type Workspace struct {
	// DisplayName is the label shown in the report. Empty => derived from id.
	DisplayName string `yaml:"display_name"`
	// Prefix is prepended to each class filename before ignore matching, so a
	// repo-root ignore file can target this workspace's files.
	Prefix string `yaml:"prefix"`
	// StripPrefix is removed from each class filename before Prefix is applied.
	// Use for Go emitters that write full module import paths.
	StripPrefix string `yaml:"strip_prefix"`
}

// Baseline configures regression detection. All fields optional.
type Baseline struct {
	// Path is where the previous run's summary JSON is read from. Empty =>
	// regression detection disabled.
	Path string `yaml:"path"`
	// FailOnDrop, when set, makes the process exit non-zero if total line
	// coverage drops by more than this many percentage points. Nil => never
	// fail, annotate only.
	FailOnDrop *float64 `yaml:"fail_on_drop"`
}

// Config is the parsed coverage.yaml with defaults applied.
type Config struct {
	FolderGroupDepth int                  `yaml:"folder_group_depth"`
	IgnoreFile       string               `yaml:"ignore_file"`
	Baseline         Baseline             `yaml:"baseline"`
	Workspaces       map[string]Workspace `yaml:"workspaces"`
	DisplayFrom      string               `yaml:"display_from"`
}

// Default returns a Config populated entirely with defaults.
func Default() *Config {
	return &Config{
		FolderGroupDepth: DefaultFolderGroupDepth,
		IgnoreFile:       "",
		Workspaces:       map[string]Workspace{},
		DisplayFrom:      DefaultDisplayFrom,
	}
}

// Load reads, parses, validates, and default-fills the config at path. The file
// must exist; a missing file is the caller's responsibility to handle (see
// app orchestration). Unknown keys and invalid values are errors.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	// Start from a zero Config so we can tell which fields were set, then
	// decode strictly to catch typos.
	var cfg Config
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		// An empty file decodes to io.EOF from the decoder; treat that as an
		// all-defaults config rather than an error.
		if errors.Is(err, io.EOF) {
			return Default(), nil
		}
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}

	applyDefaults(&cfg)
	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config %q: %w", path, err)
	}
	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.FolderGroupDepth == 0 {
		cfg.FolderGroupDepth = DefaultFolderGroupDepth
	}
	if cfg.DisplayFrom == "" {
		cfg.DisplayFrom = DefaultDisplayFrom
	}
	if cfg.Workspaces == nil {
		cfg.Workspaces = map[string]Workspace{}
	}
}

func validate(cfg *Config) error {
	if cfg.FolderGroupDepth < 1 {
		return fmt.Errorf("folder_group_depth must be >= 1, got %d", cfg.FolderGroupDepth)
	}
	switch cfg.DisplayFrom {
	case "id", "shared-slash":
	default:
		return fmt.Errorf("display_from must be \"id\" or \"shared-slash\", got %q", cfg.DisplayFrom)
	}
	if cfg.Baseline.FailOnDrop != nil && *cfg.Baseline.FailOnDrop < 0 {
		return fmt.Errorf("baseline.fail_on_drop must be >= 0, got %g", *cfg.Baseline.FailOnDrop)
	}
	return nil
}

// DisplayName returns the display label for a workspace id, honoring an
// explicit override, otherwise the DisplayFrom rule.
func (c *Config) DisplayName(id string) string {
	if ws, ok := c.Workspaces[id]; ok && ws.DisplayName != "" {
		return ws.DisplayName
	}
	switch c.DisplayFrom {
	case "shared-slash":
		return sharedSlash(id)
	default:
		return id
	}
}

// sharedSlash maps a leading "shared-" prefix to "shared/" (compat with the
// original tool); otherwise returns id unchanged.
func sharedSlash(id string) string {
	const p = "shared-"
	if len(id) > len(p) && id[:len(p)] == p {
		return "shared/" + id[len(p):]
	}
	return id
}
