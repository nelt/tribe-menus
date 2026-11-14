package acceptance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/cucumber/godog"
)

// moveState is what the move scenarios remember: the items "je" selected, by identifier, and
// the list they were selected in.
type moveState struct {
	selection []string
	from      string
}

// registerMoveSteps: the selection, the move and the new list of liste-courses.feature (C9,
// C10, plan courses-suite, D10), and their scenarios of compartimentage-tribus.feature.
func (w *world) registerMoveSteps(sc *godog.ScenarioContext) {
	sc.Step(`^(?:je sélectionne|j'ai sélectionné) "([^"]*)"(?: et "([^"]*)")? dans la liste "([^"]*)"$`, w.selectItems)
	sc.Step(`^je déplace la sélection vers la liste "([^"]*)"$`, func(ctx context.Context, list string) error {
		id, err := w.openedListID(ctx, list)
		if err != nil {
			return err
		}
		return w.moveSelection(ctx, map[string]any{"to": id})
	})
	sc.Step(`^je crée la liste "([^"]*)" à partir de la sélection$`, func(ctx context.Context, name string) error {
		return w.moveSelection(ctx, map[string]any{"name": name})
	})
	sc.Step(`^(je déplace|j'ai déplacé) "([^"]*)" de la liste "([^"]*)" vers (?:la liste "([^"]*)"|une nouvelle liste "([^"]*)")$`, w.moveItem)
	sc.Step(`^un message m'indique que (\d+) articles? n'(?:était|étaient) plus dans la liste$`, w.missingCount)
	sc.Step(`^un message m'indique que la liste "[^"]*" a été supprimée entre-temps$`, func() error {
		return w.shoppingError(http.StatusNotFound, "target_not_found")
	})
	sc.Step(`^la ligne "([^"]*)" de la liste "([^"]*)" indique qu'elle provient de (\d+) plats? servis?$`, func(ctx context.Context, ingredient, list string, n int) error {
		return w.holdsComputed(ctx, list, "-", ingredient, fmt.Sprint(n))
	})
	sc.Step(`^les listes de la tribu sont, dans l'ordre, (.+)$`, w.listsInOrder)

	// ENF-02
	sc.Step(`^je tente de déplacer "([^"]*)" vers la liste "([^"]*)" de la tribu "([^"]*)" en utilisant son identifiant$`, w.crossAccessTarget)
	sc.Step(`^la réponse est la même que pour une liste d'arrivée qui n'existe pas$`, func() error {
		if err := w.sameAsUnknownID(); err != nil {
			return err
		}
		if !strings.Contains(w.kitchen.unknown.Body.String(), `"target_not_found"`) {
			return fmt.Errorf("unknown target list: %s", w.kitchen.unknown.Body)
		}
		return nil
	})
}

// selectItems selects items of a list, as "je" reads it now; the lists are read then, as the
// screen knows them.
func (w *world) selectItems(ctx context.Context, a, b, list string) error {
	lists, err := w.shoppingLists(ctx, "je")
	if err != nil {
		return err
	}
	if w.shopping.opened == nil {
		w.shopping.opened = map[string]apiList{}
	}
	for _, l := range lists {
		w.shopping.opened[l.Name] = l
	}
	w.move = moveState{from: list}
	for _, ingredient := range []string{a, b} {
		if ingredient == "" {
			continue
		}
		_, it, err := w.itemOf(ctx, "je", list, ingredient)
		if err != nil {
			return err
		}
		w.move.selection = append(w.move.selection, it.ID)
	}
	return nil
}

// moveSelection moves the selection to the target of the body: "to" a list, or a new list
// of this "name".
func (w *world) moveSelection(ctx context.Context, body map[string]any) error {
	id, err := w.openedListID(ctx, w.move.from)
	if err != nil {
		return err
	}
	body["items"] = w.move.selection
	return w.shoppingSend(ctx, "je", http.MethodPost, "shopping-lists/"+id+"/move", body)
}

// moveItem moves the item of this ingredient, as "je" reads it now, to a list or to a new
// one; in a context step, it must succeed.
func (w *world) moveItem(ctx context.Context, verb, ingredient, list, to, name string) error {
	l, it, err := w.itemOf(ctx, "je", list, ingredient)
	if err != nil {
		return err
	}
	body := map[string]any{"items": []string{it.ID}, "name": name}
	if name == "" {
		target, err := w.listNamed(ctx, "je", to)
		if err != nil {
			return err
		}
		body = map[string]any{"items": []string{it.ID}, "to": target.ID}
	}
	if err := w.shoppingSend(ctx, "je", http.MethodPost, "shopping-lists/"+l.ID+"/move", body); err != nil {
		return err
	}
	if verb == "j'ai déplacé" {
		return w.shoppingStatus(http.StatusOK)
	}
	return nil
}

func (w *world) missingCount(n int) error {
	if err := w.shoppingStatus(http.StatusOK); err != nil {
		return err
	}
	var r struct {
		Missing *int `json:"missing"`
	}
	if err := json.Unmarshal(w.shopping.last.Body.Bytes(), &r); err != nil {
		return err
	}
	if r.Missing == nil || *r.Missing != n {
		return fmt.Errorf("missing %v, want %d; body %s", r.Missing, n, w.shopping.last.Body)
	}
	return nil
}

var quotedNames = regexp.MustCompile(`"([^"]*)"`)

// listsInOrder checks the order of the lists of the list picker (C2): the main list first,
// then by rank of creation.
func (w *world) listsInOrder(ctx context.Context, names string) error {
	lists, err := w.shoppingLists(ctx, "je")
	if err != nil {
		return err
	}
	slices.SortFunc(lists, func(a, b apiList) int {
		if a.Main != b.Main {
			return map[bool]int{true: -1, false: 1}[a.Main]
		}
		return int(a.Rank - b.Rank)
	})
	var got, want []string
	for _, l := range lists {
		got = append(got, l.Name)
	}
	for _, m := range quotedNames.FindAllStringSubmatch(names, -1) {
		want = append(want, m[1])
	}
	if !slices.Equal(got, want) {
		return fmt.Errorf("lists %q, want %q", got, want)
	}
	return nil
}

// crossAccessTarget tries to move an item of the list "Courses" of "je" to the list of another tribe by its
// identifier, then to an identifier that exists nowhere, the same way.
func (w *world) crossAccessTarget(ctx context.Context, ingredient, name, slug string) error {
	target, err := w.otherList(ctx, slug, name)
	if err != nil {
		return err
	}
	l, it, err := w.itemOf(ctx, "je", "Courses", ingredient)
	if err != nil {
		return err
	}
	attempt := func(id string) error {
		return w.shoppingSend(ctx, "je", http.MethodPost, "shopping-lists/"+l.ID+"/move", map[string]any{"items": []string{it.ID}, "to": id})
	}
	if err := attempt(string(target.ID)); err != nil {
		return err
	}
	w.kitchen.foreign = w.shopping.last
	if err := attempt(strings.Repeat("0", len(target.ID))); err != nil {
		return err
	}
	w.kitchen.unknown = w.shopping.last
	return nil
}
