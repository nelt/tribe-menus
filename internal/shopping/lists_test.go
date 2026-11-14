package shopping

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/nelt/tribe-menus/internal/collection"
	"github.com/nelt/tribe-menus/internal/dish"
)

// memStore is a store in memory: a write that fails leaves it unchanged, and a write
// increases the versions of what it changed by one.
type memStore struct {
	state    memState
	versions collection.Versions
}

type memState struct {
	lists       map[ListID]List
	ingredients map[string]memIngredient // by normalized name
	// served is the meal plan, read by the meal shopping; received, the served dishes
	// received by each list, as recorded (D5).
	served   []Served
	received map[ListID][]Served
}

type memIngredient struct {
	dish.Ingredient
	key int64
}

func (s memState) clone() memState {
	c := memState{lists: map[ListID]List{}, ingredients: maps.Clone(s.ingredients), served: s.served, received: map[ListID][]Served{}}
	for id, l := range s.lists {
		l.Items = slices.Clone(l.Items)
		for i := range l.Items {
			l.Items[i].Sources = slices.Clone(l.Items[i].Sources)
		}
		l.Received = slices.Clone(l.Received)
		c.lists[id] = l
	}
	for id, r := range s.received {
		c.received[id] = slices.Clone(r)
	}
	return c
}

// newMemStore returns the store of a new tribe: its main list "Courses" (D14).
func newMemStore() *memStore {
	return &memStore{
		state: memState{
			lists:       map[ListID]List{"courses": {ID: "courses", Name: "Courses", Main: true, Rank: 1}},
			ingredients: map[string]memIngredient{},
			received:    map[ListID][]Served{},
		},
		versions: collection.Versions{Collection: 0, dish.IngredientsCollection: 0},
	}
}

func (s *memStore) Version(context.Context) (int64, error) { return s.versions[Collection], nil }

func (s *memStore) Lists(_ context.Context, after string, limit int) (int64, collection.Page[List], error) {
	var page collection.Page[List]
	for _, id := range slices.Sorted(maps.Keys(s.state.lists)) {
		if string(id) <= after {
			continue
		}
		if len(page.Items) == limit {
			page.Next = string(page.Items[limit-1].ID)
			break
		}
		page.Items = append(page.Items, s.state.lists[id])
	}
	return s.versions[Collection], page, nil
}

func (s *memStore) Write(_ context.Context, write func(Tx) error) (collection.Versions, error) {
	saved := s.state.clone()
	if err := write(memTx{s}); err != nil {
		s.state = saved
		return nil, err
	}
	if !reflect.DeepEqual(saved.lists, s.state.lists) || !reflect.DeepEqual(saved.received, s.state.received) {
		s.versions[Collection]++
	}
	if !maps.Equal(saved.ingredients, s.state.ingredients) {
		s.versions[dish.IngredientsCollection]++
	}
	return maps.Clone(s.versions), nil
}

type memTx struct{ s *memStore }

func (t memTx) Ingredient(_ context.Context, normalized string) (dish.Ingredient, int64, bool, error) {
	i, ok := t.s.state.ingredients[normalized]
	return i.Ingredient, i.key, ok, nil
}

func (t memTx) AddIngredient(_ context.Context, ingredient dish.Ingredient, normalized string) (int64, error) {
	key := int64(len(t.s.state.ingredients) + 1)
	t.s.state.ingredients[normalized] = memIngredient{ingredient, key}
	return key, nil
}

func (t memTx) SetDefaultUnit(_ context.Context, key int64, unit dish.Unit) error {
	for n, i := range t.s.state.ingredients {
		if i.key == key && i.DefaultUnit == "" {
			i.DefaultUnit = unit
			t.s.state.ingredients[n] = i
		}
	}
	return nil
}

func (t memTx) List(_ context.Context, id ListID) (List, error) {
	l, ok := t.s.state.lists[id]
	if !ok {
		return List{}, ErrNotFound
	}
	l.Items = slices.Clone(l.Items)
	return l, nil
}

func (t memTx) ListCount(context.Context) (int, error) { return len(t.s.state.lists), nil }

func (t memTx) NextRank(context.Context) (int64, error) {
	var rank int64
	for _, l := range t.s.state.lists {
		rank = max(rank, l.Rank)
	}
	return rank + 1, nil
}

func (t memTx) InsertList(_ context.Context, l List) error {
	t.s.state.lists[l.ID] = l
	return nil
}

