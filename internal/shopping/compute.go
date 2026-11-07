package shopping

import (
	"cmp"
	"errors"
	"slices"
	"time"

	"github.com/nelt/tribe-menus/internal/dish"
	"github.com/nelt/tribe-menus/internal/mealplan"
)

// MaxPeriodDays is the length of a period of meal shopping at most, bounds included (Q38,
// D12).
const MaxPeriodDays = 31

var (
	// ErrInvalidDate is returned for a bound that is not a real date from 2000 to 2099.
	ErrInvalidDate = errors.New("invalid date")
	// ErrInvalidPeriod is returned for a period whose end precedes its start (C4).
	ErrInvalidPeriod = errors.New("period ends before it starts")
	// ErrPeriodTooLong is returned for a period of more than 31 days (D12).
	ErrPeriodTooLong = errors.New("period too long")
)

// Period is the days whose meals give their shopping (C4), bounds included.
type Period struct {
	From, To mealplan.Date
}

// ParsePeriod reads a period: two real dates of the meal plan, from 2000 to 2099 (Q35), the
// end on the start day at the earliest, 31 days at most (D12). The server knows no "today"
// (D24).
func ParsePeriod(from, to string) (Period, error) {
	f, err := mealplan.ParseDate(from)
	if err != nil {
		return Period{}, ErrInvalidDate
	}
	t, err := mealplan.ParseDate(to)
	if err != nil {
		return Period{}, ErrInvalidDate
	}
	if t < f {
		return Period{}, ErrInvalidPeriod
	}
	start, _ := time.Parse(time.DateOnly, string(f)) // checked by ParseDate
	end, _ := time.Parse(time.DateOnly, string(t))
	if end.Sub(start) >= MaxPeriodDays*24*time.Hour {
		return Period{}, ErrPeriodTooLong
	}
	return Period{From: f, To: t}, nil
}

// Served is a served dish of the period, with its dish as it is at the time of the
// computation (P2): its servings of reference and its lines.
type Served struct {
	ID           ServedDishID
	Dish         dish.ID
	Date         string
	Mealtime     string
	Servings     int
	DishServings int
	Lines        []DishLine
}

// DishLine is a line of the dish of a served dish: its ingredient, by its name and its key in
// the catalog, and its optional quantity (Q26).
type DishLine struct {
	Ingredient string
	Key        int64
	Quantity   *dish.Quantity
}

// smallestUnits are the smallest units of the families: those of a computed item and of a
// source (conception, « API »).
var smallestUnits = map[dish.Family]dish.Unit{
	dish.Mass:             dish.Gram,
	dish.Volume:           dish.Millilitre,
	dish.PieceFamily:      dish.Piece,
	dish.TablespoonFamily: dish.Tablespoon,
	dish.TeaspoonFamily:   dish.Teaspoon,
	dish.PinchFamily:      dish.Pinch,
}

// Contribution returns the quantity a line of a dish brings to a served dish: the quantity of
// the line × servings ÷ servings of reference, rounded up to the base unit (D7), in the
// smallest unit of its family.
func Contribution(q dish.Quantity, servings, reference int) dish.Quantity {
	r := int64(reference)
	amount := (q.Amount*int64(servings) + r - 1) / r
	return dish.Quantity{Amount: amount, Unit: smallestUnits[q.Unit.Family()]}
}

// RoundItem rounds up the quantity of a computed item, its contributions added, to the whole
// smallest unit of its family: pieces, spoons and pinches to the integer (Q4), masses to the
// gram, volumes to the millilitre (Q37, D7).
func RoundItem(q dish.Quantity) dish.Quantity {
	u := smallestUnits[q.Unit.Family()]
	step := u.Factor()
	return dish.Quantity{Amount: (q.Amount + step - 1) / step * step, Unit: u}
}

// groupKey is what a computed item gathers: an ingredient, by its normalized name (Q29), in
// a family of units, or without quantity (family empty, D3).
type groupKey struct {
	ingredient string
	family     dish.Family
}

func keyOf(ingredient string, q *dish.Quantity) groupKey {
	k := groupKey{ingredient: dish.NormalizeName(ingredient)}
	if q != nil {
		k.family = q.Unit.Family()
	}
	return k
}

