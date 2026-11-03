package dish

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/nelt/tribe-menus/internal/collection"
)

// Collections of the API served by the library (ADR 0029).
const (
	DishesCollection      = "dishes"
	IngredientsCollection = "ingredients"
)

// Sizes of the pages, fixed by the server (D2): a dish weighs up to 40 lines.
const (
	DishPageSize       = 50
	IngredientPageSize = 500
)

// ID identifies a dish: random, 128 bits in hexadecimal (D10), so that the identifier of a
// dish of another tribe never designates a dish of one's own.
type ID string

// Ingredient is an ingredient of the catalog: the name of its first entry, and its default
// unit, empty until a line gives it one (Q20, Q26).
type Ingredient struct {
	Name        string
	DefaultUnit Unit
}

// Line is a line of a dish: one per ingredient (Q27), with an optional quantity (Q26).
type Line struct {
	Ingredient Ingredient
	Quantity   *Quantity
}

// Dish is a dish of the library.
type Dish struct {
	ID       ID
	Name     string
	Color    Color
	Servings int
	// Revision increases on every change (Q25).
	Revision int64
	Lines    []Line
}

var (
	// ErrNotFound is returned for a dish absent from the library of the tribe, whether its
	// identifier exists in another tribe or nowhere (ENF-02).
	ErrNotFound = errors.New("dish not found")
)

// ChangedError refuses a save made on an outdated revision: another member changed the
// dish in between (Q25). Current is the dish as it is now.
type ChangedError struct {
	Current Dish
}

func (e *ChangedError) Error() string {
	return fmt.Sprintf("dish %s changed in between: revision %d", e.Current.ID, e.Current.Revision)
}

// Store reads and writes the dish library and the ingredient catalog of one tribe.
type Store interface {
	// Version reads the current version of a collection.
	Version(ctx context.Context, collection string) (int64, error)
	// Dishes reads the version of the dishes and at most limit dishes after the cursor, in
	// the order of their identifiers, in one read transaction.
	Dishes(ctx context.Context, after string, limit int) (int64, collection.Page[Dish], error)
	// Ingredients reads the version of the catalog and at most limit ingredients after the
	// cursor, in the order of their identifiers, in one read transaction; ErrBadCursor for a
	// cursor it did not give.
	Ingredients(ctx context.Context, after string, limit int) (int64, collection.Page[Ingredient], error)
	// Write runs write in a write transaction, and returns the versions of the dishes and of
	// the catalog after it.
	Write(ctx context.Context, write func(Tx) error) (collection.Versions, error)
}

// Tx reads and writes in a write transaction.
type Tx interface {
	TribeSize(ctx context.Context) (int, error)
	// Dish returns the dish, or ErrNotFound.
	Dish(ctx context.Context, id ID) (Dish, error)
	// DishNamed returns the dish with this normalized name, if any.
	DishNamed(ctx context.Context, normalized string) (ID, bool, error)
	// ColorCounts counts the dishes of each color.
	ColorCounts(ctx context.Context) (map[Color]int, error)
	// Ingredient returns the ingredient with this normalized name, if any, and its key.
	Ingredient(ctx context.Context, normalized string) (Ingredient, int64, bool, error)
	// AddIngredient adds an ingredient to the catalog and returns its key.
	AddIngredient(ctx context.Context, ingredient Ingredient, normalized string) (int64, error)
	// SetDefaultUnit gives an ingredient without default unit its default unit.
	SetDefaultUnit(ctx context.Context, key int64, unit Unit) error
	// InsertDish inserts a dish in revision 1, its lines designating their ingredients by key.
	InsertDish(ctx context.Context, d Dish, normalized string, keys []int64) error
	// UpdateDish saves the name, the servings and the lines of a dish, and increases its
	// revision.
	UpdateDish(ctx context.Context, d Dish, normalized string, keys []int64) error
}

// Library is the dish library and the ingredient catalog of a tribe (P1 to P4).
type Library struct {
	Store Store
	// NewID returns the identifier of a new dish; random by default.
	NewID func() (ID, error)
}

// NewLibrary returns the library of the tribe whose store is given.
func NewLibrary(store Store) *Library {
	return &Library{Store: store, NewID: randomID}
}

func randomID() (ID, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("dish identifier: %w", err)
	}
	return ID(hex.EncodeToString(b)), nil
}

// Version returns the current version of a collection of the library.
func (l *Library) Version(ctx context.Context, collection string) (int64, error) {
	return l.Store.Version(ctx, collection)
}

// Dishes returns a page of the library (D2), with the version of the dishes.
func (l *Library) Dishes(ctx context.Context, after string) (int64, collection.Page[Dish], error) {
	return l.Store.Dishes(ctx, after, DishPageSize)
}

