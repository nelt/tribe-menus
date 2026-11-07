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
	"strings"

	"github.com/cucumber/godog"
	"github.com/nelt/tribe-menus/internal/dish"
	"github.com/nelt/tribe-menus/internal/storage"
)

// kitchenSlug is the tribe of the dish scenarios that name none.
const kitchenSlug = "cuisine"

// emails of the people of the dish scenarios, "je" being a member of their own.
var personEmails = map[string]string{
	"je":    "moi@exemple.fr",
	"Alice": "alice@exemple.fr",
	"Bruno": "bruno@exemple.fr",
}

// kitchenState is what the dish scenarios remember from step to step (plan bibliotheque,
// D15: searches and suggestions are proved against the API by the absence of an item from
// GET dishes and GET ingredients).
type kitchenState struct {
	// people are the devices of the people of the scenario, signed in to its tribe.
	people map[string]*device
	// last is the response to the last action on a dish; saved, the last dish saved.
	last  *httptest.ResponseRecorder
	saved apiDish
	// countBefore is the number of dishes before the last creation.
	countBefore int
	// lastName is the name of the last dish the scenario created.
	lastName string
	// opened are the dishes "je" opened, by name.
	opened map[string]apiDish
	// seen is the library "je" consulted; typed, the text typed as an ingredient.
	seen  []apiDish
	typed string
	// foreign and unknown are the answers to an attempt on a dish of another tribe and on
	// an identifier that does not exist.
	foreign, unknown *httptest.ResponseRecorder
	// foreignName is the dish of the other tribe, foreignBefore what it was before the
	// attempt; versionsBefore, the versions of the collections of the other tribe.
	foreignName    string
	foreignBefore  dish.Dish
	versionsBefore [2]int64
	n              int
}

type apiDish struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Color       string    `json:"color"`
	Servings    int       `json:"servings"`
	Revision    int64     `json:"revision"`
	Ingredients []apiLine `json:"ingredients"`
}

type apiLine struct {
	Name        string `json:"name"`
	Quantity    string `json:"quantity,omitempty"`
	Unit        string `json:"unit,omitempty"`
	DefaultUnit string `json:"defaultUnit,omitempty"`
}

type apiIngredient struct {
	Name        string `json:"name"`
	DefaultUnit string `json:"defaultUnit"`
}

// dishBody is a dish as the front end sends it.
type dishBody struct {
	Revision    *int64    `json:"revision,omitempty"`
	Name        string    `json:"name"`
	Servings    int       `json:"servings,omitempty"`
	Ingredients []apiLine `json:"ingredients"`
}

func (d apiDish) body() dishBody {
	lines := make([]apiLine, len(d.Ingredients))
	for i, l := range d.Ingredients {
		lines[i] = apiLine{Name: l.Name, Quantity: l.Quantity, Unit: l.Unit}
	}
	return dishBody{Revision: &d.Revision, Name: d.Name, Servings: d.Servings, Ingredients: lines}
}

// unitCodes are the units of the scenarios, in French, by their code.
var unitCodes = map[string]string{
	"g": "g", "kg": "kg", "ml": "ml", "cl": "cl", "l": "l",
	"pièce": "piece", "pièces": "piece",
	"cuillère à soupe": "tablespoon", "cuillères à soupe": "tablespoon",
	"cuillère à café": "teaspoon", "cuillères à café": "teaspoon",
	"pincée": "pinch", "pincées": "pinch",
}

func unitCode(label string) (string, error) {
	code, ok := unitCodes[label]
	if !ok {
		return "", fmt.Errorf("unknown unit %q", label)
	}
	return code, nil
}

var quantityText = regexp.MustCompile(`^([0-9]+(?:[.,][0-9]+)?) (.+)$`)

// parseQuantity reads "500 g" or "1 cuillère à café": the quantity as the API gives it back,
// and the unit code.
func parseQuantity(text string) (quantity, unit string, err error) {
	m := quantityText.FindStringSubmatch(text)
	if m == nil {
		return "", "", fmt.Errorf("quantity %q", text)
	}
	unit, err = unitCode(m[2])
	return strings.ReplaceAll(m[1], ",", "."), unit, err
}

