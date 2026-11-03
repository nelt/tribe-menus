package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/nelt/tribe-menus/internal/collection"
	"github.com/nelt/tribe-menus/internal/dish"
	"github.com/nelt/tribe-menus/internal/storage/tribedb"
)

// DishStore is the store of the dish library of one tribe: a thin adapter around tribedb.
type DishStore struct {
	db *sql.DB
}

var _ dish.Store = (*DishStore)(nil)

// NewDishStore returns the dish store of the tribe whose database is db.
func NewDishStore(db *sql.DB) *DishStore {
	return &DishStore{db: db}
}

// Version reads the current version of a collection.
func (s *DishStore) Version(ctx context.Context, name string) (int64, error) {
	return collectionVersion(ctx, s.db, name)
}

// Dishes reads a page of dishes and the version of the collection, in one transaction.
// It reads one dish more than the limit, to know whether a next page exists.
func (s *DishStore) Dishes(ctx context.Context, after string, limit int) (int64, collection.Page[dish.Dish], error) {
	var page collection.Page[dish.Dish]
	version, err := readCollection(ctx, s.db, dish.DishesCollection, func(q *tribedb.Queries) error {
		rows, err := q.DishesAfter(ctx, tribedb.DishesAfterParams{ID: after, Limit: int64(limit) + 1})
		if err != nil {
			return fmt.Errorf("read dishes: %w", err)
		}
		if len(rows) > limit {
			rows = rows[:limit]
			page.Next = rows[limit-1].ID
		}
		if len(rows) == 0 {
			return nil
		}
		lines, err := q.DishLines(ctx, tribedb.DishLinesParams{First: rows[0].ID, Last: rows[len(rows)-1].ID})
		if err != nil {
			return fmt.Errorf("read lines of dishes: %w", err)
		}
		byDish := map[string][]dish.Line{}
		for _, l := range lines {
			byDish[l.DishID] = append(byDish[l.DishID], lineFromRow(l.Amount, l.Unit, l.Name, l.DefaultUnit))
		}
		for _, r := range rows {
			page.Items = append(page.Items, dish.Dish{
				ID: dish.ID(r.ID), Name: r.Name, Color: dish.Color(r.Color),
				Servings: int(r.Servings), Revision: r.Revision, Lines: byDish[r.ID],
			})
		}
		return nil
	})
	if err != nil {
		return 0, collection.Page[dish.Dish]{}, err
	}
	return version, page, nil
}

// Ingredients reads a page of the catalog and its version, in one transaction. The cursor
// is the key of the last ingredient of the previous page.
func (s *DishStore) Ingredients(ctx context.Context, after string, limit int) (int64, collection.Page[dish.Ingredient], error) {
	var key int64
	if after != "" {
		var err error
		if key, err = strconv.ParseInt(after, 10, 64); err != nil || key < 0 {
			return 0, collection.Page[dish.Ingredient]{}, collection.ErrBadCursor
		}
	}
	var page collection.Page[dish.Ingredient]
	version, err := readCollection(ctx, s.db, dish.IngredientsCollection, func(q *tribedb.Queries) error {
		rows, err := q.IngredientsAfter(ctx, tribedb.IngredientsAfterParams{ID: key, Limit: int64(limit) + 1})
		if err != nil {
			return fmt.Errorf("read ingredients: %w", err)
		}
		if len(rows) > limit {
			rows = rows[:limit]
			page.Next = strconv.FormatInt(rows[limit-1].ID, 10)
		}
		for _, r := range rows {
			page.Items = append(page.Items, dish.Ingredient{Name: r.Name, DefaultUnit: dish.Unit(r.DefaultUnit.String)})
		}
		return nil
	})
	if err != nil {
		return 0, collection.Page[dish.Ingredient]{}, err
	}
	return version, page, nil
}

// Write runs write in a write transaction; the versions of the dishes and of the catalog
// increase by one each if it changed them.
func (s *DishStore) Write(ctx context.Context, write func(dish.Tx) error) (collection.Versions, error) {
	return writeCollections(ctx, s.db, []string{dish.DishesCollection, dish.IngredientsCollection}, func(q *tribedb.Queries) error {
		return write(dishTx{q: q})
	})
}

func lineFromRow(amount sql.NullInt64, unit sql.NullString, name string, defaultUnit sql.NullString) dish.Line {
	l := dish.Line{Ingredient: dish.Ingredient{Name: name, DefaultUnit: dish.Unit(defaultUnit.String)}}
	if amount.Valid {
		l.Quantity = &dish.Quantity{Amount: amount.Int64, Unit: dish.Unit(unit.String)}
	}
	return l
}

type dishTx struct {
	q *tribedb.Queries
}

func (t dishTx) TribeSize(ctx context.Context) (int, error) {
	size, err := t.q.TribeSize(ctx)
	if err != nil {
		return 0, fmt.Errorf("tribe size: %w", err)
	}
	return int(size), nil
}

