// Command coverage aggregates Cobertura coverage and JUnit test-result
// artifacts into a single Markdown (or HTML) report for a CI run, with optional
// config-driven display and regression detection against a baseline.
//
// Subcommands:
//
//	coverage --input <dir> [flags]   render a coverage report (default)
//	coverage init [flags]            scaffold a workflow + config for this repo
//	coverage version                 print the build version and commit
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/aanantaco/coverage/internal/app"
	"github.com/aanantaco/coverage/internal/scaffold"
)

// version and commit are stamped at build time via -ldflags -X (see
// .goreleaser.yaml). A plain `go build` leaves the defaults, so a
// locally-built binary reports "dev".
var (
	version = "dev"
	commit  = ""
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "init":
			runInit(os.Args[2:])
			return
		case "version", "--version", "-version":
			printVersion(os.Stdout)
			return
		}
	}
	runReport(os.Args[1:])
}

// printVersion writes the build version, appending the commit SHA when it was
// stamped in — the SHA is the thing consumers pin releases by.
func printVersion(w io.Writer) {
	if commit != "" {
		fmt.Fprintf(w, "coverage %s (%s)\n", version, commit)
		return
	}
	fmt.Fprintf(w, "coverage %s\n", version)
}

func runReport(args []string) {
	fs := flag.NewFlagSet("coverage", flag.ContinueOnError)

	input := fs.String("input", "", "directory containing coverage-*.xml and tests-*.xml artifacts (required)")
	output := fs.String("output", "-", "output path; '-' is stdout. A file is appended to.")
	ignorePath := fs.String("ignore", "", "path to a .coverageignore file (gitignore syntax)")
	configPath := fs.String("config", "", "path to coverage.yaml (default: ./coverage.yaml if present)")
	baselinePath := fs.String("baseline", "", "path to a baseline coverage-summary.json for regression detection")
	failOnDrop := fs.Float64("fail-on-drop", 0, "fail (exit non-zero) if total line coverage drops by more than this many percentage points")
	emitJSON := fs.String("emit-json", "", "also write a machine-readable coverage-summary.json to this path")
	format := fs.String("format", "", "output format: markdown or html (default: auto-detect from --output extension)")
	verbose := fs.Bool("verbose", false, "log warnings for workspaces missing a config entry")

	if err := fs.Parse(args); err != nil {
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
		Format:       *format,
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

func runInit(args []string) {
	fs := flag.NewFlagSet("coverage init", flag.ContinueOnError)
	dir := fs.String("dir", ".", "repository directory to scaffold")
	dryRun := fs.Bool("dry-run", false, "print what would be created without writing files")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	err := scaffold.Run(scaffold.Options{
		Dir:    *dir,
		DryRun: *dryRun,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
	if err != nil {
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
