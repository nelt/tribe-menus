package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/nelt/tribe-menus/internal/collection"
	"github.com/nelt/tribe-menus/internal/dish"
	"github.com/nelt/tribe-menus/internal/shopping"
	"github.com/nelt/tribe-menus/internal/storage/tribedb"
)

// ShoppingStore is the store of the shopping lists of one tribe: a thin adapter around
// tribedb.
type ShoppingStore struct {
	db *sql.DB
}

var _ shopping.Store = (*ShoppingStore)(nil)

// NewShoppingStore returns the shopping list store of the tribe whose database is db.
func NewShoppingStore(db *sql.DB) *ShoppingStore {
	return &ShoppingStore{db: db}
}

// Version reads the current version of the shopping lists.
func (s *ShoppingStore) Version(ctx context.Context) (int64, error) {
	return collectionVersion(ctx, s.db, shopping.Collection)
}

// Lists reads a page of shopping lists, with their items, and the version of the collection,
// in one transaction. It reads one list more than the limit, to know whether a next page
// exists.
func (s *ShoppingStore) Lists(ctx context.Context, after string, limit int) (int64, collection.Page[shopping.List], error) {
	var page collection.Page[shopping.List]
	version, err := readCollection(ctx, s.db, shopping.Collection, func(q *tribedb.Queries) error {
		rows, err := q.ShoppingListsAfter(ctx, tribedb.ShoppingListsAfterParams{ID: after, Limit: int64(limit) + 1})
		if err != nil {
			return fmt.Errorf("read shopping lists: %w", err)
		}
		if len(rows) > limit {
			rows = rows[:limit]
			page.Next = rows[limit-1].ID
		}
		if len(rows) == 0 {
			return nil
		}
		page.Items, err = readLists(ctx, q, rows)
		return err
	})
	if err != nil {
		return 0, collection.Page[shopping.List]{}, err
	}
	return version, page, nil
}

// Write runs write in a write transaction; the versions of the shopping lists and of the
// ingredient catalog increase by one each if it changed them.
func (s *ShoppingStore) Write(ctx context.Context, write func(shopping.Tx) error) (collection.Versions, error) {
	return writeCollections(ctx, s.db, []string{shopping.Collection, dish.IngredientsCollection}, func(q *tribedb.Queries) error {
		return write(shoppingTx{catalogTx{q: q}})
	})
}

