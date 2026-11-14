package shopping

import (
	"slices"

	"github.com/nelt/tribe-menus/internal/dish"
)

// merge is a computed item of the source list joined to the computed item of the same key in
// the target list (D10): the item of the source list goes, the item of the target list after
// keeps its identifier, with the sources it gains.
type merge struct {
	from    ItemID
	item    Item
	sources []Source
}

// moving is what a move does to two lists (C9, C10, D10): the lists after, the number of
// items moved and of selected items no longer in the source list (Q42); the items that go to
// the target list as they are, those merged, and the served dishes the target list receives,
// with their kept lines, copied from the source list.
type moving struct {
	source, target List
	moved, missing int
	relocated      []ItemID
	merged         []merge
	received       []Served
}

// move takes the selected items of the source list to the target list (C9, C10, D10, Q41,
// Q42). Kept are the served dishes the source list received, as recorded. Only the selected
// items still in the source list move; the others are counted missing. A computed item joins
// the computed item of the same key in the target list: sources gathered by served dish, a
// served dish already there not counted twice, quantity added and rounded; checked if both
// were, otherwise its taken quantity is the sum of the taken quantities, the whole quantity
// of a checked item, if there is one, and the item is checked again when that reaches its
// quantity. Otherwise, and for an item added by hand, the item moves as it is. The served
// dishes of the sources moved become received by the target list, unless it already received
// them; the source list keeps them (Q22). ErrTooManyItems beyond 300 items in the target
// list.
func move(source, target List, ids []ItemID, kept []Served) (moving, error) {
	m := moving{source: source, target: target}
	selected := map[ItemID]bool{}
	for _, id := range ids {
		selected[id] = true
	}
	targetItems := make([]Item, len(target.Items))
	groups := map[groupKey]int{}
	for i, it := range target.Items {
		it.Sources = slices.Clone(it.Sources)
		targetItems[i] = it
		if k := keyOf(it.Ingredient, it.Quantity); !it.Manual {
			if _, ok := groups[k]; !ok {
				groups[k] = i
			}
		}
	}
	received := map[ServedDishID]bool{}
	for _, id := range target.Received {
		received[id] = true
	}
	keptOf := map[ServedDishID]Served{}
	for _, k := range kept {
		keptOf[k.ID] = k
	}
	m.target.Received = slices.Clone(target.Received)

	var sourceItems []Item
	for _, it := range source.Items {
		if !selected[it.ID] {
			sourceItems = append(sourceItems, it)
			continue
		}
		delete(selected, it.ID)
		m.moved++
		for _, s := range it.Sources {
			if k, ok := keptOf[s.ServedDish]; ok && !received[s.ServedDish] {
				received[s.ServedDish] = true
				m.received = append(m.received, k)
				m.target.Received = append(m.target.Received, k.ID)
			}
		}
		i, ok := groups[keyOf(it.Ingredient, it.Quantity)]
		if it.Manual || !ok {
			targetItems = append(targetItems, it)
			m.relocated = append(m.relocated, it.ID)
			continue
		}
		joined, gained := join(targetItems[i], it)
		targetItems[i] = joined
		m.merged = append(m.merged, merge{from: it.ID, item: joined, sources: gained})
	}
	m.missing = len(selected)
	if len(targetItems) > MaxItems {
		return moving{}, ErrTooManyItems
	}
	SortItems(targetItems)
	slices.Sort(m.target.Received)
	m.source.Items = sourceItems
	m.target.Items = targetItems
	return m, nil
}

// join merges the computed item from into the computed item into, of the same key (D10, Q41):
// it returns the item after, and the sources it gains, those of the served dishes it had not.
func join(into, from Item) (Item, []Source) {
	var gained []Source
	for _, s := range from.Sources {
		if !slices.ContainsFunc(into.Sources, func(t Source) bool { return t.ServedDish == s.ServedDish }) {
			gained = append(gained, s)
		}
	}
	after := into
	after.Sources = append(slices.Clone(into.Sources), gained...)
	sortSources(after.Sources)
	after.Checked, after.Taken = into.Checked && from.Checked, nil
	if after.Quantity == nil {
		return after, gained
	}
	total := dish.Quantity{Unit: after.Quantity.Unit}
	for _, s := range after.Sources {
		total.Amount += s.Quantity.Amount
	}
	total = RoundItem(total)
	after.Quantity = &total
	if after.Checked {
		return after, gained
	}
	var taken int64
	for _, it := range []Item{into, from} {
		switch {
		case it.Checked:
			taken += it.Quantity.Amount
		case it.Taken != nil:
			taken += it.Taken.Amount
		}
	}
	switch {
	case taken >= total.Amount:
		after.Checked = true
	case taken > 0:
		after.Taken = &dish.Quantity{Amount: taken, Unit: total.Unit}
	}
	return after, gained
}
