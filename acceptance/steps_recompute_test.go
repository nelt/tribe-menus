package acceptance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"github.com/cucumber/godog"
	"github.com/nelt/tribe-menus/internal/dish"
	"github.com/nelt/tribe-menus/internal/mealplan"
	"github.com/nelt/tribe-menus/internal/shopping"
)

// registerRecomputeSteps: the taken quantity, the staleness and the recomputation of
// liste-courses.feature (C8, plan courses-suite, D1), and their scenarios of
// compartimentage-tribus.feature.
func (w *world) registerRecomputeSteps(sc *godog.ScenarioContext) {
	// Taken quantity (Q41, D9).
	sc.Step(`^la liste "([^"]*)" contenant (.+) de "([^"]*)" dont (.+) déjà pris$`, w.givenTaken)
	sc.Step(`^il reste (.+) de "([^"]*)" à prendre dans la liste "([^"]*)"$`, w.remainingIs)
	sc.Step(`^il ne reste aucune part prise de "([^"]*)" dans la liste "([^"]*)"$`, func(ctx context.Context, ingredient, list string) error {
		_, it, err := w.itemOf(ctx, "je", list, ingredient)
		if err == nil && it.Taken != "" {
			err = fmt.Errorf("%s: %s taken", ingredient, it.Taken)
		}
		return err
	})
	sc.Step(`^je coche puis je décoche "([^"]*)" dans la liste "([^"]*)"$`, func(ctx context.Context, ingredient, list string) error {
		for _, checked := range []bool{true, false} {
			if err := w.check(ctx, "je", list, ingredient, checked); err != nil {
				return err
			}
			if err := w.shoppingStatus(http.StatusOK); err != nil {
				return err
			}
		}
		return nil
	})
	sc.Step(`^je déclare les courses faites de la liste "([^"]*)"$`, func(ctx context.Context, list string) error {
		return w.checkOutNow(ctx, "je", list)
	})
	sc.Step(`^j'ai coché "([^"]*)"(?: et "([^"]*)")? dans la liste "([^"]*)" et déclaré les courses faites$`, func(ctx context.Context, a, b, list string) error {
		for _, ingredient := range []string{a, b} {
			if ingredient == "" {
				continue
			}
			if err := w.check(ctx, "je", list, ingredient, true); err != nil {
				return err
			}
			if err := w.shoppingStatus(http.StatusOK); err != nil {
				return err
			}
		}
		return w.checkOutNow(ctx, "je", list)
	})

	// Changes of the meals and of the dishes (C8).
	sc.Step(`^le plat "([^"]*)" sans ingrédient servi pour (\d+) parts? le `+mealPattern+`$`, func(ctx context.Context, name string, servings int, day, moment string) error {
		m, err := parseMeal(day, moment)
		if err != nil {
			return err
		}
		return w.givenServed(ctx, name, servings, m)
	})
	sc.Step(`^le plat "([^"]*)" est ajouté au repas du `+mealPattern+`$`, func(ctx context.Context, name, day, moment string) error {
		m, err := parseMeal(day, moment)
		if err != nil {
			return err
		}
		return w.givenServed(ctx, name, 4, m)
	})
	sc.Step(`^le "([^"]*)" (?:est )?retiré du repas du `+mealPattern+`$`, func(ctx context.Context, name, day, moment string) error {
		s, err := w.servedAt(ctx, name, day, moment)
		if err != nil {
			return err
		}
		return w.planningWrite(ctx, http.MethodDelete, "meal-plan/"+s.ID, nil)
	})
	sc.Step(`^le "([^"]*)" du `+mealPattern+` réglé à (\d+) parts?$`, func(ctx context.Context, name, day, moment string, servings int) error {
		s, err := w.servedAt(ctx, name, day, moment)
		if err != nil {
			return err
		}
		return w.planningWrite(ctx, http.MethodPut, "meal-plan/"+s.ID+"/servings", map[string]int{"servings": servings})
	})
	sc.Step(`^les parts (?:du|des) "([^"]*)" du `+mealPattern+` sont modifiées$`, func(ctx context.Context, name, day, moment string) error {
		s, err := w.servedAt(ctx, name, day, moment)
		if err != nil {
			return err
		}
		return w.planningWrite(ctx, http.MethodPut, "meal-plan/"+s.ID+"/servings", map[string]int{"servings": s.Servings + 2})
	})
	sc.Step(`^la quantité de "([^"]*)" du plat "([^"]*)" change$`, func(ctx context.Context, ingredient, name string) error {
		return w.editDish(ctx, name, func(d *apiDish) error {
			for i, l := range d.Ingredients {
				if l.Name == ingredient {
					q, err := dish.ParseQuantity(l.Quantity, dish.Unit(l.Unit))
					if err != nil {
						return err
					}
					q.Amount *= 2
					d.Ingredients[i].Quantity = q.Text()
					return nil
				}
			}
			return fmt.Errorf("no %s in %+v", ingredient, d)
		})
	})
	sc.Step(`^la quantité de "([^"]*)" du plat "([^"]*)" est réécrite en (.+)$`, func(ctx context.Context, ingredient, name, quantity string) error {
		return w.changeLine(ctx, "je", ingredient, name, quantity)
	})
	sc.Step(`^l'ingrédient "([^"]*)" (?:est )?ajouté au plat "([^"]*)" pour (.+)$`, func(ctx context.Context, ingredient, name, quantity string) error {
		return w.changeLine(ctx, "je", ingredient, name, quantity)
	})
	sc.Step(`^l'ingrédient "([^"]*)" est retiré du plat "([^"]*)"$`, func(ctx context.Context, ingredient, name string) error {
		return w.editDish(ctx, name, func(d *apiDish) error {
			n := len(d.Ingredients)
			d.Ingredients = slices.DeleteFunc(d.Ingredients, func(l apiLine) bool { return l.Name == ingredient })
			if len(d.Ingredients) == n {
				return fmt.Errorf("no %s in %+v", ingredient, d)
			}
			return nil
		})
	})
	sc.Step(`^le plat "([^"]*)" est redéfini pour (\d+) parts$`, func(ctx context.Context, name string, servings int) error {
		return w.editDish(ctx, name, func(d *apiDish) error {
			d.Servings = servings
			return nil
		})
	})
	sc.Step(`^le plat "([^"]*)" est renommé "([^"]*)"$`, func(ctx context.Context, name, to string) error {
		return w.editDish(ctx, name, func(d *apiDish) error {
			d.Name = to
			return nil
		})
	})

	// Staleness and recomputation (C8, D7).
	sc.Step(`^le bandeau de liste périmée (est|n'est pas) affiché$`, w.staleBanner)
	sc.Step(`^je recalcule la liste "([^"]*)"$`, w.recompute)
	sc.Step(`^j'ai recalculé la liste "([^"]*)"$`, func(ctx context.Context, list string) error {
		if err := w.recompute(ctx, list); err != nil {
			return err
		}
		return w.shoppingStatus(http.StatusOK)
	})

	// ENF-02
	sc.Step(`^la liste "([^"]*)" de la tribu "([^"]*)" est périmée$`, w.otherListStale)
	sc.Step(`^je demande quelles listes sont périmées$`, func(ctx context.Context) error {
		ids, err := w.staleLists(ctx)
		for _, id := range ids {
			w.shopping.seen = append(w.shopping.seen, apiList{ID: id})
		}
		return err
	})
}