// registerDishSteps: bibliotheque-plats.feature, tribu.feature and the dish scenarios of
// compartimentage-tribus.feature.
func (w *world) registerDishSteps(sc *godog.ScenarioContext) {
	// P1
	sc.Step(`^je crée le plat "([^"]*)" avec les ingrédients :$`, w.createWithTable)
	sc.Step(`^je crée le plat "([^"]*)"$`, func(ctx context.Context, name string) error {
		return w.create(ctx, "je", dishBody{Name: name})
	})
	sc.Step(`^je crée le plat "([^"]*)" pour (\d+) parts$`, func(ctx context.Context, name string, servings int) error {
		return w.create(ctx, "je", dishBody{Name: name, Servings: servings})
	})
	sc.Step(`^je crée le plat "([^"]*)" sans ingrédient$`, func(ctx context.Context, name string) error {
		return w.create(ctx, "je", dishBody{Name: name})
	})
	sc.Step(`^je crée un plat sans nom$`, func(ctx context.Context) error {
		return w.create(ctx, "je", dishBody{Ingredients: []apiLine{{Name: "sel"}}})
	})
	sc.Step(`^le plat "([^"]*)" figure (?:toujours )?dans la bibliothèque$`, func(ctx context.Context, name string) error {
		_, err := w.findDish(ctx, "je", name)
		return err
	})
	sc.Step(`^le plat "([^"]*)" dans la bibliothèque$`, func(ctx context.Context, name string) error {
		_, err := w.ensureDish(ctx, name, dishBody{Name: name})
		return err
	})
	sc.Step(`^il est défini pour (\d+) parts$`, func(ctx context.Context, servings int) error {
		return w.servingsAre(ctx, w.kitchen.lastName, servings)
	})
	sc.Step(`^le plat "([^"]*)" est défini pour (\d+) parts$`, w.servingsAre)
	sc.Step(`^la bibliothèque contient un plat de chaque teinte de la palette, sauf la teinte "([^"]*)"$`, w.allHuesBut)
	sc.Step(`^le plat "([^"]*)" a (?:toujours )?la teinte "([^"]*)"$`, w.hasHue)
	sc.Step(`^je modifie le plat "([^"]*)"$`, func(ctx context.Context, name string) error {
		return w.modify(ctx, "je", name)
	})
	sc.Step(`^le plat n'est pas enregistré$`, w.notSaved)
	sc.Step(`^un message m'indique que le nom est obligatoire$`, func() error { return w.dishError(400, "name_required", "") })
	sc.Step(`^un message m'indique qu'un plat porte déjà ce nom$`, func() error { return w.dishError(400, "name_taken", "") })
	sc.Step(`^un message m'indique que l'ingrédient "([^"]*)" figure déjà dans le plat$`, func(name string) error {
		return w.dishError(400, "duplicate_ingredient", name)
	})
	sc.Step(`^je crée un plat dont le nom (du plat|d'un ingrédient) est (.+)$`, w.createWithOddName)
	sc.Step(`^le nom (du plat|d'un ingrédient) est refusé comme (vide|invalide)$`, func(of, refusal string) error {
		return w.dishError(400, oddNameErrors[of+" "+refusal], "")
	})
	sc.Step(`^l'ingrédient "([^"]*)" du plat "([^"]*)" n'a pas de quantité$`, w.lineWithoutQuantity)
	sc.Step(`^l'ingrédient "([^"]*)" n'a pas d'unité par défaut$`, func(ctx context.Context, name string) error {
		return w.defaultUnitIs(ctx, "je", name, "")
	})

	// P2
	sc.Step(`^aucune action ne permet de supprimer le plat "([^"]*)"$`, w.cannotDelete)
	sc.Step(`^le plat "([^"]*)" défini pour (\d+) parts avec (.+) de "([^"]*)"$`, w.givenDishWith)
	sc.Step(`^j'ai ouvert le plat "([^"]*)"$`, w.openDish)
	sc.Step(`^Bruno a modifié entre-temps la quantité de "([^"]*)" du plat "([^"]*)" à (.+)$`, w.changedInBetween)
	sc.Step(`^j'ajoute (.+) de "([^"]*)" au plat "([^"]*)"$`, w.addLine)
	sc.Step(`^mon enregistrement est refusé$`, func() error { return w.lastDishStatus(http.StatusConflict) })
	sc.Step(`^un message m'indique que le plat a été modifié entre-temps$`, func() error { return w.dishError(409, "dish_changed", "") })
	sc.Step(`^le plat "([^"]*)" m'est présenté à jour, avec (.+) de "([^"]*)" et sans "([^"]*)"$`, w.presentedUpToDate)

	// P4
	sc.Step(`^l'ingrédient "([^"]*)" existe dans le référentiel$`, func(ctx context.Context, name string) error {
		return w.addIngredient(ctx, name)
	})
	sc.Step(`^j'ajoute l'ingrédient "([^"]*)" à un plat$`, w.addIngredient)
	sc.Step(`^l'ingrédient "([^"]*)" n'existe pas dans le référentiel$`, func(ctx context.Context, name string) error {
		return w.inCatalog(ctx, "je", name, 0)
	})
	sc.Step(`^l'ingrédient "([^"]*)" est ajouté au référentiel$`, func(ctx context.Context, name string) error {
		return w.inCatalog(ctx, "je", name, 1)
	})
	sc.Step(`^le référentiel contient un seul ingrédient "([^"]*)"$`, func(ctx context.Context, name string) error {
		return w.inCatalog(ctx, "je", name, 1)
	})
	sc.Step(`^le plat utilise l'ingrédient "([^"]*)"$`, w.dishUses)

	// T1 to T3
	sc.Step(`^Alice et Bruno sont membres de la même tribu$`, w.aliceAndBruno)
	sc.Step(`^(?:qu')?(Alice|Bruno) (?:crée|a créé) le plat "([^"]*)"$`, func(ctx context.Context, who, name string) error {
		return w.create(ctx, who, dishBody{Name: name})
	})
	sc.Step(`^(Alice|Bruno) voit le plat "([^"]*)" dans la bibliothèque, après rafraîchissement$`, func(ctx context.Context, who, name string) error {
		_, err := w.findDish(ctx, who, name)
		return err
	})
	sc.Step(`^(Alice|Bruno) modifie le plat "([^"]*)"$`, w.modify)
	sc.Step(`^la modification est enregistrée$`, w.modificationSaved)
	sc.Step(`^une tribu qui vient d'être initialisée$`, func(ctx context.Context) error {
		_, err := w.person(ctx, "Alice")
		return err
	})
	sc.Step(`^la taille de la tribu est (\d+)$`, w.tribeSizeIs)

	// ENF-02
	sc.Step(`^la tribu "([^"]*)" a le plat "([^"]*)" et l'ingrédient "([^"]*)"$`, w.otherTribeHas)
	sc.Step(`^l'ingrédient "([^"]*)" de la tribu "([^"]*)" a (?:toujours )?l'unité par défaut "([^"]*)"$`, w.otherTribeDefaultUnit)
	sc.Step(`^je consulte la bibliothèque de plats$`, w.consultLibrary)
	sc.Step(`^je ne vois pas le plat "([^"]*)"$`, w.notSeen)
	sc.Step(`^une recherche sur "([^"]*)" ne renvoie aucun résultat$`, w.noSearchResult)
	sc.Step(`^je saisis "([^"]*)" comme ingrédient d'un plat$`, w.typeIngredient)
	sc.Step(`^"([^"]*)" ne fait pas partie des propositions$`, w.notSuggested)
	sc.Step(`^l'ingrédient "([^"]*)" est créé dans la tribu "([^"]*)" avec l'unité par défaut "([^"]*)"$`, func(ctx context.Context, name, slug, unit string) error {
		return w.defaultUnitIs(ctx, "je", name, unit)
	})
	sc.Step(`^je tente de (lire|modifier|supprimer) une donnée de la tribu "([^"]*)" en utilisant son identifiant$`, w.crossAccess)
	sc.Step(`^la requête est refusée$`, w.crossAccessRefused)
	sc.Step(`^la réponse est la même que pour un identifiant qui n'existe pas$`, w.sameAsUnknownID)
	sc.Step(`^la donnée de la tribu "([^"]*)" est inchangée$`, w.otherDishUnchanged)
	sc.Step(`^je ne vois aucun élément de la tribu "([^"]*)"$`, w.nothingOfOtherTribe)
	sc.Step(`^j'envoie une requête sur l'URL de la tribu "([^"]*)" en indiquant la tribu "([^"]*)" dans son contenu$`, w.requestNamingOtherTribe)
	sc.Step(`^la requête est traitée pour la tribu "([^"]*)" ou refusée$`, w.processedForOwnTribe)
	sc.Step(`^aucune donnée de la tribu "([^"]*)" n'est lue ni modifiée$`, w.otherTribeUntouched)
}

