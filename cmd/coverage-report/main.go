// Command coverage-report aggregates Cobertura coverage and JUnit test-result
// artifacts into a single Markdown summary for a CI run, with optional
// config-driven display and regression detection against a baseline.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/aanantaco/coverage/internal/app"
)

func main() {
	fs := flag.NewFlagSet("coverage-report", flag.ContinueOnError)

	input := fs.String("input", "", "directory containing coverage-*.xml and tests-*.xml artifacts (required)")
	output := fs.String("output", "-", "output path; '-' is stdout. A file is appended to.")
	ignorePath := fs.String("ignore", "", "path to a .coverageignore file (gitignore syntax)")
	configPath := fs.String("config", "", "path to coverage.yaml (default: ./coverage.yaml if present)")
	baselinePath := fs.String("baseline", "", "path to a baseline coverage-summary.json for regression detection")
	failOnDrop := fs.Float64("fail-on-drop", 0, "fail (exit non-zero) if total line coverage drops by more than this many percentage points")
	emitJSON := fs.String("emit-json", "", "also write a machine-readable coverage-summary.json to this path")
	verbose := fs.Bool("verbose", false, "log warnings for workspaces missing a config entry")

	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}

	if *input == "" {
		fmt.Fprintln(os.Stderr, "error: --input is required")
		fs.Usage()
		os.Exit(2)
	}

	opts := app.Options{
		Input:        *input,
		Output:       *output,
		IgnorePath:   *ignorePath,
		IgnoreSet:    flagSet(fs, "ignore"),
		ConfigPath:   *configPath,
		ConfigSet:    flagSet(fs, "config"),
		BaselinePath: *baselinePath,
		BaselineSet:  flagSet(fs, "baseline"),
		EmitJSON:     *emitJSON,
		Verbose:      *verbose,
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
	}
	if flagSet(fs, "fail-on-drop") {
		opts.FailOnDrop = failOnDrop
	}

	if err := app.Run(opts); err != nil {
		if errors.Is(err, app.ErrCoverageDropped) {
			// The report was already written; exit non-zero to fail the build.
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// flagSet reports whether the named flag was explicitly provided.
func flagSet(fs *flag.FlagSet, name string) bool {
	found := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
