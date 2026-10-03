package storage

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"regexp"
	"slices"
	"strconv"
)

// migrationName is the form of a migration file: a version number, then a description.
// Versions start at 1 and follow each other without gaps.
var migrationName = regexp.MustCompile(`^(\d{4})_[a-z0-9_]+\.sql$`)

type migration struct {
	version int
	name    string
	sql     string
}

// readMigrations reads the migration files at the root of migrations, in version order.
func readMigrations(migrations fs.FS) ([]migration, error) {
	entries, err := fs.ReadDir(migrations, ".")
	if err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}
	var list []migration
	for _, entry := range entries {
		m := migrationName.FindStringSubmatch(entry.Name())
		if entry.IsDir() || m == nil {
			return nil, fmt.Errorf("unexpected migration file %q", entry.Name())
		}
		version, _ := strconv.Atoi(m[1])
		source, err := fs.ReadFile(migrations, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		list = append(list, migration{version: version, name: entry.Name(), sql: string(source)})
	}
	slices.SortFunc(list, func(a, b migration) int { return a.version - b.version })
	for i, m := range list {
		if m.version != i+1 {
			return nil, fmt.Errorf("migration %s: version %d expected", m.name, i+1)
		}
	}
	return list, nil
}

// migrate applies the migrations newer than the database version, each in its own transaction
// with the version update, so that a failed migration leaves the version unchanged.
func migrate(ctx context.Context, db *sql.DB, migrations fs.FS) error {
	list, err := readMigrations(migrations)
	if err != nil {
		return err
	}
	var current int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&current); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	if current > len(list) {
		return fmt.Errorf("schema version %d is newer than this binary (%d)", current, len(list))
	}
	for _, m := range list[current:] {
		if err := apply(ctx, db, m); err != nil {
			return err
		}
	}
	return nil
}

func apply(ctx context.Context, db *sql.DB, m migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("migration %s: begin: %w", m.name, err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, m.sql); err != nil {
		return fmt.Errorf("migration %s: %w", m.name, err)
	}
	// PRAGMA does not take parameters; the version is an integer read from the file name.
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", m.version)); err != nil {
		return fmt.Errorf("migration %s: set version: %w", m.name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migration %s: commit: %w", m.name, err)
	}
	return nil
}
