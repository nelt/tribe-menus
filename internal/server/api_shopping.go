package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/nelt/tribe-menus/internal/collection"
	"github.com/nelt/tribe-menus/internal/dish"
	"github.com/nelt/tribe-menus/internal/shopping"
	"github.com/nelt/tribe-menus/internal/storage"
)

// maxCheckoutBodyBytes bounds the body of the shopping done: up to 300 identifiers of 32
// hexadecimal characters, about 10.5 KiB, do not fit in the 4 KiB of the rest of the API
// (conception des listes de courses, « API »).
const maxCheckoutBodyBytes = 16 << 10

// Error codes of the shopping lists, besides those of dish.ErrorCode for an item.
const (
	errListNameEmpty   = "list_name_empty"
	errListNameInvalid = "list_name_invalid"
	errListNameTooLong = "list_name_too_long"
	errTooManyLists    = "too_many_lists"
	errTooManyItems    = "too_many_items"
	errListNotEmpty    = "list_not_empty"
	errMainList        = "main_list"
	errComputedItem    = "computed_item"
	errInvalidPeriod   = "invalid_period"
	errPeriodTooLong   = "period_too_long"
)

// listResponse is a shopping list of the API: its items in the order of Q40, and the
// identifiers of the served dishes it received (Q22, D5).
type listResponse struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Main     bool           `json:"main"`
	Rank     int64          `json:"rank"`
	Items    []itemResponse `json:"items"`
	Received []string       `json:"received"`
}

// itemResponse is an item: the ingredient by its name, as the lines of a dish; the quantity
// as their text, with its unit; the taken quantity in the same unit (conception, « API »,
// Q41).
type itemResponse struct {
	ID         string           `json:"id"`
	Ingredient string           `json:"ingredient"`
	Quantity   string           `json:"quantity,omitempty"`
	Unit       string           `json:"unit,omitempty"`
	Taken      string           `json:"taken,omitempty"`
	Checked    bool             `json:"checked"`
	Manual     bool             `json:"manual"`
	Sources    []sourceResponse `json:"sources"`
}

type sourceResponse struct {
	ServedDish string `json:"servedDish"`
	Dish       string `json:"dish"`
	Date       string `json:"date"`
	Mealtime   string `json:"mealtime"`
	Servings   int    `json:"servings"`
	Quantity   string `json:"quantity,omitempty"`
	Unit       string `json:"unit,omitempty"`
}

// listWrittenResponse answers a write on a list: the list up to date, and the versions of
// the collections after the write (ADR 0029).
type listWrittenResponse struct {
	List     listResponse        `json:"list"`
	Versions collection.Versions `json:"versions"`
}

type checkedOutResponse struct {
	List     listResponse        `json:"list"`
	Removed  int                 `json:"removed"`
	Versions collection.Versions `json:"versions"`
}

// mealShoppingResponse answers the meal shopping (C4): the list, and the number of served
// dishes added and ignored, already received by the list (Q22).
type mealShoppingResponse struct {
	List     listResponse        `json:"list"`
	Added    int                 `json:"added"`
	Ignored  int                 `json:"ignored"`
	Versions collection.Versions `json:"versions"`
}

// staleResponse gives the stale lists, in the order of their identifiers (C8, D7).
type staleResponse struct {
	Lists []shopping.ListID `json:"lists"`
}

// listNotEmptyResponse refuses to delete a list that holds items, with the list up to date
// (D11).
type listNotEmptyResponse struct {
	Error string       `json:"error"`
	List  listResponse `json:"list"`
}

func quantityText(q *dish.Quantity) (quantity, unit string) {
	if q == nil {
		return "", ""
	}
	return q.Text(), string(q.Unit)
}

func newListResponse(l shopping.List) listResponse {
	r := listResponse{ID: string(l.ID), Name: l.Name, Main: l.Main, Rank: l.Rank, Items: []itemResponse{}, Received: []string{}}
	for _, it := range l.Items {
		item := itemResponse{ID: string(it.ID), Ingredient: it.Ingredient, Checked: it.Checked, Manual: it.Manual, Sources: []sourceResponse{}}
		item.Quantity, item.Unit = quantityText(it.Quantity)
		item.Taken, _ = quantityText(it.Taken)
		for _, s := range it.Sources {
			source := sourceResponse{ServedDish: string(s.ServedDish), Dish: string(s.Dish), Date: s.Date, Mealtime: s.Mealtime, Servings: s.Servings}
			source.Quantity, source.Unit = quantityText(s.Quantity)
			item.Sources = append(item.Sources, source)
		}
		r.Items = append(r.Items, item)
	}
	for _, id := range l.Received {
		r.Received = append(r.Received, string(id))
	}
	return r
}

