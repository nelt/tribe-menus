package acceptance

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"

	"github.com/cucumber/godog"
	"github.com/nelt/tribe-menus/internal/dish"
	"github.com/nelt/tribe-menus/internal/shopping"
	"github.com/nelt/tribe-menus/internal/storage"
)

// shoppingState is what the shopping list scenarios remember from step to step.
type shoppingState struct {
	// last is the response to the last write on the lists; countBefore, the number of lists
	// before the last creation.
	last        *httptest.ResponseRecorder
	countBefore int
	// opened are the lists "je" opened, alone or in the list picker, by name, as read then:
	// their identifiers stay, deleted in between or not.
	opened map[string]apiList
	// confirmation is the list whose confirmation of the shopping done "je" opened, as read
	// then.
	confirmation apiList
	// newTribe is the tribe created by the scenario, and the device of its first member.
	newTribe       string
	newTribeDevice *device
	// seen are the lists "je" consulted.
	seen []apiList
	// foreign is the list of another tribe, before the attempts on it; attempts, the answers
	// to each attempt on an identifier of another tribe and on one that exists nowhere.
	foreign  shopping.List
	attempts [][2]*httptest.ResponseRecorder
}

type apiList struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Main     bool      `json:"main"`
	Rank     int64     `json:"rank"`
	Items    []apiItem `json:"items"`
	Received []string  `json:"received"`
}

type apiItem struct {
	ID         string      `json:"id"`
	Ingredient string      `json:"ingredient"`
	Quantity   string      `json:"quantity,omitempty"`
	Unit       string      `json:"unit,omitempty"`
	Checked    bool        `json:"checked"`
	Manual     bool        `json:"manual"`
	Sources    []apiSource `json:"sources"`
}

type apiSource struct {
	ServedDish string `json:"servedDish"`
	Dish       string `json:"dish"`
	Date       string `json:"date"`
	Mealtime   string `json:"mealtime"`
	Servings   int    `json:"servings"`
	Quantity   string `json:"quantity,omitempty"`
	Unit       string `json:"unit,omitempty"`
}

// newTribeSlug is the tribe created by « Une tribu naît avec une liste principale ».
const newTribeSlug = "nouvelle"