func (t dishTx) Dish(ctx context.Context, id dish.ID) (dish.Dish, error) {
	r, err := t.q.DishByID(ctx, string(id))
	if errors.Is(err, sql.ErrNoRows) {
		return dish.Dish{}, dish.ErrNotFound
	}
	if err != nil {
		return dish.Dish{}, fmt.Errorf("read dish: %w", err)
	}
	rows, err := t.q.DishLines(ctx, tribedb.DishLinesParams{First: r.ID, Last: r.ID})
	if err != nil {
		return dish.Dish{}, fmt.Errorf("read lines of dish: %w", err)
	}
	d := dish.Dish{ID: id, Name: r.Name, Color: dish.Color(r.Color), Servings: int(r.Servings), Revision: r.Revision}
	for _, l := range rows {
		d.Lines = append(d.Lines, lineFromRow(l.Amount, l.Unit, l.Name, l.DefaultUnit))
	}
	return d, nil
}

func (t dishTx) DishNamed(ctx context.Context, normalized string) (dish.ID, bool, error) {
	id, err := t.q.DishIDByNormalizedName(ctx, normalized)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("look up dish name: %w", err)
	}
	return dish.ID(id), true, nil
}

func (t dishTx) ColorCounts(ctx context.Context) (map[dish.Color]int, error) {
	rows, err := t.q.ColorCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("count colors: %w", err)
	}
	counts := map[dish.Color]int{}
	for _, r := range rows {
		counts[dish.Color(r.Color)] = int(r.N)
	}
	return counts, nil
}

func (t dishTx) Ingredient(ctx context.Context, normalized string) (dish.Ingredient, int64, bool, error) {
	r, err := t.q.IngredientByNormalizedName(ctx, normalized)
	if errors.Is(err, sql.ErrNoRows) {
		return dish.Ingredient{}, 0, false, nil
	}
	if err != nil {
		return dish.Ingredient{}, 0, false, fmt.Errorf("look up ingredient: %w", err)
	}
	return dish.Ingredient{Name: r.Name, DefaultUnit: dish.Unit(r.DefaultUnit.String)}, r.ID, true, nil
}

func (t dishTx) AddIngredient(ctx context.Context, ingredient dish.Ingredient, normalized string) (int64, error) {
	key, err := t.q.InsertIngredient(ctx, tribedb.InsertIngredientParams{
		Name: ingredient.Name, NormalizedName: normalized, DefaultUnit: nullUnit(ingredient.DefaultUnit),
	})
	if err != nil {
		return 0, fmt.Errorf("add ingredient: %w", err)
	}
	return key, nil
}

func (t dishTx) SetDefaultUnit(ctx context.Context, key int64, unit dish.Unit) error {
	if err := t.q.SetDefaultUnit(ctx, tribedb.SetDefaultUnitParams{DefaultUnit: nullUnit(unit), ID: key}); err != nil {
		return fmt.Errorf("set default unit: %w", err)
	}
	return nil
}

func (t dishTx) InsertDish(ctx context.Context, d dish.Dish, normalized string, keys []int64) error {
	if err := t.q.InsertDish(ctx, tribedb.InsertDishParams{
		ID: string(d.ID), Name: d.Name, NormalizedName: normalized, Color: string(d.Color), Servings: int64(d.Servings),
	}); err != nil {
		return fmt.Errorf("insert dish: %w", err)
	}
	return t.insertLines(ctx, d, keys)
}

func (t dishTx) UpdateDish(ctx context.Context, d dish.Dish, normalized string, keys []int64) error {
	if err := t.q.UpdateDish(ctx, tribedb.UpdateDishParams{
		Name: d.Name, NormalizedName: normalized, Servings: int64(d.Servings), ID: string(d.ID),
	}); err != nil {
		return fmt.Errorf("update dish: %w", err)
	}
	if err := t.q.DeleteDishLines(ctx, string(d.ID)); err != nil {
		return fmt.Errorf("update dish: lines: %w", err)
	}
	return t.insertLines(ctx, d, keys)
}

func (t dishTx) insertLines(ctx context.Context, d dish.Dish, keys []int64) error {
	for i, l := range d.Lines {
		p := tribedb.InsertDishLineParams{DishID: string(d.ID), Position: int64(i), IngredientID: keys[i]}
		if l.Quantity != nil {
			p.Amount = sql.NullInt64{Int64: l.Quantity.Amount, Valid: true}
			p.Unit = nullUnit(l.Quantity.Unit)
		}
		if err := t.q.InsertDishLine(ctx, p); err != nil {
			return fmt.Errorf("insert line %d of dish: %w", i, err)
		}
	}
	return nil
}

func nullUnit(u dish.Unit) sql.NullString {
	return sql.NullString{String: string(u), Valid: u != ""}
}
