// Package shopping holds the shopping lists of a tribe (C1 to C6, C11, C12): pure rules, and
// the Lists behind a Store interface (ADR 0002). It reads the units, the quantities and the
// normalized names of internal/dish.
package shopping

import (
	"cmp"
	"errors"
	"slices"

	"github.com/nelt/tribe-menus/internal/dish"
)

// Limits (Q38, D12).
const (
	MaxListNameLength = 40
	MaxLists          = 20
	MaxItems          = 300
)

var (
	// ErrListNameEmpty is returned for a name without visible character.
	ErrListNameEmpty = errors.New("list name empty")
	// ErrListNameInvalid is returned for a name with a control character.
	ErrListNameInvalid = errors.New("list name invalid")
	// ErrListNameTooLong is returned for a name of more than 40 characters.
	ErrListNameTooLong = errors.New("list name too long")
	// ErrTooManyLists is returned for a 21st list of the tribe.
	ErrTooManyLists = errors.New("too many lists")
	// ErrTooManyItems is returned for a 301st item of a list.
	ErrTooManyItems = errors.New("too many items")
)

var listNameErrors = map[dish.NameFault]error{
	dish.NameEmpty:   ErrListNameEmpty,
	dish.NameControl: ErrListNameInvalid,
	dish.NameLong:    ErrListNameTooLong,
}

// ParseListName returns the name of a list as entered, its spaces reduced, if it follows the
// rules of the names (Q38, D12): two lists may have the same name (Q17).
func ParseListName(name string) (string, error) {
	name, fault := dish.CleanName(name, MaxListNameLength)
	if fault != dish.NameOK {
		return "", listNameErrors[fault]
	}
	return name, nil
}

// SortItems sorts the items of a list (Q40, D13): unchecked first, checked at the bottom;
// in each group by the normalized name of their ingredient (Q29), computed and manual items
// mixed; for the same ingredient, with quantity before without, computed before manual; then
// by identifier, for a stable order. Same rule as sortItems in web/src/shopping.ts.
func SortItems(items []Item) {
	slices.SortStableFunc(items, func(a, b Item) int {
		return cmp.Or(
			cmpBool(a.Checked, b.Checked),
			cmp.Compare(dish.NormalizeName(a.Ingredient), dish.NormalizeName(b.Ingredient)),
			cmpBool(a.Quantity == nil, b.Quantity == nil),
			cmpBool(a.Manual, b.Manual),
			cmp.Compare(a.ID, b.ID),
		)
	})
}

// cmpBool orders false before true.
func cmpBool(a, b bool) int {
	switch {
	case a == b:
		return 0
	case a:
		return 1
	}
	return -1
}
