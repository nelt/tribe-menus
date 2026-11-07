package acceptance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/cucumber/godog"
)

// mealShoppingState is what the meal shopping scenarios remember from step to step: the
// dishes defined by the scenario, with the number of the step that defined each, for « chacun
// de ces plats ».
type mealShoppingState struct {
	defined []definedDish
}

type definedDish struct {
	name string
	step int
}

// periodText is a period of a step: « du 6 au 12 octobre 2026 », « du 6 octobre au 5
// novembre 2026 », « du 31 décembre 1999 au 6 janvier 2000 », « du 28 décembre 2099 au 1er
// janvier 2100 ».
var periodText = regexp.MustCompile(`^du (\d+)(?:er)?(?: (\S+?))?(?: (\d{4}))? au (\d+)(?:er)? (\S+) (\d{4})$`)

// parsePeriod reads a period of a step, as AAAA-MM-JJ: the start takes the month and the year
// of the end when it does not name them. The dates are not checked: a scenario may name a
// date out of the bounds of the meal plan.
func parsePeriod(text string) (from, to string, err error) {
	m := periodText.FindStringSubmatch(text)
	if m == nil {
		return "", "", fmt.Errorf("period %q", text)
	}
	date := func(day, month, year string) (string, error) {
		mo, ok := months[month]
		if !ok {
			return "", fmt.Errorf("period %q: month %q", text, month)
		}
		d, _ := strconv.Atoi(day)
		y, _ := strconv.Atoi(year)
		return fmt.Sprintf("%04d-%02d-%02d", y, mo, d), nil
	}
	startMonth, startYear := cmpOrString(m[2], m[5]), cmpOrString(m[3], m[6])
	if from, err = date(m[1], startMonth, startYear); err != nil {
		return "", "", err
	}
	to, err = date(m[4], m[5], m[6])
	return from, to, err
}

