package storage

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/nelt/tribe-menus/internal/collection"
	"github.com/nelt/tribe-menus/internal/dish"
	"github.com/nelt/tribe-menus/internal/mealplan"
)

// TestMealPlanSchema: migration 0008 (plan planning, step 7).
func TestMealPlanSchema(t *testing.T) {
	db := newTribeDB(t)
	setup := []string{
		"INSERT INTO dishes VALUES ('d1', 'Chili', 'chili', '#C2502E', 4, 1)",
		"INSERT INTO dishes VALUES ('d2', 'Gratin', 'gratin', '#3F7F25', 4, 1)",
		"INSERT INTO served_dishes VALUES ('s1', '2026-10-06', 'dinner', 'd1', 4, 1)",
		"INSERT INTO served_dishes VALUES ('s2', '2000-01-01', 'lunch', 'd1', 1, 2)",
		"INSERT INTO served_dishes VALUES ('s3', '2099-12-31', 'dinner', 'd1', 50, 3)",
		"INSERT INTO served_dishes VALUES ('s4', '2028-02-29', 'dinner', 'd1', 4, 4)",
		"INSERT INTO served_dishes VALUES ('s5', '2026-10-06', 'lunch', 'd1', 4, 5)",
		"INSERT INTO served_dishes VALUES ('s6', '2026-10-06', 'dinner', 'd2', 4, 6)",
	}
	for _, s := range setup {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("%s: %v", s, err)
		}
	}

	refused := []struct{ name, sql string }{
		{"same dish in a meal (Q21)", "INSERT INTO served_dishes VALUES ('s9', '2026-10-06', 'dinner', 'd1', 4, 9)"},
		{"unknown dish", "INSERT INTO served_dishes VALUES ('s9', '2026-10-07', 'dinner', 'd9', 4, 9)"},
		{"servings 0", "INSERT INTO served_dishes VALUES ('s9', '2026-10-07', 'dinner', 'd1', 0, 9)"},
		{"servings 51", "INSERT INTO served_dishes VALUES ('s9', '2026-10-07', 'dinner', 'd1', 51, 9)"},
		{"rank 0", "INSERT INTO served_dishes VALUES ('s9', '2026-10-07', 'dinner', 'd1', 4, 0)"},
		{"unknown mealtime", "INSERT INTO served_dishes VALUES ('s9', '2026-10-07', 'breakfast', 'd1', 4, 9)"},
		{"before 2000", "INSERT INTO served_dishes VALUES ('s9', '1999-12-31', 'dinner', 'd1', 4, 9)"},
		{"after 2099", "INSERT INTO served_dishes VALUES ('s9', '2100-01-01', 'dinner', 'd1', 4, 9)"},
		{"30 February", "INSERT INTO served_dishes VALUES ('s9', '2026-02-30', 'dinner', 'd1', 4, 9)"},
		{"29 February of a common year", "INSERT INTO served_dishes VALUES ('s9', '2026-02-29', 'dinner', 'd1', 4, 9)"},
		{"date with a time", "INSERT INTO served_dishes VALUES ('s9', '2026-10-07 12:00', 'dinner', 'd1', 4, 9)"},
		{"short date", "INSERT INTO served_dishes VALUES ('s9', '2026-10-7', 'dinner', 'd1', 4, 9)"},
		{"same identifier", "INSERT INTO served_dishes VALUES ('s1', '2026-10-07', 'dinner', 'd1', 4, 9)"},
		{"dish served deleted", "DELETE FROM dishes WHERE id = 'd1'"},
	}
	for _, c := range refused {
		t.Run(c.name, func(t *testing.T) {
			if _, err := db.Exec(c.sql); err == nil {
				t.Errorf("%s accepted", c.sql)
			}
		})
	}
}

// TestMealPlanTriggers: every write to the served dishes increases the version of the meal
// plan, and only its own; a change of a dish does not (ADR 0029, point 2).
func TestMealPlanTriggers(t *testing.T) {
	db := newTribeDB(t)
	cases := []struct {
		name     string
		sql      string
		mealPlan int64
		dishes   int64
	}{
		{"insert a dish", "INSERT INTO dishes VALUES ('d1', 'Chili', 'chili', '#C2502E', 4, 1)", 0, 1},
		{"serve a dish", "INSERT INTO served_dishes VALUES ('s1', '2026-10-06', 'dinner', 'd1', 4, 1)", 1, 0},
		{"set servings", "UPDATE served_dishes SET servings = 3", 1, 0},
		{"change the dish", "UPDATE dishes SET servings = 6", 0, 1},
		{"remove", "DELETE FROM served_dishes", 1, 0},
		{"tribe size", "UPDATE tribe SET size = 5", 0, 0},
	}
	for _, c := range cases {
		mealPlan, dishes := versionOf(t, db, "meal_plan"), versionOf(t, db, "dishes")
		if _, err := db.Exec(c.sql); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got := versionOf(t, db, "meal_plan") - mealPlan; got != c.mealPlan {
			t.Errorf("%s: meal_plan +%d, want +%d", c.name, got, c.mealPlan)
		}
		if got := versionOf(t, db, "dishes") - dishes; got != c.dishes {
			t.Errorf("%s: dishes +%d, want +%d", c.name, got, c.dishes)
		}
	}
}

