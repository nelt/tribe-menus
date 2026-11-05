package acceptance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/nelt/tribe-menus/internal/dish"
	"github.com/nelt/tribe-menus/internal/mealplan"
	"github.com/nelt/tribe-menus/internal/storage"
)

// planningState is what the meal plan scenarios remember from step to step.
type planningState struct {
	// meal is the last meal the scenario named ("ce repas", "le même repas").
	meal meal
	// last is the response to the last write on the meal plan.
	last *httptest.ResponseRecorder
	// opened is the meal "je" opened, as read then: its served dishes keep the identifiers
	// they had, removed in between or not.
	opened []namedServed
	// seen is the meal plan "je" consulted.
	seen []apiServed
	// foreign is the served dish of another tribe, before the attempt on it.
	foreign mealplan.ServedDish
}

// meal is a date AAAA-MM-JJ and a mealtime of the API.
type meal struct {
	date, mealtime string
}

type apiServed struct {
	ID       string `json:"id"`
	Date     string `json:"date"`
	Mealtime string `json:"mealtime"`
	Dish     string `json:"dish"`
	Servings int    `json:"servings"`
	Rank     int64  `json:"rank"`
}

// namedServed is a served dish with the name of its dish.
type namedServed struct {
	apiServed
	name string
}

var (
	weekdays = map[string]time.Weekday{
		"lundi": time.Monday, "mardi": time.Tuesday, "mercredi": time.Wednesday, "jeudi": time.Thursday,
		"vendredi": time.Friday, "samedi": time.Saturday, "dimanche": time.Sunday,
	}
	months = map[string]time.Month{
		"janvier": time.January, "février": time.February, "mars": time.March, "avril": time.April,
		"mai": time.May, "juin": time.June, "juillet": time.July, "août": time.August,
		"septembre": time.September, "octobre": time.October, "novembre": time.November, "décembre": time.December,
	}
	dayText = regexp.MustCompile(`^(\S+) (\d+)(?:er)? (\S+) (\d{4})$`)
)

// parseDay reads a day of the Gherkin conventions, "mardi 6 octobre 2026" or "samedi 1er
// janvier 2000", and checks its day of the week.
func parseDay(text string) (time.Time, error) {
	m := dayText.FindStringSubmatch(text)
	if m == nil {
		return time.Time{}, fmt.Errorf("day %q", text)
	}
	weekday, okDay := weekdays[m[1]]
	month, okMonth := months[m[3]]
	day, _ := strconv.Atoi(m[2])
	year, _ := strconv.Atoi(m[4])
	if !okDay || !okMonth {
		return time.Time{}, fmt.Errorf("day %q", text)
	}
	t := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	if t.Day() != day || t.Weekday() != weekday {
		return time.Time{}, fmt.Errorf("day %q: %s is a %s", text, t.Format(time.DateOnly), t.Weekday())
	}
	return t, nil
}

// parseMeal reads a day and "à midi" or "au soir".
func parseMeal(day, moment string) (meal, error) {
	t, err := parseDay(day)
	if err != nil {
		return meal{}, err
	}
	mealtime := "lunch"
	if moment == "au soir" {
		mealtime = "dinner"
	}
	return meal{date: t.Format(time.DateOnly), mealtime: mealtime}, nil
}

// mealPattern is a meal of a step: the day, then "à midi" or "au soir".
const mealPattern = `(.+?) (à midi|au soir)`

