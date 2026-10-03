package acceptance

import (
	"context"
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/cucumber/godog"
)

const (
	featuresDir = "../docs/specs/features"
	pendingFile = "pending.txt"
	// Scenarios proved in the browser, or by hand in the staging environment (ADR 0005).
	excludedTags = "~@manuel && ~@ui"
)

var run = flag.Bool("acceptance", false, "run the Gherkin scenarios (make acceptance)")

func TestAcceptance(t *testing.T) {
	if !*run {
		t.Skip("run with make acceptance")
	}
	pending := readPending(t)

	base := t.TempDir()
	results := map[string]error{}
	suite := godog.TestSuite{
		Name: "tribe-menus",
		ScenarioInitializer: func(sc *godog.ScenarioContext) {
			initializeScenario(sc, base, results)
		},
		Options: &godog.Options{
			FS:     os.DirFS(featuresDir),
			Paths:  []string{"."},
			Tags:   excludedTags,
			Strict: true,
			Format: "progress",
			// Failures are reported below, against pending.txt.
			Output:      io.Discard,
			Concurrency: 1,
		},
	}
	// The status of the run is ignored: what counts is the comparison with pending.txt.
	_ = suite.Run()

	if len(results) == 0 {
		t.Fatal("no scenario was run")
	}
	for _, problem := range compare(results, pending) {
		t.Error(problem)
	}
	t.Logf("%d scenarios run, %d pending", len(results), len(pending))
}

func readPending(t *testing.T) []string {
	t.Helper()
	f, err := os.Open(pendingFile)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	pending, err := parsePending(f)
	if err != nil {
		t.Fatal(err)
	}
	return pending
}

// initializeScenario gives each scenario its own world, and records its result.
// A scenario outline counts as one scenario: it fails if any of its examples fails.
func initializeScenario(sc *godog.ScenarioContext, base string, results map[string]error) {
	w := &world{}
	sc.Before(func(ctx context.Context, s *godog.Scenario) (context.Context, error) {
		dir, err := os.MkdirTemp(base, "scenario-")
		if err != nil {
			return ctx, err
		}
		w.reset(dir)
		return ctx, nil
	})
	sc.After(func(ctx context.Context, s *godog.Scenario, err error) (context.Context, error) {
		if closeErr := w.close(); err == nil {
			err = closeErr
		}
		key := scenarioKey(filepath.Base(s.Uri), s.Name)
		if previous, ran := results[key]; !ran || previous == nil {
			results[key] = err
		}
		return ctx, nil
	})
	w.registerSteps(sc)
}