// givenTaken writes a computed item with a taken quantity: the item as the meal shopping
// leaves it, from the "Chili con carne" of the evening of the day of the scenario, then its
// taken quantity (Q41).
func (w *world) givenTaken(ctx context.Context, list, quantity, ingredient, taken string) error {
	if err := w.givenComputed(ctx, list, quantity, ingredient, "Chili con carne", "lundi 5 octobre 2026", "au soir"); err != nil {
		return err
	}
	_, it, err := w.itemOf(ctx, "je", list, ingredient)
	if err != nil {
		return err
	}
	q, unit, err := parseQuantity(taken)
	if err != nil {
		return err
	}
	amount, err := dish.ParseQuantity(q, dish.Unit(unit))
	if err != nil {
		return err
	}
	db, err := w.kitchenDB(ctx)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, "UPDATE shopping_items SET taken_amount = ? WHERE id = ?", amount.Amount, it.ID)
	return err
}

// remainingIs checks what remains to take of an unchecked item: its quantity less its taken
// quantity, compared in base units.
func (w *world) remainingIs(ctx context.Context, remaining, ingredient, list string) error {
	_, it, err := w.itemOf(ctx, "je", list, ingredient)
	if err != nil {
		return err
	}
	if it.Checked || it.Taken == "" {
		return fmt.Errorf("%s: %+v, without taken quantity", ingredient, it)
	}
	quantity, err := dish.ParseQuantity(it.Quantity, dish.Unit(it.Unit))
	if err != nil {
		return err
	}
	taken, err := dish.ParseQuantity(it.Taken, dish.Unit(it.Unit))
	if err != nil {
		return err
	}
	q, unit, err := parseQuantity(remaining)
	if err != nil {
		return err
	}
	want, err := dish.ParseQuantity(q, dish.Unit(unit))
	if err != nil {
		return err
	}
	if got := quantity.Amount - taken.Amount; got != want.Amount || want.Unit.Family() != quantity.Unit.Family() {
		return fmt.Errorf("%s: %s %s, %s taken; want %s left", ingredient, it.Quantity, it.Unit, it.Taken, remaining)
	}
	return nil
}

