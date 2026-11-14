package shopping

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"

	"github.com/nelt/tribe-menus/internal/collection"
	"github.com/nelt/tribe-menus/internal/dish"
)

// Collection is the collection of the API served by the shopping lists (ADR 0029, D4): a list
// carries its items, their sources and its received served dishes.
const Collection = "shopping_lists"

// PageSize is the size of a page of the shopping lists, fixed by the server (D4).
const PageSize = 20

var (
	// ErrNotFound is returned for a list or an item absent from the tribe: deleted in
	// between, unknown, or of another tribe (D11, ENF-02).
	ErrNotFound = errors.New("not found")
	// ErrMainList refuses to delete the main list (C12), maybe become main in between (D11).
	ErrMainList = errors.New("main list")
	// ErrComputedItem refuses to remove a computed item by hand: it leaves by the shopping
	// done (C11).
	ErrComputedItem = errors.New("computed item")
)

// NotEmptyError refuses to delete a list that holds items (C12), maybe added in between
// (D11). List is the list as it is now.
type NotEmptyError struct {
	List List
}

func (e *NotEmptyError) Error() string {
	return fmt.Sprintf("list %s holds %d items", e.List.ID, len(e.List.Items))
}

// ItemError refuses an item entered by hand, with the code of the rule of a line of a dish it
// breaks (Q13, Q31).
type ItemError struct {
	Code dish.ErrorCode
}

func (e *ItemError) Error() string {
	return fmt.Sprintf("item refused: %s", e.Code)
}

// Store reads and writes the shopping lists of one tribe.
type Store interface {
	// Version reads the current version of the shopping lists.
	Version(ctx context.Context) (int64, error)
	// Lists reads the version of the shopping lists and at most limit lists after the cursor,
	// in the order of their identifiers, in one read transaction.
	Lists(ctx context.Context, after string, limit int) (int64, collection.Page[List], error)
	// Write runs write in a write transaction, and returns the versions of the shopping lists
	// and of the ingredient catalog after it.
	Write(ctx context.Context, write func(Tx) error) (collection.Versions, error)
}

// Tx reads and writes in a write transaction.
type Tx interface {
	// Catalog finds and adds the ingredients of the items added by hand (Q13).
	dish.Catalog
	// List returns the list with its items, or ErrNotFound.
	List(ctx context.Context, id ListID) (List, error)
	ListCount(ctx context.Context) (int, error)
	// NextRank returns a rank greater than that of every list.
	NextRank(ctx context.Context) (int64, error)
	InsertList(ctx context.Context, l List) error
	RenameList(ctx context.Context, id ListID, name string) error
	// SetMain makes the list the main one, and the main one an ordinary list.
	SetMain(ctx context.Context, id ListID) error
	// DeleteList deletes a list without items, and its received served dishes.
	DeleteList(ctx context.Context, id ListID) error
	// InsertItem adds an item to a list, its ingredient designated by its key.
	InsertItem(ctx context.Context, list ListID, item Item, ingredient int64) error
	// DeleteItem deletes an item and its sources.
	DeleteItem(ctx context.Context, id ItemID) error
	// ServedDishes returns the served dishes of the period, ordered by meal, lunch before
	// dinner, then by rank, each with its dish as it is now (P2).
	ServedDishes(ctx context.Context, p Period) ([]Served, error)
	// UpdateItem writes the quantity, the taken quantity and the check of an item.
	UpdateItem(ctx context.Context, item Item) error
	// InsertSource adds a source to a computed item.
	InsertSource(ctx context.Context, item ItemID, s Source) error
	// InsertReceived records a served dish received by a list, with the servings of
	// reference and the lines of its dish at the time of the computation (D5).
	InsertReceived(ctx context.Context, list ListID, s Served) error
}

// Lists are the shopping lists of a tribe.
type Lists struct {
	Store Store
	// NewID returns a new identifier of a list or an item; random by default.
	NewID func() (string, error)
}

// NewLists returns the shopping lists of the tribe whose store is given.
func NewLists(store Store) *Lists {
	return &Lists{Store: store, NewID: randomID}
}

