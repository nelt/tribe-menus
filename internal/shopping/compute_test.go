package shopping

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/nelt/tribe-menus/internal/dish"
)

// mealShoppingCases is testdata/meal-shopping.json, shared with web/src/shopping.ts.
type mealShoppingCases struct {
	Contributions []struct {
		Quantity        string    `json:"quantity"`
		Unit            dish.Unit `json:"unit"`
		Servings        int       `json:"servings"`
		DishServings    int       `json:"dishServings"`
		Contributed     string    `json:"contributed"`
		ContributedUnit dish.Unit `json:"contributedUnit"`
	} `json:"contributions"`
	Items []struct {
		Contributions []string  `json:"contributions"`
		Unit          dish.Unit `json:"unit"`
		Quantity      string    `json:"quantity"`
	} `json:"items"`
}

func readMealShoppingCases(t *testing.T) mealShoppingCases {
	t.Helper()
	data, err := os.ReadFile("testdata/meal-shopping.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases mealShoppingCases
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	return cases
}

func mustQuantity(t *testing.T, text string, u dish.Unit) dish.Quantity {
	t.Helper()
	q, err := dish.ParseQuantity(text, u)
	if err != nil {
		t.Fatalf("%s %s: %v", text, u, err)
	}
	return q
}

func TestContribution(t *testing.T) {
	for _, c := range readMealShoppingCases(t).Contributions {
		got := Contribution(mustQuantity(t, c.Quantity, c.Unit), c.Servings, c.DishServings)
		if got.Text() != c.Contributed || got.Unit != c.ContributedUnit {
			t.Errorf("%s %s × %d ÷ %d = %s %s, want %s %s", c.Quantity, c.Unit, c.Servings, c.DishServings, got.Text(), got.Unit, c.Contributed, c.ContributedUnit)
		}
	}
}

func TestRoundItem(t *testing.T) {
	for _, c := range readMealShoppingCases(t).Items {
		total := dish.Quantity{Unit: c.Unit}
		for _, text := range c.Contributions {
			total.Amount += mustQuantity(t, text, c.Unit).Amount
		}
		if got := RoundItem(total); got.Text() != c.Quantity || got.Unit != c.Unit {
			t.Errorf("%v %s rounded to %s %s, want %s", c.Contributions, c.Unit, got.Text(), got.Unit, c.Quantity)
		}
	}
}

func TestParsePeriod(t *testing.T) {
	cases := []struct {
		from, to string
		err      error
	}{
		{"2026-10-06", "2026-10-12", nil},
		{"2026-10-06", "2026-10-06", nil},
		{"2026-10-06", "2026-11-05", nil}, // 31 days
		{"2026-02-28", "2026-03-30", nil}, // 31 days over February
		{"2000-01-01", "2000-01-31", nil},
		{"2099-12-01", "2099-12-31", nil},
		{"2026-10-08", "2026-10-06", ErrInvalidPeriod},
		{"2026-10-06", "2026-11-06", ErrPeriodTooLong}, // 32 days
		{"2026-10-06", "2027-10-06", ErrPeriodTooLong},
		{"1999-12-31", "2000-01-06", ErrInvalidDate},
		{"2099-12-28", "2100-01-01", ErrInvalidDate},
		{"2026-02-30", "2026-03-01", ErrInvalidDate},
		{"2026-10-6", "2026-10-12", ErrInvalidDate},
		{"", "2026-10-12", ErrInvalidDate},
		{"2026-10-06", "", ErrInvalidDate},
	}
	for _, c := range cases {
		p, err := ParsePeriod(c.from, c.to)
		if !errors.Is(err, c.err) {
			t.Errorf("ParsePeriod(%q, %q) = %v, want %v", c.from, c.to, err, c.err)
			continue
		}
		if err == nil && (string(p.From) != c.from || string(p.To) != c.to) {
			t.Errorf("ParsePeriod(%q, %q) = %+v", c.from, c.to, p)
		}
	}
}

func q(amount int64, u dish.Unit) *dish.Quantity {
	return &dish.Quantity{Amount: amount, Unit: u}
}

// Dishes of the examples of the spec, by their lines.
var (
	chiliLines  = []DishLine{{"bœuf haché", 1, q(500_000, dish.Gram)}, {"oignon", 2, q(1000, dish.Piece)}}
	pastaLines  = []DishLine{{"pâtes", 3, q(500_000, dish.Gram)}, {"sel", 4, nil}}
	breadLines  = []DishLine{{"sel", 4, q(10_000, dish.Gram)}}
	soupLines   = []DishLine{{"oignon", 2, q(300_000, dish.Gram)}}
	creamLines  = []DishLine{{"crème", 5, q(500, dish.Centilitre)}}
	gratinLines = []DishLine{{"Crème", 5, q(1000, dish.Litre)}}
	vinaigrette = []DishLine{{"huile d'olive", 6, q(1000, dish.Tablespoon)}}
	tabouleh    = []DishLine{{"huile d'olive", 6, q(2000, dish.Teaspoon)}}
	steakLines  = []DishLine{{"bœuf haché", 1, q(500_000, dish.Gram)}}
)

func served(id, dishID, date, mealtime string, servings, dishServings int, lines []DishLine) Served {
	return Served{ID: ServedDishID(id), Dish: dish.ID(dishID), Date: date, Mealtime: mealtime, Servings: servings, DishServings: dishServings, Lines: lines}
}

func src(servedID, dishID, date, mealtime string, servings int, quantity *dish.Quantity) Source {
	return Source{ServedDish: ServedDishID(servedID), Dish: dish.ID(dishID), Date: date, Mealtime: mealtime, Servings: servings, Quantity: quantity}
}

func TestAddMealShopping(t *testing.T) {
	tue := func(servings int) Source { return src("s1", "chili", "2026-10-06", "dinner", servings, nil) }
	thu := func(servings int) Source { return src("s2", "chili", "2026-10-08", "lunch", servings, nil) }
	with := func(s Source, quantity *dish.Quantity) Source { s.Quantity = quantity; return s }
	mon := src("s0", "chili", "2026-10-05", "dinner", 2, q(250_000, dish.Gram))

	cases := []struct {
		name           string
		list           List
		served         []Served
		items          []Item
		received       []ServedDishID
		added, ignored int
		err            error
	}{
		{
			name:   "quantities adjusted to the servings, oignon rounded up",
			served: []Served{served("s1", "chili", "2026-10-06", "dinner", 6, 4, chiliLines)},
			items: []Item{
				{ID: "id1", Ingredient: "bœuf haché", Quantity: q(750_000, dish.Gram), Sources: []Source{with(tue(6), q(750_000, dish.Gram))}},
				{ID: "id2", Ingredient: "oignon", Quantity: q(2000, dish.Piece), Sources: []Source{with(tue(6), q(1500, dish.Piece))}},
			},
			received: []ServedDishID{"s1"},
			added:    1,
		},
		{
			name: "quantities of the same ingredient added, rounding after the addition",
			served: []Served{
				served("s1", "chili", "2026-10-06", "dinner", 2, 4, chiliLines),
				served("s2", "chili", "2026-10-08", "lunch", 2, 4, chiliLines),
			},
			items: []Item{
				{ID: "id1", Ingredient: "bœuf haché", Quantity: q(500_000, dish.Gram), Sources: []Source{with(tue(2), q(250_000, dish.Gram)), with(thu(2), q(250_000, dish.Gram))}},
				{ID: "id2", Ingredient: "oignon", Quantity: q(1000, dish.Piece), Sources: []Source{with(tue(2), q(500, dish.Piece)), with(thu(2), q(500, dish.Piece))}},
			},
			received: []ServedDishID{"s1", "s2"},
			added:    2,
		},
		{
			name: "units of a family converted, case of the ingredient aside",
			served: []Served{
				served("s1", "creme", "2026-10-06", "dinner", 4, 4, creamLines),
				served("s2", "gratin", "2026-10-08", "lunch", 4, 4, gratinLines),
			},
			items: []Item{{ID: "id1", Ingredient: "crème", Quantity: q(1500, dish.Millilitre), Sources: []Source{
				src("s1", "creme", "2026-10-06", "dinner", 4, q(500, dish.Millilitre)),
				src("s2", "gratin", "2026-10-08", "lunch", 4, q(1000, dish.Millilitre)),
			}}},
			received: []ServedDishID{"s1", "s2"},
			added:    2,
		},
		{
			name: "families apart: pieces and grams, tablespoons and teaspoons",
			served: []Served{
				served("s1", "chili", "2026-10-06", "dinner", 4, 4, chiliLines[1:]),
				served("s2", "soupe", "2026-10-07", "dinner", 4, 4, soupLines),
				served("s3", "vinaigrette", "2026-10-08", "lunch", 4, 4, vinaigrette),
				served("s4", "taboule", "2026-10-08", "dinner", 4, 4, tabouleh),
			},
			items: []Item{
				{ID: "id3", Ingredient: "huile d'olive", Quantity: q(1000, dish.Tablespoon), Sources: []Source{src("s3", "vinaigrette", "2026-10-08", "lunch", 4, q(1000, dish.Tablespoon))}},
				{ID: "id4", Ingredient: "huile d'olive", Quantity: q(2000, dish.Teaspoon), Sources: []Source{src("s4", "taboule", "2026-10-08", "dinner", 4, q(2000, dish.Teaspoon))}},
				{ID: "id1", Ingredient: "oignon", Quantity: q(1000, dish.Piece), Sources: []Source{with(tue(4), q(1000, dish.Piece))}},
				{ID: "id2", Ingredient: "oignon", Quantity: q(300_000, dish.Gram), Sources: []Source{src("s2", "soupe", "2026-10-07", "dinner", 4, q(300_000, dish.Gram))}},
			},
			received: []ServedDishID{"s1", "s2", "s3", "s4"},
			added:    4,
		},
		{
			name:     "masses rounded up to the gram",
			served:   []Served{served("s1", "steak", "2026-10-06", "dinner", 1, 3, steakLines)},
			items:    []Item{{ID: "id1", Ingredient: "bœuf haché", Quantity: q(167_000, dish.Gram), Sources: []Source{src("s1", "steak", "2026-10-06", "dinner", 1, q(166_667, dish.Gram))}}},
			received: []ServedDishID{"s1"},
			added:    1,
		},
		{
			name:     "dish without ingredient: nothing added, received all the same",
			served:   []Served{served("s1", "restes", "2026-10-07", "lunch", 4, 4, nil)},
			received: []ServedDishID{"s1"},
			added:    1,
		},
		{
			name: "ingredient without quantity: one line, two sources, apart from the line with quantity",
			served: []Served{
				served("s1", "pates", "2026-10-06", "dinner", 4, 4, pastaLines),
				served("s2", "pates", "2026-10-08", "lunch", 2, 4, pastaLines),
				served("s3", "pain", "2026-10-09", "lunch", 4, 4, breadLines),
			},
			items: []Item{
				{ID: "id1", Ingredient: "pâtes", Quantity: q(750_000, dish.Gram), Sources: []Source{
					src("s1", "pates", "2026-10-06", "dinner", 4, q(500_000, dish.Gram)),
					src("s2", "pates", "2026-10-08", "lunch", 2, q(250_000, dish.Gram)),
				}},
				{ID: "id3", Ingredient: "sel", Quantity: q(10_000, dish.Gram), Sources: []Source{src("s3", "pain", "2026-10-09", "lunch", 4, q(10_000, dish.Gram))}},
				{ID: "id2", Ingredient: "sel", Sources: []Source{
					src("s1", "pates", "2026-10-06", "dinner", 4, nil),
					src("s2", "pates", "2026-10-08", "lunch", 2, nil),
				}},
			},
			received: []ServedDishID{"s1", "s2", "s3"},
			added:    3,
		},
		{
			name: "poured into the computed item present, checked item whose quantity increases unchecked",
			list: List{ID: "courses", Items: []Item{
				{ID: "old", Ingredient: "bœuf haché", Quantity: q(250_000, dish.Gram), Checked: true, Sources: []Source{mon}},
			}, Received: []ServedDishID{"s0"}},
			served:   []Served{served("s1", "chili", "2026-10-06", "dinner", 4, 4, chiliLines[:1])},
			items:    []Item{{ID: "old", Ingredient: "bœuf haché", Quantity: q(750_000, dish.Gram), Sources: []Source{mon, with(tue(4), q(500_000, dish.Gram))}}},
			received: []ServedDishID{"s0", "s1"},
			added:    1,
		},
		{
			name: "checked item whose rounded quantity does not increase stays checked",
			list: List{ID: "courses", Items: []Item{
				{ID: "old", Ingredient: "oignon", Quantity: q(1000, dish.Piece), Checked: true, Sources: []Source{src("s0", "chili", "2026-10-05", "dinner", 2, q(500, dish.Piece))}},
			}, Received: []ServedDishID{"s0"}},
			served: []Served{served("s1", "chili", "2026-10-06", "dinner", 2, 4, chiliLines[1:])},
			items: []Item{{ID: "old", Ingredient: "oignon", Quantity: q(1000, dish.Piece), Checked: true, Sources: []Source{
				src("s0", "chili", "2026-10-05", "dinner", 2, q(500, dish.Piece)), with(tue(2), q(500, dish.Piece)),
			}}},
			received: []ServedDishID{"s0", "s1"},
			added:    1,
		},
		{
			name: "checked item without quantity stays checked with one source more",
			list: List{ID: "courses", Items: []Item{
				{ID: "old", Ingredient: "sel", Checked: true, Sources: []Source{src("s0", "pates", "2026-10-05", "dinner", 4, nil)}},
			}, Received: []ServedDishID{"s0"}},
			served: []Served{served("s1", "pates", "2026-10-06", "dinner", 4, 4, pastaLines[1:])},
			items: []Item{{ID: "old", Ingredient: "sel", Checked: true, Sources: []Source{
				src("s0", "pates", "2026-10-05", "dinner", 4, nil), src("s1", "pates", "2026-10-06", "dinner", 4, nil),
			}}},
			received: []ServedDishID{"s0", "s1"},
			added:    1,
		},
		{
			name:   "item added by hand never merged",
			list:   List{ID: "courses", Items: []Item{{ID: "hand", Ingredient: "oignon", Quantity: q(3000, dish.Piece), Manual: true}}},
			served: []Served{served("s1", "chili", "2026-10-06", "dinner", 4, 4, chiliLines[1:])},
			items: []Item{
				{ID: "id1", Ingredient: "oignon", Quantity: q(1000, dish.Piece), Sources: []Source{with(tue(4), q(1000, dish.Piece))}},
				{ID: "hand", Ingredient: "oignon", Quantity: q(3000, dish.Piece), Manual: true},
			},
			received: []ServedDishID{"s1"},
			added:    1,
		},
		{
			name: "served dish already received ignored",
			list: List{ID: "courses", Items: []Item{
				{ID: "old", Ingredient: "bœuf haché", Quantity: q(500_000, dish.Gram), Sources: []Source{with(tue(4), q(500_000, dish.Gram))}},
			}, Received: []ServedDishID{"s1"}},
			served: []Served{
				served("s1", "chili", "2026-10-06", "dinner", 4, 4, chiliLines[:1]),
				served("s2", "chili", "2026-10-08", "lunch", 4, 4, chiliLines[:1]),
			},
			items: []Item{{ID: "old", Ingredient: "bœuf haché", Quantity: q(1_000_000, dish.Gram), Sources: []Source{
				with(tue(4), q(500_000, dish.Gram)), with(thu(4), q(500_000, dish.Gram)),
			}}},
			received: []ServedDishID{"s1", "s2"},
			added:    1,
			ignored:  1,
		},
		{
			name:     "served dish received whose items are gone: nothing again",
			list:     List{ID: "courses", Received: []ServedDishID{"s1"}},
			served:   []Served{served("s1", "chili", "2026-10-06", "dinner", 4, 4, chiliLines)},
			received: []ServedDishID{"s1"},
			ignored:  1,
		},
		{
			name: "301 items refused",
			list: func() List {
				l := List{ID: "courses"}
				for i := range 299 {
					l.Items = append(l.Items, Item{ID: ItemID(fmt.Sprint("hand", i)), Ingredient: fmt.Sprint("article ", i), Manual: true})
				}
				return l
			}(),
			served: []Served{served("s1", "chili", "2026-10-06", "dinner", 4, 4, chiliLines)},
			err:    ErrTooManyItems,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n := 0
			newID := func() (string, error) { n++; return fmt.Sprint("id", n), nil }
			before := fmt.Sprintf("%+v", c.list)
			m, err := addMealShopping(c.list, c.served, newID)
			if fmt.Sprintf("%+v", c.list) != before {
				t.Errorf("list changed in place: %+v", c.list)
			}
			if !errors.Is(err, c.err) {
				t.Fatalf("error %v, want %v", err, c.err)
			}
			if err != nil {
				return
			}
			if !reflect.DeepEqual(m.list.Items, c.items) && (len(m.list.Items) != 0 || len(c.items) != 0) {
				t.Errorf("items\n%s\nwant\n%s", dump(m.list.Items), dump(c.items))
			}
			if !reflect.DeepEqual(m.list.Received, c.received) {
				t.Errorf("received %v, want %v", m.list.Received, c.received)
			}
			if m.added != c.added || m.ignored != c.ignored || len(m.received) != c.added {
				t.Errorf("added %d (%d received), ignored %d; want %d, %d", m.added, len(m.received), m.ignored, c.added, c.ignored)
			}
		})
	}
}