func (t memTx) RenameList(_ context.Context, id ListID, name string) error {
	l := t.s.state.lists[id]
	l.Name = name
	t.s.state.lists[id] = l
	return nil
}

func (t memTx) SetMain(_ context.Context, id ListID) error {
	for lid, l := range t.s.state.lists {
		l.Main = lid == id
		t.s.state.lists[lid] = l
	}
	return nil
}

func (t memTx) DeleteList(_ context.Context, id ListID) error {
	delete(t.s.state.lists, id)
	delete(t.s.state.received, id)
	return nil
}

func (t memTx) InsertItem(_ context.Context, list ListID, item Item, _ int64) error {
	l := t.s.state.lists[list]
	l.Items = append(l.Items, item)
	SortItems(l.Items)
	t.s.state.lists[list] = l
	return nil
}

func (t memTx) DeleteItem(_ context.Context, id ItemID) error {
	for lid, l := range t.s.state.lists {
		l.Items = slices.DeleteFunc(l.Items, func(it Item) bool { return it.ID == id })
		t.s.state.lists[lid] = l
	}
	return nil
}

func (t memTx) ServedDishes(_ context.Context, p Period) ([]Served, error) {
	var out []Served
	for _, s := range t.s.state.served {
		if s.Date >= string(p.From) && s.Date <= string(p.To) {
			out = append(out, s)
		}
	}
	return out, nil
}

// item applies change to the item of this identifier.
func (t memTx) item(id ItemID, change func(*Item)) {
	for lid, l := range t.s.state.lists {
		for i := range l.Items {
			if l.Items[i].ID == id {
				change(&l.Items[i])
				SortItems(l.Items)
				t.s.state.lists[lid] = l
				return
			}
		}
	}
}

func (t memTx) UpdateItem(_ context.Context, item Item) error {
	t.item(item.ID, func(it *Item) { it.Quantity, it.Taken, it.Checked = item.Quantity, item.Taken, item.Checked })
	return nil
}

func (t memTx) InsertSource(_ context.Context, id ItemID, s Source) error {
	t.item(id, func(it *Item) {
		it.Sources = append(it.Sources, s)
		sortSources(it.Sources)
	})
	return nil
}

func (t memTx) InsertReceived(_ context.Context, list ListID, s Served) error {
	l := t.s.state.lists[list]
	l.Received = append(l.Received, s.ID)
	slices.Sort(l.Received)
	t.s.state.lists[list] = l
	t.s.state.received[list] = append(t.s.state.received[list], s)
	return nil
}

// newLists returns the lists on the store, whose identifiers are id1, id2…
func newLists(store *memStore) *Lists {
	l := NewLists(store)
	n := 0
	l.NewID = func() (string, error) {
		n++
		return fmt.Sprintf("id%d", n), nil
	}
	return l
}

