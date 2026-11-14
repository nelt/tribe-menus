package shopping

import (
	"reflect"
	"slices"

	"github.com/nelt/tribe-menus/internal/dish"
)

// recomputation is what the recomputation does to a list (D5): the list after; whether its
// items change, which makes the list stale (D6); the items deleted, and those created or
// changed, each with all its sources; the received served dishes whose kept lines, servings
// and servings of reference are replaced by their state now.
type recomputation struct {
	list     List
	stale    bool
	deleted  []ItemID
	items    []itemChange
	received []Served
}

// recompute brings a list back in line with the meals (C8, D2, D5, Q43). Kept are the served
// dishes the list received, as recorded with the lines of their dish at the time of the
// computation; current, those of them still in the meal plan, with their dish as it is now.
// Only the served dishes that still have a source in the list are recomputed: the others,
// whose items all left (bought or moved) or which never brought anything, are left as they
// are. For each of them, key by key (ingredient and family of units, or ingredient without
// quantity):
//   - a kept key whose source is in the list: the source is recomputed on the state now;
//   - a kept key without source in the list: its item left, nothing is created again;
//   - a new key (line added to the dish, or whose unit changed family): a source is created,
//     in the computed item of the key, present or created;
//   - a key no longer in the dish, or a served dish removed from the meal plan: the source
//     goes.
//
// Each computed item touched then adds its sources and rounds; an item without source goes,
// even checked; its check and taken quantity follow applyTaken. Items added by hand are
// never touched. The limit of 300 items is left to the caller. NewID gives the identifiers
// of the new items.
func recompute(list List, kept, current []Served, newID func() (string, error)) (recomputation, error) {
	r := recomputation{list: list}
	before := map[ItemID]Item{}
	items := make([]Item, len(list.Items))
	live := map[ServedDishID]bool{}
	groups := map[groupKey]int{}
	for i, it := range list.Items {
		before[it.ID] = it
		it.Sources = slices.Clone(it.Sources)
		items[i] = it
		for _, s := range it.Sources {
			live[s.ServedDish] = true
		}
		if k := keyOf(it.Ingredient, it.Quantity); !it.Manual {
			if _, ok := groups[k]; !ok {
				groups[k] = i
			}
		}
	}
	currentOf := map[ServedDishID]Served{}
	for _, c := range current {
		currentOf[c.ID] = c
	}

	touched := map[int]bool{}
	keys := map[int]int64{} // ingredient keys of the new items
	for _, k := range kept {
		if !live[k.ID] {
			continue
		}
		had := map[groupKey]bool{}
		for i := range items {
			n := len(items[i].Sources)
			items[i].Sources = slices.DeleteFunc(items[i].Sources, func(s Source) bool { return s.ServedDish == k.ID })
			if len(items[i].Sources) != n {
				had[keyOf(items[i].Ingredient, items[i].Quantity)] = true
				touched[i] = true
			}
		}
		c, ok := currentOf[k.ID]
		if !ok {
			continue
		}
		keptKeys := map[groupKey]bool{}
		for _, line := range k.Lines {
			keptKeys[keyOf(line.Ingredient, line.Quantity)] = true
		}
		for _, line := range c.Lines {
			source := Source{ServedDish: c.ID, Dish: c.Dish, Date: c.Date, Mealtime: c.Mealtime, Servings: c.Servings}
			if line.Quantity != nil {
				contributed := Contribution(*line.Quantity, c.Servings, c.DishServings)
				source.Quantity = &contributed
			}
			key := keyOf(line.Ingredient, source.Quantity)
			if keptKeys[key] && !had[key] {
				continue
			}
			i, ok := groups[key]
			if !ok {
				id, err := newID()
				if err != nil {
					return recomputation{}, err
				}
				items = append(items, Item{ID: ItemID(id), Ingredient: line.Ingredient})
				i = len(items) - 1
				groups[key] = i
				keys[i] = line.Key
			}
			items[i].Sources = append(items[i].Sources, source)
			touched[i] = true
		}
		r.received = append(r.received, c)
	}

	after := make([]Item, 0, len(items))
	for i, it := range items {
		if !touched[i] {
			after = append(after, it)
			continue
		}
		old, existed := before[it.ID]
		if len(it.Sources) == 0 {
			if existed {
				r.deleted = append(r.deleted, it.ID)
				r.stale = true
			}
			continue
		}
		sortSources(it.Sources)
		it.Quantity = nil
		if q := it.Sources[0].Quantity; q != nil {
			total := dish.Quantity{Unit: smallestUnits[q.Unit.Family()]}
			for _, s := range it.Sources {
				total.Amount += s.Quantity.Amount
			}
			total = RoundItem(total)
			it.Quantity = &total
		}
		it = applyTaken(old, it)
		after = append(after, it)
		switch {
		case !existed:
			r.items = append(r.items, itemChange{item: it, isNew: true, key: keys[i]})
			r.stale = true
		case !reflect.DeepEqual(old, it):
			r.items = append(r.items, itemChange{item: it})
			r.stale = r.stale || !sameItem(old, it)
		}
	}
	SortItems(after)
	r.list.Items = after
	return r, nil
}

// sameItem tells whether two states of an item are the same for the staleness (D6): same
// quantity and taken quantity, in base units, same check, and the same sources, by served
// dish and contributed quantity. The servings of a source without quantity, its date or
// mealtime are not compared: they change nothing to buy.
func sameItem(a, b Item) bool {
	if !sameAmount(a.Quantity, b.Quantity) || !sameAmount(a.Taken, b.Taken) || a.Checked != b.Checked || len(a.Sources) != len(b.Sources) {
		return false
	}
	for i, s := range a.Sources {
		if s.ServedDish != b.Sources[i].ServedDish || !sameAmount(s.Quantity, b.Sources[i].Quantity) {
			return false
		}
	}
	return true
}

func sameAmount(a, b *dish.Quantity) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Amount == b.Amount
}

// stale tells whether recomputing the list would change its items (D6).
func stale(list List, kept, current []Served) bool {
	r, _ := recompute(list, kept, current, func() (string, error) { return "", nil }) // no error without identifier to draw
	return r.stale
}
