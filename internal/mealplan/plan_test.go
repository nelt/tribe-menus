package mealplan

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
// increases the version by one.
type memStore struct {
	size    int
	dishes  map[dish.ID]bool
	served  map[ID]ServedDish
	version int64
}

func newMemStore(dishes ...dish.ID) *memStore {
	s := &memStore{size: 4, dishes: map[dish.ID]bool{}, served: map[ID]ServedDish{}}
	for _, d := range dishes {
		s.dishes[d] = true
	}
	return s
}

func (s *memStore) Version(context.Context) (int64, error) { return s.version, nil }

func (s *memStore) ServedDishes(_ context.Context, after string, limit int) (int64, collection.Page[ServedDish], error) {
	var page collection.Page[ServedDish]
	for _, id := range slices.Sorted(maps.Keys(s.served)) {
		if string(id) <= after {
			continue
		}
		if len(page.Items) == limit {
			page.Next = string(page.Items[limit-1].ID)
			break
		}
		page.Items = append(page.Items, s.served[id])
	}
	return s.version, page, nil
}

func (s *memStore) Write(_ context.Context, write func(Tx) error) (collection.Versions, error) {
	saved := maps.Clone(s.served)
	if err := write(memTx{s}); err != nil {
		s.served = saved
		return nil, err
	}
	if !maps.Equal(saved, s.served) {
		s.version++
	}
	return collection.Versions{Collection: s.version}, nil
}

type memTx struct{ s *memStore }

func (t memTx) DishExists(_ context.Context, id dish.ID) (bool, error) { return t.s.dishes[id], nil }
func (t memTx) TribeSize(context.Context) (int, error)                 { return t.s.size, nil }

func (t memTx) Meal(_ context.Context, date Date, mealtime Mealtime) ([]ServedDish, error) {
	var meal []ServedDish
	for _, s := range t.s.served {
		if s.Date == date && s.Mealtime == mealtime {
			meal = append(meal, s)
		}
	}
	slices.SortFunc(meal, func(a, b ServedDish) int { return int(a.Rank - b.Rank) })
	return meal, nil
}

func (t memTx) NextRank(context.Context) (int64, error) {
	var rank int64
	for _, s := range t.s.served {
		rank = max(rank, s.Rank)
	}
	return rank + 1, nil
}

func (t memTx) Insert(_ context.Context, s ServedDish) error {
	t.s.served[s.ID] = s
	return nil
}

func (t memTx) SetServings(_ context.Context, id ID, servings int) (ServedDish, error) {
	s, ok := t.s.served[id]
	if !ok {
		return ServedDish{}, ErrNotFound
	}
	s.Servings = servings
	t.s.served[id] = s
	return s, nil
}

func (t memTx) Delete(_ context.Context, id ID) error {
	if _, ok := t.s.served[id]; !ok {
		return ErrNotFound
	}
	delete(t.s.served, id)
	return nil
}

// newPlan returns a plan on the store, whose identifiers are s1, s2…
func newPlan(store *memStore) *Plan {
	p := NewPlan(store)
	n := 0
	p.NewID = func() (ID, error) {
		n++
		return ID(fmt.Sprintf("s%d", n)), nil
	}
	return p
}

func servings(n int) *int { return &n }

func TestServe(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name string
		in   ServeInput
		want ServedDish
		err  error
	}{
		{"tribe size", ServeInput{Date: "2026-10-06", Mealtime: "dinner", Dish: "chili"},
			ServedDish{ID: "s1", Date: "2026-10-06", Mealtime: Dinner, Dish: "chili", Servings: 5, Rank: 2}, nil},
		{"servings given", ServeInput{Date: "2026-10-06", Mealtime: "dinner", Dish: "chili", Servings: servings(2)},
			ServedDish{ID: "s1", Date: "2026-10-06", Mealtime: Dinner, Dish: "chili", Servings: 2, Rank: 2}, nil},
		{"past meal (R5)", ServeInput{Date: "2000-01-01", Mealtime: "lunch", Dish: "chili"},
			ServedDish{ID: "s1", Date: "2000-01-01", Mealtime: Lunch, Dish: "chili", Servings: 5, Rank: 2}, nil},
		{"same dish, another meal", ServeInput{Date: "2026-10-06", Mealtime: "lunch", Dish: "gratin"},
			ServedDish{ID: "s1", Date: "2026-10-06", Mealtime: Lunch, Dish: "gratin", Servings: 5, Rank: 2}, nil},
		{"invalid date", ServeInput{Date: "2100-01-01", Mealtime: "dinner", Dish: "chili"}, ServedDish{}, ErrInvalidDate},
		{"invalid mealtime", ServeInput{Date: "2026-10-06", Mealtime: "breakfast", Dish: "chili"}, ServedDish{}, ErrInvalidMealtime},
		{"servings 0", ServeInput{Date: "2026-10-06", Mealtime: "dinner", Dish: "chili", Servings: servings(0)}, ServedDish{}, ErrInvalidServings},
		{"servings 51", ServeInput{Date: "2026-10-06", Mealtime: "dinner", Dish: "chili", Servings: servings(51)}, ServedDish{}, ErrInvalidServings},
		{"unknown dish", ServeInput{Date: "2026-10-06", Mealtime: "dinner", Dish: "tartiflette"}, ServedDish{}, ErrUnknownDish},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			store := newMemStore("chili", "gratin")
			store.size = 5
			// Already served: the gratin, at dinner on the 6th.
			store.served["s0"] = ServedDish{ID: "s0", Date: "2026-10-06", Mealtime: Dinner, Dish: "gratin", Servings: 4, Rank: 1}
			store.version = 1
			got, versions, err := newPlan(store).Serve(ctx, c.in)
			if !errors.Is(err, c.err) {
				t.Fatalf("error %v, want %v", err, c.err)
			}
			if got != c.want {
				t.Errorf("served %+v, want %+v", got, c.want)
			}
			if c.err != nil {
				if len(store.served) != 1 || store.version != 1 {
					t.Errorf("refused write changed the plan: %v, version %d", store.served, store.version)
				}
				return
			}
			if store.served[got.ID] != got || versions[Collection] != 2 {
				t.Errorf("stored %+v, versions %v", store.served, versions)
			}
		})
	}
}

