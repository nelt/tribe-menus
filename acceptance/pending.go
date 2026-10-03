// Package acceptance runs the Gherkin scenarios of docs/specs/features against the
// application (ADR 0005), with godog.
//
// Scenarios not implemented yet are listed in pending.txt, one per line as
// "<file>: <scenario title>". The run fails when a scenario outside the list fails, and
// when a scenario of the list passes: the list can only shrink.
package acceptance

import (
	"bufio"
	"fmt"
	"io"
	"slices"
	"strings"
)

// scenarioKey identifies a scenario in pending.txt and in the reports.
func scenarioKey(file, title string) string {
	return file + ": " + title
}

// parsePending reads the keys of pending.txt, ignoring blank lines and # comments.
func parsePending(r io.Reader) ([]string, error) {
	var keys []string
	seen := map[string]bool{}
	scanner := bufio.NewScanner(r)
	for n := 1; scanner.Scan(); n++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.Contains(line, ".feature: ") {
			return nil, fmt.Errorf("pending.txt:%d: %q is not \"<file>.feature: <scenario title>\"", n, line)
		}
		if seen[line] {
			return nil, fmt.Errorf("pending.txt:%d: %q listed twice", n, line)
		}
		seen[line] = true
		keys = append(keys, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read pending.txt: %w", err)
	}
	return keys, nil
}

// compare returns the discrepancies between the results of the run, by scenario key
// (nil for a scenario that passed), and the pending list, sorted.
func compare(results map[string]error, pending []string) []string {
	var problems []string
	isPending := map[string]bool{}
	for _, key := range pending {
		isPending[key] = true
		err, ran := results[key]
		switch {
		case !ran:
			problems = append(problems, key+": listed in pending.txt but no such scenario was run")
		case err == nil:
			problems = append(problems, key+": passes, remove it from pending.txt")
		}
	}
	for key, err := range results {
		if err != nil && !isPending[key] {
			problems = append(problems, fmt.Sprintf("%s: fails: %v", key, err))
		}
	}
	slices.Sort(problems)
	return problems
}
