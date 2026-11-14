package storage

import (
	"context"
	"database/sql"
	"io/fs"
	"path/filepath"
	"regexp"
	"slices"
	"testing"
	"testing/fstest"
)

// TestShoppingListsSchema: migration 0009 (plan courses, step 7).
func TestShoppingListsSchema(t *testing.T) {
	db := newTribeDB(t)

	// The main list "Courses", of an identifier in the format of the others (D14).
	var id, name string
	var rank, main int
	if err := db.QueryRow("SELECT id, name, rank, main FROM shopping_lists").Scan(&id, &name, &rank, &main); err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(id) || name != "Courses" || rank != 1 || main != 1 {
		t.Errorf("main list %q %q %d %d", id, name, rank, main)
	}

	setup := []string{
		"INSERT INTO ingredients (id, name, normalized_name) VALUES (1, 'pain', 'pain')",
		"INSERT INTO dishes VALUES ('d1', 'Chili', 'chili', '#C2502E', 4, 1)",
		"INSERT INTO shopping_lists VALUES ('l2', 'Marché', 2, 0)",
		"INSERT INTO shopping_items VALUES ('i1', 'l2', 1, 2000, 'piece', 0, 1, NULL)",
		"INSERT INTO shopping_items VALUES ('i2', 'l2', 1, NULL, NULL, 1, 0, NULL)",
		"INSERT INTO shopping_item_sources VALUES ('i2', 's1', 'd1', '2026-10-06', 'dinner', 4, NULL, NULL)",
		"INSERT INTO shopping_received VALUES ('l2', 's1', 'd1', 4, 4)",
		"INSERT INTO shopping_received_lines VALUES ('l2', 's1', 0, 1, NULL, NULL)",
		"INSERT INTO shopping_received_lines VALUES ('l2', 's1', 1, 1, 500000, 'g')",
		// A taken quantity (migration 0010; courses-suite, D9).
		"INSERT INTO shopping_items VALUES ('i3', 'l2', 1, 750000, 'g', 0, 0, 250000)",
	}
	for _, s := range setup {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}

	refused := []struct{ name, sql string }{
		{"two main lists (Q23)", "UPDATE shopping_lists SET main = 1 WHERE id = 'l2'"},
		{"main 2", "INSERT INTO shopping_lists VALUES ('l3', 'Drive', 3, 2)"},
		{"rank 0", "INSERT INTO shopping_lists VALUES ('l3', 'Drive', 0, 0)"},
		{"item of an unknown list", "INSERT INTO shopping_items VALUES ('i9', 'l9', 1, NULL, NULL, 0, 1, NULL)"},
		{"item of an unknown ingredient", "INSERT INTO shopping_items VALUES ('i9', 'l2', 9, NULL, NULL, 0, 1, NULL)"},
		{"amount without unit", "INSERT INTO shopping_items VALUES ('i9', 'l2', 1, 2000, NULL, 0, 1, NULL)"},
		{"unit without amount", "INSERT INTO shopping_items VALUES ('i9', 'l2', 1, NULL, 'g', 0, 1, NULL)"},
		{"amount 0", "INSERT INTO shopping_items VALUES ('i9', 'l2', 1, 0, 'g', 0, 1, NULL)"},
		{"unknown unit", "INSERT INTO shopping_items VALUES ('i9', 'l2', 1, 1, 'cup', 0, 1, NULL)"},
		{"checked 2", "INSERT INTO shopping_items VALUES ('i9', 'l2', 1, NULL, NULL, 2, 1, NULL)"},
		{"source of an unknown item", "INSERT INTO shopping_item_sources VALUES ('i9', 's1', 'd1', '2026-10-06', 'dinner', 4, NULL, NULL)"},
		{"same served dish twice in an item", "INSERT INTO shopping_item_sources VALUES ('i2', 's1', 'd1', '2026-10-06', 'dinner', 4, NULL, NULL)"},
		{"source of an unknown dish", "INSERT INTO shopping_item_sources VALUES ('i2', 's2', 'd9', '2026-10-06', 'dinner', 4, NULL, NULL)"},
		{"source on 30 February", "INSERT INTO shopping_item_sources VALUES ('i2', 's2', 'd1', '2026-02-30', 'dinner', 4, NULL, NULL)"},
		{"source at breakfast", "INSERT INTO shopping_item_sources VALUES ('i2', 's2', 'd1', '2026-10-06', 'breakfast', 4, NULL, NULL)"},
		{"source of 51 servings", "INSERT INTO shopping_item_sources VALUES ('i2', 's2', 'd1', '2026-10-06', 'dinner', 51, NULL, NULL)"},
		{"source amount without unit", "INSERT INTO shopping_item_sources VALUES ('i2', 's2', 'd1', '2026-10-06', 'dinner', 4, 10, NULL)"},
		{"received twice", "INSERT INTO shopping_received VALUES ('l2', 's1', 'd1', 4, 4)"},
		{"received by an unknown list", "INSERT INTO shopping_received VALUES ('l9', 's1', 'd1', 4, 4)"},
		{"received dish servings 0", "INSERT INTO shopping_received VALUES ('l2', 's2', 'd1', 4, 0)"},
		{"line of an unknown received served dish", "INSERT INTO shopping_received_lines VALUES ('l2', 's2', 0, 1, NULL, NULL)"},
		{"same position twice", "INSERT INTO shopping_received_lines VALUES ('l2', 's1', 0, 1, NULL, NULL)"},
		{"taken quantity of a checked item", "INSERT INTO shopping_items VALUES ('i9', 'l2', 1, 750000, 'g', 1, 0, 250000)"},
		{"taken quantity of a manual item", "INSERT INTO shopping_items VALUES ('i9', 'l2', 1, 750000, 'g', 0, 1, 250000)"},
		{"taken quantity without quantity", "INSERT INTO shopping_items VALUES ('i9', 'l2', 1, NULL, NULL, 0, 0, 250000)"},
		{"taken quantity equal to the quantity", "INSERT INTO shopping_items VALUES ('i9', 'l2', 1, 750000, 'g', 0, 0, 750000)"},
		{"taken quantity 0", "INSERT INTO shopping_items VALUES ('i9', 'l2', 1, 750000, 'g', 0, 0, 0)"},
		{"checked with its taken quantity", "UPDATE shopping_items SET checked = 1 WHERE id = 'i3'"},
		{"list with items deleted", "DELETE FROM shopping_lists WHERE id = 'l2'"},
		{"item with sources deleted", "DELETE FROM shopping_items WHERE id = 'i2'"},
	}
	for _, c := range refused {
		t.Run(c.name, func(t *testing.T) {
			if _, err := db.Exec(c.sql); err == nil {
				t.Errorf("%s accepted", c.sql)
			}
		})
	}
}

