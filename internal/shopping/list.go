package shopping

import "github.com/nelt/tribe-menus/internal/dish"

// ListID identifies a shopping list, ItemID an item: random, 128 bits in hexadecimal (D10),
// so that the identifier of another tribe never designates one of one's own.
type (
	ListID string
	ItemID string
)

// ServedDishID identifies a served dish of the meal plan, kept by a source or a received
// served dish without foreign key: the served dish may be removed since (D5).
type ServedDishID string

// List is a shopping list (C1, C2): its items in the order of SortItems (Q40), and the
// identifiers of the served dishes it received (Q22).
type List struct {
	ID   ListID
	Name string
	// Main is true for the main list of the tribe, exactly one (Q23).
	Main bool
	// Rank is the rank of creation, which orders the lists after the main one (C2).
	Rank     int64
	Items    []Item
	Received []ServedDishID
}

// Item is an item of a list: an ingredient of the catalog, by its name, and an optional
// quantity (Q36). A manual item is entered by hand (C3), a computed item comes from the meal
// plan and has at least one source (C4, C5).
type Item struct {
	ID         ItemID
	Ingredient string
	Quantity   *dish.Quantity
	Checked    bool
	Manual     bool
	Sources    []Source
}

// Source is what a served dish brought to a computed item, at the time of the computation
// (C5, D5): the quantity is absent for an ingredient without quantity (D3).
type Source struct {
	ServedDish ServedDishID
	Dish       dish.ID
	Date       string
	Mealtime   string
	Servings   int
	Quantity   *dish.Quantity
}