// Read runs read in a transaction rolled back: it writes nothing.
func (s *ShoppingStore) Read(ctx context.Context, read func(shopping.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("read shopping lists: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	return read(shoppingTx{catalogTx{q: tribedb.New(tx)}})
}

// readLists reads the items, their sources and the received served dishes of the lists of
// the rows, ordered by identifier: the lists from the first to the last.
func readLists(ctx context.Context, q *tribedb.Queries, rows []tribedb.ShoppingList) ([]shopping.List, error) {
	first, last := rows[0].ID, rows[len(rows)-1].ID
	items, err := q.ShoppingItemsOfLists(ctx, tribedb.ShoppingItemsOfListsParams{First: first, Last: last})
	if err != nil {
		return nil, fmt.Errorf("read shopping items: %w", err)
	}
	sources, err := q.ShoppingItemSourcesOfLists(ctx, tribedb.ShoppingItemSourcesOfListsParams{First: first, Last: last})
	if err != nil {
		return nil, fmt.Errorf("read item sources: %w", err)
	}
	received, err := q.ShoppingReceivedOfLists(ctx, tribedb.ShoppingReceivedOfListsParams{First: first, Last: last})
	if err != nil {
		return nil, fmt.Errorf("read received served dishes: %w", err)
	}
	sourcesOf := map[string][]shopping.Source{}
	for _, r := range sources {
		sourcesOf[r.ItemID] = append(sourcesOf[r.ItemID], shopping.Source{
			ServedDish: shopping.ServedDishID(r.ServedDishID), Dish: dish.ID(r.DishID), Date: r.Date,
			Mealtime: r.Mealtime, Servings: int(r.Servings), Quantity: quantityOf(r.Amount, r.Unit),
		})
	}
	itemsOf := map[string][]shopping.Item{}
	for _, r := range items {
		itemsOf[r.ListID] = append(itemsOf[r.ListID], shopping.Item{
			ID: shopping.ItemID(r.ID), Ingredient: r.Name, Quantity: quantityOf(r.Amount, r.Unit),
			Taken: quantityOf(r.TakenAmount, r.Unit), Checked: r.Checked == 1, Manual: r.Manual == 1,
			Sources: sourcesOf[r.ID],
		})
	}
	receivedOf := map[string][]shopping.ServedDishID{}
	for _, r := range received {
		receivedOf[r.ListID] = append(receivedOf[r.ListID], shopping.ServedDishID(r.ServedDishID))
	}
	lists := make([]shopping.List, 0, len(rows))
	for _, r := range rows {
		l := shopping.List{
			ID: shopping.ListID(r.ID), Name: r.Name, Main: r.Main == 1, Rank: r.Rank,
			Items: itemsOf[r.ID], Received: receivedOf[r.ID],
		}
		shopping.SortItems(l.Items)
		lists = append(lists, l)
	}
	return lists, nil
}

func quantityOf(amount sql.NullInt64, unit sql.NullString) *dish.Quantity {
	if !amount.Valid {
		return nil
	}
	return &dish.Quantity{Amount: amount.Int64, Unit: dish.Unit(unit.String)}
}

func boolInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

type shoppingTx struct {
	catalogTx
}

func (t shoppingTx) List(ctx context.Context, id shopping.ListID) (shopping.List, error) {
	r, err := t.q.ShoppingListByID(ctx, string(id))
	if errors.Is(err, sql.ErrNoRows) {
		return shopping.List{}, shopping.ErrNotFound
	}
	if err != nil {
		return shopping.List{}, fmt.Errorf("read shopping list: %w", err)
	}
	lists, err := readLists(ctx, t.q, []tribedb.ShoppingList{r})
	if err != nil {
		return shopping.List{}, err
	}
	return lists[0], nil
}

func (t shoppingTx) AllLists(ctx context.Context) ([]shopping.List, error) {
	rows, err := t.q.AllShoppingLists(ctx)
	if err != nil {
		return nil, fmt.Errorf("read shopping lists: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return readLists(ctx, t.q, rows)
}

func (t shoppingTx) ListCount(ctx context.Context) (int, error) {
	n, err := t.q.ShoppingListCount(ctx)
	if err != nil {
		return 0, fmt.Errorf("count shopping lists: %w", err)
	}
	return int(n), nil
}

func (t shoppingTx) NextRank(ctx context.Context) (int64, error) {
	rank, err := t.q.NextShoppingListRank(ctx)
	if err != nil {
		return 0, fmt.Errorf("next rank: %w", err)
	}
	return rank, nil
}

func (t shoppingTx) InsertList(ctx context.Context, l shopping.List) error {
	if err := t.q.InsertShoppingList(ctx, tribedb.InsertShoppingListParams{ID: string(l.ID), Name: l.Name, Rank: l.Rank}); err != nil {
		return fmt.Errorf("insert shopping list: %w", err)
	}
	return nil
}

func (t shoppingTx) RenameList(ctx context.Context, id shopping.ListID, name string) error {
	if err := t.q.RenameShoppingList(ctx, tribedb.RenameShoppingListParams{Name: name, ID: string(id)}); err != nil {
		return fmt.Errorf("rename shopping list: %w", err)
	}
	return nil
}

// SetMain takes the role away from the main list first: the index allows one main list at
// most, checked statement by statement.
func (t shoppingTx) SetMain(ctx context.Context, id shopping.ListID) error {
	if err := t.q.ClearMainShoppingList(ctx); err != nil {
		return fmt.Errorf("set main list: %w", err)
	}
	if err := t.q.SetMainShoppingList(ctx, string(id)); err != nil {
		return fmt.Errorf("set main list: %w", err)
	}
	return nil
}

func (t shoppingTx) DeleteList(ctx context.Context, id shopping.ListID) error {
	if err := t.q.DeleteShoppingReceivedLinesOfList(ctx, string(id)); err != nil {
		return fmt.Errorf("delete shopping list: received lines: %w", err)
	}
	if err := t.q.DeleteShoppingReceivedOfList(ctx, string(id)); err != nil {
		return fmt.Errorf("delete shopping list: received served dishes: %w", err)
	}
	if err := t.q.DeleteShoppingList(ctx, string(id)); err != nil {
		return fmt.Errorf("delete shopping list: %w", err)
	}
	return nil
}

func (t shoppingTx) InsertItem(ctx context.Context, list shopping.ListID, item shopping.Item, ingredient int64) error {
	p := tribedb.InsertShoppingItemParams{
		ID: string(item.ID), ListID: string(list), IngredientID: ingredient,
		Checked: boolInt(item.Checked), Manual: boolInt(item.Manual),
	}
	p.Amount, p.Unit = nullQuantity(item.Quantity)
	p.TakenAmount = nullAmount(item.Taken)
	if err := t.q.InsertShoppingItem(ctx, p); err != nil {
		return fmt.Errorf("insert shopping item: %w", err)
	}
	return nil
}

func (t shoppingTx) DeleteItem(ctx context.Context, id shopping.ItemID) error {
	if err := t.q.DeleteShoppingItemSources(ctx, string(id)); err != nil {
		return fmt.Errorf("delete shopping item: sources: %w", err)
	}
	if err := t.q.DeleteShoppingItem(ctx, string(id)); err != nil {
		return fmt.Errorf("delete shopping item: %w", err)
	}
	return nil
}

// ServedDishes reads the served dishes of the period with the servings of reference and the
// lines of their dishes, in the write transaction: the dishes as they are now (P2).
func (t shoppingTx) ServedDishes(ctx context.Context, p shopping.Period) ([]shopping.Served, error) {
	bounds := tribedb.ServedDishesBetweenParams{First: string(p.From), Last: string(p.To)}
	rows, err := t.q.ServedDishesBetween(ctx, bounds)
	if err != nil {
		return nil, fmt.Errorf("read served dishes of the period: %w", err)
	}
	lines, err := t.q.DishLinesServedBetween(ctx, tribedb.DishLinesServedBetweenParams(bounds))
	if err != nil {
		return nil, fmt.Errorf("read lines of the served dishes: %w", err)
	}
	linesOf := map[string][]shopping.DishLine{}
	for _, l := range lines {
		linesOf[l.DishID] = append(linesOf[l.DishID], shopping.DishLine{Ingredient: l.Name, Key: l.IngredientID, Quantity: quantityOf(l.Amount, l.Unit)})
	}
	served := make([]shopping.Served, 0, len(rows))
	for _, r := range rows {
		served = append(served, shopping.Served{
			ID: shopping.ServedDishID(r.ID), Dish: dish.ID(r.DishID), Date: r.Date, Mealtime: r.Mealtime,
			Servings: int(r.Servings), DishServings: int(r.DishServings), Lines: linesOf[r.DishID],
		})
	}
	return served, nil
}

func nullQuantity(q *dish.Quantity) (sql.NullInt64, sql.NullString) {
	if q == nil {
		return sql.NullInt64{}, sql.NullString{}
	}
	return nullAmount(q), nullUnit(q.Unit)
}

// nullAmount is the amount of a quantity whose unit is that of its item: a taken quantity.
func nullAmount(q *dish.Quantity) sql.NullInt64 {
	if q == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: q.Amount, Valid: true}
}

// UpdateItem writes the quantity, the taken quantity and the check of an item.
func (t shoppingTx) UpdateItem(ctx context.Context, item shopping.Item) error {
	p := tribedb.UpdateShoppingItemParams{Checked: boolInt(item.Checked), ID: string(item.ID)}
	p.Amount, p.Unit = nullQuantity(item.Quantity)
	p.TakenAmount = nullAmount(item.Taken)
	if err := t.q.UpdateShoppingItem(ctx, p); err != nil {
		return fmt.Errorf("update shopping item: %w", err)
	}
	return nil
}

func (t shoppingTx) InsertSource(ctx context.Context, item shopping.ItemID, s shopping.Source) error {
	p := tribedb.InsertShoppingItemSourceParams{
		ItemID: string(item), ServedDishID: string(s.ServedDish), DishID: string(s.Dish),
		Date: s.Date, Mealtime: s.Mealtime, Servings: int64(s.Servings),
	}
	p.Amount, p.Unit = nullQuantity(s.Quantity)
	if err := t.q.InsertShoppingItemSource(ctx, p); err != nil {
		return fmt.Errorf("insert item source: %w", err)
	}
	return nil
}

func (t shoppingTx) InsertReceived(ctx context.Context, list shopping.ListID, s shopping.Served) error {
	err := t.q.InsertShoppingReceived(ctx, tribedb.InsertShoppingReceivedParams{
		ListID: string(list), ServedDishID: string(s.ID), DishID: string(s.Dish),
		Servings: int64(s.Servings), DishServings: int64(s.DishServings),
	})
	if err != nil {
		return fmt.Errorf("insert received served dish: %w", err)
	}
	for pos, l := range s.Lines {
		p := tribedb.InsertShoppingReceivedLineParams{
			ListID: string(list), ServedDishID: string(s.ID), Position: int64(pos), IngredientID: l.Key,
		}
		p.Amount, p.Unit = nullQuantity(l.Quantity)
		if err := t.q.InsertShoppingReceivedLine(ctx, p); err != nil {
			return fmt.Errorf("insert received line: %w", err)
		}
	}
	return nil
}

// Received reads the served dishes received by a list as recorded, with the lines of their
// dish at the time of the computation (D5).
func (t shoppingTx) Received(ctx context.Context, list shopping.ListID) ([]shopping.Served, error) {
	rows, err := t.q.ShoppingReceivedOfList(ctx, string(list))
	if err != nil {
		return nil, fmt.Errorf("read received served dishes: %w", err)
	}
	lines, err := t.q.ShoppingReceivedLinesOfList(ctx, string(list))
	if err != nil {
		return nil, fmt.Errorf("read received lines: %w", err)
	}
	linesOf := map[string][]shopping.DishLine{}
	for _, l := range lines {
		linesOf[l.ServedDishID] = append(linesOf[l.ServedDishID], shopping.DishLine{Ingredient: l.Name, Key: l.IngredientID, Quantity: quantityOf(l.Amount, l.Unit)})
	}
	served := make([]shopping.Served, 0, len(rows))
	for _, r := range rows {
		served = append(served, shopping.Served{
			ID: shopping.ServedDishID(r.ServedDishID), Dish: dish.ID(r.DishID),
			Servings: int(r.Servings), DishServings: int(r.DishServings), Lines: linesOf[r.ServedDishID],
		})
	}
	return served, nil
}

// CurrentReceived reads the served dishes received by a list still in the meal plan, with
// the servings of reference and the lines of their dishes as they are now (P2).
func (t shoppingTx) CurrentReceived(ctx context.Context, list shopping.ListID) ([]shopping.Served, error) {
	rows, err := t.q.ServedDishesReceivedBy(ctx, string(list))
	if err != nil {
		return nil, fmt.Errorf("read served dishes received: %w", err)
	}
	lines, err := t.q.DishLinesReceivedBy(ctx, string(list))
	if err != nil {
		return nil, fmt.Errorf("read lines of the served dishes received: %w", err)
	}
	linesOf := map[string][]shopping.DishLine{}
	for _, l := range lines {
		linesOf[l.DishID] = append(linesOf[l.DishID], shopping.DishLine{Ingredient: l.Name, Key: l.IngredientID, Quantity: quantityOf(l.Amount, l.Unit)})
	}
	served := make([]shopping.Served, 0, len(rows))
	for _, r := range rows {
		served = append(served, shopping.Served{
			ID: shopping.ServedDishID(r.ID), Dish: dish.ID(r.DishID), Date: r.Date, Mealtime: r.Mealtime,
			Servings: int(r.Servings), DishServings: int(r.DishServings), Lines: linesOf[r.DishID],
		})
	}
	return served, nil
}

func (t shoppingTx) ReplaceReceived(ctx context.Context, list shopping.ListID, s shopping.Served) error {
	if err := t.q.DeleteShoppingReceivedLines(ctx, tribedb.DeleteShoppingReceivedLinesParams{ListID: string(list), ServedDishID: string(s.ID)}); err != nil {
		return fmt.Errorf("replace received served dish: lines: %w", err)
	}
	if err := t.q.DeleteShoppingReceived(ctx, tribedb.DeleteShoppingReceivedParams{ListID: string(list), ServedDishID: string(s.ID)}); err != nil {
		return fmt.Errorf("replace received served dish: %w", err)
	}
	return t.InsertReceived(ctx, list, s)
}

func (t shoppingTx) DeleteSources(ctx context.Context, id shopping.ItemID) error {
	if err := t.q.DeleteShoppingItemSources(ctx, string(id)); err != nil {
		return fmt.Errorf("delete item sources: %w", err)
	}
	return nil
}
