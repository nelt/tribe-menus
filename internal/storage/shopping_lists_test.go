package storage

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/nelt/tribe-menus/internal/collection"
	"github.com/nelt/tribe-menus/internal/dish"
	"github.com/nelt/tribe-menus/internal/mealplan"
	"github.com/nelt/tribe-menus/internal/shopping"
)

func TestShoppingStore(t *testing.T) {
	ctx := context.Background()
	db := newTribeDB(t)
	lists := shopping.NewLists(NewShoppingStore(db))

	// A new tribe: its main list "Courses", empty (C1).
	version, page, err := lists.Page(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if version != 0 || len(page.Items) != 1 || page.Next != "" {
		t.Fatalf("version %d, page %+v", version, page)
	}
	courses := page.Items[0]
	if courses.Name != "Courses" || !courses.Main || courses.Rank != 1 || courses.Items != nil || courses.Received != nil {
		t.Errorf("main list %+v", courses)
	}

	marche, versions, err := lists.Create(ctx, "Marché")
	if err != nil {
		t.Fatal(err)
	}
	if marche.Rank != 2 || marche.Main || len(marche.ID) != 32 || !maps.Equal(versions, collection.Versions{"shopping_lists": 1}) {
		t.Errorf("created %+v, versions %v", marche, versions)
	}
	if _, versions, err = lists.Rename(ctx, marche.ID, "Marché du samedi"); err != nil || versions["shopping_lists"] != 2 {
		t.Errorf("rename: %v, %v", versions, err)
	}

	// An item creates its ingredient: both versions increase by one.
	l, versions, err := lists.AddItem(ctx, courses.ID, dish.LineInput{Name: "pain", Quantity: "2", Unit: "piece"})
	if err != nil {
		t.Fatal(err)
	}
	if !maps.Equal(versions, collection.Versions{"shopping_lists": 3, "ingredients": 1}) || len(l.Items) != 1 {
		t.Errorf("list %+v, versions %v", l, versions)
	}
	pain := l.Items[0].ID
	if l, versions, err = lists.AddItem(ctx, courses.ID, dish.LineInput{Name: "Lessive"}); err != nil || versions["ingredients"] != 2 {
		t.Fatalf("lessive: %v, %v", versions, err)
	}
	if _, _, err = lists.AddItem(ctx, courses.ID, dish.LineInput{Name: "pain"}); err != nil {
		t.Fatal(err)
	}
	if l, _, err = lists.SetChecked(ctx, courses.ID, pain, true); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, it := range l.Items {
		got = append(got, itemText(it))
	}
	if want := []string{"Lessive - manual", "pain - manual", "pain 2 piece manual checked"}; !slices.Equal(got, want) {
		t.Errorf("items %v, want %v", got, want)
	}

	// The list as read back, with a source and a received served dish written by hand, as
	// the meal shopping will (lot C).
	if _, err := db.Exec("INSERT INTO dishes VALUES ('d1', 'Chili', 'chili', '#C2502E', 4, 1)"); err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{
		"INSERT INTO ingredients (id, name, normalized_name) VALUES (9, 'oignon', 'oignon')",
		fmt.Sprintf("INSERT INTO shopping_items VALUES ('c1', '%s', 9, 2000, 'piece', 0, 0, NULL)", courses.ID),
		"INSERT INTO shopping_item_sources VALUES ('c1', 's2', 'd1', '2026-10-08', 'lunch', 4, 1000, 'piece')",
		"INSERT INTO shopping_item_sources VALUES ('c1', 's1', 'd1', '2026-10-06', 'dinner', 3, 750, 'piece')",
		"INSERT INTO shopping_item_sources VALUES ('c1', 's3', 'd1', '2026-10-06', 'lunch', 1, 250, 'piece')",
		fmt.Sprintf("INSERT INTO shopping_received VALUES ('%s', 's2', 'd1', 4, 4)", courses.ID),
		fmt.Sprintf("INSERT INTO shopping_received VALUES ('%s', 's1', 'd1', 3, 4)", courses.ID),
	} {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
	_, page, err = lists.Page(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	read := page.Items[slices.IndexFunc(page.Items, func(l shopping.List) bool { return l.ID == courses.ID })]
	oignon := read.Items[slices.IndexFunc(read.Items, func(it shopping.Item) bool { return it.ID == "c1" })]
	var sources []string
	for _, s := range oignon.Sources {
		sources = append(sources, fmt.Sprintf("%s %s %s %s %d %s", s.ServedDish, s.Dish, s.Date, s.Mealtime, s.Servings, s.Quantity.Text()))
	}
	wantSources := []string{"s3 d1 2026-10-06 lunch 1 0.25", "s1 d1 2026-10-06 dinner 3 0.75", "s2 d1 2026-10-08 lunch 4 1"}
	if oignon.Manual || !slices.Equal(sources, wantSources) {
		t.Errorf("computed item %+v, sources %v", oignon, sources)
	}
	if !slices.Equal(read.Received, []shopping.ServedDishID{"s1", "s2"}) || len(read.Items) != 4 {
		t.Errorf("list %+v", read)
	}

	// Refused: nothing written.
	before := versionOf(t, db, "shopping_lists")
	if _, _, err := lists.RemoveItem(ctx, courses.ID, "c1"); !errors.Is(err, shopping.ErrComputedItem) {
		t.Errorf("remove a computed item: %v", err)
	}
	if _, err := lists.Delete(ctx, courses.ID); !errors.Is(err, shopping.ErrMainList) {
		t.Errorf("delete the main list: %v", err)
	}
	if _, err := lists.SetMain(ctx, "0123"); !errors.Is(err, shopping.ErrNotFound) {
		t.Errorf("unknown list: %v", err)
	}
	if v := versionOf(t, db, "shopping_lists"); v != before {
		t.Errorf("version %d after refusals, %d before", v, before)
	}

	// The main list changes; the received served dishes of a list leave with it.
	if versions, err = lists.SetMain(ctx, marche.ID); err != nil || versions["shopping_lists"] != before+1 {
		t.Errorf("set main: %v, %v", versions, err)
	}
	// The list already main: nothing written, no version (review of PR #98, point 2).
	if versions, err = lists.SetMain(ctx, marche.ID); err != nil || len(versions) != 0 {
		t.Errorf("set main again: %v, %v", versions, err)
	}
	for _, s := range []string{
		fmt.Sprintf("INSERT INTO shopping_lists VALUES ('%s', 'Drive', 9, 0)", strings.Repeat("f", 32)),
		fmt.Sprintf("INSERT INTO shopping_received VALUES ('%s', 's1', 'd1', 3, 4)", strings.Repeat("f", 32)),
		fmt.Sprintf("INSERT INTO shopping_received_lines VALUES ('%s', 's1', 0, 9, 1000, 'piece')", strings.Repeat("f", 32)),
	} {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}
	if _, err := lists.Delete(ctx, shopping.ListID(strings.Repeat("f", 32))); err != nil {
		t.Errorf("delete a list with received served dishes: %v", err)
	}
	_, page, err = lists.Page(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	var mains []string
	for _, l := range page.Items {
		mains = append(mains, fmt.Sprintf("%s %v", l.Name, l.Main))
	}
	slices.Sort(mains)
	if !slices.Equal(mains, []string{"Courses false", "Marché du samedi true"}) {
		t.Errorf("lists %v", mains)
	}

	// Shopping done: pain, checked, leaves with its sources; c1, unchecked, stays.
	if _, _, err := lists.SetChecked(ctx, courses.ID, "c1", true); err != nil {
		t.Fatal(err)
	}
	if _, _, err := lists.SetChecked(ctx, courses.ID, "c1", false); err != nil {
		t.Fatal(err)
	}
	l, removed, _, err := lists.CheckOut(ctx, courses.ID, []shopping.ItemID{pain, "c1"})
	if err != nil || removed != 1 || len(l.Items) != 3 {
		t.Errorf("check out: %d removed, %+v, %v", removed, l, err)
	}
	var n int
	if err := db.QueryRow("SELECT count(*) FROM shopping_item_sources").Scan(&n); err != nil || n != 3 {
		t.Errorf("%d sources, %v", n, err)
	}
	if err := db.QueryRow("SELECT count(*) FROM shopping_received").Scan(&n); err != nil || n != 2 {
		t.Errorf("%d received served dishes after the shopping done (Q22), %v", n, err)
	}
	// A computed item checked then checked out leaves with its sources.
	if _, _, err := lists.SetChecked(ctx, courses.ID, "c1", true); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := lists.CheckOut(ctx, courses.ID, []shopping.ItemID{"c1"}); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT count(*) FROM shopping_item_sources").Scan(&n); err != nil || n != 0 {
		t.Errorf("%d sources, %v", n, err)
	}
}

func itemText(it shopping.Item) string {
	q, u := "-", ""
	if it.Quantity != nil {
		q, u = it.Quantity.Text(), " "+string(it.Quantity.Unit)
	}
	s := it.Ingredient + " " + q + u
	if it.Manual {
		s += " manual"
	}
	if it.Checked {
		s += " checked"
	}
	return s
}

func TestShoppingStorePages(t *testing.T) {
	ctx := context.Background()
	db := newTribeDB(t)
	store := NewShoppingStore(db)
	lists := shopping.NewLists(store)
	for i := range 4 {
		l, _, err := lists.Create(ctx, fmt.Sprintf("Liste %d", i))
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := lists.AddItem(ctx, l.ID, dish.LineInput{Name: fmt.Sprintf("article %d", i)}); err != nil {
			t.Fatal(err)
		}
	}
	var all []shopping.List
	pages := 0
	for after := ""; pages == 0 || after != ""; pages++ {
		version, page, err := store.Lists(ctx, after, 2)
		if err != nil {
			t.Fatal(err)
		}
		if version != 8 {
			t.Errorf("version %d", version)
		}
		all = append(all, page.Items...)
		after = page.Next
	}
	if pages != 3 || len(all) != 5 {
		t.Errorf("%d pages, %d lists", pages, len(all))
	}
	if !slices.IsSortedFunc(all, func(a, b shopping.List) int { return strings.Compare(string(a.ID), string(b.ID)) }) {
		t.Error("lists out of the order of the identifiers")
	}
	for _, l := range all {
		if !l.Main && (len(l.Items) != 1 || l.Items[0].Ingredient != "article "+strings.TrimPrefix(l.Name, "Liste ")) {
			t.Errorf("list %+v", l)
		}
	}
}

// TestShoppingStoreMealShopping: the meal shopping reads the meal plan and the dishes as they
// are now (P2), writes items, sources, received served dishes and their lines (D5) in one
// write, and reads back what it returned.
func TestShoppingStoreMealShopping(t *testing.T) {
	ctx := context.Background()
	db := newTribeDB(t)
	library := dish.NewLibrary(NewDishStore(db))
	plan := mealplan.NewPlan(NewMealPlanStore(db))
	lists := shopping.NewLists(NewShoppingStore(db))
	chili, _, err := library.Create(ctx, dish.Input{Name: "Chili con carne", Servings: 4, Lines: []dish.LineInput{
		{Name: "bœuf haché", Quantity: "500", Unit: "g"}, {Name: "oignon", Quantity: "1", Unit: "piece"}, {Name: "sel"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	restes, _, err := library.Create(ctx, dish.Input{Name: "Restes", Servings: 4})
	if err != nil {
		t.Fatal(err)
	}
	serve := func(date, mealtime string, d dish.ID, servings int) mealplan.ServedDish {
		t.Helper()
		s, _, err := plan.Serve(ctx, mealplan.ServeInput{Date: date, Mealtime: mealtime, Dish: d, Servings: &servings})
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	thu := serve("2026-10-08", "lunch", chili.ID, 4)
	tue := serve("2026-10-06", "dinner", chili.ID, 3)
	serve("2026-10-07", "lunch", restes.ID, 4)
	serve("2026-10-13", "dinner", chili.ID, 4)
	// P2: the dish as it is now, 600 g of beef.
	chili.Lines[0].Quantity = &dish.Quantity{Amount: 600_000, Unit: dish.Gram}
	if _, _, err := library.Update(ctx, chili.ID, chili.Revision, dish.Input{Name: chili.Name, Servings: 4, Lines: []dish.LineInput{
		{Name: "bœuf haché", Quantity: "600", Unit: "g"}, {Name: "oignon", Quantity: "1", Unit: "piece"}, {Name: "sel"},
	}}); err != nil {
		t.Fatal(err)
	}
	version, page, err := lists.Page(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	courses := page.Items[0]

	l, added, ignored, versions, err := lists.AddMealShopping(ctx, courses.ID, "2026-10-06", "2026-10-12")
	if err != nil {
		t.Fatal(err)
	}
	if added != 3 || ignored != 0 || !maps.Equal(versions, collection.Versions{"shopping_lists": version + 1}) {
		t.Errorf("added %d, ignored %d, versions %v", added, ignored, versions)
	}
	var got []string
	for _, it := range l.Items {
		got = append(got, itemText(it))
	}
	if want := []string{"bœuf haché 1050 g", "oignon 2 piece", "sel -"}; !slices.Equal(got, want) {
		t.Errorf("items %v, want %v", got, want)
	}
	beef := l.Items[0]
	wantSources := []shopping.Source{
		{ServedDish: shopping.ServedDishID(tue.ID), Dish: chili.ID, Date: "2026-10-06", Mealtime: "dinner", Servings: 3, Quantity: &dish.Quantity{Amount: 450_000, Unit: dish.Gram}},
		{ServedDish: shopping.ServedDishID(thu.ID), Dish: chili.ID, Date: "2026-10-08", Mealtime: "lunch", Servings: 4, Quantity: &dish.Quantity{Amount: 600_000, Unit: dish.Gram}},
	}
	if !reflect.DeepEqual(beef.Sources, wantSources) || len(l.Received) != 3 {
		t.Errorf("sources %v, want %v; received %v", beef.Sources, wantSources, l.Received)
	}
	_, page, err = lists.Page(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(page.Items[0], l) {
		t.Errorf("read back\n%+v\nreturned\n%+v", page.Items[0], l)
	}

	// The received served dishes keep the servings, the reference servings and the lines of
	// the dish (D5), "Restes" without lines.
	var received, lines int
	if err := db.QueryRow("SELECT count(*) FROM shopping_received WHERE list_id = ? AND dish_servings = 4", courses.ID).Scan(&received); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT count(*) FROM shopping_received_lines WHERE list_id = ? AND served_dish_id = ?", courses.ID, tue.ID).Scan(&lines); err != nil {
		t.Fatal(err)
	}
	var amount int64
	if err := db.QueryRow("SELECT amount FROM shopping_received_lines WHERE served_dish_id = ? AND position = 0", tue.ID).Scan(&amount); err != nil {
		t.Fatal(err)
	}
	if received != 3 || lines != 3 || amount != 600_000 {
		t.Errorf("received %d, lines %d, amount %d", received, lines, amount)
	}

	// A check, then one served dish more: poured in the items present, unchecked as the
	// quantity increases; the others ignored.
	if _, _, err := lists.SetChecked(ctx, courses.ID, beef.ID, true); err != nil {
		t.Fatal(err)
	}
	serve("2026-10-09", "dinner", chili.ID, 4)
	l, added, ignored, _, err = lists.AddMealShopping(ctx, courses.ID, "2026-10-06", "2026-10-12")
	if err != nil || added != 1 || ignored != 3 {
		t.Fatalf("again: added %d, ignored %d, %v", added, ignored, err)
	}
	if it := l.Items[0]; itemText(it) != "bœuf haché 1650 g" || len(it.Sources) != 3 || it.ID != beef.ID {
		t.Errorf("beef %s %+v", itemText(it), it)
	}
}
