// Package ignore wraps go-gitignore to provide an optional .coverageignore
// matcher. When no ignore file is present the returned matcher matches nothing,
// which makes the ignore file entirely optional.
package ignore

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	gitignore "github.com/sabhiram/go-gitignore"
)

// Matcher reports whether a repo-root-relative path should be excluded.
type Matcher interface {
	Match(path string) bool
}

// passthrough matches nothing; it is used when no ignore file is loaded.
type passthrough struct{}

func (passthrough) Match(string) bool { return false }

// gitignoreMatcher adapts a compiled go-gitignore instance to Matcher.
type gitignoreMatcher struct {
	gi *gitignore.GitIgnore
}

func (m gitignoreMatcher) Match(path string) bool {
	return m.gi.MatchesPath(path)
}

// Load returns a Matcher for the ignore file at path.
//
//   - path == ""          -> passthrough matcher, exists=false
//   - file does not exist -> passthrough matcher, exists=false
//   - file exists         -> compiled gitignore matcher, exists=true
//   - stat error (not ENOENT) or compile error -> error
func Load(path string) (matcher Matcher, exists bool, err error) {
	if path == "" {
		return passthrough{}, false, nil
	}

	if _, statErr := os.Stat(path); statErr != nil {
		if errors.Is(statErr, fs.ErrNotExist) {
			return passthrough{}, false, nil
		}
		return nil, false, fmt.Errorf("stat ignore file %q: %w", path, statErr)
	}

	gi, compileErr := gitignore.CompileIgnoreFile(path)
	if compileErr != nil {
		return nil, false, fmt.Errorf("compile ignore file %q: %w", path, compileErr)
	}
	return gitignoreMatcher{gi: gi}, true, nil
}