// itemChange is a computed item the meal shopping creates or changes: the item after, the
// sources it gains, and the key of its ingredient for a new one.
type itemChange struct {
	item    Item
	isNew   bool
	key     int64
	family  dish.Family
	sources []Source
}

// mealShopping is what the meal shopping does to a list: the list after, the served dishes
// added and ignored (Q22), the items to write and the served dishes received (D5).
type mealShopping struct {
	list     List
	added    int
	ignored  int
	items    []itemChange
	received []Served
}

// addMealShopping pours the shopping of the served dishes of a period into a list (C4, Q22,
// Q36, Q37): the served dishes the list already received are ignored; the others become
// received served dishes, even without ingredients; each line brings its contribution to the
// computed item of its ingredient and family, created if absent; items added by hand are
// never touched. The item adds the contributions of all its sources, then rounds; a checked
// item whose quantity increases is unchecked, an item without quantity keeps its check.
// ErrTooManyItems beyond 300 items (D12). NewID gives the identifiers of the new items.
func addMealShopping(list List, served []Served, newID func() (string, error)) (mealShopping, error) {
	m := mealShopping{list: list}
	items := make([]Item, len(list.Items))
	for i, it := range list.Items {
		it.Sources = slices.Clone(it.Sources)
		items[i] = it
	}
	groups := map[groupKey]int{}
	for i, it := range items {
		if k := keyOf(it.Ingredient, it.Quantity); !it.Manual {
			if _, ok := groups[k]; !ok {
				groups[k] = i
			}
		}
	}
	received := map[ServedDishID]bool{}
	for _, id := range list.Received {
		received[id] = true
	}
	m.list.Received = slices.Clone(list.Received)

	changes := map[int]*itemChange{}
	var order []int
	for _, s := range served {
		if received[s.ID] {
			m.ignored++
			continue
		}
		received[s.ID] = true
		m.added++
		m.received = append(m.received, s)
		m.list.Received = append(m.list.Received, s.ID)
		for _, line := range s.Lines {
			source := Source{ServedDish: s.ID, Dish: s.Dish, Date: s.Date, Mealtime: s.Mealtime, Servings: s.Servings}
			if line.Quantity != nil {
				c := Contribution(*line.Quantity, s.Servings, s.DishServings)
				source.Quantity = &c
			}
			k := keyOf(line.Ingredient, source.Quantity)
			i, ok := groups[k]
			if !ok {
				id, err := newID()
				if err != nil {
					return mealShopping{}, err
				}
				items = append(items, Item{ID: ItemID(id), Ingredient: line.Ingredient})
				i = len(items) - 1
				groups[k] = i
				changes[i] = &itemChange{isNew: true, key: line.Key, family: k.family}
				order = append(order, i)
			} else if changes[i] == nil {
				changes[i] = &itemChange{family: k.family}
				order = append(order, i)
			}
			items[i].Sources = append(items[i].Sources, source)
			changes[i].sources = append(changes[i].sources, source)
		}
	}
	if len(items) > MaxItems {
		return mealShopping{}, ErrTooManyItems
	}

	for _, i := range order {
		it := &items[i]
		sortSources(it.Sources)
		if family := changes[i].family; family != "" {
			total := dish.Quantity{Unit: smallestUnits[family]}
			for _, s := range it.Sources {
				if s.Quantity != nil {
					total.Amount += s.Quantity.Amount
				}
			}
			total = RoundItem(total)
			if it.Checked && (it.Quantity == nil || total.Amount > it.Quantity.Amount) {
				it.Checked = false
			}
			it.Quantity = &total
		}
		changes[i].item = *it
		m.items = append(m.items, *changes[i])
	}
	SortItems(items)
	slices.Sort(m.list.Received)
	m.list.Items = items
	return m, nil
}

// sortSources orders the sources of an item by meal, lunch before dinner, then by served
// dish: the order in which the store reads them.
func sortSources(sources []Source) {
	slices.SortFunc(sources, func(a, b Source) int {
		return cmp.Or(
			cmp.Compare(a.Date, b.Date),
			cmpBool(a.Mealtime != string(mealplan.Lunch), b.Mealtime != string(mealplan.Lunch)),
			cmp.Compare(a.ServedDish, b.ServedDish),
		)
	})
}
