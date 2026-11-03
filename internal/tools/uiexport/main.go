// Command uiexport writes the scenarios @ui of docs/specs/features in JSON, for Playwright
// (ADR 0028): one object per scenario or example of an outline, with its file, title, tags
// and steps. Written before each run of make e2e, never versioned.
//
// Usage: go run ./internal/tools/uiexport [-features dir] -out file
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/nelt/tribe-menus/internal/scenarios"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stderr))
}

func run(args []string, stderr io.Writer) int {
	flags := flag.NewFlagSet("uiexport", flag.ContinueOnError)
	flags.SetOutput(stderr)
	features := flags.String("features", "docs/specs/features", "directory of the .feature files")
	out := flags.String("out", "", "JSON file to write")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *out == "" || flags.NArg() > 0 {
		fmt.Fprintln(stderr, "usage: uiexport [-features dir] -out file")
		return 2
	}
	if err := export(os.DirFS(*features), *out); err != nil {
		fmt.Fprintf(stderr, "uiexport: %v\n", err)
		return 1
	}
	return 0
}

// manualTag marks the scenarios checked by hand in the staging environment (ADR 0005).
const manualTag = "@manuel"

// export writes the scenarios @ui, except those also @manuel.
func export(features fs.FS, out string) error {
	all, err := scenarios.Read(features)
	if err != nil {
		return err
	}
	ui := []scenarios.Scenario{}
	for _, s := range all {
		if s.HasTag(scenarios.UITag) && !s.HasTag(manualTag) {
			ui = append(ui, s)
		}
	}
	data, err := json.MarshalIndent(ui, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return fmt.Errorf("write %s: %w", out, err)
	}
	if err := os.WriteFile(out, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", out, err)
	}
	return nil
}
