package shopping

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/nelt/tribe-menus/internal/dish"
)

// poured returns the list "Courses" after the meal shopping of the served dishes, as kept:
// the received served dishes as recorded. Its new items are named after their ingredient.
func poured(t *testing.T, served ...Served) List {
	t.Helper()
	ids := 0
	m, err := addMealShopping(List{ID: "courses"}, served, func() (string, error) {
		ids++
		return fmt.Sprintf("item%d", ids), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return m.list
}

// describe writes an item on one line: ingredient, quantity, taken quantity, check, and its
// sources by served dish and contributed quantity.
func describe(it Item) string {
	s := it.Ingredient
	if it.Quantity != nil {
		s += " " + it.Quantity.Text() + " " + string(it.Quantity.Unit)
	}
	if it.Taken != nil {
		s += " taken " + it.Taken.Text()
	}
	if it.Checked {
		s += " checked"
	}
	if it.Manual {
		s += " manual"
	}
	for _, src := range it.Sources {
		s += " | " + string(src.ServedDish)
		if src.Quantity != nil {
			s += " " + src.Quantity.Text()
		}
	}
	return s
}

func describeAll(items []Item) []string {
	var out []string
	for _, it := range items {
		out = append(out, describe(it))
	}
	return out
}

// withItem changes the items of the list named after their ingredient.
func withItem(l List, ingredient string, change func(*Item)) List {
	l.Items = slices.Clone(l.Items)
	for i := range l.Items {
		if l.Items[i].Ingredient == ingredient {
			change(&l.Items[i])
		}
	}
	return l
}

func without(l List, ingredients ...string) List {
	l.Items = slices.DeleteFunc(slices.Clone(l.Items), func(it Item) bool { return slices.Contains(ingredients, it.Ingredient) })
	return l
}

func checked(it *Item) { it.Checked = true }

func TestRecompute(t *testing.T) {
	tue := func(servings, dishServings int, lines []DishLine) Served {
		return served("s1", "chili", "2026-10-06", "dinner", servings, dishServings, lines)
	}
	thu := served("s2", "chili", "2026-10-08", "lunch", 4, 4, chiliLines)
	restes := served("s3", "restes", "2026-10-07", "lunch", 4, 4, nil)
	garlic := DishLine{"ail", 7, q(1000, dish.Piece)}
	withGarlic := append(slices.Clone(chiliLines), garlic)
	saltless := []DishLine{{"sel", 4, nil}}
	salted := []DishLine{{"sel", 4, q(5000, dish.Gram)}}
	soup := served("s4", "soupe", "2026-10-07", "dinner", 4, 4, []DishLine{garlic})
	chili := poured(t, tue(4, 4, chiliLines))

	cases := []struct {
		name    string
		list    List
		kept    []Served
		current []Served
		items   []string
		stale   bool
		changed int // items created or changed
		deleted int
	}{
		{
			name: "nothing changed", list: chili, kept: []Served{tue(4, 4, chiliLines)}, current: []Served{tue(4, 4, chiliLines)},
			items: []string{"bœuf haché 500 g | s1 500", "oignon 1 piece | s1 1"},
		},
		{
			name: "servings changed (C8)", list: chili, kept: []Served{tue(4, 4, chiliLines)}, current: []Served{tue(6, 4, chiliLines)},
			items: []string{"bœuf haché 750 g | s1 750", "oignon 2 piece | s1 1.5"},
			stale: true, changed: 2,
		},
		{
			name: "dish defined for 2 servings", list: chili, kept: []Served{tue(4, 4, chiliLines)}, current: []Served{tue(4, 2, chiliLines)},
			items: []string{"bœuf haché 1000 g | s1 1000", "oignon 2 piece | s1 2"},
			stale: true, changed: 2,
		},
		{
			name: "quantity of the dish changed", list: chili, kept: []Served{tue(4, 4, chiliLines)},
			current: []Served{tue(4, 4, []DishLine{{"bœuf haché", 1, q(600_000, dish.Gram)}, chiliLines[1]})},
			items:   []string{"bœuf haché 600 g | s1 600", "oignon 1 piece | s1 1"},
			stale:   true, changed: 1,
		},
		{
			name: "checked item whose quantity does not increase stays checked", list: withItem(chili, "bœuf haché", checked),
			kept: []Served{tue(4, 4, chiliLines)}, current: []Served{tue(2, 4, chiliLines)},
			items: []string{"oignon 1 piece | s1 0.5", "bœuf haché 250 g checked | s1 250"},
			stale: true, changed: 2,
		},
		{
			name: "checked item whose quantity increases keeps its taken quantity (D1)", list: withItem(chili, "bœuf haché", checked),
			kept: []Served{tue(4, 4, chiliLines)}, current: []Served{tue(6, 4, chiliLines)},
			items: []string{"bœuf haché 750 g taken 500 | s1 750", "oignon 2 piece | s1 1.5"},
			stale: true, changed: 2,
		},
		{
			name: "item whose quantity falls to its taken quantity checked again",
			list: withItem(poured(t, tue(6, 4, chiliLines)), "bœuf haché", func(it *Item) { it.Taken = q(500_000, dish.Gram) }),
			kept: []Served{tue(6, 4, chiliLines)}, current: []Served{tue(4, 4, chiliLines)},
			items: []string{"oignon 1 piece | s1 1", "bœuf haché 500 g checked | s1 500"},
			stale: true, changed: 2,
		},
		{
			name: "served dish removed from the meal plan: its items go, even checked (R4)", list: withItem(chili, "bœuf haché", checked),
			kept:  []Served{tue(4, 4, chiliLines)},
			stale: true, deleted: 2,
		},
		{
			name: "one served dish of two removed", list: poured(t, tue(4, 4, chiliLines), thu),
			kept: []Served{tue(4, 4, chiliLines), thu}, current: []Served{thu},
			items: []string{"bœuf haché 500 g | s2 500", "oignon 1 piece | s2 1"},
			stale: true, changed: 2,
		},
		{
			name: "line added to the dish: its item created (D2)", list: chili,
			kept: []Served{tue(4, 4, chiliLines)}, current: []Served{tue(4, 4, withGarlic)},
			items: []string{"ail 1 piece | s1 1", "bœuf haché 500 g | s1 500", "oignon 1 piece | s1 1"},
			stale: true, changed: 1,
		},
		{
			name: "line added to the dish, poured into the item of another dish", list: poured(t, tue(4, 4, chiliLines), soup),
			kept: []Served{tue(4, 4, chiliLines), soup}, current: []Served{tue(4, 4, withGarlic), soup},
			items: []string{"ail 2 piece | s1 1 | s4 1", "bœuf haché 500 g | s1 500", "oignon 1 piece | s1 1"},
			stale: true, changed: 1,
		},
		{
			name: "line removed from the dish: its item goes", list: chili,
			kept: []Served{tue(4, 4, chiliLines)}, current: []Served{tue(4, 4, chiliLines[:1])},
			items: []string{"bœuf haché 500 g | s1 500"},
			stale: true, deleted: 1,
		},
		{
			name: "line whose unit changed family: its item goes, another is created", list: poured(t, tue(4, 4, saltless)),
			kept: []Served{tue(4, 4, saltless)}, current: []Served{tue(4, 4, salted)},
			items: []string{"sel 5 g | s1 5"},
			stale: true, changed: 1, deleted: 1,
		},
		{
			name: "item bought: not created again", list: without(chili, "bœuf haché"),
			kept: []Served{tue(4, 4, chiliLines)}, current: []Served{tue(6, 4, withGarlic)},
			items: []string{"ail 2 piece | s1 1.5", "oignon 2 piece | s1 1.5"},
			stale: true, changed: 2,
		},
		{
			name: "every item bought: a line added changes nothing (D2)", list: without(chili, "bœuf haché", "oignon"),
			kept: []Served{tue(4, 4, chiliLines)}, current: []Served{tue(6, 4, withGarlic)},
		},
		{
			name: "served dish without ingredient: its servings change nothing", list: poured(t, restes),
			kept: []Served{restes}, current: []Served{served("s3", "restes", "2026-10-07", "lunch", 6, 4, nil)},
		},
		{
			name: "quantity written in another unit of its family: not stale (D6)", list: chili,
			kept: []Served{tue(4, 4, chiliLines)}, current: []Served{tue(4, 4, []DishLine{{"bœuf haché", 1, q(500_000, dish.Kilogram)}, chiliLines[1]})},
			items: []string{"bœuf haché 500 g | s1 500", "oignon 1 piece | s1 1"},
		},
		{
			name: "servings of a dish without quantity: written, not stale (D6)", list: poured(t, tue(4, 4, saltless)),
			kept: []Served{tue(4, 4, saltless)}, current: []Served{tue(6, 4, saltless)},
			items:   []string{"sel | s1"},
			changed: 1,
		},
		{
			name: "items added by hand never touched", list: func() List {
				l := withItem(chili, "bœuf haché", checked)
				l.Items = append(l.Items, Item{ID: "hand", Ingredient: "oignon", Quantity: q(3000, dish.Piece), Manual: true})
				return l
			}(),
			kept:  []Served{tue(4, 4, chiliLines)},
			items: []string{"oignon 3 piece manual"},
			stale: true, deleted: 2,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ids := 0
			r, err := recompute(c.list, c.kept, c.current, func() (string, error) {
				ids++
				return fmt.Sprintf("new%d", ids), nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if got := describeAll(r.list.Items); strings.Join(got, "\n") != strings.Join(c.items, "\n") {
				t.Errorf("items\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(c.items, "\n"))
			}
			if r.stale != c.stale || stale(c.list, c.kept, c.current) != c.stale {
				t.Errorf("stale %v, want %v", r.stale, c.stale)
			}
			if len(r.items) != c.changed || len(r.deleted) != c.deleted {
				t.Errorf("%d items changed, %d deleted; want %d, %d", len(r.items), len(r.deleted), c.changed, c.deleted)
			}
			if !slices.Equal(r.list.Received, c.list.Received) {
				t.Errorf("received %v, want %v", r.list.Received, c.list.Received)
			}
		})
	}
}

// TestRecomputeChanges checks what is to be written: an item changed with all its sources, a
// new item with the key of its ingredient, and the received served dishes recomputed, with
// their dish now.
func TestRecomputeChanges(t *testing.T) {
	tue := served("s1", "chili", "2026-10-06", "dinner", 4, 4, chiliLines)
	thu := served("s2", "chili", "2026-10-08", "lunch", 4, 4, chiliLines)
	now := served("s1", "chili", "2026-10-06", "dinner", 6, 4, append(slices.Clone(chiliLines), DishLine{"ail", 7, q(1000, dish.Piece)}))
	r, err := recompute(poured(t, tue, thu), []Served{tue, thu}, []Served{now, thu}, func() (string, error) { return "garlic", nil })
	if err != nil {
		t.Fatal(err)
	}
	var changes []string
	for _, c := range r.items {
		changes = append(changes, fmt.Sprintf("%s new=%v key=%d: %s", c.item.ID, c.isNew, c.key, describe(c.item)))
	}
	want := []string{
		"item1 new=false key=0: bœuf haché 1250 g | s1 750 | s2 500",
		"item2 new=false key=0: oignon 3 piece | s1 1.5 | s2 1",
		"garlic new=true key=7: ail 2 piece | s1 1.5",
	}
	if strings.Join(changes, "\n") != strings.Join(want, "\n") {
		t.Errorf("changes\n%s\nwant\n%s", strings.Join(changes, "\n"), strings.Join(want, "\n"))
	}
	if len(r.received) != 2 || r.received[0].ID != "s1" || r.received[0].Servings != 6 || len(r.received[0].Lines) != 3 || r.received[1].ID != "s2" {
		t.Errorf("received %+v", r.received)
	}
}