// TestShoppingListsTriggers: every write to the tables of the shopping lists increases their
// version, and only theirs (ADR 0029, point 2).
func TestShoppingListsTriggers(t *testing.T) {
	db := newTribeDB(t)
	cases := []struct {
		name     string
		sql      string
		shopping int64
		others   int64 // dishes, ingredients and meal plan
	}{
		{"insert an ingredient", "INSERT INTO ingredients (id, name, normalized_name) VALUES (1, 'pain', 'pain')", 0, 1},
		{"insert a dish", "INSERT INTO dishes VALUES ('d1', 'Chili', 'chili', '#C2502E', 4, 1)", 0, 1},
		{"insert a list", "INSERT INTO shopping_lists VALUES ('l2', 'Marché', 2, 0)", 1, 0},
		{"rename a list", "UPDATE shopping_lists SET name = 'Drive' WHERE id = 'l2'", 1, 0},
		{"insert an item", "INSERT INTO shopping_items VALUES ('i1', 'l2', 1, NULL, NULL, 0, 0, NULL)", 1, 0},
		{"check an item", "UPDATE shopping_items SET checked = 1", 1, 0},
		{"insert a source", "INSERT INTO shopping_item_sources VALUES ('i1', 's1', 'd1', '2026-10-06', 'dinner', 4, NULL, NULL)", 1, 0},
		{"update a source", "UPDATE shopping_item_sources SET servings = 3", 1, 0},
		{"insert a received served dish", "INSERT INTO shopping_received VALUES ('l2', 's1', 'd1', 4, 4)", 1, 0},
		{"update a received served dish", "UPDATE shopping_received SET servings = 3", 1, 0},
		{"insert a received line", "INSERT INTO shopping_received_lines VALUES ('l2', 's1', 0, 1, NULL, NULL)", 1, 0},
		{"update a received line", "UPDATE shopping_received_lines SET amount = 5, unit = 'g'", 1, 0},
		{"delete a received line", "DELETE FROM shopping_received_lines", 1, 0},
		{"delete a received served dish", "DELETE FROM shopping_received", 1, 0},
		{"delete a source", "DELETE FROM shopping_item_sources", 1, 0},
		{"delete an item", "DELETE FROM shopping_items", 1, 0},
		{"delete a list", "DELETE FROM shopping_lists WHERE id = 'l2'", 1, 0},
	}
	others := func() int64 {
		return versionOf(t, db, "dishes") + versionOf(t, db, "ingredients") + versionOf(t, db, "meal_plan")
	}
	for _, c := range cases {
		shopping, other := versionOf(t, db, "shopping_lists"), others()
		if _, err := db.Exec(c.sql); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got := versionOf(t, db, "shopping_lists") - shopping; got != c.shopping {
			t.Errorf("%s: shopping_lists +%d, want +%d", c.name, got, c.shopping)
		}
		if got := others() - other; got != c.others {
			t.Errorf("%s: other collections +%d, want +%d", c.name, got, c.others)
		}
	}
}