// TestServeAlreadyServed: a dish already in the meal is refused with the meal as it is now,
// by rank (Q21, D3).
func TestServeAlreadyServed(t *testing.T) {
	ctx := context.Background()
	store := newMemStore("chili", "vege")
	plan := newPlan(store)
	for _, d := range []dish.ID{"vege", "chili"} {
		if _, _, err := plan.Serve(ctx, ServeInput{Date: "2026-10-06", Mealtime: "dinner", Dish: d}); err != nil {
			t.Fatal(err)
		}
	}
	_, _, err := plan.Serve(ctx, ServeInput{Date: "2026-10-06", Mealtime: "dinner", Dish: "chili"})
	var already *AlreadyServedError
	if !errors.As(err, &already) {
		t.Fatalf("error %v", err)
	}
	var dishes []dish.ID
	for _, s := range already.Meal {
		dishes = append(dishes, s.Dish)
	}
	if !slices.Equal(dishes, []dish.ID{"vege", "chili"}) {
		t.Errorf("meal %+v", already.Meal)
	}
	if len(store.served) != 2 || store.version != 2 {
		t.Errorf("plan %v, version %d", store.served, store.version)
	}
}

// TestServeAgain: a dish removed then served again is a new served dish, from the tribe
// size, after the others (D2, D11).
func TestServeAgain(t *testing.T) {
	ctx := context.Background()
	store := newMemStore("chili", "vege")
	plan := newPlan(store)
	meal := ServeInput{Date: "2026-10-06", Mealtime: "dinner"}
	first := meal
	first.Dish = "vege"
	vege, _, err := plan.Serve(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	second := meal
	second.Dish = "chili"
	if _, _, err := plan.Serve(ctx, second); err != nil {
		t.Fatal(err)
	}
	if _, _, err := plan.SetServings(ctx, vege.ID, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := plan.Remove(ctx, vege.ID); err != nil {
		t.Fatal(err)
	}
	again, _, err := plan.Serve(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	if again.ID == vege.ID || again.Servings != 4 || again.Rank <= 2 {
		t.Errorf("served again %+v, first %+v", again, vege)
	}
}

func TestSetServings(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name     string
		id       ID
		servings int
		err      error
	}{
		{"1", "s0", 1, nil},
		{"50", "s0", 50, nil},
		{"0", "s0", 0, ErrInvalidServings},
		{"51", "s0", 51, ErrInvalidServings},
		{"removed in between", "s9", 3, ErrNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			store := newMemStore("chili")
			before := ServedDish{ID: "s0", Date: "2026-10-06", Mealtime: Dinner, Dish: "chili", Servings: 4, Rank: 1}
			store.served["s0"] = before
			got, versions, err := newPlan(store).SetServings(ctx, c.id, c.servings)
			if !errors.Is(err, c.err) {
				t.Fatalf("error %v, want %v", err, c.err)
			}
			if c.err != nil {
				if store.served["s0"] != before || store.version != 0 {
					t.Errorf("refused write changed %+v", store.served["s0"])
				}
				return
			}
			want := before
			want.Servings = c.servings
			if got != want || store.served["s0"] != want || versions[Collection] != 1 {
				t.Errorf("got %+v, stored %+v, versions %v", got, store.served["s0"], versions)
			}
		})
	}
}

func TestRemove(t *testing.T) {
	ctx := context.Background()
	store := newMemStore("chili")
	store.served["s0"] = ServedDish{ID: "s0", Date: "2026-10-06", Mealtime: Dinner, Dish: "chili", Servings: 4, Rank: 1}
	plan := newPlan(store)
	versions, err := plan.Remove(ctx, "s0")
	if err != nil {
		t.Fatal(err)
	}
	if len(store.served) != 0 || versions[Collection] != 1 {
		t.Errorf("plan %v, versions %v", store.served, versions)
	}
	if !store.dishes["chili"] {
		t.Error("dish removed from the library")
	}
	// Removed already, by another member (D19).
	if _, err := plan.Remove(ctx, "s0"); !errors.Is(err, ErrNotFound) {
		t.Errorf("removed twice: %v", err)
	}
}

func TestServedDishes(t *testing.T) {
	ctx := context.Background()
	store := newMemStore("chili")
	for i := range PageSize + 1 {
		id := ID(fmt.Sprintf("s%03d", i))
		store.served[id] = ServedDish{ID: id, Date: "2026-10-06", Mealtime: Lunch, Dish: "chili", Servings: 4, Rank: int64(i + 1)}
	}
	plan := NewPlan(store)
	_, first, err := plan.ServedDishes(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	_, second, err := plan.ServedDishes(ctx, first.Next)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != PageSize || len(second.Items) != 1 || second.Next != "" {
		t.Errorf("pages of %d and %d, next %q", len(first.Items), len(second.Items), second.Next)
	}
	if !reflect.DeepEqual(second.Items[0], store.served["s200"]) {
		t.Errorf("last %+v", second.Items[0])
	}
}

func TestRandomID(t *testing.T) {
	id, err := randomID()
	if err != nil {
		t.Fatal(err)
	}
	if len(id) != 32 || strings.Trim(string(id), "0123456789abcdef") != "" {
		t.Errorf("identifier %q", id)
	}
}