// person returns the device of a person of the scenario, signed in to its tribe on first use:
// "je" is the member signed in by an earlier step, if any, otherwise a member of the tribe
// of the scenario, the kitchen if none.
func (w *world) person(ctx context.Context, who string) (*device, error) {
	if d, ok := w.kitchen.people[who]; ok {
		return d, nil
	}
	if who == "je" && w.auth.device != nil && w.auth.home != "" && hasSession(w.auth.device, w.auth.home) {
		w.kitchen.people[who] = w.auth.device
		return w.auth.device, nil
	}
	email, ok := personEmails[who]
	if !ok {
		return nil, fmt.Errorf("unknown person %q", who)
	}
	slug := w.kitchenTribe()
	if _, err := w.ensureMember(ctx, email, slug); err != nil {
		return nil, err
	}
	d := w.otherDevice()
	if err := w.signIn(ctx, d, slug, email); err != nil {
		return nil, fmt.Errorf("sign in %s: %w", who, err)
	}
	w.kitchen.people[who] = d
	return d, nil
}

// kitchenTribe is the tribe of the dish scenarios: the first one the scenario names, the
// kitchen otherwise.
func (w *world) kitchenTribe() string {
	if w.auth.home == "" {
		w.auth.home = kitchenSlug
	}
	return w.auth.home
}

func hasSession(d *device, slug string) bool {
	return slices.ContainsFunc(d.jar.Cookies(originURL("/")), func(c *http.Cookie) bool {
		return c.Name == cookieName(sessionCookie, slug)
	})
}

// api sends a request of a person to the API of the tribe of the scenario.
func (w *world) api(ctx context.Context, who, method, path string, body any) (*httptest.ResponseRecorder, error) {
	d, err := w.person(ctx, who)
	if err != nil {
		return nil, err
	}
	return w.do(ctx, d, method, tribePath(w.kitchenTribe(), "api/"+path), body)
}