// registerShoppingSteps: liste-courses.feature, and the shopping list scenarios of
// tribu.feature and compartimentage-tribus.feature.
func (w *world) registerShoppingSteps(sc *godog.ScenarioContext) {
	sc.Step(`^la bibliothèque contient le plat "([^"]*)" défini pour (\d+) parts :$`, w.givenDishWithTable)

	// C1
	sc.Step(`^la liste principale de la tribu est (?:toujours )?la liste "([^"]*)"$`, w.mainListIs)
	sc.Step(`^une nouvelle tribu est créée$`, w.createNewTribe)
	sc.Step(`^(?:elle|la tribu "([^"]*)") a une seule liste de courses, "([^"]*)", vide$`, w.onlyList)
	sc.Step(`^"([^"]*)" est sa liste principale$`, w.newTribeMainList)
	sc.Step(`^les listes "([^"]*)" et "([^"]*)"$`, func(ctx context.Context, a, b string) error {
		for _, name := range []string{a, b} {
			if _, err := w.givenList(ctx, name); err != nil {
				return err
			}
		}
		return nil
	})
	sc.Step(`^je désigne la liste "([^"]*)" comme principale$`, func(ctx context.Context, name string) error {
		return w.setMain(ctx, "je", name)
	})
	sc.Step(`^la liste "([^"]*)" n'est plus principale$`, func(ctx context.Context, name string) error {
		l, err := w.listNamed(ctx, "je", name)
		if err == nil && l.Main {
			err = fmt.Errorf("%s is still the main list", name)
		}
		return err
	})
	sc.Step(`^aucune action ne permet de retirer son rôle à la liste principale sans en désigner une autre$`, w.mainListStays)

	// C2
	sc.Step(`^je crée la liste "([^"]*)"$`, func(ctx context.Context, name string) error {
		return w.createList(ctx, name)
	})
	sc.Step(`^je crée une liste sans nom$`, func(ctx context.Context) error {
		return w.createList(ctx, "")
	})
	sc.Step(`^je crée une liste dont le nom est (.+)$`, func(ctx context.Context, label string) error {
		name, ok := oddListNames[label]
		if !ok {
			return fmt.Errorf("unknown name %s", label)
		}
		return w.createList(ctx, name)
	})
	sc.Step(`^la liste "([^"]*)"( vide)?$`, func(ctx context.Context, name, _ string) error {
		_, err := w.givenList(ctx, name)
		return err
	})
	sc.Step(`^la liste "([^"]*)" existe( et est vide)?$`, func(ctx context.Context, name, empty string) error {
		l, err := w.listNamed(ctx, "je", name)
		if err == nil && empty != "" && len(l.Items) != 0 {
			err = fmt.Errorf("list %s holds %+v", name, l.Items)
		}
		return err
	})
	sc.Step(`^cette liste existe et est vide$`, w.createdListEmpty)
	sc.Step(`^la création de la liste est impossible$`, w.listNotCreated)
	sc.Step(`^un message m'indique que la liste doit avoir un nom$`, func() error { return w.shoppingError(400, "list_name_empty") })
	sc.Step(`^le nom de la liste est refusé comme (trop long|vide|invalide)$`, func(refusal string) error {
		return w.shoppingError(400, map[string]string{"trop long": "list_name_too_long", "vide": "list_name_empty", "invalide": "list_name_invalid"}[refusal])
	})
	sc.Step(`^je renomme la liste "([^"]*)" en "([^"]*)"$`, func(ctx context.Context, name, to string) error {
		return w.rename(ctx, name, to)
	})
	sc.Step(`^la tribu a deux listes "([^"]*)"$`, w.twoListsNamed)
	sc.Step(`^la tribu a 20 listes de courses$`, w.twentyLists)
	sc.Step(`^un message m'indique que la tribu ne peut pas avoir plus de 20 listes$`, func() error { return w.shoppingError(409, "too_many_lists") })

	// C3
	sc.Step(`^(j'|Alice |Bruno )ajoute l'article "([^"]*)"(?: pour (.+?)| sans quantité)? à la liste "([^"]*)"$`, func(ctx context.Context, who, ingredient, quantity, list string) error {
		return w.addItem(ctx, subject(who), list, ingredient, quantity)
	})
	sc.Step(`^la liste "([^"]*)" contient (?:toujours )?l'article ajouté "([^"]*)"( pour .+| sans quantité)?(, non coché)?$`, func(ctx context.Context, list, ingredient, quantity, unchecked string) error {
		return w.holdsManual(ctx, list, ingredient, quantity, unchecked != "")
	})
	sc.Step(`^l'article "([^"]*)" n'est rattaché à aucun plat$`, w.notFromAnyDish)
	sc.Step(`^(?:que )?le référentiel d'ingrédients ne contient pas "([^"]*)"$`, func(ctx context.Context, name string) error {
		return w.inCatalog(ctx, "je", name, 0)
	})
	sc.Step(`^"([^"]*)" est proposé par l'autocomplétion lors de la saisie d'un ingrédient$`, func(ctx context.Context, name string) error {
		return w.inCatalog(ctx, "je", name, 1)
	})
	sc.Step(`^la liste "([^"]*)" contenant l'article ajouté "([^"]*)"(?: pour (.+))?$`, func(ctx context.Context, list, ingredient, quantity string) error {
		return w.givenManual(ctx, list, ingredient, quantity)
	})
	sc.Step(`^la liste "([^"]*)" contenant les articles ajoutés "([^"]*)" et "([^"]*)"$`, func(ctx context.Context, list, a, b string) error {
		if err := w.givenManual(ctx, list, a, ""); err != nil {
			return err
		}
		return w.givenManual(ctx, list, b, "")
	})
	sc.Step(`^la liste "([^"]*)" contenant les articles ajoutés :$`, w.givenManualTable)
	sc.Step(`^la liste "([^"]*)" contenant (\d+) articles ajoutés$`, w.givenManyItems)
	sc.Step(`^je retire "([^"]*)" de la liste "([^"]*)"$`, func(ctx context.Context, ingredient, list string) error {
		return w.removeItem(ctx, "je", list, ingredient)
	})
	sc.Step(`^j'ai ouvert la liste "([^"]*)"$`, w.openList)
	sc.Step(`^Bruno a retiré entre-temps "([^"]*)" de la liste "([^"]*)"$`, func(ctx context.Context, ingredient, list string) error {
		if err := w.removeItem(ctx, "Bruno", list, ingredient); err != nil {
			return err
		}
		return w.shoppingStatus(http.StatusOK)
	})
	sc.Step(`^la liste "([^"]*)" ne contient (?:plus|pas) "([^"]*)"$`, w.holdsNo)
	sc.Step(`^aucun message ne m'est présenté$`, w.noMessage)
	sc.Step(`^l'ajout de l'article est impossible$`, func() error { return w.shoppingRefused() })
	sc.Step(`^un message m'indique que la liste ne peut pas dépasser 300 articles$`, func() error { return w.shoppingError(409, "too_many_items") })
	sc.Step(`^la liste "([^"]*)" contenant (.+) de "([^"]*)" provenant du "([^"]*)" du `+mealPattern+`$`, func(ctx context.Context, list, quantity, ingredient, dishName, day, moment string) error {
		return w.givenComputed(ctx, list, quantity, ingredient, dishName, day, moment)
	})
	sc.Step(`^la liste "([^"]*)" contenant "([^"]*)" sans quantité provenant du "([^"]*)" du `+mealPattern+`$`, func(ctx context.Context, list, ingredient, dishName, day, moment string) error {
		return w.givenComputed(ctx, list, "", ingredient, dishName, day, moment)
	})
	sc.Step(`^les articles de la liste "([^"]*)" sont dans l'ordre :$`, w.itemsInOrder)

	// C6
	sc.Step(`^"([^"]*)" est coché dans la liste "([^"]*)"$`, func(ctx context.Context, ingredient, list string) error {
		if err := w.check(ctx, "je", list, ingredient, true); err != nil {
			return err
		}
		return w.shoppingStatus(http.StatusOK)
	})
	sc.Step(`^(je|Bruno) (coche|décoche) "([^"]*)" dans la liste "([^"]*)"$`, func(ctx context.Context, who, action, ingredient, list string) error {
		return w.check(ctx, who, list, ingredient, action == "coche")
	})
	sc.Step(`^Bruno a (coché|décoché) entre-temps "([^"]*)" dans la liste "([^"]*)"$`, func(ctx context.Context, action, ingredient, list string) error {
		if err := w.check(ctx, "Bruno", list, ingredient, action == "coché"); err != nil {
			return err
		}
		return w.shoppingStatus(http.StatusOK)
	})
	sc.Step(`^Bruno a coché entre-temps "([^"]*)" dans la liste "([^"]*)" et déclaré les courses faites$`, func(ctx context.Context, ingredient, list string) error {
		if err := w.check(ctx, "Bruno", list, ingredient, true); err != nil {
			return err
		}
		return w.checkOutNow(ctx, "Bruno", list)
	})
	sc.Step(`^"([^"]*)" apparaît comme (non )?coché dans la liste "([^"]*)"$`, func(ctx context.Context, ingredient, not, list string) error {
		return w.checkedIs(ctx, "je", list, ingredient, not == "")
	})
	sc.Step(`^(Alice|Bruno) voit "([^"]*)" coché dans la liste "([^"]*)", après rafraîchissement$`, func(ctx context.Context, who, ingredient, list string) error {
		return w.checkedIs(ctx, who, list, ingredient, true)
	})

	// C11
	sc.Step(`^je déclare les courses faites pour la liste "([^"]*)"$`, func(ctx context.Context, list string) error {
		return w.checkOutNow(ctx, "je", list)
	})
	sc.Step(`^j'ai coché tous les articles de la liste "([^"]*)"( et déclaré les courses faites)?$`, w.checkAll)
	sc.Step(`^j'ai ouvert la confirmation des courses faites de la liste "([^"]*)"$`, func(ctx context.Context, list string) error {
		l, err := w.listNamed(ctx, "je", list)
		w.shopping.confirmation = l
		return err
	})
	sc.Step(`^je confirme les courses faites$`, func(ctx context.Context) error {
		return w.checkOut(ctx, "je", w.shopping.confirmation)
	})
	sc.Step(`^un message m'indique que (\d+) articles? (?:a|ont) été retirés?$`, w.removedCount)

	// C12
	sc.Step(`^je (?:supprime|tente de supprimer) la liste "([^"]*)"$`, func(ctx context.Context, name string) error {
		id, err := w.openedListID(ctx, name)
		if err != nil {
			return err
		}
		return w.shoppingSend(ctx, "je", http.MethodDelete, "shopping-lists/"+id, nil)
	})
	sc.Step(`^la liste "([^"]*)" n'existe plus$`, func(ctx context.Context, name string) error {
		lists, err := w.shoppingLists(ctx, "je")
		if err == nil && slices.ContainsFunc(lists, func(l apiList) bool { return l.Name == name }) {
			err = fmt.Errorf("lists %+v", lists)
		}
		return err
	})
	sc.Step(`^la suppression est impossible$`, func() error { return w.shoppingRefused() })
	sc.Step(`^un message m'indique que la liste doit être vide pour être supprimée$`, func() error { return w.shoppingError(409, "list_not_empty") })
	sc.Step(`^un message m'indique que la liste n'est plus vide$`, func() error { return w.shoppingError(409, "list_not_empty") })
	sc.Step(`^un message m'indique qu'il faut d'abord désigner une autre liste comme principale$`, func() error { return w.shoppingError(409, "main_list") })
	sc.Step(`^j'ai ouvert le choix des listes$`, w.openListPicker)
	sc.Step(`^Bruno a ajouté entre-temps l'article "([^"]*)" à la liste "([^"]*)"$`, func(ctx context.Context, ingredient, list string) error {
		if err := w.addItem(ctx, "Bruno", list, ingredient, ""); err != nil {
			return err
		}
		return w.shoppingStatus(http.StatusCreated)
	})
	sc.Step(`^Bruno a désigné entre-temps la liste "([^"]*)" comme principale$`, func(ctx context.Context, name string) error {
		if err := w.setMain(ctx, "Bruno", name); err != nil {
			return err
		}
		return w.shoppingStatus(http.StatusOK)
	})
	sc.Step(`^la liste "([^"]*)" m'est présentée à jour, avec l'article ajouté "([^"]*)"$`, w.presentedList)
	sc.Step(`^Bruno a supprimé entre-temps la liste "([^"]*)"$`, func(ctx context.Context, name string) error {
		l, err := w.listNamed(ctx, "Bruno", name)
		if err != nil {
			return err
		}
		if err := w.shoppingSend(ctx, "Bruno", http.MethodDelete, "shopping-lists/"+l.ID, nil); err != nil {
			return err
		}
		return w.shoppingStatus(http.StatusOK)
	})
	sc.Step(`^je tente de ajouter l'article "([^"]*)" à la liste "([^"]*)"$`, func(ctx context.Context, ingredient, list string) error {
		return w.addItem(ctx, "je", list, ingredient, "")
	})
	sc.Step(`^je tente de renommer la liste "([^"]*)" en "([^"]*)"$`, w.rename)
	sc.Step(`^je tente de désigner la liste "([^"]*)" comme principale$`, func(ctx context.Context, name string) error {
		return w.setMain(ctx, "je", name)
	})
	sc.Step(`^un message m'indique que la liste a été supprimée entre-temps$`, func() error { return w.shoppingError(404, "not_found") })

	// ENF-02
	sc.Step(`^la liste "([^"]*)" de la tribu "([^"]*)" contient l'article ajouté "([^"]*)"$`, w.otherTribeItem)
	sc.Step(`^la tribu "([^"]*)" a la liste "([^"]*)" vide$`, func(ctx context.Context, slug, name string) error {
		lists, err := w.otherLists(ctx, slug)
		if err != nil {
			return err
		}
		_, _, err = lists.Create(ctx, name)
		return err
	})
	sc.Step(`^je consulte les listes de courses$`, func(ctx context.Context) error {
		lists, err := w.shoppingLists(ctx, "je")
		w.shopping.seen = lists
		return err
	})
	sc.Step(`^je tente de (renommer|désigner comme principale|ajouter un article à|ajouter les courses des repas à|déclarer les courses faites de|supprimer) la liste "([^"]*)" de la tribu "([^"]*)" en utilisant son identifiant$`, w.crossAccessList)
	sc.Step(`^la liste "([^"]*)" de la tribu "([^"]*)" est inchangée$`, w.otherListUnchanged)
	sc.Step(`^je tente de cocher puis de retirer cet article de la tribu "([^"]*)" en utilisant son identifiant et celui de sa liste$`, w.crossAccessItem)
	sc.Step(`^chaque requête est refusée$`, func() error {
		for _, a := range w.shopping.attempts {
			if c := a[0].Code; c < 400 {
				return fmt.Errorf("status %d, body %s", c, a[0].Body)
			}
		}
		return nil
	})
	sc.Step(`^l'article "([^"]*)" de la tribu "([^"]*)" est inchangé$`, func(ctx context.Context, _, slug string) error {
		return w.otherListUnchanged(ctx, w.shopping.foreign.Name, slug)
	})
}

