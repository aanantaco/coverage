package junit

import (
	"strings"
	"testing"
)

func TestParseTestsuitesRollup(t *testing.T) {
	const doc = `<testsuites tests="412" failures="2" errors="1" skipped="3">
      <testsuite tests="200" failures="1"/>
      <testsuite tests="212" failures="1"/>
    </testsuites>`
	r, err := Parse(strings.NewReader(doc))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if r.Tests != 412 || r.Failures != 2 || r.Errors != 1 || r.Skipped != 3 {
		t.Errorf("rollup mismatch: %+v", r)
	}
}

func TestParseTestsuitesSumsWhenRollupMissing(t *testing.T) {
	// gotestsum sometimes omits the tests attribute on the root.
	const doc = `<testsuites>
      <testsuite tests="10" failures="1" errors="0" skipped="2"/>
      <testsuite tests="5" failures="0" errors="1" skipped="0"/>
    </testsuites>`
	r, err := Parse(strings.NewReader(doc))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if r.Tests != 15 || r.Failures != 1 || r.Errors != 1 || r.Skipped != 2 {
		t.Errorf("sum mismatch: %+v", r)
	}
}

func TestParseBareTestsuite(t *testing.T) {
	const doc = `<testsuite tests="7" failures="1" errors="0" skipped="1"/>`
	r, err := Parse(strings.NewReader(doc))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if r.Tests != 7 || r.Failures != 1 || r.Skipped != 1 {
		t.Errorf("bare testsuite mismatch: %+v", r)
	}
}

func TestParseEmptyInputIsZero(t *testing.T) {
	r, err := Parse(strings.NewReader(""))
	if err != nil {
		t.Fatalf("Parse empty: %v", err)
	}
	if (*r != Report{}) {
		t.Errorf("expected zero report, got %+v", r)
	}
}

func TestParseZeroTestsPlaceholder(t *testing.T) {
	const doc = `<?xml version="1.0" encoding="UTF-8"?>
<testsuites tests="0" failures="0" errors="0" skipped="0"/>`
	r, err := Parse(strings.NewReader(doc))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if r.Tests != 0 {
		t.Errorf("expected 0 tests, got %d", r.Tests)
	}
}