// shoppingRoutes registers the routes of the shopping lists, behind the session; bodies of
// 4 KiB at most, 16 KiB for the shopping done.
func (a *api) shoppingRoutes(handle func(pattern string, h http.Handler)) {
	handle("GET /tribes/{tribe}/api/shopping-lists", a.withMember(a.listShoppingLists))
	handle("POST /tribes/{tribe}/api/shopping-lists", a.withMember(a.createList))
	handle("PUT /tribes/{tribe}/api/shopping-lists/main", a.withMember(a.setMainList))
	handle("PUT /tribes/{tribe}/api/shopping-lists/{id}/name", a.withMember(a.renameList))
	handle("DELETE /tribes/{tribe}/api/shopping-lists/{id}", a.withMember(a.deleteList))
	handle("POST /tribes/{tribe}/api/shopping-lists/{id}/items", a.withMember(a.addItem))
	handle("DELETE /tribes/{tribe}/api/shopping-lists/{id}/items/{item}", a.withMember(a.removeItem))
	handle("PUT /tribes/{tribe}/api/shopping-lists/{id}/items/{item}/checked", a.withMember(a.setChecked))
	handle("POST /tribes/{tribe}/api/shopping-lists/{id}/checkout", a.withMember(a.checkOut))
	handle("POST /tribes/{tribe}/api/shopping-lists/{id}/meal-shopping", a.withMember(a.addMealShopping))
	handle("GET /tribes/{tribe}/api/shopping-lists/stale", a.withMember(a.staleLists))
	handle("POST /tribes/{tribe}/api/shopping-lists/{id}/recompute", a.withMember(a.recomputeList))
}

// listsOf is the shopping lists of the tribe of the request: its own database only (ENF-02).
func listsOf(r *http.Request) *shopping.Lists {
	return shopping.NewLists(storage.NewShoppingStore(tribeDBOf(r)))
}

func listID(r *http.Request) shopping.ListID {
	return shopping.ListID(r.PathValue("id"))
}

func itemID(r *http.Request) shopping.ItemID {
	return shopping.ItemID(r.PathValue("item"))
}

// listShoppingLists: GET shopping-lists?after=…, a page of the lists with their items (D4,
// ADR 0029).
func (a *api) listShoppingLists(w http.ResponseWriter, r *http.Request) {
	lists := listsOf(r)
	err := collection.Serve(w, r, collection.Source[listResponse]{
		Name:    shopping.Collection,
		Version: lists.Version,
		Page: func(ctx context.Context, after string) (int64, collection.Page[listResponse], error) {
			version, page, err := lists.Page(ctx, after)
			out := collection.Page[listResponse]{Next: page.Next}
			for _, l := range page.Items {
				out.Items = append(out.Items, newListResponse(l))
			}
			return version, out, err
		},
	})
	if err != nil {
		a.server.fail(w, err)
	}
}

// createList: POST shopping-lists {name}, an empty list, not the main one (C2).
func (a *api) createList(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	l, versions, err := listsOf(r).Create(r.Context(), body.Name)
	if err != nil {
		a.writeShoppingError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, listWrittenResponse{List: newListResponse(l), Versions: versions})
}

// renameList: PUT shopping-lists/{id}/name {name}, the last save wins (D11).
func (a *api) renameList(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	l, versions, err := listsOf(r).Rename(r.Context(), listID(r), body.Name)
	if err != nil {
		a.writeShoppingError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, listWrittenResponse{List: newListResponse(l), Versions: versions})
}

