package dish

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/nelt/tribe-menus/internal/collection"
)

// memStore is a store in memory: a write that fails leaves it unchanged.
type memStore struct {
	size        int
	dishes      map[ID]memDish
	ingredients []memIngredient // key = index + 1
	versions    collection.Versions
}

type memDish struct {
	dish       Dish
	normalized string
	keys       []int64
}

type memIngredient struct {
	ingredient Ingredient
	normalized string
}

func newMemStore() *memStore {
	return &memStore{size: 4, dishes: map[ID]memDish{}, versions: collection.Versions{DishesCollection: 0, IngredientsCollection: 0}}
}

func (s *memStore) Version(_ context.Context, name string) (int64, error) {
	return s.versions[name], nil
}

func (s *memStore) Dishes(_ context.Context, after string, limit int) (int64, collection.Page[Dish], error) {
	ids := slices.Sorted(maps.Keys(s.dishes))
	var page collection.Page[Dish]
	for _, id := range ids {
		if string(id) <= after {
			continue
		}
		if len(page.Items) == limit {
			page.Next = string(page.Items[limit-1].ID)
			break
		}
		page.Items = append(page.Items, s.dish(id))
	}
	return s.versions[DishesCollection], page, nil
}

func (s *memStore) Ingredients(context.Context, string, int) (int64, collection.Page[Ingredient], error) {
	var page collection.Page[Ingredient]
	for _, i := range s.ingredients {
		page.Items = append(page.Items, i.ingredient)
	}
	return s.versions[IngredientsCollection], page, nil
}

// dish returns the dish with the ingredients of the catalog as they are now.
func (s *memStore) dish(id ID) Dish {
	m := s.dishes[id]
	d := m.dish
	d.Lines = slices.Clone(d.Lines)
	for i, key := range m.keys {
		d.Lines[i].Ingredient = s.ingredients[key-1].ingredient
	}
	return d
}

func (s *memStore) Write(_ context.Context, write func(Tx) error) (collection.Versions, error) {
	saved := memStore{size: s.size, dishes: maps.Clone(s.dishes), ingredients: slices.Clone(s.ingredients), versions: maps.Clone(s.versions)}
	if err := write(memTx{s}); err != nil {
		*s = saved
		return nil, err
	}
	return maps.Clone(s.versions), nil
}

type memTx struct{ s *memStore }

func (t memTx) TribeSize(context.Context) (int, error) { return t.s.size, nil }

func (t memTx) Dish(_ context.Context, id ID) (Dish, error) {
	if _, ok := t.s.dishes[id]; !ok {
		return Dish{}, ErrNotFound
	}
	return t.s.dish(id), nil
}

func (t memTx) DishNamed(_ context.Context, normalized string) (ID, bool, error) {
	for id, d := range t.s.dishes {
		if d.normalized == normalized {
			return id, true, nil
		}
	}
	return "", false, nil
}

func (t memTx) ColorCounts(context.Context) (map[Color]int, error) {
	counts := map[Color]int{}
	for _, d := range t.s.dishes {
		counts[d.dish.Color]++
	}
	return counts, nil
}

func (t memTx) Ingredient(_ context.Context, normalized string) (Ingredient, int64, bool, error) {
	for i, ing := range t.s.ingredients {
		if ing.normalized == normalized {
			return ing.ingredient, int64(i + 1), true, nil
		}
	}
	return Ingredient{}, 0, false, nil
}

func (t memTx) AddIngredient(_ context.Context, ingredient Ingredient, normalized string) (int64, error) {
	t.s.ingredients = append(t.s.ingredients, memIngredient{ingredient, normalized})
	t.s.versions[IngredientsCollection]++
	return int64(len(t.s.ingredients)), nil
}

func (t memTx) SetDefaultUnit(_ context.Context, key int64, unit Unit) error {
	t.s.ingredients[key-1].ingredient.DefaultUnit = unit
	t.s.versions[IngredientsCollection]++
	return nil
}

func (t memTx) InsertDish(_ context.Context, d Dish, normalized string, keys []int64) error {
	t.s.dishes[d.ID] = memDish{d, normalized, keys}
	t.s.versions[DishesCollection]++
	return nil
}

func (t memTx) UpdateDish(_ context.Context, d Dish, normalized string, keys []int64) error {
	t.s.dishes[d.ID] = memDish{d, normalized, keys}
	t.s.versions[DishesCollection]++
	return nil
}

// newTestLibrary returns a library on a store in memory, with identifiers d1, d2…
func newTestLibrary() (*Library, *memStore) {
	s := newMemStore()
	n := 0
	return &Library{Store: s, NewID: func() (ID, error) {
		n++
		return ID(fmt.Sprintf("d%d", n)), nil
	}}, s
}