// pages reads every page of a collection, as the front end does (ADR 0029).
func pages[T any](ctx context.Context, w *world, who, path string) ([]T, error) {
	var items []T
	for after := ""; ; {
		target := path
		if after != "" {
			target += "?after=" + after
		}
		rec, err := w.api(ctx, who, http.MethodGet, target, nil)
		if err != nil {
			return nil, err
		}
		if rec.Code != http.StatusOK {
			return nil, fmt.Errorf("GET %s: status %d, body %s", target, rec.Code, rec.Body)
		}
		var page struct {
			Items []T    `json:"items"`
			Next  string `json:"next"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
			return nil, err
		}
		items = append(items, page.Items...)
		if page.Next == "" {
			return items, nil
		}
		after = page.Next
	}
}

func (w *world) library(ctx context.Context, who string) ([]apiDish, error) {
	return pages[apiDish](ctx, w, who, "dishes")
}

func (w *world) catalog(ctx context.Context, who string) ([]apiIngredient, error) {
	return pages[apiIngredient](ctx, w, who, "ingredients")
}

func (w *world) findDish(ctx context.Context, who, name string) (apiDish, error) {
	dishes, err := w.library(ctx, who)
	if err != nil {
		return apiDish{}, err
	}
	for _, d := range dishes {
		if d.Name == name {
			return d, nil
		}
	}
	return apiDish{}, fmt.Errorf("no dish %q in the library of %s: %+v", name, who, dishes)
}

// create sends a new dish; the response is kept for the outcome steps.
func (w *world) create(ctx context.Context, who string, body dishBody) error {
	dishes, err := w.library(ctx, who)
	if err != nil {
		return err
	}
	w.kitchen.countBefore = len(dishes)
	w.kitchen.lastName = body.Name
	return w.send(ctx, who, http.MethodPost, "dishes", body)
}

// save sends a change of a dish, on the revision it was read in.
func (w *world) save(ctx context.Context, who string, d apiDish) error {
	return w.send(ctx, who, http.MethodPut, "dishes/"+d.ID, d.body())
}

func (w *world) send(ctx context.Context, who, method, path string, body dishBody) error {
	if body.Ingredients == nil {
		body.Ingredients = []apiLine{}
	}
	rec, err := w.api(ctx, who, method, path, body)
	if err != nil {
		return err
	}
	w.kitchen.last = rec
	if rec.Code == http.StatusOK || rec.Code == http.StatusCreated {
		var saved struct {
			Dish apiDish `json:"dish"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &saved); err != nil {
			return err
		}
		w.kitchen.saved = saved.Dish
	}
	return nil
}

// mustSave sends a dish in a context step: it must be saved.
func (w *world) mustSave(ctx context.Context, who, method, path string, body dishBody) (apiDish, error) {
	if err := w.send(ctx, who, method, path, body); err != nil {
		return apiDish{}, err
	}
	if c := w.kitchen.last.Code; c != http.StatusOK && c != http.StatusCreated {
		return apiDish{}, fmt.Errorf("%s %s: status %d, body %s", method, path, c, w.kitchen.last.Body)
	}
	return w.kitchen.saved, nil
}

// ensureDish returns the dish of this name, created by "je" from body if absent.
func (w *world) ensureDish(ctx context.Context, name string, body dishBody) (apiDish, error) {
	if d, err := w.findDish(ctx, "je", name); err == nil {
		return d, nil
	}
	return w.mustSave(ctx, "je", http.MethodPost, "dishes", body)
}

func (w *world) createWithTable(ctx context.Context, name string, table *godog.Table) error {
	body := dishBody{Name: name}
	for _, row := range table.Rows[1:] {
		line := apiLine{Name: row.Cells[0].Value, Quantity: row.Cells[1].Value}
		if label := row.Cells[2].Value; label != "" {
			code, err := unitCode(label)
			if err != nil {
				return err
			}
			line.Unit = code
		}
		body.Ingredients = append(body.Ingredients, line)
	}
	return w.create(ctx, "je", body)
}

func (w *world) lastDishStatus(status int) error {
	if w.kitchen.last == nil {
		return errors.New("no dish sent")
	}
	if w.kitchen.last.Code != status {
		return fmt.Errorf("status %d, want %d; body %s", w.kitchen.last.Code, status, w.kitchen.last.Body)
	}
	return nil
}

func (w *world) servingsAre(ctx context.Context, name string, servings int) error {
	if err := w.lastDishStatus(http.StatusCreated); err != nil {
		return err
	}
	d, err := w.findDish(ctx, "je", name)
	if err != nil {
		return err
	}
	if d.Servings != servings {
		return fmt.Errorf("%s for %d servings, want %d", name, d.Servings, servings)
	}
	return nil
}

// allHuesBut fills the library with a dish of each hue of the palette but one. The dishes
// are created through the API; their colors are then set in the database if the hue left
// out is not the last one given.
func (w *world) allHuesBut(ctx context.Context, missing string) error {
	var others []dish.Color
	for _, c := range dish.Palette {
		if string(c) != missing {
			others = append(others, c)
		}
	}
	if len(others) != len(dish.Palette)-1 {
		return fmt.Errorf("%s is not a hue of the palette", missing)
	}
	for i, c := range others {
		d, err := w.mustSave(ctx, "je", http.MethodPost, "dishes", dishBody{Name: fmt.Sprintf("Plat de teinte %d", i+1)})
		if err != nil {
			return err
		}
		if err := w.setColor(ctx, d.ID, c); err != nil {
			return err
		}
	}
	return nil
}

// setColor sets the color of a dish in the database: the API never changes it (Q19).
func (w *world) setColor(ctx context.Context, id string, color dish.Color) error {
	store, err := w.openStore(ctx)
	if err != nil {
		return err
	}
	db, err := store.Tribe(ctx, w.kitchenTribe())
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, "UPDATE dishes SET color = ? WHERE id = ?", string(color), id)
	return err
}

func (w *world) hasHue(ctx context.Context, name, color string) error {
	if w.isContext() {
		d, err := w.ensureDish(ctx, name, dishBody{Name: name})
		if err != nil {
			return err
		}
		if d.Color == color {
			return nil
		}
		return w.setColor(ctx, d.ID, dish.Color(color))
	}
	d, err := w.findDish(ctx, "je", name)
	if err != nil {
		return err
	}
	if d.Color != color {
		return fmt.Errorf("%s has the hue %s, want %s", name, d.Color, color)
	}
	return nil
}

