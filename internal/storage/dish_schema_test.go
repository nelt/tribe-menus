package storage

import (
	"context"
	"database/sql"
	"testing"
)

// newTribeDB returns the migrated database of a new tribe named "Les Martin".
func newTribeDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()
	s := openStore(t, t.TempDir())
	if err := s.CreateTribe(ctx, "martin", "Les Martin", setName("Les Martin")); err != nil {
		t.Fatalf("CreateTribe: %v", err)
	}
	db, err := s.Tribe(ctx, "martin")
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func versionOf(t *testing.T, db *sql.DB, name string) int64 {
	t.Helper()
	var v int64
	if err := db.QueryRow("SELECT version FROM collection_versions WHERE name = ?", name).Scan(&v); err != nil {
		t.Fatalf("version of %s: %v", name, err)
	}
	return v
}

// TestDishLibrarySchema: migration 0007 (plan bibliotheque, step 9).
func TestDishLibrarySchema(t *testing.T) {
	db := newTribeDB(t)

	var size int
	if err := db.QueryRow("SELECT size FROM tribe").Scan(&size); err != nil {
		t.Fatal(err)
	}
	if size != 4 {
		t.Errorf("tribe size %d, want 4 (T3)", size)
	}

	setup := []string{
		"INSERT INTO ingredients (id, name, normalized_name, default_unit) VALUES (1, 'Tomate', 'tomate', 'piece')",
		"INSERT INTO ingredients (id, name, normalized_name) VALUES (2, 'sel', 'sel')",
		"INSERT INTO dishes VALUES ('d1', 'Chili', 'chili', '#C2502E', 4, 1)",
		"INSERT INTO dish_ingredients VALUES ('d1', 0, 1, 3000, 'piece')",
		"INSERT INTO dish_ingredients VALUES ('d1', 1, 2, NULL, NULL)",
	}
	for _, s := range setup {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}

	refused := []struct{ name, sql string }{
		{"size 0", "UPDATE tribe SET size = 0"},
		{"size 51", "UPDATE tribe SET size = 51"},
		{"same normalized ingredient name", "INSERT INTO ingredients (name, normalized_name) VALUES ('TOMATE', 'tomate')"},
		{"unknown default unit", "INSERT INTO ingredients (name, normalized_name, default_unit) VALUES ('riz', 'riz', 'cup')"},
		{"same normalized dish name", "INSERT INTO dishes VALUES ('d2', 'CHILI', 'chili', '#C2502E', 4, 1)"},
		{"servings 0", "INSERT INTO dishes VALUES ('d2', 'Gratin', 'gratin', '#C2502E', 0, 1)"},
		{"revision 0", "INSERT INTO dishes VALUES ('d2', 'Gratin', 'gratin', '#C2502E', 4, 0)"},
		{"ingredient twice in a dish", "INSERT INTO dish_ingredients VALUES ('d1', 2, 1, NULL, NULL)"},
		{"amount without unit", "INSERT INTO dish_ingredients VALUES ('d1', 2, 2, 1000, NULL)"},
		{"unit without amount", "INSERT INTO dish_ingredients VALUES ('d1', 2, 2, NULL, 'g')"},
		{"amount 0", "INSERT INTO dish_ingredients VALUES ('d1', 2, 2, 0, 'g')"},
		{"unknown dish", "INSERT INTO dish_ingredients VALUES ('d9', 0, 1, NULL, NULL)"},
	}
	for _, c := range refused {
		t.Run(c.name, func(t *testing.T) {
			if _, err := db.Exec(c.sql); err == nil {
				t.Errorf("%s accepted", c.sql)
			}
		})
	}
}

// TestCollectionTriggers: every write to a table of a collection increases its version, and
// only its own (ADR 0029, point 2).
func TestCollectionTriggers(t *testing.T) {
	db := newTribeDB(t)
	cases := []struct {
		name            string
		sql             string
		dishes, ingreds int64 // increase of each version
	}{
		{"insert an ingredient", "INSERT INTO ingredients (id, name, normalized_name) VALUES (1, 'sel', 'sel')", 0, 1},
		{"update an ingredient", "UPDATE ingredients SET default_unit = 'g' WHERE id = 1", 0, 1},
		{"insert a dish", "INSERT INTO dishes VALUES ('d1', 'Chili', 'chili', '#C2502E', 4, 1)", 1, 0},
		{"update a dish", "UPDATE dishes SET servings = 6 WHERE id = 'd1'", 1, 0},
		{"insert a line", "INSERT INTO dish_ingredients VALUES ('d1', 0, 1, NULL, NULL)", 1, 0},
		{"update a line", "UPDATE dish_ingredients SET amount = 5, unit = 'g'", 1, 0},
		{"delete a line", "DELETE FROM dish_ingredients", 1, 0},
		{"delete a dish", "DELETE FROM dishes", 1, 0},
		{"delete an ingredient", "DELETE FROM ingredients", 0, 1},
		{"tribe size", "UPDATE tribe SET size = 5", 0, 0},
	}
	for _, c := range cases {
		dishes, ingreds := versionOf(t, db, "dishes"), versionOf(t, db, "ingredients")
		if _, err := db.Exec(c.sql); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got := versionOf(t, db, "dishes") - dishes; got != c.dishes {
			t.Errorf("%s: dishes +%d, want +%d", c.name, got, c.dishes)
		}
		if got := versionOf(t, db, "ingredients") - ingreds; got != c.ingreds {
			t.Errorf("%s: ingredients +%d, want +%d", c.name, got, c.ingreds)
		}
	}
}