// oddListNames are the names of lists that cannot be written as such in a scenario (D12).
var oddListNames = map[string]string{
	"un texte de 41 caractères":                      strings.Repeat("é", 41),
	"un texte de 40 caractères":                      strings.Repeat("é", 40),
	"l'espace sans chasse U+200B":                    "\u200b",
	`"Marché" suivi du caractère de contrôle U+0007`: "Marché\a",
}

// shoppingLists reads every page of the shopping lists, as the front end does (ADR 0029).
func (w *world) shoppingLists(ctx context.Context, who string) ([]apiList, error) {
	return pages[apiList](ctx, w, who, "shopping-lists")
}

// listNamed returns the only list of this name, as the person reads it now.
func (w *world) listNamed(ctx context.Context, who, name string) (apiList, error) {
	lists, err := w.shoppingLists(ctx, who)
	if err != nil {
		return apiList{}, err
	}
	var found []apiList
	for _, l := range lists {
		if l.Name == name {
			found = append(found, l)
		}
	}
	if len(found) != 1 {
		return apiList{}, fmt.Errorf("%d lists %q in %+v", len(found), name, lists)
	}
	return found[0], nil
}

// openedListID returns the identifier of the list of this name: as "je" opened it, if so,
// otherwise as read now.
func (w *world) openedListID(ctx context.Context, name string) (string, error) {
	if l, ok := w.shopping.opened[name]; ok {
		return l.ID, nil
	}
	l, err := w.listNamed(ctx, "je", name)
	return l.ID, err
}