func randomID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("shopping identifier: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// Version returns the current version of the shopping lists.
func (l *Lists) Version(ctx context.Context) (int64, error) {
	return l.Store.Version(ctx)
}

// Page returns a page of the shopping lists (D4), with their version.
func (l *Lists) Page(ctx context.Context, after string) (int64, collection.Page[List], error) {
	return l.Store.Lists(ctx, after, PageSize)
}

// Create creates an empty list (C2), which is not the main one: ErrTooManyLists beyond 20
// lists (D12).
func (l *Lists) Create(ctx context.Context, name string) (List, collection.Versions, error) {
	name, err := ParseListName(name)
	if err != nil {
		return List{}, nil, err
	}
	id, err := l.NewID()
	if err != nil {
		return List{}, nil, err
	}
	var list List
	versions, err := l.Store.Write(ctx, func(tx Tx) error {
		n, err := tx.ListCount(ctx)
		if err != nil {
			return err
		}
		if n >= MaxLists {
			return ErrTooManyLists
		}
		rank, err := tx.NextRank(ctx)
		if err != nil {
			return err
		}
		list = List{ID: ListID(id), Name: name, Rank: rank}
		return tx.InsertList(ctx, list)
	})
	if err != nil {
		return List{}, nil, fmt.Errorf("create list: %w", err)
	}
	return list, versions, nil
}

// Rename renames a list (C2): the last save wins (D11).
func (l *Lists) Rename(ctx context.Context, id ListID, name string) (List, collection.Versions, error) {
	name, err := ParseListName(name)
	if err != nil {
		return List{}, nil, err
	}
	var list List
	versions, err := l.Store.Write(ctx, func(tx Tx) error {
		var err error
		if list, err = tx.List(ctx, id); err != nil {
			return err
		}
		list.Name = name
		return tx.RenameList(ctx, id, name)
	})
	if err != nil {
		return List{}, nil, fmt.Errorf("rename list %s: %w", id, err)
	}
	return list, versions, nil
}

// SetMain makes a list the main one (C1), the former main list an ordinary one: the last
// save wins (D11). Nothing is written for the main list itself.
func (l *Lists) SetMain(ctx context.Context, id ListID) (collection.Versions, error) {
	versions, err := l.Store.Write(ctx, func(tx Tx) error {
		list, err := tx.List(ctx, id)
		if err != nil || list.Main {
			return err
		}
		return tx.SetMain(ctx, id)
	})
	if err != nil {
		return nil, fmt.Errorf("set main list %s: %w", id, err)
	}
	return versions, nil
}

// Delete deletes an empty list (C12): ErrMainList for the main list, *NotEmptyError for a
// list that holds items, maybe since in between (D11).
func (l *Lists) Delete(ctx context.Context, id ListID) (collection.Versions, error) {
	versions, err := l.Store.Write(ctx, func(tx Tx) error {
		list, err := tx.List(ctx, id)
		switch {
		case err != nil:
			return err
		case list.Main:
			return ErrMainList
		case len(list.Items) > 0:
			return &NotEmptyError{List: list}
		}
		return tx.DeleteList(ctx, id)
	})
	if err != nil {
		return nil, fmt.Errorf("delete list %s: %w", id, err)
	}
	return versions, nil
}

// AddItem adds an item by hand to a list (C3): its ingredient found in the catalog by its
// normalized name or added to it, with an optional quantity and unit (Q13, Q20, Q31); never
// merged with another item (D23). ErrTooManyItems beyond 300 items (D12).
func (l *Lists) AddItem(ctx context.Context, id ListID, in dish.LineInput) (List, collection.Versions, error) {
	line, code := dish.ParseLine(in)
	if code != "" {
		return List{}, nil, &ItemError{Code: code}
	}
	itemID, err := l.NewID()
	if err != nil {
		return List{}, nil, err
	}
	var list List
	versions, err := l.Store.Write(ctx, func(tx Tx) error {
		var err error
		if list, err = tx.List(ctx, id); err != nil {
			return err
		}
		if len(list.Items) >= MaxItems {
			return ErrTooManyItems
		}
		ingredient, key, err := dish.ResolveIngredient(ctx, tx, line)
		if err != nil {
			return err
		}
		item := Item{ID: ItemID(itemID), Ingredient: ingredient.Name, Quantity: line.Quantity, Manual: true}
		list.Items = append(list.Items, item)
		return tx.InsertItem(ctx, id, item, key)
	})
	if err != nil {
		return List{}, nil, fmt.Errorf("add item to list %s: %w", id, err)
	}
	SortItems(list.Items)
	return list, versions, nil
}

// RemoveItem removes an item added by hand (C3): ErrComputedItem for a computed item,
// ErrNotFound for an item absent from the list, maybe removed in between, which the screen
// takes as done (D11).
func (l *Lists) RemoveItem(ctx context.Context, id ListID, itemID ItemID) (List, collection.Versions, error) {
	var list List
	versions, err := l.Store.Write(ctx, func(tx Tx) error {
		var err error
		if list, err = tx.List(ctx, id); err != nil {
			return err
		}
		i := slices.IndexFunc(list.Items, func(it Item) bool { return it.ID == itemID })
		switch {
		case i < 0:
			return ErrNotFound
		case !list.Items[i].Manual:
			return ErrComputedItem
		}
		list.Items = slices.Delete(list.Items, i, i+1)
		return tx.DeleteItem(ctx, itemID)
	})
	if err != nil {
		return List{}, nil, fmt.Errorf("remove item %s: %w", itemID, err)
	}
	return list, versions, nil
}

// SetChecked checks or unchecks an item (C6): the last save wins (D8); ErrNotFound for an
// item absent from the list, maybe gone in between. Checking an item erases its taken
// quantity: everything is taken; unchecking a checked item gives it none (Q41, D9).
func (l *Lists) SetChecked(ctx context.Context, id ListID, itemID ItemID, checked bool) (List, collection.Versions, error) {
	var list List
	versions, err := l.Store.Write(ctx, func(tx Tx) error {
		var err error
		if list, err = tx.List(ctx, id); err != nil {
			return err
		}
		i := slices.IndexFunc(list.Items, func(it Item) bool { return it.ID == itemID })
		if i < 0 {
			return ErrNotFound
		}
		list.Items[i].Checked = checked
		if checked {
			list.Items[i].Taken = nil
		}
		return tx.UpdateItem(ctx, list.Items[i])
	})
	if err != nil {
		return List{}, nil, fmt.Errorf("check item %s: %w", itemID, err)
	}
	SortItems(list.Items)
	return list, versions, nil
}

// AddMealShopping adds the shopping of the meals of a period to a list (C4, P2): the served
// dishes of the period, with their dishes as they are now, read in the write transaction, so
// that two members adding at the same time do not pour the same served dishes twice (Q22,
// D6). It returns the list, and the number of served dishes added and ignored, already
// received by the list. ErrTooManyItems beyond 300 items, and nothing is written (D12).
func (l *Lists) AddMealShopping(ctx context.Context, id ListID, from, to string) (List, int, int, collection.Versions, error) {
	period, err := ParsePeriod(from, to)
	if err != nil {
		return List{}, 0, 0, nil, err
	}
	var m mealShopping
	versions, err := l.Store.Write(ctx, func(tx Tx) error {
		list, err := tx.List(ctx, id)
		if err != nil {
			return err
		}
		served, err := tx.ServedDishes(ctx, period)
		if err != nil {
			return err
		}
		if m, err = addMealShopping(list, served, l.NewID); err != nil {
			return err
		}
		for _, c := range m.items {
			if c.isNew {
				item := c.item
				item.Sources = nil
				err = tx.InsertItem(ctx, id, item, c.key)
			} else {
				err = tx.UpdateItem(ctx, c.item)
			}
			if err != nil {
				return err
			}
			for _, s := range c.sources {
				if err := tx.InsertSource(ctx, c.item.ID, s); err != nil {
					return err
				}
			}
		}
		for _, s := range m.received {
			if err := tx.InsertReceived(ctx, id, s); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return List{}, 0, 0, nil, fmt.Errorf("add meal shopping to list %s: %w", id, err)
	}
	return m.list, m.added, m.ignored, versions, nil
}

// CheckOut declares the shopping done (C11): the items the member saw checked leave the
// list if they still are; an item checked or unchecked in between stays (D9). The planning
// is not touched, and the received served dishes stay (Q22). It returns the list and the
// number of items removed.
func (l *Lists) CheckOut(ctx context.Context, id ListID, seen []ItemID) (List, int, collection.Versions, error) {
	var list List
	removed := 0
	versions, err := l.Store.Write(ctx, func(tx Tx) error {
		var err error
		if list, err = tx.List(ctx, id); err != nil {
			return err
		}
		var kept []Item
		for _, it := range list.Items {
			if !it.Checked || !slices.Contains(seen, it.ID) {
				kept = append(kept, it)
				continue
			}
			if err := tx.DeleteItem(ctx, it.ID); err != nil {
				return err
			}
			removed++
		}
		list.Items = kept
		return nil
	})
	if err != nil {
		return List{}, 0, nil, fmt.Errorf("check out list %s: %w", id, err)
	}
	return list, removed, versions, nil
}
