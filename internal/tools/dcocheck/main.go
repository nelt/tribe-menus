// Command dcocheck reports commits without a Developer Certificate of Origin
// sign-off from their author (ADR 0011, point 7).
//
// Usage: go run ./internal/tools/dcocheck <base>..<head>
//
// Merge commits and Dependabot commits are skipped. The exit code is 1 when
// a commit lacks a sign-off.
package main

import (
	"fmt"
	"io"
	"net/mail"
	"os"
	"os/exec"
	"strings"
)

// dependabotEmail is the author address of the commits made by Dependabot,
// which has nothing to certify when it bumps a version.
const dependabotEmail = "49699333+dependabot[bot]@users.noreply.github.com"

const (
	fieldSeparator  = "\x00"
	recordSeparator = "\x1e"
	// logFormat prints, for each commit: hash, author name, author email,
	// parent hashes and raw message.
	logFormat = "%H%x00%an%x00%ae%x00%P%x00%B%x1e"
)

// commit is what the check needs to know about a commit.
type commit struct {
	hash        string
	authorName  string
	authorEmail string
	parents     int
	message     string
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 || !strings.Contains(args[0], "..") {
		fmt.Fprintln(stderr, "usage: dcocheck <base>..<head>")
		return 2
	}
	out, err := exec.Command("git", "log", "--format="+logFormat, args[0]).Output()
	if err != nil {
		fmt.Fprintf(stderr, "dcocheck: git log %s: %v\n", args[0], err)
		return 2
	}
	commits, err := parseLog(string(out))
	if err != nil {
		fmt.Fprintf(stderr, "dcocheck: %v\n", err)
		return 2
	}
	unsigned := check(commits)
	for _, c := range unsigned {
		fmt.Fprintf(stdout, "%s %s: missing \"Signed-off-by: %s <%s>\" (DCO, see CONTRIBUTING.md)\n",
			shortHash(c.hash), subject(c.message), c.authorName, c.authorEmail)
	}
	if len(unsigned) > 0 {
		return 1
	}
	return 0
}

// parseLog reads the output of git log with logFormat.
func parseLog(out string) ([]commit, error) {
	var commits []commit
	for _, record := range strings.Split(out, recordSeparator) {
		record = strings.TrimLeft(record, "\n")
		if record == "" {
			continue
		}
		fields := strings.SplitN(record, fieldSeparator, 5)
		if len(fields) != 5 {
			return nil, fmt.Errorf("unexpected git log record %q", record)
		}
		commits = append(commits, commit{
			hash:        fields[0],
			authorName:  fields[1],
			authorEmail: fields[2],
			parents:     len(strings.Fields(fields[3])),
			message:     fields[4],
		})
	}
	return commits, nil
}

// check returns the commits that need a sign-off from their author and lack it.
func check(commits []commit) []commit {
	var unsigned []commit
	for _, c := range commits {
		if c.parents > 1 || strings.EqualFold(c.authorEmail, dependabotEmail) {
			continue
		}
		if !signedOffByAuthor(c) {
			unsigned = append(unsigned, c)
		}
	}
	return unsigned
}

// signedOffByAuthor reports whether the message has a Signed-off-by line
// with the author's name and email.
func signedOffByAuthor(c commit) bool {
	for _, line := range strings.Split(c.message, "\n") {
		value, ok := strings.CutPrefix(strings.TrimSpace(line), "Signed-off-by:")
		if !ok {
			continue
		}
		addr, err := mail.ParseAddress(strings.TrimSpace(value))
		if err != nil {
			continue
		}
		if addr.Name == c.authorName && strings.EqualFold(addr.Address, c.authorEmail) {
			return true
		}
	}
	return false
}

func shortHash(hash string) string {
	if len(hash) > 7 {
		return hash[:7]
	}
	return hash
}

func subject(message string) string {
	first, _, _ := strings.Cut(message, "\n")
	return first
}