// servedAt returns the served dish of this name in the meal, as "je" reads it now.
func (w *world) servedAt(ctx context.Context, name, day, moment string) (namedServed, error) {
	m, err := parseMeal(day, moment)
	if err != nil {
		return namedServed{}, err
	}
	served, err := w.mealOf(ctx, "je", m)
	if err != nil {
		return namedServed{}, err
	}
	for _, s := range served {
		if s.name == name {
			return s, nil
		}
	}
	return namedServed{}, fmt.Errorf("no %s in the meal %v: %+v", name, m, served)
}

// planningWrite writes on the meal plan as "je", in a context step: it must succeed.
func (w *world) planningWrite(ctx context.Context, method, path string, body any) error {
	rec, err := w.api(ctx, "je", method, path, body)
	if err != nil {
		return err
	}
	w.planning.last = rec
	if rec.Code != http.StatusOK {
		return fmt.Errorf("%s %s: status %d, body %s", method, path, rec.Code, rec.Body)
	}
	return nil
}

// editDish changes the dish of this name, as "je" reads it now, and saves it.
func (w *world) editDish(ctx context.Context, name string, change func(*apiDish) error) error {
	d, err := w.findDish(ctx, "je", name)
	if err != nil {
		return err
	}
	if err := change(&d); err != nil {
		return err
	}
	_, err = w.mustSave(ctx, "je", http.MethodPut, "dishes/"+d.ID, d.body())
	return err
}

// staleLists reads the stale lists, as "je" (D7).
func (w *world) staleLists(ctx context.Context) ([]string, error) {
	rec, err := w.api(ctx, "je", http.MethodGet, "shopping-lists/stale", nil)
	if err != nil {
		return nil, err
	}
	if rec.Code != http.StatusOK {
		return nil, fmt.Errorf("stale lists: status %d, body %s", rec.Code, rec.Body)
	}
	var r struct {
		Lists []string `json:"lists"`
	}
	err = json.Unmarshal(rec.Body.Bytes(), &r)
	return r.Lists, err
}

// staleBanner checks whether the list of the last write on the lists is stale: the banner
// shows what GET shopping-lists/stale says (D16).
func (w *world) staleBanner(ctx context.Context, shown string) error {
	l, err := w.writtenList()
	if err != nil {
		return err
	}
	ids, err := w.staleLists(ctx)
	if err != nil {
		return err
	}
	if stale := slices.Contains(ids, l.ID); stale != (shown == "est") {
		return fmt.Errorf("list %s stale %v; stale lists %v", l.Name, stale, ids)
	}
	return nil
}

func (w *world) recompute(ctx context.Context, list string) error {
	id, err := w.openedListID(ctx, list)
	if err != nil {
		return err
	}
	return w.shoppingSend(ctx, "je", http.MethodPost, "shopping-lists/"+id+"/recompute", nil)
}

// otherListStale makes the list of another tribe stale: the dish served, its shopping added
// to the list, then the served dish removed from the meal plan.
func (w *world) otherListStale(ctx context.Context, name, slug string) error {
	l, err := w.otherList(ctx, slug, name)
	if err != nil {
		return err
	}
	id, err := w.otherDishID(ctx, slug, w.kitchen.foreignName)
	if err != nil {
		return err
	}
	plan, err := w.otherPlan(ctx, slug)
	if err != nil {
		return err
	}
	servings := 4
	s, _, err := plan.Serve(ctx, mealplan.ServeInput{Date: "2026-10-06", Mealtime: "dinner", Dish: id, Servings: &servings})
	if err != nil {
		return err
	}
	lists, err := w.otherLists(ctx, slug)
	if err != nil {
		return err
	}
	if _, _, _, _, err := lists.AddMealShopping(ctx, l.ID, "2026-10-06", "2026-10-06"); err != nil {
		return err
	}
	if _, err := plan.Remove(ctx, s.ID); err != nil {
		return err
	}
	stale, err := lists.Stale(ctx)
	if err == nil && !slices.Equal(stale, []shopping.ListID{l.ID}) {
		err = fmt.Errorf("stale lists of %s: %v", slug, stale)
	}
	return err
}
