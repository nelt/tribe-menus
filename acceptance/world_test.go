package acceptance

import (
	"github.com/cucumber/godog"
	"github.com/nelt/tribe-menus/internal/storage"
)

// world is the state of one scenario: its own data directory, hence its own databases.
type world struct {
	dir   string
	store *storage.Store
}

// registerSteps registers the step definitions, grouped by feature in the steps_*_test.go files.
func (w *world) registerSteps(sc *godog.ScenarioContext) {}

func (w *world) close() error {
	if w.store == nil {
		return nil
	}
	err := w.store.Close()
	w.store = nil
	return err
}
