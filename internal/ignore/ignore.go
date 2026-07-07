// Package ignore provides an optional .coverageignore matcher using
// gitignore-pattern syntax. When no ignore file is present the returned matcher
// matches nothing, which makes the ignore file entirely optional.
//
// Pattern matching is implemented in-repo (see gitignore.go) with no external
// dependency.
package ignore

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// Matcher reports whether a repo-root-relative path should be excluded.
type Matcher interface {
	Match(path string) bool
}

// passthrough matches nothing; it is used when no ignore file is loaded.
type passthrough struct{}

func (passthrough) Match(string) bool { return false }

// gitignoreMatcher adapts a compiled gitignore pattern set to Matcher.
type gitignoreMatcher struct {
	gi *gitIgnore
}

func (m gitignoreMatcher) Match(path string) bool {
	return m.gi.MatchesPath(path)
}

// Load returns a Matcher for the ignore file at path.
//
//   - path == ""          -> passthrough matcher, exists=false
//   - file does not exist -> passthrough matcher, exists=false
//   - file exists         -> compiled gitignore matcher, exists=true
//   - stat/read error (not ENOENT) -> error
func Load(path string) (matcher Matcher, exists bool, err error) {
	if path == "" {
		return passthrough{}, false, nil
	}

	data, readErr := os.ReadFile(path)
	if readErr != nil {
		if errors.Is(readErr, fs.ErrNotExist) {
			return passthrough{}, false, nil
		}
		return nil, false, fmt.Errorf("read ignore file %q: %w", path, readErr)
	}

	lines := strings.Split(string(data), "\n")
	return gitignoreMatcher{gi: compileGitIgnoreLines(lines)}, true, nil
}
