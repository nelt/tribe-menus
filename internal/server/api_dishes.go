package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/nelt/tribe-menus/internal/collection"
	"github.com/nelt/tribe-menus/internal/dish"
	"github.com/nelt/tribe-menus/internal/storage"
	"github.com/nelt/tribe-menus/internal/tribe"
)

// maxDishBodyBytes bounds the body of a request on a dish: 40 lines do not fit in the 4 KiB
// of the rest of the API (D20).
const maxDishBodyBytes = 16 << 10

// Error codes of the dish library, besides those of dish.ErrorCode.
const (
	errDishChanged = "dish_changed"
	errInvalidSize = "invalid_size"
)

// dishResponse is a dish of the API. The lines of a dish returned by a write carry the
// default unit of their ingredient (D27); those of GET dishes do not, the catalog does.
type dishResponse struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Color       string         `json:"color"`
	Servings    int            `json:"servings"`
	Revision    int64          `json:"revision"`
	Ingredients []lineResponse `json:"ingredients"`
}

type lineResponse struct {
	Name        string `json:"name"`
	Quantity    string `json:"quantity,omitempty"`
	Unit        string `json:"unit,omitempty"`
	DefaultUnit string `json:"defaultUnit,omitempty"`
}

type ingredientResponse struct {
	Name        string `json:"name"`
	DefaultUnit string `json:"defaultUnit,omitempty"`
}

// savedResponse answers a write of a dish: the dish, and the versions of the collections
// after the write (ADR 0029).
type savedResponse struct {
	Dish     dishResponse        `json:"dish"`
	Versions collection.Versions `json:"versions"`
}

type dishErrorResponse struct {
	Error      string        `json:"error"`
	Line       *int          `json:"line,omitempty"`
	Ingredient string        `json:"ingredient,omitempty"`
	Dish       *dishResponse `json:"dish,omitempty"`
}

type tribeInfoResponse struct {
	Name          string `json:"name"`
	Size          int    `json:"size"`
	ActiveMembers int    `json:"activeMembers"`
}

// dishBody is a dish as sent by the front end: each line names its ingredient (Q20), the
// quantity is the text entered, with its unit code.
type dishBody struct {
	Name        string `json:"name"`
	Servings    int    `json:"servings"`
	Ingredients []struct {
		Name     string `json:"name"`
		Quantity string `json:"quantity"`
		Unit     string `json:"unit"`
	} `json:"ingredients"`
}

func (b dishBody) input() dish.Input {
	in := dish.Input{Name: b.Name, Servings: b.Servings}
	for _, l := range b.Ingredients {
		in.Lines = append(in.Lines, dish.LineInput{Name: l.Name, Quantity: l.Quantity, Unit: l.Unit})
	}
	return in
}

func newDishResponse(d dish.Dish, withDefaultUnits bool) dishResponse {
	r := dishResponse{ID: string(d.ID), Name: d.Name, Color: string(d.Color), Servings: d.Servings, Revision: d.Revision, Ingredients: []lineResponse{}}
	for _, l := range d.Lines {
		line := lineResponse{Name: l.Ingredient.Name}
		if l.Quantity != nil {
			line.Quantity, line.Unit = l.Quantity.Text(), string(l.Quantity.Unit)
		}
		if withDefaultUnits {
			line.DefaultUnit = string(l.Ingredient.DefaultUnit)
		}
		r.Ingredients = append(r.Ingredients, line)
	}
	return r
}

// dishRoutes registers the routes of the dish library and of the tribe, behind the session
// (plan bibliotheque, D28). No DELETE on a dish: the mux answers 405 (P2, Q5b).
func (a *api) dishRoutes(handle func(pattern string, h http.Handler)) {
	handle("GET /tribes/{tribe}/api/dishes", a.withMember(a.listDishes))
	handle("POST /tribes/{tribe}/api/dishes", a.withMember(a.createDish))
	handle("PUT /tribes/{tribe}/api/dishes/{id}", a.withMember(a.updateDish))
	handle("GET /tribes/{tribe}/api/ingredients", a.withMember(a.listIngredients))
	handle("GET /tribes/{tribe}/api/tribe", a.withMember(a.getTribe))
	handle("PUT /tribes/{tribe}/api/tribe/size", a.withMember(a.setTribeSize))
}

// withMember serves the request of a member only: 401 without session.
func (a *api) withMember(next http.HandlerFunc) http.Handler {
	return a.withSession(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := sessionOf(r); !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: errNoSession})
			return
		}
		next(w, r)
	})
}

// library is the dish library of the tribe of the request: its own database only (ENF-02).
func libraryOf(r *http.Request) *dish.Library {
	return dish.NewLibrary(storage.NewDishStore(tribeDBOf(r)))
}