// setMainList: PUT shopping-lists/main {list}, the versions only: the copy swaps the two
// flags itself (conception, « Données dans le navigateur »).
func (a *api) setMainList(w http.ResponseWriter, r *http.Request) {
	var body struct {
		List string `json:"list"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	versions, err := listsOf(r).SetMain(r.Context(), shopping.ListID(body.List))
	if err != nil {
		a.writeShoppingError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, versionsResponse{Versions: versions})
}

// deleteList: DELETE shopping-lists/{id}, an empty list other than the main one (C12); 409
// list_not_empty with the list up to date, 409 main_list (D11).
func (a *api) deleteList(w http.ResponseWriter, r *http.Request) {
	versions, err := listsOf(r).Delete(r.Context(), listID(r))
	if err != nil {
		a.writeShoppingError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, versionsResponse{Versions: versions})
}

// addItem: POST shopping-lists/{id}/items {ingredient, quantity?, unit?}, an item added by
// hand (C3), with the rules of a line of a dish (Q13, Q31).
func (a *api) addItem(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Ingredient string `json:"ingredient"`
		Quantity   string `json:"quantity"`
		Unit       string `json:"unit"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	l, versions, err := listsOf(r).AddItem(r.Context(), listID(r), dish.LineInput{Name: body.Ingredient, Quantity: body.Quantity, Unit: body.Unit})
	if err != nil {
		a.writeShoppingError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, listWrittenResponse{List: newListResponse(l), Versions: versions})
}

// removeItem: DELETE shopping-lists/{id}/items/{item}, an item added by hand (C3); 409
// computed_item; 404 for an item already removed, which the screen takes as done (D11).
func (a *api) removeItem(w http.ResponseWriter, r *http.Request) {
	l, versions, err := listsOf(r).RemoveItem(r.Context(), listID(r), itemID(r))
	if err != nil {
		a.writeShoppingError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, listWrittenResponse{List: newListResponse(l), Versions: versions})
}

// setChecked: PUT shopping-lists/{id}/items/{item}/checked {checked}, the last save wins
// (D8).
func (a *api) setChecked(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Checked *bool `json:"checked"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	if body.Checked == nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: errBadRequest})
		return
	}
	l, versions, err := listsOf(r).SetChecked(r.Context(), listID(r), itemID(r), *body.Checked)
	if err != nil {
		a.writeShoppingError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, listWrittenResponse{List: newListResponse(l), Versions: versions})
}

// checkOut: POST shopping-lists/{id}/checkout {items}, the items the member saw checked
// (C11, D9); the number of items removed.
func (a *api) checkOut(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Items []shopping.ItemID `json:"items"`
	}
	if !readJSONLimit(w, r, &body, maxCheckoutBodyBytes) {
		return
	}
	l, removed, versions, err := listsOf(r).CheckOut(r.Context(), listID(r), body.Items)
	if err != nil {
		a.writeShoppingError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, checkedOutResponse{List: newListResponse(l), Removed: removed, Versions: versions})
}

// addMealShopping: POST shopping-lists/{id}/meal-shopping {from, to}, the shopping of the
// meals of the period, bounds included (C4, P2).
func (a *api) addMealShopping(w http.ResponseWriter, r *http.Request) {
	var body struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	l, added, ignored, versions, err := listsOf(r).AddMealShopping(r.Context(), listID(r), body.From, body.To)
	if err != nil {
		a.writeShoppingError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, mealShoppingResponse{List: newListResponse(l), Added: added, Ignored: ignored, Versions: versions})
}

// staleLists: GET shopping-lists/stale, the lists a recomputation would change, computed
// at each call: no ETag nor 304, a derived state outside the collections (ADR 0031).
func (a *api) staleLists(w http.ResponseWriter, r *http.Request) {
	ids, err := listsOf(r).Stale(r.Context())
	if err != nil {
		a.server.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, staleResponse{Lists: append([]shopping.ListID{}, ids...)})
}

// recomputeList: POST shopping-lists/{id}/recompute, the list brought back in line with the
// meals (C8, D5); versions empty when nothing changes; 409 too_many_items.
func (a *api) recomputeList(w http.ResponseWriter, r *http.Request) {
	l, versions, err := listsOf(r).Recompute(r.Context(), listID(r))
	if err != nil {
		a.writeShoppingError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, listWrittenResponse{List: newListResponse(l), Versions: versions})
}

func (a *api) writeShoppingError(w http.ResponseWriter, err error) {
	var itemErr *shopping.ItemError
	var notEmpty *shopping.NotEmptyError
	switch {
	case errors.Is(err, shopping.ErrListNameEmpty):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: errListNameEmpty})
	case errors.Is(err, shopping.ErrListNameInvalid):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: errListNameInvalid})
	case errors.Is(err, shopping.ErrListNameTooLong):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: errListNameTooLong})
	case errors.As(err, &itemErr):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: string(itemErr.Code)})
	case errors.Is(err, shopping.ErrInvalidDate):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: errInvalidDate})
	case errors.Is(err, shopping.ErrInvalidPeriod):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: errInvalidPeriod})
	case errors.Is(err, shopping.ErrPeriodTooLong):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: errPeriodTooLong})
	case errors.Is(err, shopping.ErrTooManyLists):
		writeJSON(w, http.StatusConflict, errorResponse{Error: errTooManyLists})
	case errors.Is(err, shopping.ErrTooManyItems):
		writeJSON(w, http.StatusConflict, errorResponse{Error: errTooManyItems})
	case errors.As(err, &notEmpty):
		writeJSON(w, http.StatusConflict, listNotEmptyResponse{Error: errListNotEmpty, List: newListResponse(notEmpty.List)})
	case errors.Is(err, shopping.ErrMainList):
		writeJSON(w, http.StatusConflict, errorResponse{Error: errMainList})
	case errors.Is(err, shopping.ErrComputedItem):
		writeJSON(w, http.StatusConflict, errorResponse{Error: errComputedItem})
	case errors.Is(err, shopping.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: errNotFound})
	default:
		a.server.fail(w, err)
	}
}
