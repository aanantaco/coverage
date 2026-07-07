// Package junit parses JUnit XML purely to surface a test count per workspace.
//
// It accepts both a <testsuites> root (Jest, Vitest, gotestsum default) and a
// bare <testsuite> root (older emitters). When the root omits the rolled-up
// tests attribute, the child <testsuite> elements are summed.
package junit

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
)

// Report is the aggregated test-count view of a JUnit document.
type Report struct {
	Tests    int
	Failures int
	Errors   int
	Skipped  int
}

type xmlSuites struct {
	XMLName  xml.Name   `xml:"testsuites"`
	Tests    *int       `xml:"tests,attr"`
	Failures *int       `xml:"failures,attr"`
	Errors   *int       `xml:"errors,attr"`
	Skipped  *int       `xml:"skipped,attr"`
	Suites   []xmlSuite `xml:"testsuite"`
}

type xmlSuite struct {
	XMLName  xml.Name `xml:"testsuite"`
	Tests    int      `xml:"tests,attr"`
	Failures int      `xml:"failures,attr"`
	Errors   int      `xml:"errors,attr"`
	Skipped  int      `xml:"skipped,attr"`
}

// ParseFile reads and parses a JUnit XML file at path.
func ParseFile(path string) (*Report, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

// Parse parses JUnit XML from r. An empty input yields a zero Report with a nil
// error, so "no tests" is a valid state rather than a crash.
func Parse(r io.Reader) (*Report, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(string(data))
	if len(trimmed) == 0 {
		return &Report{}, nil
	}

	// Determine the root element by decoding the first start token.
	root, err := rootElement(trimmed)
	if err != nil {
		return nil, err
	}

	switch root {
	case "testsuites":
		var suites xmlSuites
		if err := xml.Unmarshal(data, &suites); err != nil {
			return nil, fmt.Errorf("parse junit xml: %w", err)
		}
		return fromSuites(suites), nil
	case "testsuite":
		var suite xmlSuite
		if err := xml.Unmarshal(data, &suite); err != nil {
			return nil, fmt.Errorf("parse junit xml: %w", err)
		}
		return &Report{
			Tests:    suite.Tests,
			Failures: suite.Failures,
			Errors:   suite.Errors,
			Skipped:  suite.Skipped,
		}, nil
	default:
		return nil, fmt.Errorf("parse junit xml: unexpected root element %q", root)
	}
}

// fromSuites prefers the rolled-up attributes on <testsuites>; when a field is
// absent it sums the corresponding child <testsuite> values.
func fromSuites(s xmlSuites) *Report {
	sum := func(get func(xmlSuite) int) int {
		total := 0
		for _, suite := range s.Suites {
			total += get(suite)
		}
		return total
	}
	pick := func(rollup *int, get func(xmlSuite) int) int {
		if rollup != nil {
			return *rollup
		}
		return sum(get)
	}
	return &Report{
		Tests:    pick(s.Tests, func(x xmlSuite) int { return x.Tests }),
		Failures: pick(s.Failures, func(x xmlSuite) int { return x.Failures }),
		Errors:   pick(s.Errors, func(x xmlSuite) int { return x.Errors }),
		Skipped:  pick(s.Skipped, func(x xmlSuite) int { return x.Skipped }),
	}
}

// rootElement returns the local name of the first XML start element.
func rootElement(s string) (string, error) {
	dec := xml.NewDecoder(strings.NewReader(s))
	for {
		tok, err := dec.Token()
		if err != nil {
			return "", fmt.Errorf("parse junit xml: %w", err)
		}
		if start, ok := tok.(xml.StartElement); ok {
			return start.Name.Local, nil
		}
	}
}