func cmpOrString(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// thePeriod is « la période » of a scenario: the one proposed by default, from tomorrow to
// a week later (D24); its dishes are served on its first evening.
func (w *world) thePeriod() (from, to string, first meal) {
	start := w.now.AddDate(0, 0, 1)
	from, to = start.Format(time.DateOnly), start.AddDate(0, 0, 6).Format(time.DateOnly)
	return from, to, meal{date: from, mealtime: "dinner"}
}

// registerMealShoppingSteps: the meal shopping of liste-courses.feature (C4, C5), and P2 of
// bibliotheque-plats.feature.
func (w *world) registerMealShoppingSteps(sc *godog.ScenarioContext) {
	sc.StepContext().Before(func(ctx context.Context, _ *godog.Step) (context.Context, error) {
		w.step++
		return ctx, nil
	})
	sc.Step(`^le plat "([^"]*)"( sans ingrédient)? servi pour (\d+) parts? dans la période$`, func(ctx context.Context, name, _ string, servings int) error {
		_, _, m := w.thePeriod()
		return w.givenServed(ctx, name, servings, m)
	})
	sc.Step(`^chacun de ces plats servi pour (\d+) parts? dans la période$`, w.eachServed)
	sc.Step(`^(?:j'ajoute|je tente de ajouter) les courses des repas (de la période|du .+?) à la liste "([^"]*)"$`, func(ctx context.Context, period, list string) error {
		return w.addMealShopping(ctx, period, list)
	})
	sc.Step(`^j'ai ajouté les courses des repas (du .+?) à la liste "([^"]*)"$`, func(ctx context.Context, period, list string) error {
		if err := w.addMealShopping(ctx, period, list); err != nil {
			return err
		}
		return w.shoppingStatus(http.StatusOK)
	})
	sc.Step(`^l'ajout des courses est impossible$`, func() error { return w.shoppingRefused() })
	sc.Step(`^un message m'indique que la date de fin doit suivre la date de début$`, func() error { return w.shoppingError(400, "invalid_period") })
	sc.Step(`^un message m'indique que la période ne peut pas dépasser 31 jours$`, func() error { return w.shoppingError(400, "period_too_long") })
	sc.Step(`^un message m'indique que (\d+) plats? servis? déjà reçus? par la liste (?:a|ont) été ignorés?$`, w.ignoredCount)

	sc.Step(`^la liste "([^"]*)" est vide$`, func(ctx context.Context, list string) error {
		l, err := w.listNamed(ctx, "je", list)
		if err == nil && len(l.Items) != 0 {
			err = fmt.Errorf("list %s holds %+v", list, l.Items)
		}
		return err
	})
	sc.Step(`^la liste "([^"]*)" contient (\d\S* [^"]+?) de "([^"]*)"(?: provenant de (\d+) plats? servis?)?$`, func(ctx context.Context, list, quantity, ingredient, sources string) error {
		return w.holdsComputed(ctx, list, quantity, ingredient, sources)
	})
	sc.Step(`^chacune des listes "([^"]*)" et "([^"]*)" contient (\d\S* [^"]+?) de "([^"]*)"$`, func(ctx context.Context, a, b, quantity, ingredient string) error {
		for _, list := range []string{a, b} {
			if err := w.holdsComputed(ctx, list, quantity, ingredient, ""); err != nil {
				return err
			}
		}
		return nil
	})
	sc.Step(`^la liste "([^"]*)" contient une seule ligne "([^"]*)"(?: de (.+)|, sans quantité)$`, w.onlyLine)
	sc.Step(`^la liste "([^"]*)" contient "([^"]*)" sans quantité$`, func(ctx context.Context, list, ingredient string) error {
		return w.holdsComputed(ctx, list, "", ingredient, "")
	})
	sc.Step(`^la ligne "([^"]*)" indique qu'elle provient de (\d+) plats? servis?$`, w.sourcesCount)

	// P2
	sc.Step(`^je modifie la quantité de "([^"]*)" du plat "([^"]*)" à (.+)$`, func(ctx context.Context, ingredient, name, quantity string) error {
		return w.changeLine(ctx, "je", ingredient, name, quantity)
	})
}

// recordDish remembers a dish defined by the scenario, for « chacun de ces plats ».
func (w *world) recordDish(name string) {
	w.mealShopping.defined = append(w.mealShopping.defined, definedDish{name: name, step: w.step})
}

// eachServed serves, on the first evening of the period, each dish defined by the steps just
// before this one: the run of definitions that ends at the previous step. A dish defined by
// the background just before the scenario belongs to the run: in liste-courses.feature, the
// "Chili con carne" brings no ingredient of the lines these scenarios check.
func (w *world) eachServed(ctx context.Context, servings int) error {
	var names []string
	step := w.step - 1
	for i := len(w.mealShopping.defined) - 1; i >= 0 && w.mealShopping.defined[i].step == step; i-- {
		names = append([]string{w.mealShopping.defined[i].name}, names...)
		step--
	}
	if len(names) < 2 {
		return fmt.Errorf("dishes defined just before: %v", names)
	}
	_, _, m := w.thePeriod()
	for _, name := range names {
		if err := w.givenServed(ctx, name, servings, m); err != nil {
			return err
		}
	}
	return nil
}

// addMealShopping adds the shopping of the meals of the period to the list: as "je" opened
// it, if so (D11).
func (w *world) addMealShopping(ctx context.Context, period, list string) error {
	from, to, _ := w.thePeriod()
	if period != "de la période" {
		var err error
		if from, to, err = parsePeriod(period); err != nil {
			return err
		}
	}
	id, err := w.openedListID(ctx, list)
	if err != nil {
		return err
	}
	return w.shoppingSend(ctx, "je", http.MethodPost, "shopping-lists/"+id+"/meal-shopping", map[string]string{"from": from, "to": to})
}

func (w *world) ignoredCount(n int) error {
	if err := w.shoppingStatus(http.StatusOK); err != nil {
		return err
	}
	var r struct {
		Ignored *int `json:"ignored"`
	}
	if err := json.Unmarshal(w.shopping.last.Body.Bytes(), &r); err != nil {
		return err
	}
	if r.Ignored == nil || *r.Ignored != n {
		return fmt.Errorf("ignored %v, want %d; body %s", r.Ignored, n, w.shopping.last.Body)
	}
	return nil
}

// holdsComputed checks that the list holds one computed item of this ingredient and
// quantity ("" for an item without quantity), and from this number of served dishes if given.
func (w *world) holdsComputed(ctx context.Context, list, quantity, ingredient, sources string) error {
	l, err := w.listNamed(ctx, "je", list)
	if err != nil {
		return err
	}
	found, err := findItems(l, ingredient, "calculé", quantity)
	if err != nil {
		return err
	}
	if len(found) != 1 {
		return fmt.Errorf("list %s: %d computed items %q %q in %+v", list, len(found), ingredient, quantity, l.Items)
	}
	if n, _ := strconv.Atoi(sources); sources != "" && len(found[0].Sources) != n {
		return fmt.Errorf("list %s: %q from %d served dishes, want %s", list, ingredient, len(found[0].Sources), sources)
	}
	return nil
}

// onlyLine checks that the list holds one line of this ingredient, of this quantity or
// without one.
func (w *world) onlyLine(ctx context.Context, list, ingredient, quantity string) error {
	l, err := w.listNamed(ctx, "je", list)
	if err != nil {
		return err
	}
	all, err := findItems(l, ingredient, "", "-")
	if err != nil {
		return err
	}
	found, err := findItems(l, ingredient, "", quantity)
	if err != nil {
		return err
	}
	if len(all) != 1 || len(found) != 1 {
		return fmt.Errorf("list %s: lines %q %+v, want one of %q", list, ingredient, all, quantity)
	}
	return nil
}

// sourcesCount checks the number of served dishes the line of this ingredient comes from, in
// the list of the last write, as read now (C5).
func (w *world) sourcesCount(ctx context.Context, ingredient string, n int) error {
	if w.shopping.last == nil {
		return errors.New("nothing sent on the lists")
	}
	written, err := w.writtenList()
	if err != nil {
		return err
	}
	lists, err := w.shoppingLists(ctx, "je")
	if err != nil {
		return err
	}
	for _, l := range lists {
		if l.ID != written.ID {
			continue
		}
		found, err := findItems(l, ingredient, "", "-")
		if err != nil {
			return err
		}
		if len(found) != 1 || len(found[0].Sources) != n {
			return fmt.Errorf("list %s: lines %q %+v, want one from %d served dishes", l.Name, ingredient, found, n)
		}
		return nil
	}
	return fmt.Errorf("no list %s", written.ID)
}