// TestMealShoppingChanges checks what is to be written: new items with the key of their
// ingredient, items changed with the sources they gain only.
func TestMealShoppingChanges(t *testing.T) {
	list := List{ID: "courses", Items: []Item{
		{ID: "old", Ingredient: "Bœuf haché", Quantity: q(250_000, dish.Gram), Sources: []Source{src("s0", "chili", "2026-10-05", "dinner", 2, q(250_000, dish.Gram))}},
		{ID: "hand", Ingredient: "pain", Manual: true},
	}, Received: []ServedDishID{"s0"}}
	m, err := addMealShopping(list, []Served{served("s1", "chili", "2026-10-06", "dinner", 4, 4, chiliLines)}, func() (string, error) { return "new", nil })
	if err != nil {
		t.Fatal(err)
	}
	tue := src("s1", "chili", "2026-10-06", "dinner", 4, nil)
	beef, onion := tue, tue
	beef.Quantity, onion.Quantity = q(500_000, dish.Gram), q(1000, dish.Piece)
	want := []itemChange{
		{item: Item{ID: "old", Ingredient: "Bœuf haché", Quantity: q(750_000, dish.Gram), Sources: []Source{list.Items[0].Sources[0], beef}}, family: dish.Mass, sources: []Source{beef}},
		{item: Item{ID: "new", Ingredient: "oignon", Quantity: q(1000, dish.Piece), Sources: []Source{onion}}, isNew: true, key: 2, family: dish.PieceFamily, sources: []Source{onion}},
	}
	if !reflect.DeepEqual(m.items, want) {
		t.Errorf("changes\n%+v\nwant\n%+v", m.items, want)
	}
	if !reflect.DeepEqual(m.received, []Served{served("s1", "chili", "2026-10-06", "dinner", 4, 4, chiliLines)}) {
		t.Errorf("received %+v", m.received)
	}
}

func dump(items []Item) string {
	s := ""
	for _, it := range items {
		s += fmt.Sprintf("  %s %s %v checked=%v manual=%v\n", it.ID, it.Ingredient, it.Quantity, it.Checked, it.Manual)
		for _, src := range it.Sources {
			s += fmt.Sprintf("    %+v %v\n", src, src.Quantity)
		}
	}
	return s
}