// registerMealPlanSteps: planning-repas.feature, and the meal plan scenarios of tribu.feature
// and compartimentage-tribus.feature.
func (w *world) registerMealPlanSteps(sc *godog.ScenarioContext) {
	sc.Step(`^(?:que )?nous sommes le (.+)$`, w.weAre)
	sc.Step(`^la bibliothèque contient les plats "([^"]*)" et "([^"]*)"$`, func(ctx context.Context, a, b string) error {
		for _, name := range []string{a, b} {
			if _, err := w.ensureDish(ctx, name, dishBody{Name: name}); err != nil {
				return err
			}
		}
		return nil
	})

	// R2 to R5
	sc.Step(`^(j'|Alice |Bruno )ajoute le plat "([^"]*)" au repas du `+mealPattern+`$`, func(ctx context.Context, who, name, day, moment string) error {
		m, err := parseMeal(day, moment)
		if err != nil {
			return err
		}
		return w.serveNamed(ctx, subject(who), name, m)
	})
	sc.Step(`^j'ajoute (?:de nouveau )?(?:le plat )?"([^"]*)" (?:au même repas|à ce repas)$`, func(ctx context.Context, name string) error {
		return w.serveNamed(ctx, "je", name, w.planning.meal)
	})
	sc.Step(`^le plat "([^"]*)" servi pour (\d+) parts? le `+mealPattern+`$`, func(ctx context.Context, name string, servings int, day, moment string) error {
		m, err := parseMeal(day, moment)
		if err != nil {
			return err
		}
		return w.givenServed(ctx, name, servings, m)
	})
	sc.Step(`^le plat "([^"]*)" servi pour (\d+) parts? le même repas$`, func(ctx context.Context, name string, servings int) error {
		return w.givenServed(ctx, name, servings, w.planning.meal)
	})
	sc.Step(`^le repas du `+mealPattern+` contient (?:toujours )?"([^"]*)" pour (\d+) parts?$`, func(ctx context.Context, day, moment, name string, servings int) error {
		return w.mealHolds(ctx, "je", day, moment, name, servings)
	})
	sc.Step(`^(Alice|Bruno) voit "([^"]*)" au repas du `+mealPattern+`, après rafraîchissement$`, func(ctx context.Context, who, name, day, moment string) error {
		return w.mealHolds(ctx, who, day, moment, name, 0)
	})
	sc.Step(`^le repas du `+mealPattern+` contient :$`, func(ctx context.Context, day, moment string, table *godog.Table) error {
		return w.mealIs(ctx, day, moment, "", table)
	})
	sc.Step(`^le repas du `+mealPattern+` contient, dans cet ordre :$`, func(ctx context.Context, day, moment string, table *godog.Table) error {
		return w.mealIs(ctx, day, moment, "in order", table)
	})
	sc.Step(`^le repas du `+mealPattern+` ne contient aucun plat$`, func(ctx context.Context, day, moment string) error {
		m, err := parseMeal(day, moment)
		if err != nil {
			return err
		}
		served, err := w.mealOf(ctx, "je", m)
		if err != nil {
			return err
		}
		if len(served) != 0 {
			return fmt.Errorf("meal %v holds %+v", m, served)
		}
		return nil
	})
	sc.Step(`^le repas du `+mealPattern+` ne contient plus "([^"]*)"$`, func(ctx context.Context, day, moment, name string) error {
		m, err := parseMeal(day, moment)
		if err != nil {
			return err
		}
		served, err := w.mealOf(ctx, "je", m)
		if err != nil {
			return err
		}
		if slices.ContainsFunc(served, func(s namedServed) bool { return s.name == name }) {
			return fmt.Errorf("meal %v holds %+v", m, served)
		}
		return nil
	})
	sc.Step(`^j'ajoute au repas du `+mealPattern+` un plat dont l'identifiant est inconnu de la bibliothèque$`, func(ctx context.Context, day, moment string) error {
		m, err := parseMeal(day, moment)
		if err != nil {
			return err
		}
		return w.serve(ctx, "je", m, strings.Repeat("0", 32), nil)
	})
	sc.Step(`^l'ajout est refusé$`, w.planningRefused)
	sc.Step(`^le réglage est refusé$`, w.planningRefused)
	sc.Step(`^un message m'indique que "([^"]*)" est déjà servi dans ce repas$`, func(ctx context.Context, name string) error {
		_, err := w.alreadyServed(ctx, name)
		return err
	})
	sc.Step(`^le repas à jour m'est présenté, avec "([^"]*)" pour (\d+) parts?$`, func(ctx context.Context, name string, servings int) error {
		meal, err := w.alreadyServed(ctx, name)
		if err != nil {
			return err
		}
		if len(meal) != 1 || meal[0].Servings != servings {
			return fmt.Errorf("meal presented %+v, want %s for %d", meal, name, servings)
		}
		return nil
	})
	sc.Step(`^j'ai ouvert le repas du `+mealPattern+`$`, func(ctx context.Context, day, moment string) error {
		m, err := parseMeal(day, moment)
		if err != nil {
			return err
		}
		w.planning.meal = m
		w.planning.opened, err = w.mealOf(ctx, "je", m)
		return err
	})
	sc.Step(`^Bruno a ajouté entre-temps le plat "([^"]*)" à ce repas$`, func(ctx context.Context, name string) error {
		if err := w.serveNamed(ctx, "Bruno", name, w.planning.meal); err != nil {
			return err
		}
		return w.planningStatus(http.StatusCreated)
	})
	sc.Step(`^je retire "([^"]*)" de ce repas$`, func(ctx context.Context, name string) error {
		return w.remove(ctx, "je", name)
	})
	sc.Step(`^Bruno a retiré entre-temps "([^"]*)" de ce repas$`, func(ctx context.Context, name string) error {
		if err := w.remove(ctx, "Bruno", name); err != nil {
			return err
		}
		return w.planningStatus(http.StatusOK)
	})
	sc.Step(`^je règle "([^"]*)" à (\d+) parts? et "([^"]*)" à (\d+) parts?$`, func(ctx context.Context, a string, na int, b string, nb int) error {
		if err := w.setServings(ctx, "je", a, na); err != nil {
			return err
		}
		if err := w.planningStatus(http.StatusOK); err != nil {
			return err
		}
		return w.setServings(ctx, "je", b, nb)
	})
	sc.Step(`^je règle "([^"]*)" à (\d+) parts?$`, func(ctx context.Context, name string, servings int) error {
		return w.setServings(ctx, "je", name, servings)
	})
	sc.Step(`^Bruno a réglé entre-temps "([^"]*)" à (\d+) parts? dans ce repas$`, func(ctx context.Context, name string, servings int) error {
		if err := w.setServings(ctx, "Bruno", name, servings); err != nil {
			return err
		}
		return w.planningStatus(http.StatusOK)
	})
	sc.Step(`^un message m'indique que le plat a été retiré du repas entre-temps$`, func() error {
		if err := w.planningStatus(http.StatusNotFound); err != nil {
			return err
		}
		return w.planningError("not_found")
	})

	// T3
	sc.Step(`^(Alice|Bruno) règle la taille de la tribu à (\d+)$`, func(ctx context.Context, who string, size int) error {
		rec, err := w.api(ctx, who, http.MethodPut, "tribe/size", map[string]int{"size": size})
		if err != nil {
			return err
		}
		if rec.Code != http.StatusOK {
			return fmt.Errorf("PUT tribe/size: status %d, body %s", rec.Code, rec.Body)
		}
		return nil
	})

	// ENF-02
	sc.Step(`^le plat "([^"]*)" servi pour (\d+) parts? le `+mealPattern+` dans la tribu "([^"]*)"$`, w.servedInOtherTribe)
	sc.Step(`^je consulte le planning$`, func(ctx context.Context) error {
		served, err := w.mealPlan(ctx, "je")
		w.planning.seen = served
		return err
	})
	sc.Step(`^je tente de (régler les parts de|retirer) ce plat servi de la tribu "([^"]*)" en utilisant son identifiant$`, w.crossAccessServed)
	sc.Step(`^le plat servi de la tribu "([^"]*)" est inchangé$`, w.otherServedUnchanged)
	sc.Step(`^j'ajoute au repas du `+mealPattern+` le plat "([^"]*)" de la tribu "([^"]*)" en utilisant son identifiant$`, w.serveOtherTribeDish)
}

