package ignore

import "testing"

func matcherFrom(lines ...string) *gitIgnore {
	return compileGitIgnoreLines(lines)
}

func TestGitIgnoreDoubleStarPrefix(t *testing.T) {
	gi := matcherFrom("**/*_test.go", "**/node_modules/**")
	cases := map[string]bool{
		"foo_test.go":                    true,
		"a/b/foo_test.go":                true,
		"a/b/foo.go":                     false,
		"pkg/node_modules/left-pad/x.js": true,
		"node_modules/x.js":              true,
		"src/lib.js":                     false,
	}
	for path, want := range cases {
		if got := gi.MatchesPath(path); got != want {
			t.Errorf("MatchesPath(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestGitIgnoreTrailingDoubleStar(t *testing.T) {
	gi := matcherFrom("shared/migration/**", "cmd/**", "schemata/**")
	cases := map[string]bool{
		"shared/migration/001.sql":     true,
		"shared/migration/sub/002.sql": true,
		"cmd/tool/main.go":             true,
		"schemata/x.sql":               true,
		"internal/cmd.go":              false,
		// A mid-slash pattern still floats under a leading path segment. This
		// matches the semantics of the go-gitignore library this code was
		// ported from (it prepends "^(|.*/)" to non-"/"-anchored patterns),
		// which is what the original .coverageignore patterns were written for.
		"services/shared/migration/001.sql": true,
	}
	for path, want := range cases {
		if got := gi.MatchesPath(path); got != want {
			t.Errorf("MatchesPath(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestGitIgnoreStarDoesNotCrossSlash(t *testing.T) {
	gi := matcherFrom("**/store/*.go")
	cases := map[string]bool{
		"services/worker/internal/store/gen.go": true,
		"store/gen.go":                          true,
		"store/nested/gen.go":                   false, // *.go must not cross a slash
	}
	for path, want := range cases {
		if got := gi.MatchesPath(path); got != want {
			t.Errorf("MatchesPath(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestGitIgnoreNegation(t *testing.T) {
	// Exclude all *.go under gen/, but re-include keep.go.
	gi := matcherFrom("**/gen/**", "!**/gen/keep.go")
	if !gi.MatchesPath("a/gen/thing.go") {
		t.Error("expected a/gen/thing.go to be ignored")
	}
	if gi.MatchesPath("a/gen/keep.go") {
		t.Error("expected a/gen/keep.go to be re-included by negation")
	}
}

func TestGitIgnoreCommentsAndBlankLines(t *testing.T) {
	gi := matcherFrom("# a comment", "", "   ", "vendor/**")
	if len(gi.patterns) != 1 {
		t.Fatalf("expected 1 compiled pattern, got %d", len(gi.patterns))
	}
	if !gi.MatchesPath("vendor/x/y.go") {
		t.Error("expected vendor/x/y.go to match")
	}
}

func TestGitIgnoreDotIsLiteral(t *testing.T) {
	gi := matcherFrom("**/*.test.ts")
	if !gi.MatchesPath("src/a.test.ts") {
		t.Error("expected a.test.ts to match")
	}
	if gi.MatchesPath("src/axtestxts") {
		t.Error("dot should be literal, not any-char")
	}
}
