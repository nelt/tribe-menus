package shopping

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/nelt/tribe-menus/internal/dish"
)

// manual returns an item added by hand.
func manual(id, ingredient string, quantity *dish.Quantity) Item {
	return Item{ID: ItemID(id), Ingredient: ingredient, Quantity: quantity, Manual: true}
}

// renamed gives the list another identifier, and its items identifiers prefixed by it.
func renamed(l List, id ListID) List {
	l.ID = id
	l.Items = slices.Clone(l.Items)
	for i := range l.Items {
		l.Items[i].ID = ItemID(string(id) + "-" + string(l.Items[i].ID))
	}
	return l
}

func idOf(l List, ingredient string) ItemID {
	it, _ := itemNamed(l, ingredient)
	return it.ID
}

func TestMove(t *testing.T) {
	tue := served("s1", "chili", "2026-10-06", "dinner", 4, 4, chiliLines)
	thu := served("s2", "chili", "2026-10-08", "lunch", 4, 4, chiliLines)
	steak := served("s3", "steak", "2026-10-09", "dinner", 4, 4, steakLines)
	chiliTue := poured(t, tue)
	chiliThu := renamed(poured(t, thu), "marche")
	breadAndSoap := List{ID: "courses", Items: []Item{manual("pain", "pain", q(2000, dish.Piece)), manual("lessive", "lessive", nil)}}
	marche := List{ID: "marche"}

	cases := []struct {
		name     string
		source   List
		target   List
		select_  []string // ingredients of the source list, or identifiers absent from it
		kept     []Served
		left     []string
		arrived  []string
		received []ServedDishID
		moved    int
		missing  int
		merged   int
	}{
		{
			name: "item added by hand (C9)", source: breadAndSoap, target: marche, select_: []string{"pain"},
			left: []string{"lessive manual"}, arrived: []string{"pain 2 piece manual"}, moved: 1,
		},
		{
			name: "check and sources kept", source: withItem(chiliTue, "bœuf haché", checked), target: marche,
			select_: []string{"bœuf haché"}, kept: []Served{tue},
			left: []string{"oignon 1 piece | s1 1"}, arrived: []string{"bœuf haché 500 g checked | s1 500"},
			received: []ServedDishID{"s1"}, moved: 1,
		},
		{
			name: "merged with the same ingredient", source: chiliTue, target: chiliThu,
			select_: []string{"bœuf haché"}, kept: []Served{tue},
			left:     []string{"oignon 1 piece | s1 1"},
			arrived:  []string{"bœuf haché 1000 g | s1 500 | s2 500", "oignon 1 piece | s2 1"},
			received: []ServedDishID{"s1", "s2"}, moved: 1, merged: 1,
		},
		{
			name: "served dish already in the target list not counted twice", source: chiliTue, target: renamed(chiliTue, "marche"),
			select_: []string{"bœuf haché"}, kept: []Served{tue},
			left:     []string{"oignon 1 piece | s1 1"},
			arrived:  []string{"bœuf haché 500 g | s1 500", "oignon 1 piece | s1 1"},
			received: []ServedDishID{"s1"}, moved: 1, merged: 1,
		},
		{
			name: "merged checked only if both were (Q41)", source: withItem(chiliTue, "bœuf haché", checked), target: chiliThu,
			select_: []string{"bœuf haché"}, kept: []Served{tue},
			left:     []string{"oignon 1 piece | s1 1"},
			arrived:  []string{"bœuf haché 1000 g taken 500 | s1 500 | s2 500", "oignon 1 piece | s2 1"},
			received: []ServedDishID{"s1", "s2"}, moved: 1, merged: 1,
		},
		{
			name: "both checked", source: withItem(chiliTue, "bœuf haché", checked), target: withItem(chiliThu, "bœuf haché", checked),
			select_: []string{"bœuf haché"}, kept: []Served{tue},
			left:     []string{"oignon 1 piece | s1 1"},
			arrived:  []string{"oignon 1 piece | s2 1", "bœuf haché 1000 g checked | s1 500 | s2 500"},
			received: []ServedDishID{"s1", "s2"}, moved: 1, merged: 1,
		},
		{
			name: "taken quantities added", kept: []Served{tue},
			source:   withItem(chiliTue, "bœuf haché", func(it *Item) { it.Taken = q(200_000, dish.Gram) }),
			target:   withItem(chiliThu, "bœuf haché", func(it *Item) { it.Taken = q(100_000, dish.Gram) }),
			select_:  []string{"bœuf haché"},
			left:     []string{"oignon 1 piece | s1 1"},
			arrived:  []string{"bœuf haché 1000 g taken 300 | s1 500 | s2 500", "oignon 1 piece | s2 1"},
			received: []ServedDishID{"s1", "s2"}, moved: 1, merged: 1,
		},
		{
			name: "taken reaching the quantity: checked", source: withItem(chiliTue, "bœuf haché", checked), target: withItem(renamed(chiliTue, "marche"), "bœuf haché", checked),
			select_: []string{"oignon", "bœuf haché"}, kept: []Served{tue},
			arrived:  []string{"oignon 1 piece | s1 1", "bœuf haché 500 g checked | s1 500"},
			received: []ServedDishID{"s1"}, moved: 2, merged: 2,
		},
		{
			name: "one checked, the same served dish: checked, all is taken", source: withItem(chiliTue, "bœuf haché", checked), target: renamed(chiliTue, "marche"),
			select_: []string{"bœuf haché"}, kept: []Served{tue},
			left:     []string{"oignon 1 piece | s1 1"},
			arrived:  []string{"oignon 1 piece | s1 1", "bœuf haché 500 g checked | s1 500"},
			received: []ServedDishID{"s1"}, moved: 1, merged: 1,
		},
		{
			name: "without quantity", kept: []Served{served("s4", "pates", "2026-10-07", "lunch", 4, 4, pastaLines)},
			source:   withItem(poured(t, served("s4", "pates", "2026-10-07", "lunch", 4, 4, pastaLines)), "sel", checked),
			target:   renamed(poured(t, served("s5", "pates", "2026-10-09", "lunch", 4, 4, pastaLines)), "marche"),
			select_:  []string{"sel"},
			left:     []string{"pâtes 500 g | s4 500"},
			arrived:  []string{"pâtes 500 g | s5 500", "sel | s4 | s5"},
			received: []ServedDishID{"s4", "s5"}, moved: 1, merged: 1,
		},
		{
			name: "another family of units: not merged", kept: []Served{served("s6", "pain", "2026-10-07", "lunch", 4, 4, breadLines)},
			source:   poured(t, served("s6", "pain", "2026-10-07", "lunch", 4, 4, breadLines)),
			target:   renamed(poured(t, served("s4", "pates", "2026-10-07", "lunch", 4, 4, pastaLines)), "marche"),
			select_:  []string{"sel"},
			arrived:  []string{"pâtes 500 g | s4 500", "sel 10 g | s6 10", "sel | s4"},
			received: []ServedDishID{"s4", "s6"}, moved: 1,
		},
		{
			name: "item added by hand not merged", source: breadAndSoap,
			target:  List{ID: "marche", Items: []Item{manual("pain1", "pain", q(1000, dish.Piece))}},
			select_: []string{"pain"},
			left:    []string{"lessive manual"}, arrived: []string{"pain 2 piece manual", "pain 1 piece manual"}, moved: 1,
		},
		{
			name: "computed item not merged with an item added by hand", source: chiliTue, kept: []Served{tue},
			target:  List{ID: "marche", Items: []Item{manual("bœuf", "bœuf haché", q(300_000, dish.Gram))}},
			select_: []string{"bœuf haché"},
			left:    []string{"oignon 1 piece | s1 1"}, arrived: []string{"bœuf haché 500 g | s1 500", "bœuf haché 300 g manual"},
			received: []ServedDishID{"s1"}, moved: 1,
		},
		{
			name: "items no longer in the source list (Q42)", source: without(breadAndSoap, "lessive"), target: marche,
			select_: []string{"pain", "lessive", "pain"},
			arrived: []string{"pain 2 piece manual"}, moved: 1, missing: 1,
		},
		{
			name: "every item gone", source: List{ID: "courses"}, target: marche, select_: []string{"pain"},
			missing: 1,
		},
		{
			name: "served dishes of the moved sources only", source: poured(t, tue, steak), target: marche,
			select_: []string{"oignon"}, kept: []Served{tue, steak},
			left:     []string{"bœuf haché 1000 g | s1 500 | s3 500"},
			arrived:  []string{"oignon 1 piece | s1 1"},
			received: []ServedDishID{"s1"}, moved: 1,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var ids []ItemID
			for _, s := range c.select_ {
				if id := idOf(c.source, s); id != "" {
					ids = append(ids, id)
				} else {
					ids = append(ids, ItemID(s))
				}
			}
			m, err := move(c.source, c.target, ids, c.kept)
			if err != nil {
				t.Fatal(err)
			}
			if got := describeAll(m.source.Items); !slices.Equal(got, c.left) {
				t.Errorf("source list\n got %q\nwant %q", got, c.left)
			}
			if got := describeAll(m.target.Items); !slices.Equal(got, c.arrived) {
				t.Errorf("target list\n got %q\nwant %q", got, c.arrived)
			}
			want := slices.Sorted(slices.Values(append(slices.Clone(c.target.Received), c.received...)))
			want = slices.Compact(want)
			if !slices.Equal(m.target.Received, want) {
				t.Errorf("received %v, want %v", m.target.Received, want)
			}
			if !slices.Equal(m.source.Received, c.source.Received) {
				t.Errorf("source received %v, want %v", m.source.Received, c.source.Received)
			}
			if m.moved != c.moved || m.missing != c.missing || len(m.merged) != c.merged || len(m.relocated) != c.moved-c.merged {
				t.Errorf("moved %d, missing %d, merged %d, relocated %d", m.moved, m.missing, len(m.merged), len(m.relocated))
			}
			for _, r := range m.received {
				if !slices.ContainsFunc(c.kept, func(k Served) bool { return k.ID == r.ID && slices.Equal(k.Lines, r.Lines) }) {
					t.Errorf("received %s not copied from the kept served dishes", r.ID)
				}
			}
		})
	}
}