// shoppingSend sends a write on the lists; the response is kept for the outcome steps.
func (w *world) shoppingSend(ctx context.Context, who, method, path string, body any) error {
	rec, err := w.api(ctx, who, method, path, body)
	w.shopping.last = rec
	return err
}

func (w *world) shoppingStatus(status int) error {
	if w.shopping.last == nil {
		return errors.New("nothing sent on the lists")
	}
	if w.shopping.last.Code != status {
		return fmt.Errorf("status %d, want %d; body %s", w.shopping.last.Code, status, w.shopping.last.Body)
	}
	return nil
}

func (w *world) shoppingError(status int, code string) error {
	if err := w.shoppingStatus(status); err != nil {
		return err
	}
	var e struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(w.shopping.last.Body.Bytes(), &e); err != nil {
		return err
	}
	if e.Error != code {
		return fmt.Errorf("error %q, want %q", e.Error, code)
	}
	return nil
}

func (w *world) shoppingRefused() error {
	if w.shopping.last == nil {
		return errors.New("nothing sent on the lists")
	}
	if c := w.shopping.last.Code; c < 400 || c >= 500 {
		return fmt.Errorf("status %d, body %s", c, w.shopping.last.Body)
	}
	return nil
}

// writtenList returns the list of the last response.
func (w *world) writtenList() (apiList, error) {
	var r struct {
		List apiList `json:"list"`
	}
	if err := json.Unmarshal(w.shopping.last.Body.Bytes(), &r); err != nil {
		return apiList{}, err
	}
	return r.List, nil
}

func (w *world) givenDishWithTable(ctx context.Context, name string, servings int, table *godog.Table) error {
	body := dishBody{Name: name, Servings: servings}
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
	_, err := w.ensureDish(ctx, name, body)
	w.recordDish(name)
	return err
}

// givenList returns the list of this name, created by "je" if absent.
func (w *world) givenList(ctx context.Context, name string) (apiList, error) {
	if l, err := w.listNamed(ctx, "je", name); err == nil {
		return l, nil
	}
	if err := w.createList(ctx, name); err != nil {
		return apiList{}, err
	}
	if err := w.shoppingStatus(http.StatusCreated); err != nil {
		return apiList{}, err
	}
	return w.writtenList()
}

func (w *world) createList(ctx context.Context, name string) error {
	lists, err := w.shoppingLists(ctx, "je")
	if err != nil {
		return err
	}
	w.shopping.countBefore = len(lists)
	return w.shoppingSend(ctx, "je", http.MethodPost, "shopping-lists", map[string]string{"name": name})
}

func (w *world) createdListEmpty(ctx context.Context) error {
	if err := w.shoppingStatus(http.StatusCreated); err != nil {
		return err
	}
	created, err := w.writtenList()
	if err != nil {
		return err
	}
	lists, err := w.shoppingLists(ctx, "je")
	if err != nil {
		return err
	}
	for _, l := range lists {
		if l.ID == created.ID {
			if len(l.Items) != 0 || l.Main {
				return fmt.Errorf("list %+v", l)
			}
			return nil
		}
	}
	return fmt.Errorf("no list %+v in %+v", created, lists)
}

func (w *world) listNotCreated(ctx context.Context) error {
	if err := w.shoppingRefused(); err != nil {
		return err
	}
	lists, err := w.shoppingLists(ctx, "je")
	if err != nil {
		return err
	}
	if len(lists) != w.shopping.countBefore {
		return fmt.Errorf("%d lists, %d before", len(lists), w.shopping.countBefore)
	}
	return nil
}

// mainListIs makes the list of this name the main one in a context step, or checks it.
func (w *world) mainListIs(ctx context.Context, name string) error {
	if w.isContext() {
		if _, err := w.givenList(ctx, name); err != nil {
			return err
		}
		if err := w.setMain(ctx, "je", name); err != nil {
			return err
		}
		return w.shoppingStatus(http.StatusOK)
	}
	lists, err := w.shoppingLists(ctx, "je")
	if err != nil {
		return err
	}
	var mains []string
	for _, l := range lists {
		if l.Main {
			mains = append(mains, l.Name)
		}
	}
	if !slices.Equal(mains, []string{name}) {
		return fmt.Errorf("main lists %v, want %s", mains, name)
	}
	return nil
}

func (w *world) setMain(ctx context.Context, who, name string) error {
	id, err := w.openedListID(ctx, name)
	if who != "je" {
		var l apiList
		l, err = w.listNamed(ctx, who, name)
		id = l.ID
	}
	if err != nil {
		return err
	}
	return w.shoppingSend(ctx, who, http.MethodPut, "shopping-lists/main", map[string]string{"list": id})
}

func (w *world) rename(ctx context.Context, name, to string) error {
	id, err := w.openedListID(ctx, name)
	if err != nil {
		return err
	}
	return w.shoppingSend(ctx, "je", http.MethodPut, "shopping-lists/"+id+"/name", map[string]string{"name": to})
}

func (w *world) twoListsNamed(ctx context.Context, name string) error {
	lists, err := w.shoppingLists(ctx, "je")
	if err != nil {
		return err
	}
	n := 0
	for _, l := range lists {
		if l.Name == name {
			n++
		}
	}
	if n != 2 {
		return fmt.Errorf("%d lists %q in %+v", n, name, lists)
	}
	return nil
}

func (w *world) twentyLists(ctx context.Context) error {
	lists, err := w.shoppingLists(ctx, "je")
	if err != nil {
		return err
	}
	for i := len(lists) + 1; i <= shopping.MaxLists; i++ {
		if _, err := w.givenList(ctx, fmt.Sprintf("Liste %d", i)); err != nil {
			return err
		}
	}
	return nil
}

// createNewTribe creates a tribe with the admin command, and signs its first member in.
func (w *world) createNewTribe(ctx context.Context) error {
	email := "premier@exemple.fr"
	if err := w.initTribe(ctx, "La nouvelle tribu", newTribeSlug, email); err != nil {
		return err
	}
	d := w.otherDevice()
	if err := w.signIn(ctx, d, newTribeSlug, email); err != nil {
		return err
	}
	w.shopping.newTribe, w.shopping.newTribeDevice = newTribeSlug, d
	return nil
}