// modify changes the servings of a dish, as read just before.
func (w *world) modify(ctx context.Context, who, name string) error {
	d, err := w.findDish(ctx, who, name)
	if err != nil {
		return err
	}
	d.Servings = d.Servings%dish.MaxServings + 1
	return w.save(ctx, who, d)
}

func (w *world) modificationSaved(ctx context.Context) error {
	if err := w.lastDishStatus(http.StatusOK); err != nil {
		return err
	}
	d, err := w.findDish(ctx, "Alice", w.kitchen.saved.Name)
	if err != nil {
		return err
	}
	if d.Revision != w.kitchen.saved.Revision || d.Servings != w.kitchen.saved.Servings {
		return fmt.Errorf("dish %+v, saved %+v", d, w.kitchen.saved)
	}
	return nil
}

func (w *world) notSaved(ctx context.Context) error {
	if err := w.lastDishStatus(http.StatusBadRequest); err != nil {
		return err
	}
	dishes, err := w.library(ctx, "je")
	if err != nil {
		return err
	}
	if len(dishes) != w.kitchen.countBefore {
		return fmt.Errorf("%d dishes, %d before", len(dishes), w.kitchen.countBefore)
	}
	return nil
}

// oddNames are the names that cannot be written as such in a scenario (D30).
var oddNames = map[string]string{
	"l'espace sans chasse U+200B":                   "\u200b",
	`"Soupe" suivi du caractère de contrôle U+0007`: "Soupe\a",
	`"sel" suivi du caractère de contrôle U+0007`:   "sel\a",
}

// oddNameErrors are the error codes of a name refused, by what it names and why.
var oddNameErrors = map[string]string{
	"du plat vide":             "name_required",
	"du plat invalide":         "name_invalid",
	"d'un ingrédient vide":     "ingredient_name_required",
	"d'un ingrédient invalide": "ingredient_name_invalid",
}

func (w *world) createWithOddName(ctx context.Context, of, label string) error {
	name, ok := oddNames[label]
	if !ok {
		return fmt.Errorf("unknown name %s", label)
	}
	if of == "du plat" {
		return w.create(ctx, "je", dishBody{Name: name})
	}
	w.kitchen.n++
	return w.create(ctx, "je", dishBody{Name: fmt.Sprintf("Plat %d", w.kitchen.n), Ingredients: []apiLine{{Name: name}}})
}

// dishError checks the error of the last response, and the ingredient it names.
func (w *world) dishError(status int, code, ingredient string) error {
	if err := w.lastDishStatus(status); err != nil {
		return err
	}
	var e struct {
		Error      string `json:"error"`
		Ingredient string `json:"ingredient"`
	}
	if err := json.Unmarshal(w.kitchen.last.Body.Bytes(), &e); err != nil {
		return err
	}
	if e.Error != code || e.Ingredient != ingredient {
		return fmt.Errorf("error %+v, want %s %q", e, code, ingredient)
	}
	return nil
}

func (w *world) lineWithoutQuantity(ctx context.Context, ingredient, name string) error {
	d, err := w.findDish(ctx, "je", name)
	if err != nil {
		return err
	}
	for _, l := range d.Ingredients {
		if l.Name == ingredient {
			if l.Quantity != "" || l.Unit != "" {
				return fmt.Errorf("%s: %s %s", ingredient, l.Quantity, l.Unit)
			}
			return nil
		}
	}
	return fmt.Errorf("no %q in %+v", ingredient, d)
}

func (w *world) defaultUnitIs(ctx context.Context, who, name, label string) error {
	want := ""
	if label != "" {
		code, err := unitCode(label)
		if err != nil {
			return err
		}
		want = code
	}
	ingredients, err := w.catalog(ctx, who)
	if err != nil {
		return err
	}
	for _, i := range ingredients {
		if i.Name == name {
			if i.DefaultUnit != want {
				return fmt.Errorf("%s: default unit %q, want %q", name, i.DefaultUnit, want)
			}
			return nil
		}
	}
	return fmt.Errorf("no %q in the catalog %+v", name, ingredients)
}

// cannotDelete: no route deletes a dish (P2; ADR 0005, point 4): DELETE on a dish answers 405,
// and the dish stays.
func (w *world) cannotDelete(ctx context.Context, name string) error {
	d, err := w.findDish(ctx, "je", name)
	if err != nil {
		return err
	}
	rec, err := w.api(ctx, "je", http.MethodDelete, "dishes/"+d.ID, nil)
	if err != nil {
		return err
	}
	if rec.Code != http.StatusMethodNotAllowed {
		return fmt.Errorf("DELETE: status %d, want 405", rec.Code)
	}
	if allow := rec.Header().Get("Allow"); strings.Contains(allow, http.MethodDelete) {
		return fmt.Errorf("Allow: %s", allow)
	}
	_, err = w.findDish(ctx, "je", name)
	return err
}

func (w *world) givenDishWith(ctx context.Context, name string, servings int, quantity, ingredient string) error {
	q, unit, err := parseQuantity(quantity)
	if err != nil {
		return err
	}
	_, err = w.mustSave(ctx, "je", http.MethodPost, "dishes", dishBody{
		Name: name, Servings: servings, Ingredients: []apiLine{{Name: ingredient, Quantity: q, Unit: unit}},
	})
	w.recordDish(name)
	return err
}

func (w *world) openDish(ctx context.Context, name string) error {
	d, err := w.findDish(ctx, "je", name)
	if err != nil {
		return err
	}
	w.kitchen.opened[name] = d
	return nil
}

