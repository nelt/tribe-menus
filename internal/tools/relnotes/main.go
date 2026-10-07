// Command relnotes prints the section of a version in CHANGELOG.md: the notes of its release
// (ADR 0012, point 5).
//
// Usage: go run ./internal/tools/relnotes [-changelog CHANGELOG.md] vX.Y.Z
//
// The section of version X.Y.Z starts with the heading "## X.Y.Z — date" and ends at the next
// heading of the same level. The exit code is 1 when the section is missing, empty or written
// twice.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

var (
	// tagPattern is the tag of a release; "Non publié" is not a version.
	tagPattern = regexp.MustCompile(`^v([0-9]+\.[0-9]+\.[0-9]+)$`)

	errMissing = errors.New("no section for this version")
	errEmpty   = errors.New("empty section")
	errTwice   = errors.New("section written twice")
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("relnotes", flag.ContinueOnError)
	flags.SetOutput(stderr)
	changelog := flags.String("changelog", "CHANGELOG.md", "journal of the changes")
	if err := flags.Parse(args); err != nil || flags.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: relnotes [-changelog CHANGELOG.md] vX.Y.Z")
		return 2
	}
	data, err := os.ReadFile(*changelog)
	if err != nil {
		fmt.Fprintf(stderr, "relnotes: %v\n", err)
		return 2
	}
	notes, err := section(string(data), flags.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "relnotes: %s: %v\n", flags.Arg(0), err)
		return 1
	}
	fmt.Fprint(stdout, notes)
	return 0
}

// section returns the lines of the section of the version tagged tag, without its heading
// and the blank lines around it, ending with a newline.
func section(changelog, tag string) (string, error) {
	match := tagPattern.FindStringSubmatch(tag)
	if match == nil {
		return "", errors.New("not a release tag vX.Y.Z")
	}
	version := match[1]
	var (
		found bool
		in    bool
		lines []string
	)
	for line := range strings.Lines(changelog) {
		line = strings.TrimRight(line, "\r\n")
		if strings.HasPrefix(line, "## ") {
			in = line == "## "+version || strings.HasPrefix(line, "## "+version+" ")
			if in && found {
				return "", errTwice
			}
			found = found || in
			continue
		}
		if in {
			lines = append(lines, line)
		}
	}
	if !found {
		return "", errMissing
	}
	notes := strings.TrimSpace(strings.Join(lines, "\n"))
	if notes == "" {
		return "", errEmpty
	}
	return notes + "\n", nil
}
