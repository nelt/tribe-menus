package storage

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/nelt/tribe-menus/internal/collection"
	"github.com/nelt/tribe-menus/internal/dish"
)

// TestCollectionsHaveTriggers: each table declared in a collection has its three triggers,
// which increase the version of this collection (ADR 0029, point 2), and each collection has
// its version.
func TestCollectionsHaveTriggers(t *testing.T) {
	db := newTribeDB(t)

	rows, err := db.Query("SELECT name FROM collection_versions ORDER BY name")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	if want := slices.Sorted(maps.Keys(Collections)); !slices.Equal(names, want) {
		t.Errorf("collection_versions holds %v, Collections declares %v", names, want)
	}

	for name, tables := range Collections {
		for _, table := range tables {
			for _, event := range []string{"INSERT", "UPDATE", "DELETE"} {
				var sql string
				err := db.QueryRow("SELECT sql FROM sqlite_schema WHERE type = 'trigger' AND tbl_name = ? AND sql LIKE ?",
					table, "%AFTER "+event+" ON "+table+"%").Scan(&sql)
				if err != nil {
					t.Errorf("%s: no trigger AFTER %s on %s: %v", name, event, table, err)
					continue
				}
				increase := regexp.MustCompile(`UPDATE collection_versions SET version = version \+ 1 WHERE name = '` + name + `'`)
				if !increase.MatchString(sql) {
					t.Errorf("trigger AFTER %s on %s does not increase %s: %s", event, table, name, sql)
				}
			}
		}
	}
}