func line(name, quantity, unit string) LineInput {
	return LineInput{Name: name, Quantity: quantity, Unit: unit}
}

func create(t *testing.T, l *Library, in Input) Dish {
	t.Helper()
	d, _, err := l.Create(context.Background(), in)
	if err != nil {
		t.Fatalf("Create %q: %v", in.Name, err)
	}
	return d
}

func wantInputError(t *testing.T, err error, code ErrorCode) {
	t.Helper()
	var inputErr *InputError
	if !errors.As(err, &inputErr) || inputErr.Code != code {
		t.Fatalf("error %v, want %s", err, code)
	}
}

func TestCreate(t *testing.T) {
	ctx := context.Background()
	l, s := newTestLibrary()

	d, versions, err := l.Create(ctx, Input{Name: " Chili con carne ", Lines: []LineInput{
		line("bœuf haché", "500", "g"), line("haricots rouges", "400", "g"), line("oignon", "2", "piece"), line("sel", "", ""),
	}})
	if err != nil {
		t.Fatal(err)
	}
	want := Dish{ID: "d1", Name: "Chili con carne", Color: Palette[0], Servings: 4, Revision: 1, Lines: []Line{
		{Ingredient{"bœuf haché", Gram}, &Quantity{500_000, Gram}},
		{Ingredient{"haricots rouges", Gram}, &Quantity{400_000, Gram}},
		{Ingredient{"oignon", Piece}, &Quantity{2000, Piece}},
		{Ingredient{"sel", ""}, nil},
	}}
	if !equalDish(d, want) {
		t.Errorf("dish %+v, want %+v", d, want)
	}
	if versions[DishesCollection] != 1 || versions[IngredientsCollection] != 4 {
		t.Errorf("versions %v", versions)
	}
	if got := s.dish("d1"); !equalDish(got, want) {
		t.Errorf("stored %+v", got)
	}
}

func TestCreateServings(t *testing.T) {
	l, s := newTestLibrary()
	if d := create(t, l, Input{Name: "Gratin dauphinois", Servings: 6}); d.Servings != 6 {
		t.Errorf("servings %d, want 6", d.Servings)
	}
	s.size = 5
	if d := create(t, l, Input{Name: "Curry de lentilles"}); d.Servings != 5 {
		t.Errorf("servings %d, want the tribe size 5 (T3)", d.Servings)
	}
}

func TestCreateColors(t *testing.T) {
	l, _ := newTestLibrary()
	for i, want := range append(slices.Clone(Palette), Palette[0], Palette[1]) {
		if d := create(t, l, Input{Name: fmt.Sprintf("Plat %d", i)}); d.Color != want {
			t.Errorf("dish %d: color %s, want %s", i, d.Color, want)
		}
	}
}

func TestCreateRefused(t *testing.T) {
	ctx := context.Background()
	l, s := newTestLibrary()
	create(t, l, Input{Name: "Soupe à l'oignon", Lines: []LineInput{line("oignon", "4", "piece")}})
	before := s.versions[DishesCollection]

	_, _, err := l.Create(ctx, Input{Name: " soupe a l’oignon ", Lines: []LineInput{line("gruyère", "100", "g")}})
	wantInputError(t, err, NameTaken)
	_, _, err = l.Create(ctx, Input{Name: ""})
	wantInputError(t, err, NameRequired)
	_, _, err = l.Create(ctx, Input{Name: "Ratatouille", Lines: []LineInput{line("courgette", "3", "piece"), line("Courgette", "200", "g")}})
	wantInputError(t, err, DuplicateIngredient)

	if len(s.dishes) != 1 || len(s.ingredients) != 1 || s.versions[DishesCollection] != before {
		t.Errorf("refused dishes written: %d dishes, %d ingredients", len(s.dishes), len(s.ingredients))
	}
}