// newTribeLists reads the lists of the new tribe, or those of the tribe named.
func (w *world) newTribeLists(ctx context.Context, slug string) ([]apiList, error) {
	if slug != "" && slug != w.shopping.newTribe {
		if slug != w.kitchenTribe() {
			return nil, fmt.Errorf("not signed in to %s", slug)
		}
		return w.shoppingLists(ctx, "je")
	}
	rec, err := w.do(ctx, w.shopping.newTribeDevice, http.MethodGet, tribePath(w.shopping.newTribe, "api/shopping-lists"), nil)
	if err != nil {
		return nil, err
	}
	var page struct {
		Items []apiList `json:"items"`
		Next  string    `json:"next"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil || page.Next != "" {
		return nil, fmt.Errorf("GET shopping-lists: status %d, body %s: %v", rec.Code, rec.Body, err)
	}
	return page.Items, nil
}

func (w *world) onlyList(ctx context.Context, slug, name string) error {
	lists, err := w.newTribeLists(ctx, slug)
	if err != nil {
		return err
	}
	if len(lists) != 1 || lists[0].Name != name || len(lists[0].Items) != 0 || len(lists[0].Received) != 0 {
		return fmt.Errorf("lists %+v, want %s alone and empty", lists, name)
	}
	return nil
}

func (w *world) newTribeMainList(ctx context.Context, name string) error {
	lists, err := w.newTribeLists(ctx, "")
	if err != nil {
		return err
	}
	if len(lists) != 1 || lists[0].Name != name || !lists[0].Main {
		return fmt.Errorf("lists %+v, want %s main", lists, name)
	}
	return nil
}

// mainListStays: no route takes the role of the main list away without giving it to another
// (D16): its deletion is refused, PUT shopping-lists/main designates an existing list only,
// and a rename keeps the role. The main list is the same at the end, and alone.
func (w *world) mainListStays(ctx context.Context) error {
	lists, err := w.shoppingLists(ctx, "je")
	if err != nil {
		return err
	}
	i := slices.IndexFunc(lists, func(l apiList) bool { return l.Main })
	if i < 0 {
		return fmt.Errorf("no main list in %+v", lists)
	}
	main := lists[i]
	attempts := []struct {
		method, path string
		body         any
		status       int
	}{
		{http.MethodDelete, "shopping-lists/" + main.ID, nil, http.StatusConflict},
		{http.MethodPut, "shopping-lists/main", map[string]string{"list": ""}, http.StatusNotFound},
		{http.MethodPut, "shopping-lists/main", map[string]any{}, http.StatusNotFound},
		{http.MethodPut, "shopping-lists/main", map[string]string{"list": strings.Repeat("0", len(main.ID))}, http.StatusNotFound},
		{http.MethodPut, "shopping-lists/main", map[string]string{"list": main.ID}, http.StatusOK},
		{http.MethodPut, "shopping-lists/" + main.ID + "/name", map[string]string{"name": main.Name}, http.StatusOK},
		{http.MethodPut, "shopping-lists/" + main.ID, map[string]any{"main": false}, http.StatusMethodNotAllowed},
	}
	for _, a := range attempts {
		rec, err := w.api(ctx, "je", a.method, a.path, a.body)
		if err != nil {
			return err
		}
		if rec.Code != a.status {
			return fmt.Errorf("%s %s: status %d, want %d; body %s", a.method, a.path, rec.Code, a.status, rec.Body)
		}
	}
	after, err := w.shoppingLists(ctx, "je")
	if err != nil {
		return err
	}
	var mains []string
	for _, l := range after {
		if l.Main {
			mains = append(mains, l.ID)
		}
	}
	if !slices.Equal(mains, []string{main.ID}) {
		return fmt.Errorf("main lists %v, before %s", mains, main.ID)
	}
	return nil
}

// addItem adds an item by hand: quantity is "2 pièces", or empty for an item without
// quantity.
func (w *world) addItem(ctx context.Context, who, list, ingredient, quantity string) error {
	body := map[string]string{"ingredient": ingredient}
	if quantity != "" {
		q, unit, err := parseQuantity(quantity)
		if err != nil {
			return err
		}
		body["quantity"], body["unit"] = q, unit
	}
	id := ""
	if who == "je" {
		var err error
		if id, err = w.openedListID(ctx, list); err != nil {
			return err
		}
	} else {
		l, err := w.listNamed(ctx, who, list)
		if err != nil {
			return err
		}
		id = l.ID
	}
	return w.shoppingSend(ctx, who, http.MethodPost, "shopping-lists/"+id+"/items", body)
}

func (w *world) givenManual(ctx context.Context, list, ingredient, quantity string) error {
	if _, err := w.givenList(ctx, list); err != nil {
		return err
	}
	if err := w.addItem(ctx, "je", list, ingredient, quantity); err != nil {
		return err
	}
	return w.shoppingStatus(http.StatusCreated)
}

func (w *world) givenManualTable(ctx context.Context, list string, table *godog.Table) error {
	for _, row := range table.Rows[1:] {
		quantity := ""
		if q := row.Cells[1].Value; q != "" {
			quantity = q + " " + row.Cells[2].Value
		}
		if err := w.givenManual(ctx, list, row.Cells[0].Value, quantity); err != nil {
			return err
		}
	}
	return nil
}

func (w *world) givenManyItems(ctx context.Context, list string, n int) error {
	for i := range n {
		if err := w.givenManual(ctx, list, fmt.Sprintf("article %d", i+1), ""); err != nil {
			return err
		}
	}
	return nil
}

// sameQuantity tells whether the quantity of an item is the one of the scenario ("2
// pièces", "1,5 l"), "" for an item without quantity: the same amount in the same family,
// whatever the unit of the API.
func sameQuantity(it apiItem, label string) (bool, error) {
	if label == "" || it.Quantity == "" {
		return label == "" && it.Quantity == "", nil
	}
	q, unit, err := parseQuantity(label)
	if err != nil {
		return false, err
	}
	want, err := dish.ParseQuantity(q, dish.Unit(unit))
	if err != nil {
		return false, err
	}
	got, err := dish.ParseQuantity(it.Quantity, dish.Unit(it.Unit))
	if err != nil {
		return false, fmt.Errorf("item %+v: %w", it, err)
	}
	return got.Amount == want.Amount && got.Unit.Family() == want.Unit.Family(), nil
}

// findItem returns the items of the list of this ingredient, of this kind ("ajouté",
// "calculé", or any if empty) and quantity ("-" for any).
func findItems(l apiList, ingredient, kind, quantity string) ([]apiItem, error) {
	var found []apiItem
	for _, it := range l.Items {
		if it.Ingredient != ingredient || kind == "ajouté" && !it.Manual || kind == "calculé" && it.Manual {
			continue
		}
		if quantity != "-" {
			same, err := sameQuantity(it, quantity)
			if err != nil {
				return nil, err
			}
			if !same {
				continue
			}
		}
		found = append(found, it)
	}
	return found, nil
}

// holdsManual checks that the list holds one item added by hand of this ingredient and
// quantity (" pour 2 pièces", " sans quantité", or any if empty), unchecked if asked.
func (w *world) holdsManual(ctx context.Context, list, ingredient, quantity string, unchecked bool) error {
	l, err := w.listNamed(ctx, "je", list)
	if err != nil {
		return err
	}
	switch {
	case quantity == "":
		quantity = "-"
	case quantity == " sans quantité":
		quantity = ""
	default:
		quantity = strings.TrimPrefix(quantity, " pour ")
	}
	found, err := findItems(l, ingredient, "ajouté", quantity)
	if err != nil {
		return err
	}
	if len(found) != 1 || unchecked && found[0].Checked {
		return fmt.Errorf("list %s: %d items %q %q in %+v", list, len(found), ingredient, quantity, l.Items)
	}
	return nil
}

func (w *world) notFromAnyDish(ctx context.Context, ingredient string) error {
	if err := w.shoppingStatus(http.StatusCreated); err != nil {
		return err
	}
	l, err := w.writtenList()
	if err != nil {
		return err
	}
	found, err := findItems(l, ingredient, "", "-")
	if err != nil {
		return err
	}
	if len(found) != 1 || !found[0].Manual || len(found[0].Sources) != 0 {
		return fmt.Errorf("items %q: %+v", ingredient, found)
	}
	return nil
}

// itemOf returns the only item of this ingredient in the list as the person reads it, or in
// the list "je" opened.
func (w *world) itemOf(ctx context.Context, who, list, ingredient string) (apiList, apiItem, error) {
	l, opened := w.shopping.opened[list]
	if who != "je" || !opened {
		var err error
		if l, err = w.listNamed(ctx, who, list); err != nil {
			return apiList{}, apiItem{}, err
		}
	}
	found, err := findItems(l, ingredient, "", "-")
	if err != nil {
		return apiList{}, apiItem{}, err
	}
	if len(found) != 1 {
		return apiList{}, apiItem{}, fmt.Errorf("list %s: %d items %q in %+v", list, len(found), ingredient, l.Items)
	}
	return l, found[0], nil
}

func (w *world) removeItem(ctx context.Context, who, list, ingredient string) error {
	l, it, err := w.itemOf(ctx, who, list, ingredient)
	if err != nil {
		return err
	}
	return w.shoppingSend(ctx, who, http.MethodDelete, "shopping-lists/"+l.ID+"/items/"+it.ID, nil)
}

func (w *world) openList(ctx context.Context, name string) error {
	l, err := w.listNamed(ctx, "je", name)
	if err != nil {
		return err
	}
	if w.shopping.opened == nil {
		w.shopping.opened = map[string]apiList{}
	}
	w.shopping.opened[name] = l
	return nil
}

func (w *world) openListPicker(ctx context.Context) error {
	lists, err := w.shoppingLists(ctx, "je")
	if err != nil {
		return err
	}
	w.shopping.opened = map[string]apiList{}
	for _, l := range lists {
		w.shopping.opened[l.Name] = l
	}
	return nil
}

func (w *world) holdsNo(ctx context.Context, list, ingredient string) error {
	l, err := w.listNamed(ctx, "je", list)
	if err != nil {
		return err
	}
	if slices.ContainsFunc(l.Items, func(it apiItem) bool { return it.Ingredient == ingredient }) {
		return fmt.Errorf("list %s holds %q: %+v", list, ingredient, l.Items)
	}
	return nil
}

// noMessage: the last write succeeded, or found its item gone, which the screen takes as
// done without a word (D8, D11: 404 not_found on an item).
func (w *world) noMessage() error {
	if w.shopping.last != nil && w.shopping.last.Code == http.StatusNotFound {
		return w.shoppingError(http.StatusNotFound, "not_found")
	}
	return w.shoppingStatus(http.StatusOK)
}

// smallestUnits are the units of a computed item, by the unit of a line: the smallest of
// its family (conception, « API »).
var smallestUnits = map[dish.Unit]dish.Unit{dish.Kilogram: dish.Gram, dish.Centilitre: dish.Millilitre, dish.Litre: dish.Millilitre}

// givenComputed writes a computed item in the list, as the meal shopping leaves it (lot C):
// the dish served in the meal for the servings that bring this quantity, the item in the
// smallest unit of its family, its source, and the served dish received by the list with the
// lines of its dish (D5). Quantity is empty for an ingredient without quantity.
func (w *world) givenComputed(ctx context.Context, list, quantity, ingredient, dishName, day, moment string) error {
	l, err := w.givenList(ctx, list)
	if err != nil {
		return err
	}
	d, err := w.findDish(ctx, "je", dishName)
	if err != nil {
		return err
	}
	i := slices.IndexFunc(d.Ingredients, func(line apiLine) bool { return line.Name == ingredient })
	if i < 0 {
		return fmt.Errorf("no %s in %+v", ingredient, d)
	}
	servings := d.Servings
	var amount *dish.Quantity
	if quantity != "" {
		q, unit, err := parseQuantity(quantity)
		if err != nil {
			return err
		}
		want, err := dish.ParseQuantity(q, dish.Unit(unit))
		if err != nil {
			return err
		}
		line, err := dish.ParseQuantity(d.Ingredients[i].Quantity, dish.Unit(d.Ingredients[i].Unit))
		if err != nil {
			return fmt.Errorf("line %+v: %w", d.Ingredients[i], err)
		}
		if want.Amount*int64(d.Servings)%line.Amount != 0 {
			return fmt.Errorf("%s of %s: no whole servings of %s", quantity, ingredient, dishName)
		}
		servings = int(want.Amount * int64(d.Servings) / line.Amount)
		want.Unit = cmpOr(smallestUnits[want.Unit], want.Unit)
		amount = &want
	}
	m, err := parseMeal(day, moment)
	if err != nil {
		return err
	}
	if err := w.givenServed(ctx, dishName, servings, m); err != nil {
		return err
	}
	var served struct {
		ServedDish apiServed `json:"servedDish"`
	}
	if err := json.Unmarshal(w.planning.last.Body.Bytes(), &served); err != nil {
		return err
	}

	db, err := w.kitchenDB(ctx)
	if err != nil {
		return err
	}
	key := func(name string) (int64, error) {
		var k int64
		err := db.QueryRowContext(ctx, "SELECT id FROM ingredients WHERE normalized_name = ?", dish.NormalizeName(name)).Scan(&k)
		return k, err
	}
	ingredientKey, err := key(ingredient)
	if err != nil {
		return err
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return err
	}
	itemID := hex.EncodeToString(b)
	var itemAmount, itemUnit any
	if amount != nil {
		itemAmount, itemUnit = amount.Amount, string(amount.Unit)
	}
	writes := []struct {
		sql  string
		args []any
	}{
		{"INSERT INTO shopping_items VALUES (?, ?, ?, ?, ?, 0, 0, NULL)", []any{itemID, l.ID, ingredientKey, itemAmount, itemUnit}},
		{"INSERT INTO shopping_item_sources VALUES (?, ?, ?, ?, ?, ?, ?, ?)", []any{itemID, served.ServedDish.ID, d.ID, m.date, m.mealtime, servings, itemAmount, itemUnit}},
		{"INSERT INTO shopping_received VALUES (?, ?, ?, ?, ?)", []any{l.ID, served.ServedDish.ID, d.ID, servings, d.Servings}},
	}
	for pos, line := range d.Ingredients {
		k, err := key(line.Name)
		if err != nil {
			return err
		}
		var lineAmount, lineUnit any
		if line.Quantity != "" {
			q, err := dish.ParseQuantity(line.Quantity, dish.Unit(line.Unit))
			if err != nil {
				return err
			}
			lineAmount, lineUnit = q.Amount, line.Unit
		}
		writes = append(writes, struct {
			sql  string
			args []any
		}{"INSERT INTO shopping_received_lines VALUES (?, ?, ?, ?, ?, ?)", []any{l.ID, served.ServedDish.ID, pos, k, lineAmount, lineUnit}})
	}
	for _, wr := range writes {
		if _, err := db.ExecContext(ctx, wr.sql, wr.args...); err != nil {
			return fmt.Errorf("%s: %w", wr.sql, err)
		}
	}
	return nil
}

func cmpOr(a, b dish.Unit) dish.Unit {
	if a != "" {
		return a
	}
	return b
}

// kitchenDB returns the database of the tribe of the scenario.
func (w *world) kitchenDB(ctx context.Context) (*sql.DB, error) {
	store, err := w.openStore(ctx)
	if err != nil {
		return nil, err
	}
	return store.Tribe(ctx, w.kitchenTribe())
}

func (w *world) itemsInOrder(ctx context.Context, list string, table *godog.Table) error {
	l, err := w.listNamed(ctx, "je", list)
	if err != nil {
		return err
	}
	var got, want []string
	for _, it := range l.Items {
		kind := "ajouté"
		if !it.Manual {
			kind = "calculé"
		}
		checked := "non"
		if it.Checked {
			checked = "oui"
		}
		quantity := ""
		if it.Quantity != "" {
			quantity = it.Quantity + " " + it.Unit
		}
		got = append(got, strings.Join([]string{it.Ingredient, quantity, kind, checked}, " | "))
	}
	for _, row := range table.Rows[1:] {
		quantity := row.Cells[1].Value
		if quantity != "" {
			q, unit, err := parseQuantity(quantity)
			if err != nil {
				return err
			}
			quantity = q + " " + unit
		}
		want = append(want, strings.Join([]string{row.Cells[0].Value, quantity, row.Cells[2].Value, row.Cells[3].Value}, " | "))
	}
	if !slices.Equal(got, want) {
		return fmt.Errorf("items\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	return nil
}

func (w *world) check(ctx context.Context, who, list, ingredient string, checked bool) error {
	l, it, err := w.itemOf(ctx, who, list, ingredient)
	if err != nil {
		return err
	}
	return w.shoppingSend(ctx, who, http.MethodPut, "shopping-lists/"+l.ID+"/items/"+it.ID+"/checked", map[string]bool{"checked": checked})
}

func (w *world) checkedIs(ctx context.Context, who, list, ingredient string, checked bool) error {
	l, err := w.listNamed(ctx, who, list)
	if err != nil {
		return err
	}
	found, err := findItems(l, ingredient, "", "-")
	if err != nil {
		return err
	}
	if len(found) != 1 || found[0].Checked != checked {
		return fmt.Errorf("list %s: items %q %+v, want checked %v", list, ingredient, found, checked)
	}
	return nil
}

// checkOut declares the shopping done, with the items checked in the list as read.
func (w *world) checkOut(ctx context.Context, who string, l apiList) error {
	seen := []string{}
	for _, it := range l.Items {
		if it.Checked {
			seen = append(seen, it.ID)
		}
	}
	return w.shoppingSend(ctx, who, http.MethodPost, "shopping-lists/"+l.ID+"/checkout", map[string][]string{"items": seen})
}

// checkOutNow declares the shopping done of the list as the person reads it now.
func (w *world) checkOutNow(ctx context.Context, who, list string) error {
	l, err := w.listNamed(ctx, who, list)
	if err != nil {
		return err
	}
	if err := w.checkOut(ctx, who, l); err != nil {
		return err
	}
	return w.shoppingStatus(http.StatusOK)
}

func (w *world) checkAll(ctx context.Context, list, done string) error {
	l, err := w.listNamed(ctx, "je", list)
	if err != nil {
		return err
	}
	for _, it := range l.Items {
		if err := w.shoppingSend(ctx, "je", http.MethodPut, "shopping-lists/"+l.ID+"/items/"+it.ID+"/checked", map[string]bool{"checked": true}); err != nil {
			return err
		}
		if err := w.shoppingStatus(http.StatusOK); err != nil {
			return err
		}
	}
	if done == "" {
		return nil
	}
	return w.checkOutNow(ctx, "je", list)
}

func (w *world) removedCount(n int) error {
	if err := w.shoppingStatus(http.StatusOK); err != nil {
		return err
	}
	var r struct {
		Removed *int `json:"removed"`
	}
	if err := json.Unmarshal(w.shopping.last.Body.Bytes(), &r); err != nil {
		return err
	}
	if r.Removed == nil || *r.Removed != n {
		return fmt.Errorf("removed %v, want %d; body %s", r.Removed, n, w.shopping.last.Body)
	}
	return nil
}

// presentedList checks the list of a refused deletion: up to date, with the item added in
// between (D11).
func (w *world) presentedList(name, ingredient string) error {
	l, err := w.writtenList()
	if err != nil {
		return err
	}
	if l.Name != name || l.ID != w.shopping.opened[name].ID {
		return fmt.Errorf("list presented %+v, opened %+v", l, w.shopping.opened[name])
	}
	found, err := findItems(l, ingredient, "ajouté", "-")
	if err != nil {
		return err
	}
	if len(found) != 1 {
		return fmt.Errorf("list presented %+v, without %s", l, ingredient)
	}
	return nil
}

// otherLists returns the shopping lists of another tribe, used through the domain: the
// scenario signs in to the tribe of "je" only.
func (w *world) otherLists(ctx context.Context, slug string) (*shopping.Lists, error) {
	store, err := w.openStore(ctx)
	if err != nil {
		return nil, err
	}
	db, err := store.Tribe(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("tribe %q: %w", slug, err)
	}
	return shopping.NewLists(storage.NewShoppingStore(db)), nil
}

// otherList returns the list of this name of another tribe.
func (w *world) otherList(ctx context.Context, slug, name string) (shopping.List, error) {
	lists, err := w.otherLists(ctx, slug)
	if err != nil {
		return shopping.List{}, err
	}
	_, page, err := lists.Page(ctx, "")
	if err != nil {
		return shopping.List{}, err
	}
	for _, l := range page.Items {
		if l.Name == name {
			return l, nil
		}
	}
	return shopping.List{}, fmt.Errorf("no list %q in %s", name, slug)
}

func (w *world) otherTribeItem(ctx context.Context, list, slug, ingredient string) error {
	l, err := w.otherList(ctx, slug, list)
	if err != nil {
		return err
	}
	lists, err := w.otherLists(ctx, slug)
	if err != nil {
		return err
	}
	l, _, err = lists.AddItem(ctx, l.ID, dish.LineInput{Name: ingredient})
	w.shopping.foreign = l
	return err
}

// nothingOfOtherLists: neither the lists nor the items of the other tribe are in the lists
// "je" consulted.
func (w *world) nothingOfOtherLists(ctx context.Context, slug string) error {
	lists, err := w.otherLists(ctx, slug)
	if err != nil {
		return err
	}
	_, page, err := lists.Page(ctx, "")
	if err != nil {
		return err
	}
	seen, err := json.Marshal(w.shopping.seen)
	if err != nil {
		return err
	}
	for _, l := range page.Items {
		secrets := []string{string(l.ID)}
		for _, it := range l.Items {
			secrets = append(secrets, string(it.ID), it.Ingredient)
		}
		for _, secret := range secrets {
			if strings.Contains(string(seen), secret) {
				return fmt.Errorf("%q of %s seen in %s", secret, slug, seen)
			}
		}
	}
	return nil
}

// crossAccessList tries a write on the list of another tribe by its identifier, then on an
// identifier that exists nowhere, the same way. Each example names its route.
func (w *world) crossAccessList(ctx context.Context, action, name, slug string) error {
	target, err := w.otherList(ctx, slug, name)
	if err != nil {
		return err
	}
	w.shopping.foreign = target
	attempt := func(id string) (*httptest.ResponseRecorder, error) {
		switch action {
		case "renommer":
			return w.api(ctx, "je", http.MethodPut, "shopping-lists/"+id+"/name", map[string]string{"name": "Drive"})
		case "désigner comme principale":
			return w.api(ctx, "je", http.MethodPut, "shopping-lists/main", map[string]string{"list": id})
		case "ajouter un article à":
			return w.api(ctx, "je", http.MethodPost, "shopping-lists/"+id+"/items", map[string]string{"ingredient": "pain"})
		case "ajouter les courses des repas à":
			return w.api(ctx, "je", http.MethodPost, "shopping-lists/"+id+"/meal-shopping", map[string]string{"from": "2026-10-06", "to": "2026-10-12"})
		case "déclarer les courses faites de":
			return w.api(ctx, "je", http.MethodPost, "shopping-lists/"+id+"/checkout", map[string][]string{"items": {}})
		default:
			return w.api(ctx, "je", http.MethodDelete, "shopping-lists/"+id, nil)
		}
	}
	if w.kitchen.foreign, err = attempt(string(target.ID)); err != nil {
		return err
	}
	if w.kitchen.unknown, err = attempt(strings.Repeat("0", len(target.ID))); err != nil {
		return err
	}
	// Two 405 would look the same: each example must reach its route (revue #98).
	if w.kitchen.unknown.Code == http.StatusMethodNotAllowed {
		return fmt.Errorf("%s: no route, %s", action, w.kitchen.unknown.Body)
	}
	return nil
}

// crossAccessItem tries to check, then to remove, the item of another tribe by its
// identifier and that of its list, then by identifiers that exist nowhere, the same way.
func (w *world) crossAccessItem(ctx context.Context, slug string) error {
	l := w.shopping.foreign
	if len(l.Items) != 1 {
		return fmt.Errorf("list of %s %+v", slug, l)
	}
	unknownList, unknownItem := strings.Repeat("0", len(l.ID)), strings.Repeat("0", len(l.Items[0].ID))
	attempts := []struct {
		method, path string
		body         any
	}{
		{http.MethodPut, "shopping-lists/%s/items/%s/checked", map[string]bool{"checked": true}},
		{http.MethodDelete, "shopping-lists/%s/items/%s", nil},
	}
	for _, a := range attempts {
		foreign, err := w.api(ctx, "je", a.method, fmt.Sprintf(a.path, l.ID, l.Items[0].ID), a.body)
		if err != nil {
			return err
		}
		unknown, err := w.api(ctx, "je", a.method, fmt.Sprintf(a.path, unknownList, unknownItem), a.body)
		if err != nil {
			return err
		}
		w.shopping.attempts = append(w.shopping.attempts, [2]*httptest.ResponseRecorder{foreign, unknown})
	}
	return nil
}

func (w *world) otherListUnchanged(ctx context.Context, name, slug string) error {
	before := w.shopping.foreign
	now, err := w.otherList(ctx, slug, name)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(now, before) {
		return fmt.Errorf("list of %s %+v, before %+v", slug, now, before)
	}
	return nil
}
