package storage

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func sqlFile(s string) *fstest.MapFile { return &fstest.MapFile{Data: []byte(s)} }

func userVersion(t *testing.T, db *sql.DB) int {
	t.Helper()
	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func tableExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var n int
	if err := db.QueryRow("SELECT count(*) FROM sqlite_schema WHERE type = 'table' AND name = ?", name).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n > 0
}

func TestMigrate(t *testing.T) {
	v1 := fstest.MapFS{"0001_a.sql": sqlFile("CREATE TABLE a (x INTEGER) STRICT;")}
	v2 := fstest.MapFS{
		"0001_a.sql": sqlFile("CREATE TABLE a (x INTEGER) STRICT;"),
		"0002_b.sql": sqlFile("CREATE TABLE b (y TEXT) STRICT; INSERT INTO a VALUES (1);"),
	}
	cases := []struct {
		name        string
		first       fstest.MapFS // applied first, must succeed
		then        fstest.MapFS
		wantErr     string
		wantVersion int
		wantTables  []string
		wantMissing []string
	}{
		{name: "empty database", then: v2, wantVersion: 2, wantTables: []string{"a", "b"}},
		{name: "replayed without effect", first: v2, then: v2, wantVersion: 2, wantTables: []string{"a", "b"}},
		{name: "newer migration applied", first: v1, then: v2, wantVersion: 2, wantTables: []string{"a", "b"}},
		{name: "no migration", then: fstest.MapFS{}, wantVersion: 0},
		{
			name:  "failed migration leaves the version unchanged",
			first: v1,
			then: fstest.MapFS{
				"0001_a.sql": sqlFile("CREATE TABLE a (x INTEGER) STRICT;"),
				"0002_b.sql": sqlFile("CREATE TABLE b (y TEXT) STRICT; INSERT INTO missing VALUES (1);"),
			},
			wantErr: "0002_b.sql", wantVersion: 1, wantTables: []string{"a"}, wantMissing: []string{"b"},
		},
		{name: "database newer than the binary", first: v2, then: v1, wantErr: "newer than this binary", wantVersion: 2},
		{name: "gap in versions", then: fstest.MapFS{"0002_b.sql": sqlFile("SELECT 1;")}, wantErr: "version 1 expected", wantVersion: 0},
		{name: "unexpected file", then: fstest.MapFS{"0001-a.sql": sqlFile("SELECT 1;")}, wantErr: "unexpected migration file", wantVersion: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db, err := openDB(ctx, filepath.Join(t.TempDir(), "test.db"), true)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if tc.first != nil {
				if err := migrate(ctx, db, tc.first); err != nil {
					t.Fatalf("first migration: %v", err)
				}
			}

			err = migrate(ctx, db, tc.then)
			if tc.wantErr == "" && err != nil {
				t.Fatalf("migrate: %v", err)
			}
			if tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
				t.Fatalf("migrate error = %v, want it to contain %q", err, tc.wantErr)
			}
			if got := userVersion(t, db); got != tc.wantVersion {
				t.Errorf("user_version = %d, want %d", got, tc.wantVersion)
			}
			for _, name := range tc.wantTables {
				if !tableExists(t, db, name) {
					t.Errorf("table %s missing", name)
				}
			}
			for _, name := range tc.wantMissing {
				if tableExists(t, db, name) {
					t.Errorf("table %s exists, want it rolled back", name)
				}
			}
		})
	}
}

func TestConnectionSettings(t *testing.T) {
	ctx := context.Background()
	db, err := openDB(ctx, filepath.Join(t.TempDir(), "test.db"), true)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for pragma, want := range map[string]string{
		"journal_mode": "wal",
		"foreign_keys": "1",
		"busy_timeout": "5000",
	} {
		var got string
		if err := db.QueryRow("PRAGMA " + pragma).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("%s = %s, want %s", pragma, got, want)
		}
	}
}

func TestOpenMissingFileWithoutCreate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.db")
	if _, err := openDB(context.Background(), path, false); err == nil {
		t.Fatal("openDB without create: want an error for a missing file")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("file created: %v", err)
	}
}