func TestMealPlanStore(t *testing.T) {
	ctx := context.Background()
	db := newTribeDB(t)
	library := dish.NewLibrary(NewDishStore(db))
	plan := mealplan.NewPlan(NewMealPlanStore(db))
	var ids []dish.ID
	for _, name := range []string{"Chili con carne", "Chili végétarien"} {
		d, _, err := library.Create(ctx, dish.Input{Name: name})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, d.ID)
	}
	chili, vege := ids[0], ids[1]
	if _, err := db.Exec("UPDATE tribe SET size = 5"); err != nil {
		t.Fatal(err)
	}
	dinner := func(d dish.ID) mealplan.ServeInput {
		return mealplan.ServeInput{Date: "2026-10-06", Mealtime: "dinner", Dish: d}
	}

	first, versions, err := plan.Serve(ctx, dinner(vege))
	if err != nil {
		t.Fatal(err)
	}
	want := mealplan.ServedDish{ID: first.ID, Date: "2026-10-06", Mealtime: mealplan.Dinner, Dish: vege, Servings: 5, Rank: 1}
	if first != want || len(first.ID) != 32 {
		t.Errorf("served %+v, want %+v", first, want)
	}
	if !maps.Equal(versions, collection.Versions{"meal_plan": 1}) {
		t.Errorf("versions %v", versions)
	}
	second, versions, err := plan.Serve(ctx, dinner(chili))
	if err != nil {
		t.Fatal(err)
	}
	if second.Rank != 2 || versions["meal_plan"] != 2 {
		t.Errorf("second %+v, versions %v", second, versions)
	}

	// Already served (Q21): the meal by rank, nothing written.
	_, _, err = plan.Serve(ctx, dinner(chili))
	var already *mealplan.AlreadyServedError
	if !errors.As(err, &already) || len(already.Meal) != 2 || already.Meal[0] != first || already.Meal[1] != second {
		t.Errorf("served twice: %v", err)
	}
	if _, _, err := plan.Serve(ctx, dinner("0123")); !errors.Is(err, mealplan.ErrUnknownDish) {
		t.Errorf("unknown dish: %v", err)
	}
	if v := versionOf(t, db, "meal_plan"); v != 2 {
		t.Errorf("version %d after refused writes", v)
	}

	set, versions, err := plan.SetServings(ctx, first.ID, 3)
	if err != nil {
		t.Fatal(err)
	}
	wantSet := first
	wantSet.Servings = 3
	if set != wantSet || versions["meal_plan"] != 3 {
		t.Errorf("set %+v, versions %v", set, versions)
	}
	if versions, err = plan.Remove(ctx, first.ID); err != nil || versions["meal_plan"] != 4 {
		t.Errorf("remove: %v, versions %v", err, versions)
	}
	if _, _, err := plan.SetServings(ctx, first.ID, 2); !errors.Is(err, mealplan.ErrNotFound) {
		t.Errorf("servings of a removed dish: %v", err)
	}
	if _, err := plan.Remove(ctx, first.ID); !errors.Is(err, mealplan.ErrNotFound) {
		t.Errorf("removed twice: %v", err)
	}

	// Served again (D2): a new served dish, after the other (D11).
	again, _, err := plan.Serve(ctx, dinner(vege))
	if err != nil {
		t.Fatal(err)
	}
	if again.ID == first.ID || again.Servings != 5 || again.Rank != 3 {
		t.Errorf("served again %+v", again)
	}

	version, page, err := plan.ServedDishes(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	got := slices.Clone(page.Items)
	wantAll := []mealplan.ServedDish{second, again}
	slices.SortFunc(wantAll, func(a, b mealplan.ServedDish) int { return strings.Compare(string(a.ID), string(b.ID)) })
	if version != 5 || !slices.Equal(got, wantAll) || page.Next != "" {
		t.Errorf("version %d, page %+v, want %+v", version, page, wantAll)
	}
}

func TestMealPlanStorePages(t *testing.T) {
	ctx := context.Background()
	db := newTribeDB(t)
	store := NewMealPlanStore(db)
	d, _, err := dish.NewLibrary(NewDishStore(db)).Create(ctx, dish.Input{Name: "Chili"})
	if err != nil {
		t.Fatal(err)
	}
	plan := mealplan.NewPlan(store)
	for day := 1; day <= 5; day++ {
		if _, _, err := plan.Serve(ctx, mealplan.ServeInput{Date: fmt.Sprintf("2026-10-%02d", day), Mealtime: "lunch", Dish: d.ID}); err != nil {
			t.Fatal(err)
		}
	}
	var served []mealplan.ServedDish
	pages := 0
	for after := ""; pages == 0 || after != ""; pages++ {
		version, page, err := store.ServedDishes(ctx, after, 2)
		if err != nil {
			t.Fatal(err)
		}
		if version != 5 {
			t.Errorf("version %d", version)
		}
		served = append(served, page.Items...)
		after = page.Next
	}
	if pages != 3 || len(served) != 5 {
		t.Errorf("%d pages, %d served dishes", pages, len(served))
	}
	if !slices.IsSortedFunc(served, func(a, b mealplan.ServedDish) int { return strings.Compare(string(a.ID), string(b.ID)) }) {
		t.Error("served dishes out of the order of the identifiers")
	}
}
