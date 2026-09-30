// Command webcheck reports escape hatches from Lit's escaped rendering in the
// front-end sources (ADR 0004, point 9).
//
// Usage: go run ./internal/tools/webcheck [dir]
//
// dir defaults to web/src. The exit code is 1 when something is found.
package main

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
)

var (
	checkedExtensions = map[string]bool{".ts": true, ".js": true, ".mjs": true, ".html": true}
	forbidden         = regexp.MustCompile(`\b(unsafeHTML|unsafeSVG|innerHTML|outerHTML|insertAdjacentHTML|document\.writeln|document\.write)\b`)
)

// finding is a forbidden construct at a given line of a file.
type finding struct {
	file  string
	line  int
	match string
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	dir := filepath.Join("web", "src")
	if len(args) > 0 {
		dir = args[0]
	}
	findings, err := check(os.DirFS(dir))
	if err != nil {
		fmt.Fprintf(stderr, "webcheck: %v\n", err)
		return 2
	}
	for _, f := range findings {
		fmt.Fprintf(stdout, "%s:%d: %s is forbidden (ADR 0004)\n", filepath.Join(dir, filepath.FromSlash(f.file)), f.line, f.match)
	}
	if len(findings) > 0 {
		return 1
	}
	return 0
}

// check walks fsys and returns the forbidden constructs found in the checked files.
func check(fsys fs.FS) ([]finding, error) {
	var findings []finding
	err := fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !checkedExtensions[path.Ext(name)] {
			return nil
		}
		found, err := checkFile(fsys, name)
		if err != nil {
			return err
		}
		findings = append(findings, found...)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk: %w", err)
	}
	return findings, nil
}

func checkFile(fsys fs.FS, name string) ([]finding, error) {
	file, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var findings []finding
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for line := 1; scanner.Scan(); line++ {
		for _, match := range forbidden.FindAllString(scanner.Text(), -1) {
			findings = append(findings, finding{file: name, line: line, match: match})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	return findings, nil
}