func (w *world) changedInBetween(ctx context.Context, ingredient, name, quantity string) error {
	return w.changeLine(ctx, "Bruno", ingredient, name, quantity)
}

// changeLine sets the quantity of the line of the ingredient in the dish, as the person reads
// it now.
func (w *world) changeLine(ctx context.Context, who, ingredient, name, quantity string) error {
	d, err := w.findDish(ctx, who, name)
	if err != nil {
		return err
	}
	if err := setLine(&d, ingredient, quantity); err != nil {
		return err
	}
	_, err = w.mustSave(ctx, who, http.MethodPut, "dishes/"+d.ID, d.body())
	return err
}

// setLine sets the quantity of the line of the ingredient, added if absent.
func setLine(d *apiDish, ingredient, quantity string) error {
	q, unit, err := parseQuantity(quantity)
	if err != nil {
		return err
	}
	for i, l := range d.Ingredients {
		if l.Name == ingredient {
			d.Ingredients[i].Quantity, d.Ingredients[i].Unit = q, unit
			return nil
		}
	}
	d.Ingredients = append(d.Ingredients, apiLine{Name: ingredient, Quantity: q, Unit: unit})
	return nil
}

// addLine adds a line to the dish "je" opened, to the dish of the library otherwise, or to
// a new dish of this name.
func (w *world) addLine(ctx context.Context, quantity, ingredient, name string) error {
	d, opened := w.kitchen.opened[name]
	if !opened {
		found, err := w.findDish(ctx, "je", name)
		if err != nil {
			body := dishBody{Name: name}
			q, unit, err := parseQuantity(quantity)
			if err != nil {
				return err
			}
			body.Ingredients = []apiLine{{Name: ingredient, Quantity: q, Unit: unit}}
			return w.create(ctx, "je", body)
		}
		d = found
	}
	if err := setLine(&d, ingredient, quantity); err != nil {
		return err
	}
	return w.save(ctx, "je", d)
}

func (w *world) presentedUpToDate(name, quantity, ingredient, absent string) error {
	q, unit, err := parseQuantity(quantity)
	if err != nil {
		return err
	}
	var e struct {
		Dish apiDish `json:"dish"`
	}
	if err := json.Unmarshal(w.kitchen.last.Body.Bytes(), &e); err != nil {
		return err
	}
	d := e.Dish
	if d.Name != name || d.Revision != w.kitchen.opened[name].Revision+1 {
		return fmt.Errorf("dish %+v, opened in revision %d", d, w.kitchen.opened[name].Revision)
	}
	found := false
	for _, l := range d.Ingredients {
		switch l.Name {
		case ingredient:
			found = l.Quantity == q && l.Unit == unit
		case absent:
			return fmt.Errorf("%s in %+v", absent, d)
		}
	}
	if !found {
		return fmt.Errorf("no %s %s of %s in %+v", q, unit, ingredient, d)
	}
	return nil
}

// addIngredient adds the ingredient to the catalog by entering it in a new dish, without
// quantity: an ingredient is created only by entering a dish (Q20).
func (w *world) addIngredient(ctx context.Context, name string) error {
	w.kitchen.n++
	_, err := w.mustSave(ctx, "je", http.MethodPost, "dishes", dishBody{
		Name: fmt.Sprintf("Plat %d", w.kitchen.n), Ingredients: []apiLine{{Name: name}},
	})
	return err
}

// inCatalog checks that the catalog holds n ingredients of the same normalized name as
// name, displayed as name.
func (w *world) inCatalog(ctx context.Context, who, name string, n int) error {
	ingredients, err := w.catalog(ctx, who)
	if err != nil {
		return err
	}
	var same []apiIngredient
	for _, i := range ingredients {
		if dish.NormalizeName(i.Name) == dish.NormalizeName(name) {
			same = append(same, i)
		}
	}
	if len(same) != n || n > 0 && same[0].Name != name {
		return fmt.Errorf("catalog %+v, want %d %q", ingredients, n, name)
	}
	return nil
}

func (w *world) dishUses(name string) error {
	for _, l := range w.kitchen.saved.Ingredients {
		if l.Name == name {
			return nil
		}
	}
	return fmt.Errorf("dish %+v does not use %q", w.kitchen.saved, name)
}

func (w *world) aliceAndBruno(ctx context.Context) error {
	for _, who := range []string{"Alice", "Bruno"} {
		if _, err := w.person(ctx, who); err != nil {
			return err
		}
	}
	return nil
}