func TestCatalog(t *testing.T) {
	ctx := context.Background()
	l, s := newTestLibrary()

	create(t, l, Input{Name: "Pâtes au beurre", Lines: []LineInput{line("crème fraîche", "", ""), line("sel", "", "")}})
	d := create(t, l, Input{Name: "Gratin", Lines: []LineInput{line(" Creme  Fraiche ", "20", "cl"), line("Sel", "1", "pinch")}})
	create(t, l, Input{Name: "Tarte", Lines: []LineInput{line("CRÈME FRAÎCHE", "200", "g")}})

	if len(s.ingredients) != 2 {
		t.Fatalf("catalog %+v", s.ingredients)
	}
	want := []Ingredient{{"crème fraîche", Centilitre}, {"sel", Pinch}}
	_, page, err := l.Catalog(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(page.Items, want) {
		t.Errorf("catalog %+v, want %+v (name of the first entry, first unit entered)", page.Items, want)
	}
	if d.Lines[0].Ingredient != want[0] || d.Lines[1].Ingredient != want[1] {
		t.Errorf("lines of the dish %+v, want the ingredients of the catalog (D27)", d.Lines)
	}
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	l, s := newTestLibrary()
	chili := create(t, l, Input{Name: "Chili con carne", Lines: []LineInput{line("bœuf haché", "500", "g")}})
	create(t, l, Input{Name: "Ratatouille"})

	updated, versions, err := l.Update(ctx, chili.ID, 1, Input{Name: "Chili", Servings: 6, Lines: []LineInput{
		line("bœuf haché", "600", "g"), line("cumin", "1", "teaspoon"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	want := Dish{ID: chili.ID, Name: "Chili", Color: chili.Color, Servings: 6, Revision: 2, Lines: []Line{
		{Ingredient{"bœuf haché", Gram}, &Quantity{600_000, Gram}},
		{Ingredient{"cumin", Teaspoon}, &Quantity{1000, Teaspoon}},
	}}
	if !equalDish(updated, want) || !equalDish(s.dish(chili.ID), want) {
		t.Errorf("dish %+v, want %+v", updated, want)
	}
	if versions[DishesCollection] != 3 {
		t.Errorf("versions %v", versions)
	}

	// Same name, its own: accepted.
	if _, _, err := l.Update(ctx, chili.ID, 2, Input{Name: "CHILI", Servings: 6}); err != nil {
		t.Errorf("own name: %v", err)
	}
}

func TestUpdateRefused(t *testing.T) {
	ctx := context.Background()
	l, s := newTestLibrary()
	chili := create(t, l, Input{Name: "Chili con carne", Lines: []LineInput{line("bœuf haché", "500", "g")}})
	create(t, l, Input{Name: "Ratatouille"})
	if _, _, err := l.Update(ctx, chili.ID, 1, Input{Name: "Chili con carne", Servings: 4, Lines: []LineInput{line("bœuf haché", "600", "g")}}); err != nil {
		t.Fatal(err)
	}
	current := s.dish(chili.ID)

	_, _, err := l.Update(ctx, chili.ID, 1, Input{Name: "Chili con carne", Servings: 4, Lines: []LineInput{line("bœuf haché", "500", "g"), line("cumin", "1", "teaspoon")}})
	var changed *ChangedError
	if !errors.As(err, &changed) {
		t.Fatalf("error %v, want ChangedError (Q25)", err)
	}
	if !equalDish(changed.Current, current) {
		t.Errorf("current dish %+v, want %+v", changed.Current, current)
	}

	_, _, err = l.Update(ctx, "inconnu", 1, Input{Name: "Chili", Servings: 4})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown dish: %v", err)
	}
	_, _, err = l.Update(ctx, chili.ID, 2, Input{Name: "ratatouille", Servings: 4})
	wantInputError(t, err, NameTaken)
	_, _, err = l.Update(ctx, chili.ID, 2, Input{Name: "Chili"})
	wantInputError(t, err, InvalidServings)

	if got := s.dish(chili.ID); !equalDish(got, current) {
		t.Errorf("dish changed by a refused save: %+v", got)
	}
	if _, _, found, _ := (memTx{s}).Ingredient(ctx, "cumin"); found {
		t.Error("cumin added by a refused save")
	}
}

func TestDishesPages(t *testing.T) {
	ctx := context.Background()
	l, _ := newTestLibrary()
	for i := range DishPageSize + 1 {
		create(t, l, Input{Name: fmt.Sprintf("Plat %03d", i)})
	}
	_, first, err := l.Dishes(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != DishPageSize || first.Next == "" {
		t.Fatalf("first page: %d dishes, next %q", len(first.Items), first.Next)
	}
	_, second, err := l.Dishes(ctx, first.Next)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 1 || second.Next != "" {
		t.Errorf("second page: %d dishes, next %q", len(second.Items), second.Next)
	}
}

func equalDish(a, b Dish) bool {
	return a.ID == b.ID && a.Name == b.Name && a.Color == b.Color && a.Servings == b.Servings &&
		a.Revision == b.Revision && slices.EqualFunc(a.Lines, b.Lines, func(x, y Line) bool {
		return x.Ingredient == y.Ingredient && (x.Quantity == nil) == (y.Quantity == nil) && (x.Quantity == nil || *x.Quantity == *y.Quantity)
	})
}