// subject reads "j'", "Alice " or "Bruno ".
func subject(who string) string {
	if who == "j'" {
		return "je"
	}
	return strings.TrimSpace(who)
}

// weAre sets the date of the scenario. The server knows no "today" (D4): only the steps of
// the browser use it.
func (w *world) weAre(text string) error {
	t, err := parseDay(text)
	if err != nil {
		return err
	}
	w.now = t.Add(time.Duration(today.Hour()) * time.Hour)
	return nil
}

// mealPlan reads every page of the meal plan, as the front end does (ADR 0029).
func (w *world) mealPlan(ctx context.Context, who string) ([]apiServed, error) {
	return pages[apiServed](ctx, w, who, "meal-plan")
}

// mealOf returns the dishes served in the meal, by rank, with the names of their dishes.
func (w *world) mealOf(ctx context.Context, who string, m meal) ([]namedServed, error) {
	plan, err := w.mealPlan(ctx, who)
	if err != nil {
		return nil, err
	}
	var served []apiServed
	for _, s := range plan {
		if s.Date == m.date && s.Mealtime == m.mealtime {
			served = append(served, s)
		}
	}
	return w.named(ctx, who, served)
}

// named gives the served dishes the names of their dishes, by rank.
func (w *world) named(ctx context.Context, who string, served []apiServed) ([]namedServed, error) {
	dishes, err := w.library(ctx, who)
	if err != nil {
		return nil, err
	}
	names := map[string]string{}
	for _, d := range dishes {
		names[d.ID] = d.Name
	}
	out := make([]namedServed, 0, len(served))
	for _, s := range served {
		name, ok := names[s.Dish]
		if !ok {
			return nil, fmt.Errorf("served dish %+v of a dish absent from the library", s)
		}
		out = append(out, namedServed{apiServed: s, name: name})
	}
	slices.SortFunc(out, func(a, b namedServed) int { return int(a.Rank - b.Rank) })
	return out, nil
}

