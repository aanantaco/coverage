package ignore

// This file is an in-repo, dependency-free implementation of gitignore-pattern
// matching. The line-to-regexp compilation is a faithful port of
// github.com/sabhiram/go-gitignore (MIT License, Copyright (c) 2015 Shaba
// Abhiram), which this project previously depended on. Porting it in preserves
// the exact matching semantics our .coverageignore patterns were written
// against while dropping the external module.
//
// Original: https://github.com/sabhiram/go-gitignore/blob/master/ignore.go

import (
	"regexp"
	"strings"
)

// gitIgnore is a compiled set of gitignore patterns.
type gitIgnore struct {
	patterns []gitPattern
}

type gitPattern struct {
	re     *regexp.Regexp
	negate bool
}

// compileGitIgnoreLines compiles raw gitignore lines into a matcher. Blank
// lines and comments are skipped; a line that fails to compile is skipped.
func compileGitIgnoreLines(lines []string) *gitIgnore {
	gi := &gitIgnore{}
	for _, line := range lines {
		re, negate, ok := compilePattern(line)
		if !ok {
			continue
		}
		gi.patterns = append(gi.patterns, gitPattern{re: re, negate: negate})
	}
	return gi
}

// MatchesPath reports whether f is matched by the pattern set, honoring later
// negation (`!`) patterns that re-include a previously matched path.
func (gi *gitIgnore) MatchesPath(f string) bool {
	f = strings.ReplaceAll(f, "\\", "/")
	matched := false
	for _, p := range gi.patterns {
		if p.re.MatchString(f) {
			if !p.negate {
				matched = true
			} else if matched {
				matched = false
			}
		}
	}
	return matched
}

// magicStar is a placeholder used while translating `**` sequences so that a
// later `*` -> `[^/]*` substitution does not clobber them.
const magicStar = "#$~"

var (
	reLeadingHashOrBang = regexp.MustCompile(`^(\#|\!)`)
	reFolderGlobPrefix  = regexp.MustCompile(`([^\/+])/.*\*\.`)
	reDot               = regexp.MustCompile(`\.`)
	reDoubleStarSlash   = regexp.MustCompile(`/\*\*/`)
	reTrailingDStar     = regexp.MustCompile(`/\*\*`)
	reLeadingDStar      = regexp.MustCompile(`\*\*/`)
	reEscapedStar       = regexp.MustCompile(`\\\*`)
	reStar              = regexp.MustCompile(`\*`)
)

// compilePattern converts a single gitignore line into a regexp. ok is false
// for comments, blank lines, and lines that fail to compile.
func compilePattern(line string) (re *regexp.Regexp, negate bool, ok bool) {
	// Trim OS-specific carriage returns.
	line = strings.TrimRight(line, "\r")

	// Strip comment lines.
	if strings.HasPrefix(line, "#") {
		return nil, false, false
	}

	// Trim surrounding spaces.
	line = strings.Trim(line, " ")
	if line == "" {
		return nil, false, false
	}

	// A leading "!" negates the pattern.
	if line[0] == '!' {
		negate = true
		line = line[1:]
	}

	// An escaped leading "#" or "!" is a literal.
	if reLeadingHashOrBang.MatchString(line) {
		line = line[1:]
	}

	// foo/*.blah inside a folder gets a leading "/" so it anchors to the folder.
	if reFolderGlobPrefix.MatchString(line) && line[0] != '/' {
		line = "/" + line
	}

	// Escape literal dots.
	line = reDot.ReplaceAllString(line, `\.`)

	// Translate "**" sequences.
	if strings.HasPrefix(line, "/**/") {
		line = line[1:]
	}
	line = reDoubleStarSlash.ReplaceAllString(line, `(/|/.+/)`)
	line = reLeadingDStar.ReplaceAllString(line, `(|.`+magicStar+`/)`)
	line = reTrailingDStar.ReplaceAllString(line, `(|/.`+magicStar+`)`)

	// Escaped "*" becomes a literal star (via the magic placeholder); a bare
	// "*" matches any run of non-separator characters.
	line = reEscapedStar.ReplaceAllString(line, `\`+magicStar)
	line = reStar.ReplaceAllString(line, `[^/]*`)

	// Escape "?".
	line = strings.ReplaceAll(line, "?", `\?`)

	// Restore the magic placeholder to a real "*".
	line = strings.ReplaceAll(line, magicStar, "*")

	// Anchor the expression.
	var expr string
	if strings.HasSuffix(line, "/") {
		expr = line + "(|.*)$"
	} else {
		expr = line + "(|/.*)$"
	}
	if strings.HasPrefix(expr, "/") {
		expr = "^(|/)" + expr[1:]
	} else {
		expr = "^(|.*/)" + expr
	}

	compiled, err := regexp.Compile(expr)
	if err != nil {
		return nil, false, false
	}
	return compiled, negate, true
}