// migratedBefore returns a tribe database migrated up to the migration named, excluded.
func migratedBefore(t *testing.T, migration string) *sql.DB {
	t.Helper()
	all, err := EmbeddedMigrations()
	if err != nil {
		t.Fatal(err)
	}
	before := fstest.MapFS{}
	err = fs.WalkDir(all.Tribe, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || path >= migration {
			return err
		}
		data, err := fs.ReadFile(all.Tribe, path)
		before[path] = &fstest.MapFile{Data: data}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	db, err := openDB(context.Background(), filepath.Join(t.TempDir(), "tribe.db"), true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := migrate(context.Background(), db, before); err != nil {
		t.Fatal(err)
	}
	return db
}

// migrateAll applies every migration of a tribe database.
func migrateAll(t *testing.T, db *sql.DB) {
	t.Helper()
	all, err := EmbeddedMigrations()
	if err != nil {
		t.Fatal(err)
	}
	if err := migrate(context.Background(), db, all.Tribe); err != nil {
		t.Fatal(err)
	}
}

// TestShoppingListsMigrationOfAnExistingTribe: a tribe created before the migration 0009
// receives its main list "Courses" (D14).
func TestShoppingListsMigrationOfAnExistingTribe(t *testing.T) {
	db := migratedBefore(t, "0009")
	if _, err := db.Exec("INSERT INTO dishes VALUES ('d1', 'Chili', 'chili', '#C2502E', 4, 1)"); err != nil {
		t.Fatal(err)
	}
	migrateAll(t, db)
	var lists []string
	rows, err := db.Query("SELECT name || ' ' || main FROM shopping_lists")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var l string
		if err := rows.Scan(&l); err != nil {
			t.Fatal(err)
		}
		lists = append(lists, l)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(lists, []string{"Courses 1"}) {
		t.Errorf("lists %v", lists)
	}
}

// TestTakenAmountMigration: the items of a tribe of the migration 0009 stay intact, without
// taken quantity (courses-suite, step 8).
func TestTakenAmountMigration(t *testing.T) {
	db := migratedBefore(t, "0010")
	for _, s := range []string{
		"INSERT INTO ingredients (id, name, normalized_name) VALUES (1, 'bœuf haché', 'boeuf hache')",
		"INSERT INTO shopping_items SELECT 'i1', id, 1, 500000, 'g', 1, 0 FROM shopping_lists",
		"INSERT INTO shopping_items SELECT 'i2', id, 1, NULL, NULL, 0, 1 FROM shopping_lists",
	} {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
	migrateAll(t, db)
	var items []string
	rows, err := db.Query("SELECT id || ' ' || coalesce(amount, '-') || ' ' || coalesce(taken_amount, '-') || ' ' || checked || ' ' || manual FROM shopping_items ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var it string
		if err := rows.Scan(&it); err != nil {
			t.Fatal(err)
		}
		items = append(items, it)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	if want := []string{"i1 500000 - 1 0", "i2 - - 0 1"}; !slices.Equal(items, want) {
		t.Errorf("items %v, want %v", items, want)
	}
}