// serve sends a dish to serve in a meal; the response is kept for the outcome steps.
func (w *world) serve(ctx context.Context, who string, m meal, dishID string, servings *int) error {
	w.planning.meal = m
	body := map[string]any{"date": m.date, "mealtime": m.mealtime, "dish": dishID}
	if servings != nil {
		body["servings"] = *servings
	}
	rec, err := w.api(ctx, who, http.MethodPost, "meal-plan", body)
	w.planning.last = rec
	return err
}

// serveNamed serves the dish of this name, created by the same person if the library does
// not hold it yet.
func (w *world) serveNamed(ctx context.Context, who, name string, m meal) error {
	d, err := w.findDish(ctx, who, name)
	if err != nil {
		if d, err = w.mustSave(ctx, who, http.MethodPost, "dishes", dishBody{Name: name}); err != nil {
			return err
		}
	}
	return w.serve(ctx, who, m, d.ID, nil)
}

// givenServed serves the dish for these servings in a context step: it must be served.
func (w *world) givenServed(ctx context.Context, name string, servings int, m meal) error {
	d, err := w.ensureDish(ctx, name, dishBody{Name: name})
	if err != nil {
		return err
	}
	if err := w.serve(ctx, "je", m, d.ID, &servings); err != nil {
		return err
	}
	return w.planningStatus(http.StatusCreated)
}

func (w *world) planningStatus(status int) error {
	if w.planning.last == nil {
		return errors.New("nothing sent on the meal plan")
	}
	if w.planning.last.Code != status {
		return fmt.Errorf("status %d, want %d; body %s", w.planning.last.Code, status, w.planning.last.Body)
	}
	return nil
}

func (w *world) planningError(code string) error {
	var e struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.planning.last.Body.Bytes(), &e); err != nil {
		return err
	}
	if e.Error != code {
		return fmt.Errorf("error %q, want %q", e.Error, code)
	}
	return nil
}

func (w *world) planningRefused() error {
	if w.planning.last == nil {
		return errors.New("nothing sent on the meal plan")
	}
	if c := w.planning.last.Code; c < 400 || c >= 500 {
		return fmt.Errorf("status %d, body %s", c, w.planning.last.Body)
	}
	return nil
}

