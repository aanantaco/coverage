package junit

import (
	"errors"
	"os"
	"path/filepath"
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

func TestParseFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tests-x.xml")
	if err := os.WriteFile(path, []byte(`<testsuite tests="4" failures="0"/>`), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if r.Tests != 4 {
		t.Errorf("Tests = %d, want 4", r.Tests)
	}
}

func TestParseFileMissing(t *testing.T) {
	if _, err := ParseFile(filepath.Join(t.TempDir(), "nope.xml")); err == nil {
		t.Fatal("expected an error for a missing file")
	}
}

func TestParseUnexpectedRoot(t *testing.T) {
	_, err := Parse(strings.NewReader(`<report tests="1"/>`))
	if err == nil || !strings.Contains(err.Error(), "unexpected root element") {
		t.Fatalf("expected unexpected-root error, got %v", err)
	}
}

func TestParseMalformedTestsuitesAttr(t *testing.T) {
	// Well-formed enough to identify the root, but the attribute is not an int,
	// so the <testsuites> unmarshal fails.
	_, err := Parse(strings.NewReader(`<testsuites><testsuite tests="lots"/></testsuites>`))
	if err == nil {
		t.Fatal("expected a parse error for a non-numeric tests attribute")
	}
}

func TestParseMalformedBareTestsuiteAttr(t *testing.T) {
	_, err := Parse(strings.NewReader(`<testsuite tests="lots"/>`))
	if err == nil {
		t.Fatal("expected a parse error for a non-numeric tests attribute")
	}
}

func TestParseNoStartElement(t *testing.T) {
	// Non-empty but contains no start element; rootElement hits EOF first.
	_, err := Parse(strings.NewReader("<!-- just a comment -->"))
	if err == nil {
		t.Fatal("expected an error when there is no root element")
	}
}

// errReader fails on the first read, exercising the io.ReadAll error path.
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestParseReadError(t *testing.T) {
	if _, err := Parse(errReader{}); err == nil {
		t.Fatal("expected a read error to propagate")
	}
}