// listDishes: GET dishes?after=…, a page of the library (D2, ADR 0029).
func (a *api) listDishes(w http.ResponseWriter, r *http.Request) {
	library := libraryOf(r)
	err := collection.Serve(w, r, collection.Source[dishResponse]{
		Name:    dish.DishesCollection,
		Version: func(ctx context.Context) (int64, error) { return library.Version(ctx, dish.DishesCollection) },
		Page: func(ctx context.Context, after string) (int64, collection.Page[dishResponse], error) {
			version, page, err := library.Dishes(ctx, after)
			out := collection.Page[dishResponse]{Next: page.Next}
			for _, d := range page.Items {
				out.Items = append(out.Items, newDishResponse(d, false))
			}
			return version, out, err
		},
	})
	if err != nil {
		a.server.fail(w, err)
	}
}

// listIngredients: GET ingredients?after=…, a page of the catalog (D2, ADR 0029).
func (a *api) listIngredients(w http.ResponseWriter, r *http.Request) {
	library := libraryOf(r)
	err := collection.Serve(w, r, collection.Source[ingredientResponse]{
		Name:    dish.IngredientsCollection,
		Version: func(ctx context.Context) (int64, error) { return library.Version(ctx, dish.IngredientsCollection) },
		Page: func(ctx context.Context, after string) (int64, collection.Page[ingredientResponse], error) {
			version, page, err := library.Catalog(ctx, after)
			out := collection.Page[ingredientResponse]{Next: page.Next}
			for _, i := range page.Items {
				out.Items = append(out.Items, ingredientResponse{Name: i.Name, DefaultUnit: string(i.DefaultUnit)})
			}
			return version, out, err
		},
	})
	if err != nil {
		a.server.fail(w, err)
	}
}

// createDish: POST dishes {name, servings, ingredients}. Servings left out are the tribe
// size (D16).
func (a *api) createDish(w http.ResponseWriter, r *http.Request) {
	var body dishBody
	if !readJSONLimit(w, r, &body, maxDishBodyBytes) {
		return
	}
	d, versions, err := libraryOf(r).Create(r.Context(), body.input())
	if err != nil {
		a.writeDishError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, savedResponse{Dish: newDishResponse(d, true), Versions: versions})
}

// updateDish: PUT dishes/{id} {revision, name, servings, ingredients}. 409 dish_changed with
// the current dish if the revision is outdated (Q25); 404 for a dish absent from the
// library of the tribe, whether it exists elsewhere or not (ENF-02).
func (a *api) updateDish(w http.ResponseWriter, r *http.Request) {
	var body struct {
		dishBody
		Revision *int64 `json:"revision"`
	}
	if !readJSONLimit(w, r, &body, maxDishBodyBytes) {
		return
	}
	if body.Revision == nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: errBadRequest})
		return
	}
	d, versions, err := libraryOf(r).Update(r.Context(), dish.ID(r.PathValue("id")), *body.Revision, body.input())
	if err != nil {
		a.writeDishError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, savedResponse{Dish: newDishResponse(d, true), Versions: versions})
}

func (a *api) writeDishError(w http.ResponseWriter, err error) {
	var inputErr *dish.InputError
	var changed *dish.ChangedError
	switch {
	case errors.As(err, &inputErr):
		resp := dishErrorResponse{Error: string(inputErr.Code), Ingredient: inputErr.Ingredient}
		if inputErr.Line >= 0 {
			resp.Line = &inputErr.Line
		}
		writeJSON(w, http.StatusBadRequest, resp)
	case errors.As(err, &changed):
		current := newDishResponse(changed.Current, true)
		writeJSON(w, http.StatusConflict, dishErrorResponse{Error: errDishChanged, Dish: &current})
	case errors.Is(err, dish.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: errNotFound})
	default:
		a.server.fail(w, err)
	}
}

// getTribe: GET tribe, its name, its size and its number of active members.
func (a *api) getTribe(w http.ResponseWriter, r *http.Request) {
	a.writeTribe(w, r, tribeOf(r))
}

// setTribeSize: PUT tribe/size {size}, from 1 to 50 (T3, Q31).
func (a *api) setTribeSize(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Size int `json:"size"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	store := tribeOf(r)
	err := store.SetSize(r.Context(), body.Size)
	if errors.Is(err, tribe.ErrInvalidSize) {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: errInvalidSize})
		return
	}
	if err != nil {
		a.server.fail(w, err)
		return
	}
	a.writeTribe(w, r, store)
}

func (a *api) writeTribe(w http.ResponseWriter, r *http.Request, store *tribe.Store) {
	ctx := r.Context()
	name, err := store.Name(ctx)
	if err != nil {
		a.server.fail(w, err)
		return
	}
	size, err := store.Size(ctx)
	if err != nil {
		a.server.fail(w, err)
		return
	}
	members, err := store.ActiveMembers(ctx)
	if err != nil {
		a.server.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tribeInfoResponse{Name: name, Size: size, ActiveMembers: members})
}