// alreadyServed checks the refusal of a dish already served in the meal (Q21, D3), and
// returns the meal it presents, with the names of its dishes.
func (w *world) alreadyServed(ctx context.Context, name string) ([]namedServed, error) {
	if err := w.planningStatus(http.StatusConflict); err != nil {
		return nil, err
	}
	if err := w.planningError("dish_already_served"); err != nil {
		return nil, err
	}
	var e struct {
		Meal []apiServed `json:"meal"`
	}
	if err := json.Unmarshal(w.planning.last.Body.Bytes(), &e); err != nil {
		return nil, err
	}
	meal, err := w.named(ctx, "je", e.Meal)
	if err != nil {
		return nil, err
	}
	if !slices.ContainsFunc(meal, func(s namedServed) bool { return s.name == name }) {
		return nil, fmt.Errorf("meal presented %+v, without %s", meal, name)
	}
	return meal, nil
}

// mealHolds checks that the meal holds the dish, for these servings unless 0.
func (w *world) mealHolds(ctx context.Context, who, day, moment, name string, servings int) error {
	m, err := parseMeal(day, moment)
	if err != nil {
		return err
	}
	served, err := w.mealOf(ctx, who, m)
	if err != nil {
		return err
	}
	for _, s := range served {
		if s.name == name {
			if servings != 0 && s.Servings != servings {
				return fmt.Errorf("%s for %d servings, want %d", name, s.Servings, servings)
			}
			return nil
		}
	}
	return fmt.Errorf("meal %v holds %+v, without %s", m, served, name)
}

// mealIs checks the dishes of the meal and their servings, in their order if asked.
func (w *world) mealIs(ctx context.Context, day, moment, ordered string, table *godog.Table) error {
	m, err := parseMeal(day, moment)
	if err != nil {
		return err
	}
	served, err := w.mealOf(ctx, "je", m)
	if err != nil {
		return err
	}
	var got, want []string
	for _, s := range served {
		got = append(got, fmt.Sprintf("%s %d", s.name, s.Servings))
	}
	for _, row := range table.Rows[1:] {
		want = append(want, row.Cells[0].Value+" "+row.Cells[1].Value)
	}
	if ordered == "" {
		slices.Sort(got)
		slices.Sort(want)
	}
	if !slices.Equal(got, want) {
		return fmt.Errorf("meal %v holds %q, want %q", m, got, want)
	}
	return nil
}

// servedIn returns the served dish of this name in the current meal: as "je" opened it, if
// so, otherwise as the person reads it now.
func (w *world) servedIn(ctx context.Context, who, name string) (namedServed, error) {
	served := w.planning.opened
	if who != "je" || served == nil {
		var err error
		if served, err = w.mealOf(ctx, who, w.planning.meal); err != nil {
			return namedServed{}, err
		}
	}
	for _, s := range served {
		if s.name == name {
			return s, nil
		}
	}
	return namedServed{}, fmt.Errorf("no %s in the meal %v: %+v", name, w.planning.meal, served)
}

func (w *world) remove(ctx context.Context, who, name string) error {
	s, err := w.servedIn(ctx, who, name)
	if err != nil {
		return err
	}
	rec, err := w.api(ctx, who, http.MethodDelete, "meal-plan/"+s.ID, nil)
	w.planning.last = rec
	return err
}

func (w *world) setServings(ctx context.Context, who, name string, servings int) error {
	s, err := w.servedIn(ctx, who, name)
	if err != nil {
		return err
	}
	rec, err := w.api(ctx, who, http.MethodPut, "meal-plan/"+s.ID+"/servings", map[string]int{"servings": servings})
	w.planning.last = rec
	return err
}

// otherPlan returns the meal plan of another tribe, used through the domain.
func (w *world) otherPlan(ctx context.Context, slug string) (*mealplan.Plan, error) {
	store, err := w.openStore(ctx)
	if err != nil {
		return nil, err
	}
	db, err := store.Tribe(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("tribe %q: %w", slug, err)
	}
	return mealplan.NewPlan(storage.NewMealPlanStore(db)), nil
}