// TestMoveChanges checks what the store writes: an item merged loses its source item, keeps
// the identifier of the target item and gains only the sources of served dishes it had not.
func TestMoveChanges(t *testing.T) {
	tue := served("s1", "chili", "2026-10-06", "dinner", 4, 4, chiliLines)
	thu := served("s2", "chili", "2026-10-08", "lunch", 4, 4, chiliLines)
	source := poured(t, tue, thu)
	target := renamed(poured(t, thu), "marche")
	m, err := move(source, target, []ItemID{idOf(source, "bœuf haché"), idOf(source, "oignon")}, []Served{tue, thu})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.merged) != 2 || len(m.relocated) != 0 {
		t.Fatalf("merged %d, relocated %d", len(m.merged), len(m.relocated))
	}
	for _, g := range m.merged {
		if !slices.Contains([]ItemID{idOf(source, "bœuf haché"), idOf(source, "oignon")}, g.from) || g.item.ID != idOf(target, g.item.Ingredient) {
			t.Errorf("merge %s into %s", g.from, g.item.ID)
		}
		if len(g.sources) != 1 || g.sources[0].ServedDish != "s1" {
			t.Errorf("sources gained %+v", g.sources)
		}
	}
	if len(m.received) != 1 || m.received[0].ID != "s1" {
		t.Errorf("received %+v", m.received)
	}
}

func TestMoveTooManyItems(t *testing.T) {
	target := List{ID: "marche"}
	for i := range MaxItems {
		target.Items = append(target.Items, manual(fmt.Sprint("a", i), fmt.Sprint("article ", i), nil))
	}
	source := List{ID: "courses", Items: []Item{manual("pain", "pain", nil)}}
	if _, err := move(source, target, []ItemID{"pain"}, nil); !errors.Is(err, ErrTooManyItems) {
		t.Errorf("301 items: %v", err)
	}
	// A merge adds no item: allowed at 300.
	tue := served("s1", "chili", "2026-10-06", "dinner", 4, 4, chiliLines)
	chili := poured(t, tue)
	full := renamed(chili, "marche")
	for i := range MaxItems - 2 {
		full.Items = append(full.Items, manual(fmt.Sprint("a", i), fmt.Sprint("article ", i), nil))
	}
	if _, err := move(chili, full, []ItemID{idOf(chili, "bœuf haché")}, []Served{tue}); err != nil {
		t.Errorf("merge at 300 items: %v", err)
	}
}