func mustCreate(t *testing.T, lists *Lists, name string) List {
	t.Helper()
	l, _, err := lists.Create(context.Background(), name)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func mustAdd(t *testing.T, lists *Lists, id ListID, in dish.LineInput) List {
	t.Helper()
	l, _, err := lists.AddItem(context.Background(), id, in)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func itemNamed(l List, ingredient string) (Item, bool) {
	i := slices.IndexFunc(l.Items, func(it Item) bool { return it.Ingredient == ingredient })
	if i < 0 {
		return Item{}, false
	}
	return l.Items[i], true
}

func TestCreateAndRename(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	lists := newLists(store)

	l, versions, err := lists.Create(ctx, "  Marché ")
	if err != nil {
		t.Fatal(err)
	}
	want := List{ID: "id1", Name: "Marché", Rank: 2}
	if !reflect.DeepEqual(l, want) || versions[Collection] != 1 {
		t.Errorf("created %+v, versions %v; want %+v", l, versions, want)
	}
	// Same name: allowed (Q17), a later rank.
	if l := mustCreate(t, lists, "Marché"); l.Rank != 3 || l.Main {
		t.Errorf("second Marché %+v", l)
	}

	l, versions, err = lists.Rename(ctx, "id1", "Marché du samedi")
	if err != nil || l.Name != "Marché du samedi" || l.ID != "id1" || versions[Collection] != 3 {
		t.Errorf("renamed %+v, %v, %v", l, versions, err)
	}

	refused := []struct {
		name string
		err  error
		do   func() error
	}{
		{"create without name", ErrListNameEmpty, func() error { _, _, err := lists.Create(ctx, " "); return err }},
		{"create with a control character", ErrListNameInvalid, func() error { _, _, err := lists.Create(ctx, "a\x07"); return err }},
		{"create a long name", ErrListNameTooLong, func() error { _, _, err := lists.Create(ctx, strings.Repeat("a", 41)); return err }},
		{"rename without name", ErrListNameEmpty, func() error { _, _, err := lists.Rename(ctx, "id1", ""); return err }},
		{"rename an unknown list", ErrNotFound, func() error { _, _, err := lists.Rename(ctx, "id9", "Drive"); return err }},
	}
	for _, c := range refused {
		if err := c.do(); !errors.Is(err, c.err) {
			t.Errorf("%s: %v, want %v", c.name, err, c.err)
		}
	}
	if store.versions[Collection] != 3 {
		t.Errorf("version %d after refusals, want 3", store.versions[Collection])
	}
}

func TestTwentyListsAtMost(t *testing.T) {
	store := newMemStore()
	lists := newLists(store)
	for i := 2; i <= MaxLists; i++ {
		mustCreate(t, lists, fmt.Sprintf("Liste %d", i))
	}
	if _, _, err := lists.Create(context.Background(), "Marché"); !errors.Is(err, ErrTooManyLists) {
		t.Errorf("21st list: %v", err)
	}
	if len(store.state.lists) != MaxLists {
		t.Errorf("%d lists", len(store.state.lists))
	}
}

func TestSetMainAndDelete(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	lists := newLists(store)
	mustCreate(t, lists, "Marché")

	// The main list itself: nothing written.
	versions, err := lists.SetMain(ctx, "courses")
	if err != nil || versions[Collection] != 1 {
		t.Errorf("set main on the main list: %v, %v", versions, err)
	}
	versions, err = lists.SetMain(ctx, "id1")
	if err != nil || versions[Collection] != 2 {
		t.Errorf("set main: %v, %v", versions, err)
	}
	if !store.state.lists["id1"].Main || store.state.lists["courses"].Main {
		t.Errorf("lists %+v", store.state.lists)
	}
	if _, err := lists.SetMain(ctx, "id9"); !errors.Is(err, ErrNotFound) {
		t.Errorf("set main on an unknown list: %v", err)
	}

	// The main list is not deleted (C12), nor a list that holds items.
	if _, err := lists.Delete(ctx, "id1"); !errors.Is(err, ErrMainList) {
		t.Errorf("delete the main list: %v", err)
	}
	mustAdd(t, lists, "courses", dish.LineInput{Name: "pain"})
	_, err = lists.Delete(ctx, "courses")
	var notEmpty *NotEmptyError
	if !errors.As(err, &notEmpty) || notEmpty.List.ID != "courses" || len(notEmpty.List.Items) != 1 {
		t.Errorf("delete a list with items: %v", err)
	}
	if len(store.state.lists) != 2 {
		t.Errorf("lists %+v", store.state.lists)
	}

	// An empty list is deleted, once.
	if _, err := lists.SetMain(ctx, "courses"); err != nil {
		t.Fatal(err)
	}
	if _, err := lists.Delete(ctx, "id1"); err != nil {
		t.Errorf("delete: %v", err)
	}
	if _, ok := store.state.lists["id1"]; ok {
		t.Error("id1 still there")
	}
	if _, err := lists.Delete(ctx, "id1"); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete again: %v", err)
	}
}

func TestAddItem(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	lists := newLists(store)

	l, versions, err := lists.AddItem(ctx, "courses", dish.LineInput{Name: " pain ", Quantity: "2", Unit: "piece"})
	if err != nil {
		t.Fatal(err)
	}
	want := []Item{{ID: "id1", Ingredient: "pain", Quantity: &dish.Quantity{Amount: 2000, Unit: dish.Piece}, Manual: true}}
	if !reflect.DeepEqual(l.Items, want) || versions[Collection] != 1 || versions[dish.IngredientsCollection] != 1 {
		t.Errorf("list %+v, versions %v", l, versions)
	}
	// The catalog: the ingredient added with the unit as default unit (Q20).
	if i := store.state.ingredients["pain"]; i.Name != "pain" || i.DefaultUnit != dish.Piece {
		t.Errorf("ingredient %+v", i)
	}

	// Without quantity (C3), an ingredient of the catalog found by its normalized name: the
	// catalog is not touched.
	l = mustAdd(t, lists, "courses", dish.LineInput{Name: "PAIN"})
	l = mustAdd(t, lists, "courses", dish.LineInput{Name: "lessive", Unit: "piece"})
	if len(l.Items) != 3 {
		t.Fatalf("items %+v", l.Items)
	}
	// Two items of the same ingredient stay distinct (D23), in the order of Q40.
	if got := []ItemID{l.Items[0].ID, l.Items[1].ID, l.Items[2].ID}; !slices.Equal(got, []ItemID{"id3", "id1", "id2"}) {
		t.Errorf("order %v", got)
	}
	if it := l.Items[2]; it.Ingredient != "pain" || it.Quantity != nil {
		t.Errorf("item without quantity %+v", it)
	}
	if i := store.state.ingredients["lessive"]; i.DefaultUnit != "" {
		t.Errorf("lessive %+v: a unit without quantity is dropped (Q26)", i)
	}
	// An ingredient without default unit gets the unit of the item (Q20).
	mustAdd(t, lists, "courses", dish.LineInput{Name: "lessive", Quantity: "1", Unit: "l"})
	if i := store.state.ingredients["lessive"]; i.DefaultUnit != dish.Litre {
		t.Errorf("lessive %+v", i)
	}

	refused := []struct {
		name string
		in   dish.LineInput
		code dish.ErrorCode
	}{
		{"no name", dish.LineInput{Name: " "}, dish.IngredientNameRequired},
		{"control character", dish.LineInput{Name: "sel\a"}, dish.IngredientNameInvalid},
		{"long name", dish.LineInput{Name: strings.Repeat("a", 41)}, dish.IngredientNameTooLong},
		{"quantity without unit", dish.LineInput{Name: "riz", Quantity: "1"}, dish.UnitRequired},
		{"unknown unit", dish.LineInput{Name: "riz", Quantity: "1", Unit: "cup"}, dish.InvalidUnit},
		{"invalid quantity", dish.LineInput{Name: "riz", Quantity: "0", Unit: "g"}, dish.InvalidQuantity},
		{"too precise", dish.LineInput{Name: "riz", Quantity: "1.5", Unit: "ml"}, dish.QuantityTooPrecise},
	}
	for _, c := range refused {
		_, _, err := lists.AddItem(ctx, "courses", c.in)
		var itemErr *ItemError
		if !errors.As(err, &itemErr) || itemErr.Code != c.code {
			t.Errorf("%s: %v, want %s", c.name, err, c.code)
		}
	}
	if _, _, err := lists.AddItem(ctx, "id9", dish.LineInput{Name: "pain"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown list: %v", err)
	}
	if n := len(store.state.lists["courses"].Items); n != 4 {
		t.Errorf("%d items after refusals", n)
	}
}

func TestThreeHundredItemsAtMost(t *testing.T) {
	store := newMemStore()
	lists := newLists(store)
	for i := range MaxItems {
		mustAdd(t, lists, "courses", dish.LineInput{Name: fmt.Sprintf("article %d", i)})
	}
	if _, _, err := lists.AddItem(context.Background(), "courses", dish.LineInput{Name: "pain"}); !errors.Is(err, ErrTooManyItems) {
		t.Errorf("301st item: %v", err)
	}
	if _, ok := store.state.ingredients["pain"]; ok {
		t.Error("pain added to the catalog")
	}
}

func TestRemoveItem(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	lists := newLists(store)
	mustAdd(t, lists, "courses", dish.LineInput{Name: "pain"})
	mustAdd(t, lists, "courses", dish.LineInput{Name: "lessive"})
	computed := Item{ID: "c1", Ingredient: "oignon", Quantity: &dish.Quantity{Amount: 1000, Unit: dish.Piece}, Sources: []Source{{ServedDish: "s1"}}}
	l := store.state.lists["courses"]
	l.Items = append(l.Items, computed)
	store.state.lists["courses"] = l

	l, versions, err := lists.RemoveItem(ctx, "courses", "id1")
	if err != nil || versions[Collection] != 3 {
		t.Fatalf("remove: %v, %v", versions, err)
	}
	if _, ok := itemNamed(l, "pain"); ok || len(l.Items) != 2 {
		t.Errorf("list %+v", l)
	}
	if _, _, err := lists.RemoveItem(ctx, "courses", "id1"); !errors.Is(err, ErrNotFound) {
		t.Errorf("remove again: %v", err)
	}
	if _, _, err := lists.RemoveItem(ctx, "courses", "c1"); !errors.Is(err, ErrComputedItem) {
		t.Errorf("remove a computed item: %v", err)
	}
	mustCreate(t, lists, "Marché")
	if _, _, err := lists.RemoveItem(ctx, "id3", "id2"); !errors.Is(err, ErrNotFound) {
		t.Errorf("remove the item of another list: %v", err)
	}
	if _, _, err := lists.RemoveItem(ctx, "id9", "id2"); !errors.Is(err, ErrNotFound) {
		t.Errorf("remove from an unknown list: %v", err)
	}
}

func TestSetCheckedAndCheckOut(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	lists := newLists(store)
	for _, name := range []string{"pain", "lessive", "ail", "beurre"} {
		mustAdd(t, lists, "courses", dish.LineInput{Name: name})
	}
	// pain id1, lessive id2, ail id3, beurre id4.
	for _, id := range []ItemID{"id1", "id2", "id3"} {
		if _, _, err := lists.SetChecked(ctx, "courses", id, true); err != nil {
			t.Fatal(err)
		}
	}
	l, _, err := lists.SetChecked(ctx, "courses", "id1", true)
	if err != nil {
		t.Fatal(err)
	}
	// Checked at the bottom (Q40).
	var order []string
	for _, it := range l.Items {
		order = append(order, fmt.Sprintf("%s %v", it.Ingredient, it.Checked))
	}
	if want := []string{"beurre false", "ail true", "lessive true", "pain true"}; !slices.Equal(order, want) {
		t.Errorf("order %v, want %v", order, want)
	}
	if _, _, err := lists.SetChecked(ctx, "courses", "id9", true); !errors.Is(err, ErrNotFound) {
		t.Errorf("check an unknown item: %v", err)
	}

	// The member saw pain, lessive and ail checked; ail was unchecked in between, beurre
	// checked in between: only pain and lessive leave (D9).
	if _, _, err := lists.SetChecked(ctx, "courses", "id3", false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := lists.SetChecked(ctx, "courses", "id4", true); err != nil {
		t.Fatal(err)
	}
	before := store.versions[Collection]
	l, removed, versions, err := lists.CheckOut(ctx, "courses", []ItemID{"id1", "id2", "id3", "id9"})
	if err != nil || removed != 2 || versions[Collection] != before+1 {
		t.Fatalf("check out: %d removed, %v, %v", removed, versions, err)
	}
	order = nil
	for _, it := range l.Items {
		order = append(order, fmt.Sprintf("%s %v", it.Ingredient, it.Checked))
	}
	if want := []string{"ail false", "beurre true"}; !slices.Equal(order, want) {
		t.Errorf("after check out %v, want %v", order, want)
	}
	// Nothing seen checked: nothing written.
	if _, removed, versions, err := lists.CheckOut(ctx, "courses", nil); err != nil || removed != 0 || versions[Collection] != before+1 {
		t.Errorf("empty check out: %d, %v, %v", removed, versions, err)
	}
	if _, _, _, err := lists.CheckOut(ctx, "id9", nil); !errors.Is(err, ErrNotFound) {
		t.Errorf("check out an unknown list: %v", err)
	}
}

func TestListsAddMealShopping(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	lists := newLists(store)
	marche := mustCreate(t, lists, "Marché")
	tue := served("s1", "chili", "2026-10-06", "dinner", 4, 4, chiliLines)
	thu := served("s2", "chili", "2026-10-08", "lunch", 4, 4, chiliLines)
	next := served("s3", "chili", "2026-10-13", "dinner", 4, 4, chiliLines)
	restes := served("s4", "restes", "2026-10-07", "lunch", 4, 4, nil)
	store.state.served = []Served{tue, restes, thu, next}

	// The period is read before anything.
	for _, c := range []struct {
		from, to string
		err      error
	}{
		{"2026-10-08", "2026-10-06", ErrInvalidPeriod},
		{"2026-10-06", "2026-11-06", ErrPeriodTooLong},
		{"1999-12-31", "2000-01-06", ErrInvalidDate},
	} {
		if _, _, _, _, err := lists.AddMealShopping(ctx, "courses", c.from, c.to); !errors.Is(err, c.err) {
			t.Errorf("%s to %s: %v, want %v", c.from, c.to, err, c.err)
		}
	}
	if _, _, _, _, err := lists.AddMealShopping(ctx, "unknown", "2026-10-06", "2026-10-12"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown list: %v", err)
	}
	if store.versions[Collection] != 1 {
		t.Fatalf("versions %v after refusals", store.versions)
	}

	// The list chosen only, the meals of the period only (C4).
	l, added, ignored, versions, err := lists.AddMealShopping(ctx, marche.ID, "2026-10-06", "2026-10-12")
	if err != nil {
		t.Fatal(err)
	}
	if added != 3 || ignored != 0 || versions[Collection] != 2 {
		t.Errorf("added %d, ignored %d, versions %v", added, ignored, versions)
	}
	beef, ok := itemNamed(l, "bœuf haché")
	if !ok || *beef.Quantity != (dish.Quantity{Amount: 1_000_000, Unit: dish.Gram}) || len(beef.Sources) != 2 || beef.Manual {
		t.Errorf("bœuf haché %+v in %+v", beef, l)
	}
	if !reflect.DeepEqual(l.Received, []ServedDishID{"s1", "s2", "s4"}) {
		t.Errorf("received %v", l.Received)
	}
	if !reflect.DeepEqual(store.state.lists[marche.ID], l) {
		t.Errorf("stored\n%+v\nreturned\n%+v", store.state.lists[marche.ID], l)
	}
	if got := store.state.received[marche.ID]; !reflect.DeepEqual(got, []Served{tue, restes, thu}) {
		t.Errorf("received recorded %+v", got)
	}
	if len(store.state.lists["courses"].Items) != 0 {
		t.Errorf("Courses %+v", store.state.lists["courses"])
	}

	// Again: every served dish ignored, nothing written (Q22).
	_, added, ignored, versions, err = lists.AddMealShopping(ctx, marche.ID, "2026-10-06", "2026-10-12")
	if err != nil || added != 0 || ignored != 3 || versions[Collection] != 2 {
		t.Errorf("again: added %d, ignored %d, versions %v, %v", added, ignored, versions, err)
	}
	// The same served dishes on another list (C4).
	if l, added, _, _, err := lists.AddMealShopping(ctx, "courses", "2026-10-06", "2026-10-06"); err != nil || added != 1 || len(l.Items) != 2 {
		t.Errorf("Courses: added %d, %+v, %v", added, l, err)
	}

	// Beyond 300 items, nothing is written (D12).
	full := mustCreate(t, lists, "Pleine")
	for i := range MaxItems - 1 {
		mustAdd(t, lists, full.ID, dish.LineInput{Name: fmt.Sprint("article ", i)})
	}
	before := store.state.clone()
	if _, _, _, _, err := lists.AddMealShopping(ctx, full.ID, "2026-10-06", "2026-10-12"); !errors.Is(err, ErrTooManyItems) {
		t.Errorf("301 items: %v", err)
	}
	if !reflect.DeepEqual(store.state, before) {
		t.Error("301 items: store changed")
	}
}

// TestSetCheckedErasesTaken: checking an item erases its taken quantity, unchecking it gives
// it none (Q41, D9); an item with a taken quantity stays after the shopping done.
func TestSetCheckedErasesTaken(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	lists := newLists(store)
	beef := Item{ID: "beef", Ingredient: "bœuf haché", Quantity: q(750_000, dish.Gram), Taken: q(250_000, dish.Gram),
		Sources: []Source{src("s1", "chili", "2026-10-06", "dinner", 6, q(750_000, dish.Gram))}}
	l := store.state.lists["courses"]
	l.Items = []Item{beef}
	store.state.lists["courses"] = l

	l, _, _, err := lists.CheckOut(ctx, "courses", []ItemID{"beef"})
	if err != nil || len(l.Items) != 1 || *l.Items[0].Taken != *beef.Taken {
		t.Fatalf("check out: %+v, %v", l, err)
	}
	if l, _, err = lists.SetChecked(ctx, "courses", "beef", true); err != nil || l.Items[0].Taken != nil || !l.Items[0].Checked {
		t.Fatalf("checked: %+v, %v", l, err)
	}
	if l, _, err = lists.SetChecked(ctx, "courses", "beef", false); err != nil || l.Items[0].Taken != nil || l.Items[0].Checked {
		t.Fatalf("unchecked: %+v, %v", l, err)
	}
	if !reflect.DeepEqual(store.state.lists["courses"], l) {
		t.Errorf("stored %+v, returned %+v", store.state.lists["courses"], l)
	}
}
