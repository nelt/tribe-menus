package shopping

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/nelt/tribe-menus/internal/dish"
)

func TestParseListName(t *testing.T) {
	cases := []struct {
		name, in, want string
		err            error
	}{
		{"kept", "Marché", "Marché", nil},
		{"spaces reduced", "  Marché   du  samedi ", "Marché du samedi", nil},
		{"40 characters", strings.Repeat("é", 40), strings.Repeat("é", 40), nil},
		{"41 characters", strings.Repeat("é", 41), "", ErrListNameTooLong},
		{"empty", "", "", ErrListNameEmpty},
		{"spaces only", "   ", "", ErrListNameEmpty},
		{"zero width space", "​", "", ErrListNameEmpty},
		{"control character", "Marché\a", "", ErrListNameInvalid},
		{"emoji with joiner", "🧑‍🍳 Drive", "🧑‍🍳 Drive", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseListName(c.in)
			if got != c.want || !errors.Is(err, c.err) {
				t.Errorf("ParseListName(%q) = %q, %v; want %q, %v", c.in, got, err, c.want, c.err)
			}
		})
	}
}

// TestSortItems: the example of « Ordre des articles » (Q40, D13), shuffled.
func TestSortItems(t *testing.T) {
	q := func(n int64) *dish.Quantity { return &dish.Quantity{Amount: n * 1000, Unit: dish.Piece} }
	want := []Item{
		{ID: "e", Ingredient: "Échalote", Quantity: q(4), Manual: true},
		{ID: "l", Ingredient: "lessive", Manual: true},
		{ID: "o2", Ingredient: "oignon", Quantity: q(1)},
		{ID: "o3", Ingredient: "Oignon", Quantity: q(2)},
		{ID: "o1", Ingredient: "oignon", Quantity: q(3), Manual: true},
		{ID: "o4", Ingredient: "oignon"},
		{ID: "o5", Ingredient: "oignon", Manual: true},
		{ID: "a", Ingredient: "ail", Quantity: q(1), Checked: true, Manual: true},
		{ID: "p", Ingredient: "pain", Quantity: q(2), Checked: true, Manual: true},
	}
	items := slices.Clone(want)
	slices.Reverse(items)
	items[0], items[4] = items[4], items[0]
	SortItems(items)
	if !slices.EqualFunc(items, want, func(a, b Item) bool { return a.ID == b.ID }) {
		var ids []ItemID
		for _, it := range items {
			ids = append(ids, it.ID)
		}
		t.Errorf("order %v", ids)
	}
}