// otherDishID returns the identifier of the dish of this name in another tribe.
func (w *world) otherDishID(ctx context.Context, slug, name string) (dish.ID, error) {
	library, err := w.otherLibrary(ctx, slug)
	if err != nil {
		return "", err
	}
	d, err := otherDish(ctx, library, name)
	return d.ID, err
}

func (w *world) servedInOtherTribe(ctx context.Context, name string, servings int, day, moment, slug string) error {
	m, err := parseMeal(day, moment)
	if err != nil {
		return err
	}
	id, err := w.otherDishID(ctx, slug, name)
	if err != nil {
		return err
	}
	plan, err := w.otherPlan(ctx, slug)
	if err != nil {
		return err
	}
	w.planning.foreign, _, err = plan.Serve(ctx, mealplan.ServeInput{Date: m.date, Mealtime: m.mealtime, Dish: id, Servings: &servings})
	return err
}

// otherServed returns the served dishes of another tribe.
func (w *world) otherServed(ctx context.Context, slug string) ([]mealplan.ServedDish, error) {
	plan, err := w.otherPlan(ctx, slug)
	if err != nil {
		return nil, err
	}
	_, page, err := plan.ServedDishes(ctx, "")
	return page.Items, err
}

// crossAccessServed tries to set the servings of, or remove, the served dish of another
// tribe by its identifier, then an identifier that exists nowhere, the same way.
func (w *world) crossAccessServed(ctx context.Context, action, slug string) error {
	attempt := func(id string) (*httptest.ResponseRecorder, error) {
		if action == "retirer" {
			return w.api(ctx, "je", http.MethodDelete, "meal-plan/"+id, nil)
		}
		return w.api(ctx, "je", http.MethodPut, "meal-plan/"+id+"/servings", map[string]int{"servings": w.planning.foreign.Servings + 1})
	}
	var err error
	if w.kitchen.foreign, err = attempt(string(w.planning.foreign.ID)); err != nil {
		return err
	}
	w.kitchen.unknown, err = attempt(strings.Repeat("0", len(w.planning.foreign.ID)))
	return err
}

func (w *world) otherServedUnchanged(ctx context.Context, slug string) error {
	served, err := w.otherServed(ctx, slug)
	if err != nil {
		return err
	}
	if len(served) != 1 || served[0] != w.planning.foreign {
		return fmt.Errorf("meal plan of %s %+v, before %+v", slug, served, w.planning.foreign)
	}
	return nil
}

// serveOtherTribeDish serves the dish of another tribe by its identifier, then a dish that
// exists nowhere, the same way.
func (w *world) serveOtherTribeDish(ctx context.Context, day, moment, name, slug string) error {
	m, err := parseMeal(day, moment)
	if err != nil {
		return err
	}
	id, err := w.otherDishID(ctx, slug, name)
	if err != nil {
		return err
	}
	if err := w.serve(ctx, "je", m, strings.Repeat("0", len(id)), nil); err != nil {
		return err
	}
	w.kitchen.unknown = w.planning.last
	if err := w.serve(ctx, "je", m, string(id), nil); err != nil {
		return err
	}
	w.kitchen.foreign = w.planning.last
	return nil
}

// nothingOfOtherPlan: neither the served dishes nor the dishes of the other tribe are in the
// meal plan "je" consulted.
func (w *world) nothingOfOtherPlan(ctx context.Context, slug string) error {
	served, err := w.otherServed(ctx, slug)
	if err != nil {
		return err
	}
	seen, err := json.Marshal(w.planning.seen)
	if err != nil {
		return err
	}
	for _, s := range served {
		for _, secret := range []string{string(s.ID), string(s.Dish)} {
			if strings.Contains(string(seen), secret) {
				return fmt.Errorf("%q of %s seen in %s", secret, slug, seen)
			}
		}
	}
	return nil
}