func TestWriteCollections(t *testing.T) {
	ctx := context.Background()
	db := newTribeDB(t)
	library := dish.NewLibrary(NewDishStore(db))

	// Three ingredients and a dish with three lines: many rows, one increase each.
	_, versions, err := library.Create(ctx, dish.Input{Name: "Chili", Lines: []dish.LineInput{
		{Name: "bœuf"}, {Name: "haricots"}, {Name: "oignon"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if want := (collection.Versions{"dishes": 1, "ingredients": 1}); !maps.Equal(versions, want) {
		t.Errorf("versions %v, want %v", versions, want)
	}
	if got := versionOf(t, db, "dishes"); got != 1 {
		t.Errorf("stored version of dishes %d", got)
	}

	// No ingredient written: the catalog keeps its version, absent from the versions.
	_, versions, err = library.Create(ctx, dish.Input{Name: "Restes"})
	if err != nil {
		t.Fatal(err)
	}
	if want := (collection.Versions{"dishes": 2}); !maps.Equal(versions, want) {
		t.Errorf("versions %v, want %v", versions, want)
	}

	// A refused write changes nothing.
	if _, _, err := library.Create(ctx, dish.Input{Name: "chili", Lines: []dish.LineInput{{Name: "sel"}}}); err == nil {
		t.Fatal("name taken accepted")
	}
	if versionOf(t, db, "dishes") != 2 || versionOf(t, db, "ingredients") != 1 {
		t.Error("versions changed by a refused write")
	}
}

func TestDishStore(t *testing.T) {
	ctx := context.Background()
	db := newTribeDB(t)
	store := NewDishStore(db)
	library := dish.NewLibrary(store)

	if _, err := db.Exec("UPDATE tribe SET size = 5"); err != nil {
		t.Fatal(err)
	}
	chili, _, err := library.Create(ctx, dish.Input{Name: "Chili con carne", Lines: []dish.LineInput{
		{Name: "bœuf haché", Quantity: "500", Unit: "g"}, {Name: "sel"}, {Name: "oignon", Quantity: "1,5", Unit: "piece"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if chili.Servings != 5 || len(chili.ID) != 32 {
		t.Errorf("dish %+v", chili)
	}
	if _, _, err := library.Update(ctx, chili.ID, 1, dish.Input{Name: "Chili", Servings: 6, Lines: []dish.LineInput{
		{Name: "oignon", Quantity: "2", Unit: "piece"}, {Name: "Sel", Quantity: "1", Unit: "pinch"},
	}}); err != nil {
		t.Fatal(err)
	}

	_, page, err := library.Dishes(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	want := dish.Dish{ID: chili.ID, Name: "Chili", Color: chili.Color, Servings: 6, Revision: 2, Lines: []dish.Line{
		{Ingredient: dish.Ingredient{Name: "oignon", DefaultUnit: dish.Piece}, Quantity: &dish.Quantity{Amount: 2000, Unit: dish.Piece}},
		{Ingredient: dish.Ingredient{Name: "sel", DefaultUnit: dish.Pinch}, Quantity: &dish.Quantity{Amount: 1000, Unit: dish.Pinch}},
	}}
	if len(page.Items) != 1 || !reflect.DeepEqual(page.Items[0], want) || page.Next != "" {
		t.Errorf("dishes %+v, want %+v", page.Items, want)
	}

	// Changed in between (Q25): the current dish, as read in the database.
	_, _, err = library.Update(ctx, chili.ID, 1, dish.Input{Name: "Chili", Servings: 4})
	var changed *dish.ChangedError
	if !errors.As(err, &changed) || changed.Current.Revision != 2 || len(changed.Current.Lines) != 2 {
		t.Errorf("error %v", err)
	}
	if _, _, err := library.Update(ctx, "0123", 1, dish.Input{Name: "Chili", Servings: 4}); !errors.Is(err, dish.ErrNotFound) {
		t.Errorf("unknown dish: %v", err)
	}

	_, catalog, err := library.Catalog(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	wantCatalog := []dish.Ingredient{{Name: "bœuf haché", DefaultUnit: dish.Gram}, {Name: "sel", DefaultUnit: dish.Pinch}, {Name: "oignon", DefaultUnit: dish.Piece}}
	if !slices.Equal(catalog.Items, wantCatalog) {
		t.Errorf("catalog %v, want %v", catalog.Items, wantCatalog)
	}
}

func TestDishStorePages(t *testing.T) {
	ctx := context.Background()
	store := NewDishStore(newTribeDB(t))
	library := dish.NewLibrary(store)
	for i := range 5 {
		if _, _, err := library.Create(ctx, dish.Input{Name: fmt.Sprintf("Plat %d", i), Lines: []dish.LineInput{
			{Name: fmt.Sprintf("ingrédient %d", i)}, {Name: "sel"},
		}}); err != nil {
			t.Fatal(err)
		}
	}

	var dishes []dish.Dish
	pages := 0
	for after := ""; pages == 0 || after != ""; pages++ {
		version, page, err := store.Dishes(ctx, after, 2)
		if err != nil {
			t.Fatal(err)
		}
		if version != 5 {
			t.Errorf("version %d", version)
		}
		dishes = append(dishes, page.Items...)
		after = page.Next
	}
	if pages != 3 || len(dishes) != 5 {
		t.Errorf("%d pages, %d dishes", pages, len(dishes))
	}
	if !slices.IsSortedFunc(dishes, func(a, b dish.Dish) int { return strings.Compare(string(a.ID), string(b.ID)) }) {
		t.Error("dishes out of the order of the identifiers")
	}
	for _, d := range dishes {
		if len(d.Lines) != 2 {
			t.Errorf("%s: lines %v", d.Name, d.Lines)
		}
	}

	var names []string
	pages = 0
	for after := ""; pages == 0 || after != ""; pages++ {
		_, page, err := store.Ingredients(ctx, after, 3)
		if err != nil {
			t.Fatal(err)
		}
		for _, i := range page.Items {
			names = append(names, i.Name)
		}
		after = page.Next
	}
	if pages != 2 || len(names) != 6 {
		t.Errorf("%d pages, ingredients %v", pages, names)
	}

	for _, cursor := range []string{"x", "-1", "1.5"} {
		if _, _, err := store.Ingredients(ctx, cursor, 3); !errors.Is(err, collection.ErrBadCursor) {
			t.Errorf("cursor %q: %v", cursor, err)
		}
	}
}