func (w *world) tribeSizeIs(ctx context.Context, size int) error {
	rec, err := w.api(ctx, "Alice", http.MethodGet, "tribe", nil)
	if err != nil {
		return err
	}
	var got struct {
		Size int `json:"size"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		return fmt.Errorf("GET tribe: status %d, body %q: %w", rec.Code, rec.Body, err)
	}
	if got.Size != size {
		return fmt.Errorf("tribe size %d, want %d", got.Size, size)
	}
	return nil
}

// otherLibrary returns the library of another tribe, used through the domain: the scenario
// signs in to the tribe of "je" only.
func (w *world) otherLibrary(ctx context.Context, slug string) (*dish.Library, error) {
	store, err := w.openStore(ctx)
	if err != nil {
		return nil, err
	}
	db, err := store.Tribe(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("tribe %q: %w", slug, err)
	}
	return dish.NewLibrary(storage.NewDishStore(db)), nil
}

func (w *world) otherTribeHas(ctx context.Context, slug, name, ingredient string) error {
	library, err := w.otherLibrary(ctx, slug)
	if err != nil {
		return err
	}
	_, _, err = library.Create(ctx, dish.Input{Name: name, Lines: []dish.LineInput{{Name: ingredient}}})
	w.kitchen.foreignName = name
	return err
}

// otherDish returns the only dish of the first page of another tribe that has this name.
func otherDish(ctx context.Context, library *dish.Library, name string) (dish.Dish, error) {
	_, page, err := library.Dishes(ctx, "")
	if err != nil {
		return dish.Dish{}, err
	}
	for _, d := range page.Items {
		if d.Name == name {
			return d, nil
		}
	}
	return dish.Dish{}, fmt.Errorf("no dish %q", name)
}

// otherTribeDefaultUnit gives the ingredient of another tribe its default unit, by entering
// it in a dish with this unit, or checks it.
func (w *world) otherTribeDefaultUnit(ctx context.Context, name, slug, label string) error {
	unit, err := unitCode(label)
	if err != nil {
		return err
	}
	library, err := w.otherLibrary(ctx, slug)
	if err != nil {
		return err
	}
	if w.isContext() {
		_, _, err := library.Create(ctx, dish.Input{Name: "Plat au " + name, Lines: []dish.LineInput{{Name: name, Quantity: "200", Unit: unit}}})
		if err != nil {
			return err
		}
	}
	_, page, err := library.Catalog(ctx, "")
	if err != nil {
		return err
	}
	for _, i := range page.Items {
		if i.Name == name {
			if string(i.DefaultUnit) != unit {
				return fmt.Errorf("%s of %s: default unit %q, want %q", name, slug, i.DefaultUnit, unit)
			}
			return nil
		}
	}
	return fmt.Errorf("no %q in the catalog of %s", name, slug)
}

func (w *world) consultLibrary(ctx context.Context) error {
	dishes, err := w.library(ctx, "je")
	w.kitchen.seen = dishes
	return err
}

func (w *world) notSeen(name string) error {
	for _, d := range w.kitchen.seen {
		if d.Name == name {
			return fmt.Errorf("%q in the library %+v", name, w.kitchen.seen)
		}
	}
	return nil
}

// words are the words of a normalized name, for the search and the suggestions (Q30).
func words(name string) []string {
	return strings.FieldsFunc(dish.NormalizeName(name), func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	})
}

// matches reports whether each term of the text starts a word of the names (Q30).
func matches(text string, names ...string) bool {
	var all []string
	for _, n := range names {
		all = append(all, words(n)...)
	}
	for _, term := range words(text) {
		if !slices.ContainsFunc(all, func(word string) bool { return strings.HasPrefix(word, term) }) {
			return false
		}
	}
	return true
}

// noSearchResult: no dish of the library seen matches the search (D15).
func (w *world) noSearchResult(text string) error {
	for _, d := range w.kitchen.seen {
		names := []string{d.Name}
		for _, l := range d.Ingredients {
			names = append(names, l.Name)
		}
		if matches(text, names...) {
			return fmt.Errorf("%q finds %+v", text, d)
		}
	}
	return nil
}

func (w *world) typeIngredient(ctx context.Context, text string) error {
	if _, err := w.catalog(ctx, "je"); err != nil {
		return err
	}
	w.kitchen.typed = text
	return nil
}

// notSuggested: the ingredient is not among the ingredients of the catalog the text typed
// would propose (D15).
func (w *world) notSuggested(ctx context.Context, name string) error {
	ingredients, err := w.catalog(ctx, "je")
	if err != nil {
		return err
	}
	for _, i := range ingredients {
		if matches(w.kitchen.typed, i.Name) && dish.NormalizeName(i.Name) == dish.NormalizeName(name) {
			return fmt.Errorf("%q proposed for %q", i.Name, w.kitchen.typed)
		}
	}
	return nil
}

// crossAccess tries to read, change or delete the dish of another tribe by its identifier,
// then an identifier that exists nowhere, the same way. Each example names its route, so
// that a future route takes the place of the one answering today (plan bibliotheque, notes
// of execution, point 6):
//   - lire: GET dishes/{id}, which no route serves yet: the API answers 404 to any GET it
//     does not know;
//   - modifier: PUT dishes/{id}, the route of a change;
//   - supprimer: DELETE dishes/{id}, which no route serves (P2): the mux answers 405.
func (w *world) crossAccess(ctx context.Context, action, slug string) error {
	library, err := w.otherLibrary(ctx, slug)
	if err != nil {
		return err
	}
	target, err := otherDish(ctx, library, w.kitchen.foreignName)
	if err != nil {
		return err
	}
	w.kitchen.foreignBefore = target
	attempt := func(id string) (*httptest.ResponseRecorder, error) {
		switch action {
		case "lire":
			return w.api(ctx, "je", http.MethodGet, "dishes/"+id, nil)
		case "modifier":
			return w.api(ctx, "je", http.MethodPut, "dishes/"+id, dishBody{Revision: &target.Revision, Name: target.Name, Servings: target.Servings + 1, Ingredients: []apiLine{}})
		default:
			return w.api(ctx, "je", http.MethodDelete, "dishes/"+id, nil)
		}
	}
	if w.kitchen.foreign, err = attempt(string(target.ID)); err != nil {
		return err
	}
	w.kitchen.unknown, err = attempt(strings.Repeat("0", len(target.ID)))
	return err
}

// crossAccessRefused checks the answer to the attempt on another tribe or, without one, to
// the last write on a shopping list.
func (w *world) crossAccessRefused() error {
	rec := w.kitchen.foreign
	if rec == nil {
		rec = w.shopping.last
	}
	if rec == nil {
		return errors.New("no request sent")
	}
	if c := rec.Code; c < 400 {
		return fmt.Errorf("status %d, body %s", c, rec.Body)
	}
	return nil
}

// sameAsUnknownID compares the answers to each attempt on an identifier of another tribe
// with those on an identifier that exists nowhere.
func (w *world) sameAsUnknownID() error {
	pairs := w.shopping.attempts
	if w.kitchen.foreign != nil {
		pairs = append(pairs, [2]*httptest.ResponseRecorder{w.kitchen.foreign, w.kitchen.unknown})
	}
	if len(pairs) == 0 {
		return errors.New("no attempt made")
	}
	for _, p := range pairs {
		foreign, unknown := answer(p[0]), answer(p[1])
		if foreign != unknown {
			return fmt.Errorf("%q, for an unknown identifier %q", foreign, unknown)
		}
	}
	return nil
}

func (w *world) otherDishUnchanged(ctx context.Context, slug string) error {
	library, err := w.otherLibrary(ctx, slug)
	if err != nil {
		return err
	}
	before := w.kitchen.foreignBefore
	now, err := otherDish(ctx, library, before.Name)
	if err != nil {
		return err
	}
	if now.Revision != before.Revision || now.Servings != before.Servings {
		return fmt.Errorf("dish of %s %+v, before %+v", slug, now, before)
	}
	return nil
}

// nothingOfOtherTribe: neither the dishes nor the ingredients of the other tribe are in what
// "je" consulted, nor its meal plan, nor its shopping lists.
func (w *world) nothingOfOtherTribe(ctx context.Context, slug string) error {
	if err := w.nothingOfOtherPlan(ctx, slug); err != nil {
		return err
	}
	if err := w.nothingOfOtherLists(ctx, slug); err != nil {
		return err
	}
	library, err := w.otherLibrary(ctx, slug)
	if err != nil {
		return err
	}
	_, dishes, err := library.Dishes(ctx, "")
	if err != nil {
		return err
	}
	_, catalog, err := library.Catalog(ctx, "")
	if err != nil {
		return err
	}
	seen, err := json.Marshal(w.kitchen.seen)
	if err != nil {
		return err
	}
	for _, d := range dishes.Items {
		if strings.Contains(string(seen), d.Name) {
			return fmt.Errorf("%q seen in %s", d.Name, seen)
		}
	}
	for _, i := range catalog.Items {
		if strings.Contains(string(seen), i.Name) {
			return fmt.Errorf("%q seen in %s", i.Name, seen)
		}
	}
	return nil
}

// otherVersions reads the versions of the collections of another tribe.
func (w *world) otherVersions(ctx context.Context, slug string) ([2]int64, error) {
	library, err := w.otherLibrary(ctx, slug)
	if err != nil {
		return [2]int64{}, err
	}
	dishes, err := library.Version(ctx, dish.DishesCollection)
	if err != nil {
		return [2]int64{}, err
	}
	ingredients, err := library.Version(ctx, dish.IngredientsCollection)
	return [2]int64{dishes, ingredients}, err
}

// requestNamingOtherTribe creates a dish on the URL of the tribe of the session, the body
// naming the other tribe.
func (w *world) requestNamingOtherTribe(ctx context.Context, slug, other string) error {
	if slug != w.kitchenTribe() {
		return fmt.Errorf("signed in to %s, not %s", w.kitchenTribe(), slug)
	}
	versions, err := w.otherVersions(ctx, other)
	if err != nil {
		return err
	}
	w.kitchen.versionsBefore = versions
	rec, err := w.api(ctx, "je", http.MethodPost, "dishes", map[string]any{
		"tribe": other, "slug": other, "name": "Raclette", "ingredients": []apiLine{{Name: "raclette"}},
	})
	w.kitchen.last = rec
	return err
}

func (w *world) processedForOwnTribe(ctx context.Context, slug string) error {
	switch c := w.kitchen.last.Code; {
	case c >= 400:
		return nil
	case c != http.StatusCreated:
		return fmt.Errorf("status %d, body %s", c, w.kitchen.last.Body)
	}
	_, err := w.findDish(ctx, "je", "Raclette")
	return err
}

func (w *world) otherTribeUntouched(ctx context.Context, slug string) error {
	versions, err := w.otherVersions(ctx, slug)
	if err != nil {
		return err
	}
	if versions != w.kitchen.versionsBefore {
		return fmt.Errorf("versions of %s %v, before %v", slug, versions, w.kitchen.versionsBefore)
	}
	t, err := w.tribeStore(ctx, slug)
	if err != nil {
		return err
	}
	name, err := t.Name(ctx)
	if err != nil {
		return err
	}
	for _, secret := range []string{name, slug, "david@exemple.fr"} {
		if strings.Contains(w.kitchen.last.Body.String(), secret) {
			return fmt.Errorf("%q in the response %s", secret, w.kitchen.last.Body)
		}
	}
	return nil
}
