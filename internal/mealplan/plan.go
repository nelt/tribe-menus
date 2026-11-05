package mealplan

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

// Collection is the collection of the API served by the meal plan (ADR 0029, D1).
const Collection = "meal_plan"

// PageSize is the size of a page of the meal plan, fixed by the server (D1).
const PageSize = 200

// ID identifies a served dish: random, 128 bits in hexadecimal (D10), so that the
// identifier of a served dish of another tribe never designates one of one's own.
type ID string

// ServedDish is a dish of the library served in a meal (R2): its servings are its own (Q8),
// its rank orders the dishes of the meal (D11).
type ServedDish struct {
	ID       ID
	Date     Date
	Mealtime Mealtime
	Dish     dish.ID
	Servings int
	Rank     int64
}

var (
	// ErrUnknownDish is returned for a dish absent from the library of the tribe, whether
	// its identifier exists in another tribe or nowhere (ENF-02).
	ErrUnknownDish = errors.New("unknown dish")
	// ErrNotFound is returned for a served dish absent from the meal plan of the tribe:
	// removed in between, unknown, or of another tribe (D3, D19, ENF-02).
	ErrNotFound = errors.New("served dish not found")
)

// AlreadyServedError refuses to serve a dish already served in the meal (Q21), maybe by
// another member in between (D3). Meal is the dishes of the meal as they are now.
type AlreadyServedError struct {
	Meal []ServedDish
}

func (e *AlreadyServedError) Error() string {
	return "dish already served in the meal"
}

// Store reads and writes the meal plan of one tribe.
type Store interface {
	// Version reads the current version of the meal plan.
	Version(ctx context.Context) (int64, error)
	// ServedDishes reads the version of the meal plan and at most limit served dishes after
	// the cursor, in the order of their identifiers, in one read transaction.
	ServedDishes(ctx context.Context, after string, limit int) (int64, collection.Page[ServedDish], error)
	// Write runs write in a write transaction, and returns the version of the meal plan
	// after it.
	Write(ctx context.Context, write func(Tx) error) (collection.Versions, error)
}

// Tx reads and writes in a write transaction.
type Tx interface {
	// DishExists tells whether the library of the tribe holds the dish.
	DishExists(ctx context.Context, id dish.ID) (bool, error)
	TribeSize(ctx context.Context) (int, error)
	// Meal returns the dishes served in the meal, by rank.
	Meal(ctx context.Context, date Date, mealtime Mealtime) ([]ServedDish, error)
	// NextRank returns a rank greater than that of every served dish.
	NextRank(ctx context.Context) (int64, error)
	Insert(ctx context.Context, s ServedDish) error
	// SetServings sets the servings of a served dish and returns it, or ErrNotFound.
	SetServings(ctx context.Context, id ID, servings int) (ServedDish, error)
	// Delete removes a served dish, or returns ErrNotFound.
	Delete(ctx context.Context, id ID) error
}

// Plan is the meal plan of a tribe (R1 to R5).
type Plan struct {
	Store Store
	// NewID returns the identifier of a new served dish; random by default.
	NewID func() (ID, error)
}

// NewPlan returns the meal plan of the tribe whose store is given.
func NewPlan(store Store) *Plan {
	return &Plan{Store: store, NewID: randomID}
}

func randomID() (ID, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("served dish identifier: %w", err)
	}
	return ID(hex.EncodeToString(b)), nil
}

// Version returns the current version of the meal plan.
func (p *Plan) Version(ctx context.Context) (int64, error) {
	return p.Store.Version(ctx)
}

// ServedDishes returns a page of the meal plan (D1), with its version.
func (p *Plan) ServedDishes(ctx context.Context, after string) (int64, collection.Page[ServedDish], error) {
	return p.Store.ServedDishes(ctx, after, PageSize)
}

// ServeInput is a dish to serve in a meal, as received. Servings left out (nil) are the
// tribe size at the time of the addition (D9).
type ServeInput struct {
	Date     string
	Mealtime string
	Dish     dish.ID
	Servings *int
}

// Serve serves a dish of the library in a meal (R2), past or future (R5), after the dishes
// already served in it (D11): ErrUnknownDish for a dish absent from the library,
// *AlreadyServedError for a dish already in the meal (Q21, D3). It returns the served dish
// and the version of the meal plan after the write.
func (p *Plan) Serve(ctx context.Context, in ServeInput) (ServedDish, collection.Versions, error) {
	date, err := ParseDate(in.Date)
	if err != nil {
		return ServedDish{}, nil, err
	}
	mealtime, err := ParseMealtime(in.Mealtime)
	if err != nil {
		return ServedDish{}, nil, err
	}
	if in.Servings != nil {
		if err := CheckServings(*in.Servings); err != nil {
			return ServedDish{}, nil, err
		}
	}
	id, err := p.NewID()
	if err != nil {
		return ServedDish{}, nil, err
	}
	var s ServedDish
	versions, err := p.Store.Write(ctx, func(tx Tx) error {
		exists, err := tx.DishExists(ctx, in.Dish)
		if err != nil {
			return err
		}
		if !exists {
			return ErrUnknownDish
		}
		meal, err := tx.Meal(ctx, date, mealtime)
		if err != nil {
			return err
		}
		if slices.ContainsFunc(meal, func(m ServedDish) bool { return m.Dish == in.Dish }) {
			return &AlreadyServedError{Meal: meal}
		}
		var servings int
		if in.Servings != nil {
			servings = *in.Servings
		} else if servings, err = tx.TribeSize(ctx); err != nil {
			return err
		}
		rank, err := tx.NextRank(ctx)
		if err != nil {
			return err
		}
		s = ServedDish{ID: id, Date: date, Mealtime: mealtime, Dish: in.Dish, Servings: servings, Rank: rank}
		return tx.Insert(ctx, s)
	})
	if err != nil {
		return ServedDish{}, nil, fmt.Errorf("serve dish %s: %w", in.Dish, err)
	}
	return s, versions, nil
}

// SetServings sets the servings of a served dish (R3): the last save wins (D3);
// ErrNotFound for a served dish absent from the meal plan.
func (p *Plan) SetServings(ctx context.Context, id ID, servings int) (ServedDish, collection.Versions, error) {
	if err := CheckServings(servings); err != nil {
		return ServedDish{}, nil, err
	}
	var s ServedDish
	versions, err := p.Store.Write(ctx, func(tx Tx) error {
		var err error
		s, err = tx.SetServings(ctx, id, servings)
		return err
	})
	if err != nil {
		return ServedDish{}, nil, fmt.Errorf("set servings of %s: %w", id, err)
	}
	return s, versions, nil
}

// Remove removes a served dish from its meal (R4); the dish stays in the library.
// ErrNotFound for a served dish absent from the meal plan (D19).
func (p *Plan) Remove(ctx context.Context, id ID) (collection.Versions, error) {
	versions, err := p.Store.Write(ctx, func(tx Tx) error {
		return tx.Delete(ctx, id)
	})
	if err != nil {
		return nil, fmt.Errorf("remove %s: %w", id, err)
	}
	return versions, nil
}
