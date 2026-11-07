package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/nelt/tribe-menus/internal/collection"
	"github.com/nelt/tribe-menus/internal/storage/tribedb"
)

// Collections are the collections of the API (ADR 0029), by name, with the tables of the
// tribe database that make them: a write to any of them increases the version of the
// collection, by the triggers of the migration that creates the table. A test checks that
// each of them has its three triggers.
var Collections = map[string][]string{
	"dishes":         {"dishes", "dish_ingredients"},
	"ingredients":    {"ingredients"},
	"meal_plan":      {"served_dishes"},
	"shopping_lists": {"shopping_lists", "shopping_items", "shopping_item_sources", "shopping_received", "shopping_received_lines"},
}

// readCollection runs read in a transaction of the tribe database and returns the version
// of the collection read in the same transaction (ADR 0029, point 3).
func readCollection(ctx context.Context, db *sql.DB, name string, read func(*tribedb.Queries) error) (int64, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", name, err)
	}
	defer func() { _ = tx.Rollback() }()
	q := tribedb.New(tx)
	version, err := q.CollectionVersion(ctx, name)
	if err != nil {
		return 0, fmt.Errorf("read %s: version: %w", name, err)
	}
	if err := read(q); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("read %s: %w", name, err)
	}
	return version, nil
}

// collectionVersion reads the current version of a collection.
func collectionVersion(ctx context.Context, db *sql.DB, name string) (int64, error) {
	version, err := tribedb.New(db).CollectionVersion(ctx, name)
	if err != nil {
		return 0, fmt.Errorf("version of %s: %w", name, err)
	}
	return version, nil
}

// writeCollections runs write in a transaction of the tribe database and returns the
// versions of the collections named, after the write.
//
// The triggers increase a version on every row written, so that a write of several rows
// would increase it by as many. Before the commit, each collection changed is brought back
// to its version before the write plus one: a write increases the version of a collection
// by one exactly, and the browser that made it knows whether another write came in between
// (ADR 0029, point 4). A version is never lowered below a committed one: the transaction is
// immediate, nobody else writes in between. A write that does not go through here only
// increases the version by more: the copies reload, which stays correct.
func writeCollections(ctx context.Context, db *sql.DB, names []string, write func(*tribedb.Queries) error) (collection.Versions, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("write: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := tribedb.New(tx)
	before := collection.Versions{}
	for _, name := range names {
		v, err := q.CollectionVersion(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("write: version of %s: %w", name, err)
		}
		before[name] = v
	}
	if err := write(q); err != nil {
		return nil, err
	}
	after := collection.Versions{}
	for _, name := range names {
		v, err := q.CollectionVersion(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("write: version of %s: %w", name, err)
		}
		if v > before[name] {
			v = before[name] + 1
			if err := q.SetCollectionVersion(ctx, tribedb.SetCollectionVersionParams{Version: v, Name: name}); err != nil {
				return nil, fmt.Errorf("write: version of %s: %w", name, err)
			}
		}
		after[name] = v
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("write: %w", err)
	}
	return after, nil
}