func testMigrations(t *testing.T) Migrations {
	t.Helper()
	m, err := EmbeddedMigrations()
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func openStore(t *testing.T, dir string) *Store {
	t.Helper()
	s, err := Open(context.Background(), dir, testMigrations(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func setName(name string) func(context.Context, *sql.DB) error {
	return func(ctx context.Context, db *sql.DB) error {
		_, err := db.ExecContext(ctx, "INSERT INTO tribe (id, name) VALUES (1, ?)", name)
		return err
	}
}

func tribeName(t *testing.T, s *Store, slug string) string {
	t.Helper()
	db, err := s.Tribe(context.Background(), slug)
	if err != nil {
		t.Fatalf("Tribe(%q): %v", slug, err)
	}
	var name string
	if err := db.QueryRow("SELECT name FROM tribe").Scan(&name); err != nil {
		t.Fatal(err)
	}
	return name
}

func tribeFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(dir, tribesDir))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestStore(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s := openStore(t, dir)

	if err := s.CreateTribe(ctx, "martin", "Les Martin", setName("Les Martin")); err != nil {
		t.Fatalf("CreateTribe martin: %v", err)
	}
	if err := s.CreateTribe(ctx, "durand", "Les Durand", setName("Les Durand")); err != nil {
		t.Fatalf("CreateTribe durand: %v", err)
	}

	t.Run("each tribe has its own database", func(t *testing.T) {
		if got := tribeName(t, s, "martin"); got != "Les Martin" {
			t.Errorf("martin: name = %q", got)
		}
		if got := tribeName(t, s, "durand"); got != "Les Durand" {
			t.Errorf("durand: name = %q", got)
		}
	})

	t.Run("file names are not derived from the slug", func(t *testing.T) {
		for _, name := range tribeFiles(t, dir) {
			if strings.Contains(name, "martin") || strings.Contains(name, "durand") {
				t.Errorf("file %s reveals a slug", name)
			}
		}
	})

	t.Run("unknown slug", func(t *testing.T) {
		if _, err := s.Tribe(ctx, "inconnue"); !errors.Is(err, ErrUnknownTribe) {
			t.Errorf("Tribe(inconnue) error = %v, want ErrUnknownTribe", err)
		}
	})

	t.Run("slug taken", func(t *testing.T) {
		before := tribeFiles(t, dir)
		err := s.CreateTribe(ctx, "martin", "Autres Martin", setName("Autres Martin"))
		if !errors.Is(err, ErrSlugTaken) {
			t.Errorf("CreateTribe error = %v, want ErrSlugTaken", err)
		}
		if got := tribeFiles(t, dir); len(got) != len(before) {
			t.Errorf("files = %v, want %v", got, before)
		}
	})

	t.Run("failed setup leaves no trace", func(t *testing.T) {
		before := tribeFiles(t, dir)
		failure := errors.New("setup failed")
		err := s.CreateTribe(ctx, "dupont", "Les Dupont", func(context.Context, *sql.DB) error { return failure })
		if !errors.Is(err, failure) {
			t.Errorf("CreateTribe error = %v, want %v", err, failure)
		}
		if _, err := s.Tribe(ctx, "dupont"); !errors.Is(err, ErrUnknownTribe) {
			t.Errorf("Tribe(dupont) error = %v, want ErrUnknownTribe", err)
		}
		if got := tribeFiles(t, dir); len(got) != len(before) {
			t.Errorf("files = %v, want %v", got, before)
		}
	})

	t.Run("tribe created by another process", func(t *testing.T) {
		other := openStore(t, dir)
		if err := other.CreateTribe(ctx, "leroy", "Les Leroy", setName("Les Leroy")); err != nil {
			t.Fatalf("CreateTribe leroy: %v", err)
		}
		if got := tribeName(t, s, "leroy"); got != "Les Leroy" {
			t.Errorf("leroy: name = %q", got)
		}
	})

	t.Run("reopened store", func(t *testing.T) {
		reopened := openStore(t, dir)
		if got := tribeName(t, reopened, "martin"); got != "Les Martin" {
			t.Errorf("martin: name = %q", got)
		}
	})
}

func TestOpenFailsOnBrokenTribe(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s := openStore(t, dir)
	if err := s.CreateTribe(ctx, "martin", "Les Martin", setName("Les Martin")); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	broken := testMigrations(t)
	broken.Tribe = fstest.MapFS{
		"0001_tribe.sql": sqlFile("SELECT 1;"),
		"0002_bad.sql":   sqlFile("INSERT INTO missing VALUES (1);"),
	}
	_, err := Open(ctx, dir, broken)
	if err == nil || !strings.Contains(err.Error(), "0002_bad.sql") {
		t.Fatalf("Open error = %v, want the failed migration", err)
	}
}