// Catalog returns a page of the ingredient catalog (D2), with its version.
func (l *Library) Catalog(ctx context.Context, after string) (int64, collection.Page[Ingredient], error) {
	return l.Store.Ingredients(ctx, after, IngredientPageSize)
}

// Create adds a dish to the library (P1): servings left out are the tribe size (T3, D16),
// the color is the hue least used (Q19), the ingredients are found in the catalog by their
// normalized name or added to it (Q20). It returns the dish and the versions after the write.
func (l *Library) Create(ctx context.Context, in Input) (Dish, collection.Versions, error) {
	v, err := in.validate()
	if err != nil {
		return Dish{}, nil, err
	}
	id, err := l.NewID()
	if err != nil {
		return Dish{}, nil, err
	}
	var d Dish
	versions, err := l.Store.Write(ctx, func(tx Tx) error {
		if err := checkName(ctx, tx, v.normalized, ""); err != nil {
			return err
		}
		servings := v.servings
		if servings == 0 {
			size, err := tx.TribeSize(ctx)
			if err != nil {
				return err
			}
			servings = size
		}
		lines, keys, err := resolve(ctx, tx, v.lines)
		if err != nil {
			return err
		}
		counts, err := tx.ColorCounts(ctx)
		if err != nil {
			return err
		}
		d = Dish{ID: id, Name: v.name, Color: PickColor(counts), Servings: servings, Revision: 1, Lines: lines}
		return tx.InsertDish(ctx, d, v.normalized, keys)
	})
	if err != nil {
		return Dish{}, nil, fmt.Errorf("create dish: %w", err)
	}
	return d, versions, nil
}

// Update saves a change of a dish read in the revision given (P2): ErrNotFound for a dish
// absent from the library, *ChangedError if it changed since (Q25). The servings are
// required, the color does not change (Q19).
func (l *Library) Update(ctx context.Context, id ID, revision int64, in Input) (Dish, collection.Versions, error) {
	if in.Servings == 0 {
		return Dish{}, nil, dishError(InvalidServings)
	}
	v, err := in.validate()
	if err != nil {
		return Dish{}, nil, err
	}
	var d Dish
	versions, err := l.Store.Write(ctx, func(tx Tx) error {
		current, err := tx.Dish(ctx, id)
		if err != nil {
			return err
		}
		if current.Revision != revision {
			return &ChangedError{Current: current}
		}
		if err := checkName(ctx, tx, v.normalized, id); err != nil {
			return err
		}
		lines, keys, err := resolve(ctx, tx, v.lines)
		if err != nil {
			return err
		}
		d = Dish{ID: id, Name: v.name, Color: current.Color, Servings: v.servings, Revision: current.Revision + 1, Lines: lines}
		return tx.UpdateDish(ctx, d, v.normalized, keys)
	})
	if err != nil {
		return Dish{}, nil, fmt.Errorf("update dish %s: %w", id, err)
	}
	return d, versions, nil
}

// checkName refuses a name already taken by another dish than self (Q28).
func checkName(ctx context.Context, tx Tx, normalized string, self ID) error {
	other, found, err := tx.DishNamed(ctx, normalized)
	if err != nil {
		return err
	}
	if found && other != self {
		return dishError(NameTaken)
	}
	return nil
}

// resolve finds the ingredient of each line in the catalog by its normalized name, or adds
// it with the unit of the line as default unit; an ingredient without default unit gets the
// unit of the line, if it has one (Q20, D4).
func resolve(ctx context.Context, tx Tx, valid []validLine) ([]Line, []int64, error) {
	lines := make([]Line, 0, len(valid))
	keys := make([]int64, 0, len(valid))
	for _, v := range valid {
		var unit Unit
		if v.quantity != nil {
			unit = v.quantity.Unit
		}
		ingredient, key, found, err := tx.Ingredient(ctx, v.normalized)
		if err != nil {
			return nil, nil, err
		}
		switch {
		case !found:
			ingredient = Ingredient{Name: v.name, DefaultUnit: unit}
			if key, err = tx.AddIngredient(ctx, ingredient, v.normalized); err != nil {
				return nil, nil, err
			}
		case ingredient.DefaultUnit == "" && unit != "":
			if err := tx.SetDefaultUnit(ctx, key, unit); err != nil {
				return nil, nil, err
			}
			ingredient.DefaultUnit = unit
		}
		lines = append(lines, Line{Ingredient: ingredient, Quantity: v.quantity})
		keys = append(keys, key)
	}
	return lines, keys, nil
}
