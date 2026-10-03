package acceptance

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/cucumber/godog"
	"github.com/nelt/tribe-menus/internal/admin"
	"github.com/nelt/tribe-menus/internal/storage"
	"github.com/nelt/tribe-menus/internal/tribe"
)

const baseURL = "http://localhost:8080"

// now is the date of the scenarios, until the test clock arrives with the login (lot B).
var now = time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC)

// world is the state of one scenario: its own data directory, hence its own databases.
type world struct {
	dir   string
	store *storage.Store

	// admin is the admin command started by the scenario, if any.
	admin *adminSession
	// tribe is the slug of the last tribe the scenario named or created.
	tribe tribe.Slug
	// mails are the messages sent; nothing sends any before the login (lot B).
	mails []sentMail
}

type sentMail struct {
	to tribe.Email
}

// registerSteps registers the step definitions, grouped by feature in the steps_*_test.go files.
func (w *world) registerSteps(sc *godog.ScenarioContext) {
	w.registerAdministrationSteps(sc)
}

// openStore returns the databases of the scenario, opened on first use.
func (w *world) openStore(ctx context.Context) (*storage.Store, error) {
	if w.store != nil {
		return w.store, nil
	}
	migrations, err := storage.EmbeddedMigrations()
	if err != nil {
		return nil, err
	}
	store, err := storage.Open(ctx, filepath.Join(w.dir, "data"), migrations)
	if err != nil {
		return nil, fmt.Errorf("open databases: %w", err)
	}
	w.store = store
	return store, nil
}

// adminCommand returns the admin command on the databases of the scenario.
func (w *world) adminCommand(ctx context.Context) (*admin.Command, error) {
	store, err := w.openStore(ctx)
	if err != nil {
		return nil, err
	}
	return &admin.Command{Store: store, BaseURL: baseURL, Now: func() time.Time { return now }}, nil
}

// tribeStore returns the store of the tribe with this slug.
func (w *world) tribeStore(ctx context.Context, slug string) (*tribe.Store, error) {
	store, err := w.openStore(ctx)
	if err != nil {
		return nil, err
	}
	db, err := store.Tribe(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("tribe %q: %w", slug, err)
	}
	return tribe.NewStore(db), nil
}

func (w *world) close() error {
	var errs []error
	if w.admin != nil {
		errs = append(errs, w.admin.close())
	}
	if w.store != nil {
		errs = append(errs, w.store.Close())
	}
	return errors.Join(errs...)
}
