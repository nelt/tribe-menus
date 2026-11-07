package shopping

import (
	"encoding/json"
	"errors"
	"os"
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
		{"spaces reduced", "  Marché \u00a0 du  samedi ", "Marché du samedi", nil},
		{"40 characters", strings.Repeat("é", 40), strings.Repeat("é", 40), nil},
		{"41 characters", strings.Repeat("é", 41), "", ErrListNameTooLong},
		{"empty", "", "", ErrListNameEmpty},
		{"spaces only", "   ", "", ErrListNameEmpty},
		{"zero width space", "\u200b", "", ErrListNameEmpty},
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

// TestSortItems: testdata/sort-items.json, shared with web/src/shopping.ts (Q40, D13; review
// of PR #98, point 3).
func TestSortItems(t *testing.T) {
	data, err := os.ReadFile("testdata/sort-items.json")
	if err != nil {
		t.Fatal(err)
	}
	var table struct {
		Cases []struct {
			Name  string `json:"name"`
			Items []struct {
				ID         ItemID    `json:"id"`
				Ingredient string    `json:"ingredient"`
				Quantity   string    `json:"quantity"`
				Unit       dish.Unit `json:"unit"`
				Checked    bool      `json:"checked"`
				Manual     bool      `json:"manual"`
			} `json:"items"`
			Order []ItemID `json:"order"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &table); err != nil {
		t.Fatal(err)
	}
	if len(table.Cases) == 0 {
		t.Fatal("no case")
	}
	for _, c := range table.Cases {
		t.Run(c.Name, func(t *testing.T) {
			var items []Item
			for _, it := range c.Items {
				item := Item{ID: it.ID, Ingredient: it.Ingredient, Checked: it.Checked, Manual: it.Manual}
				if it.Quantity != "" {
					q := mustQuantity(t, it.Quantity, it.Unit)
					item.Quantity = &q
				}
				items = append(items, item)
			}
			SortItems(items)
			var ids []ItemID
			for _, it := range items {
				ids = append(ids, it.ID)
			}
			if !slices.Equal(ids, c.Order) {
				t.Errorf("order %v, want %v", ids, c.Order)
			}
		})
	}
}
