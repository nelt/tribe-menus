package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/nelt/tribe-menus/internal/collection"
	"github.com/nelt/tribe-menus/internal/dish"
	"github.com/nelt/tribe-menus/internal/mealplan"
	"github.com/nelt/tribe-menus/internal/storage/tribedb"
)

// MealPlanStore is the store of the meal plan of one tribe: a thin adapter around tribedb.
type MealPlanStore struct {
	db *sql.DB
}

var _ mealplan.Store = (*MealPlanStore)(nil)

// NewMealPlanStore returns the meal plan store of the tribe whose database is db.
func NewMealPlanStore(db *sql.DB) *MealPlanStore {
	return &MealPlanStore{db: db}
}

// Version reads the current version of the meal plan.
func (s *MealPlanStore) Version(ctx context.Context) (int64, error) {
	return collectionVersion(ctx, s.db, mealplan.Collection)
}

// ServedDishes reads a page of served dishes and the version of the meal plan, in one
// transaction. It reads one served dish more than the limit, to know whether a next page
// exists.
func (s *MealPlanStore) ServedDishes(ctx context.Context, after string, limit int) (int64, collection.Page[mealplan.ServedDish], error) {
	var page collection.Page[mealplan.ServedDish]
	version, err := readCollection(ctx, s.db, mealplan.Collection, func(q *tribedb.Queries) error {
		rows, err := q.ServedDishesAfter(ctx, tribedb.ServedDishesAfterParams{ID: after, Limit: int64(limit) + 1})
		if err != nil {
			return fmt.Errorf("read served dishes: %w", err)
		}
		if len(rows) > limit {
			rows = rows[:limit]
			page.Next = rows[limit-1].ID
		}
		for _, r := range rows {
			page.Items = append(page.Items, servedDishFromRow(r))
		}
		return nil
	})
	if err != nil {
		return 0, collection.Page[mealplan.ServedDish]{}, err
	}
	return version, page, nil
}

// Write runs write in a write transaction; the version of the meal plan increases by one
// if it changed it.
func (s *MealPlanStore) Write(ctx context.Context, write func(mealplan.Tx) error) (collection.Versions, error) {
	return writeCollections(ctx, s.db, []string{mealplan.Collection}, func(q *tribedb.Queries) error {
		return write(mealPlanTx{q: q})
	})
}

func servedDishFromRow(r tribedb.ServedDish) mealplan.ServedDish {
	return mealplan.ServedDish{
		ID: mealplan.ID(r.ID), Date: mealplan.Date(r.Date), Mealtime: mealplan.Mealtime(r.Mealtime),
		Dish: dish.ID(r.DishID), Servings: int(r.Servings), Rank: r.Rank,
	}
}

type mealPlanTx struct {
	q *tribedb.Queries
}

func (t mealPlanTx) DishExists(ctx context.Context, id dish.ID) (bool, error) {
	exists, err := t.q.DishExists(ctx, string(id))
	if err != nil {
		return false, fmt.Errorf("look up dish: %w", err)
	}
	return exists, nil
}

func (t mealPlanTx) TribeSize(ctx context.Context) (int, error) {
	size, err := t.q.TribeSize(ctx)
	if err != nil {
		return 0, fmt.Errorf("tribe size: %w", err)
	}
	return int(size), nil
}

func (t mealPlanTx) Meal(ctx context.Context, date mealplan.Date, mealtime mealplan.Mealtime) ([]mealplan.ServedDish, error) {
	rows, err := t.q.MealServedDishes(ctx, tribedb.MealServedDishesParams{Date: string(date), Mealtime: string(mealtime)})
	if err != nil {
		return nil, fmt.Errorf("read meal: %w", err)
	}
	meal := make([]mealplan.ServedDish, 0, len(rows))
	for _, r := range rows {
		meal = append(meal, servedDishFromRow(r))
	}
	return meal, nil
}

func (t mealPlanTx) NextRank(ctx context.Context) (int64, error) {
	rank, err := t.q.NextServedDishRank(ctx)
	if err != nil {
		return 0, fmt.Errorf("next rank: %w", err)
	}
	return rank, nil
}

func (t mealPlanTx) Insert(ctx context.Context, s mealplan.ServedDish) error {
	if err := t.q.InsertServedDish(ctx, tribedb.InsertServedDishParams{
		ID: string(s.ID), Date: string(s.Date), Mealtime: string(s.Mealtime), DishID: string(s.Dish),
		Servings: int64(s.Servings), Rank: s.Rank,
	}); err != nil {
		return fmt.Errorf("insert served dish: %w", err)
	}
	return nil
}

func (t mealPlanTx) SetServings(ctx context.Context, id mealplan.ID, servings int) (mealplan.ServedDish, error) {
	r, err := t.q.SetServedDishServings(ctx, tribedb.SetServedDishServingsParams{Servings: int64(servings), ID: string(id)})
	if errors.Is(err, sql.ErrNoRows) {
		return mealplan.ServedDish{}, mealplan.ErrNotFound
	}
	if err != nil {
		return mealplan.ServedDish{}, fmt.Errorf("set servings: %w", err)
	}
	return servedDishFromRow(r), nil
}

func (t mealPlanTx) Delete(ctx context.Context, id mealplan.ID) error {
	n, err := t.q.DeleteServedDish(ctx, string(id))
	if err != nil {
		return fmt.Errorf("delete served dish: %w", err)
	}
	if n == 0 {
		return mealplan.ErrNotFound
	}
	return nil
}
