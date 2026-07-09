package main

import (
	"strings"
	"testing"
)

func TestPrintVersion(t *testing.T) {
	origVersion, origCommit := version, commit
	t.Cleanup(func() { version, commit = origVersion, origCommit })

	tests := []struct {
		name    string
		version string
		commit  string
		want    string
	}{
		{"dev build", "dev", "", "coverage dev\n"},
		{"with commit", "0.0.0-abc1234", "abc1234def", "coverage 0.0.0-abc1234 (abc1234def)\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			version, commit = tc.version, tc.commit
			var b strings.Builder
			printVersion(&b)
			if got := b.String(); got != tc.want {
				t.Errorf("printVersion() = %q, want %q", got, tc.want)
			}
		})
	}
}
