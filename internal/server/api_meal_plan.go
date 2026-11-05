package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/nelt/tribe-menus/internal/collection"
	"github.com/nelt/tribe-menus/internal/dish"
	"github.com/nelt/tribe-menus/internal/mealplan"
	"github.com/nelt/tribe-menus/internal/storage"
)

// Error codes of the meal plan.
const (
	errInvalidDate       = "invalid_date"
	errInvalidMealtime   = "invalid_mealtime"
	errInvalidServings   = "invalid_servings"
	errUnknownDish       = "unknown_dish"
	errDishAlreadyServed = "dish_already_served"
)

// servedDishResponse is a served dish of the API: the dish by its identifier only, its name
// and color come from the library (conception du planning, « API »).
type servedDishResponse struct {
	ID       string `json:"id"`
	Date     string `json:"date"`
	Mealtime string `json:"mealtime"`
	Dish     string `json:"dish"`
	Servings int    `json:"servings"`
	Rank     int64  `json:"rank"`
}

// servedResponse answers a write of a served dish: the served dish, and the version of the
// meal plan after the write (ADR 0029).
type servedResponse struct {
	ServedDish servedDishResponse  `json:"servedDish"`
	Versions   collection.Versions `json:"versions"`
}

type versionsResponse struct {
	Versions collection.Versions `json:"versions"`
}

// alreadyServedResponse refuses a dish already served in the meal, with the dishes of the
// meal as they are now (Q21, D3).
type alreadyServedResponse struct {
	Error string               `json:"error"`
	Meal  []servedDishResponse `json:"meal"`
}

func newServedDishResponse(s mealplan.ServedDish) servedDishResponse {
	return servedDishResponse{
		ID: string(s.ID), Date: string(s.Date), Mealtime: string(s.Mealtime),
		Dish: string(s.Dish), Servings: s.Servings, Rank: s.Rank,
	}
}

// mealPlanRoutes registers the routes of the meal plan, behind the session; bodies of
// 4 KiB at most.
func (a *api) mealPlanRoutes(handle func(pattern string, h http.Handler)) {
	handle("GET /tribes/{tribe}/api/meal-plan", a.withMember(a.listServedDishes))
	handle("POST /tribes/{tribe}/api/meal-plan", a.withMember(a.serveDish))
	handle("PUT /tribes/{tribe}/api/meal-plan/{id}/servings", a.withMember(a.setServings))
	handle("DELETE /tribes/{tribe}/api/meal-plan/{id}", a.withMember(a.removeServedDish))
}

// planOf is the meal plan of the tribe of the request: its own database only (ENF-02).
func planOf(r *http.Request) *mealplan.Plan {
	return mealplan.NewPlan(storage.NewMealPlanStore(tribeDBOf(r)))
}

// listServedDishes: GET meal-plan?after=…, a page of the meal plan (D1, ADR 0029).
func (a *api) listServedDishes(w http.ResponseWriter, r *http.Request) {
	plan := planOf(r)
	err := collection.Serve(w, r, collection.Source[servedDishResponse]{
		Name:    mealplan.Collection,
		Version: plan.Version,
		Page: func(ctx context.Context, after string) (int64, collection.Page[servedDishResponse], error) {
			version, page, err := plan.ServedDishes(ctx, after)
			out := collection.Page[servedDishResponse]{Next: page.Next}
			for _, s := range page.Items {
				out.Items = append(out.Items, newServedDishResponse(s))
			}
			return version, out, err
		},
	})
	if err != nil {
		a.server.fail(w, err)
	}
}

// serveDish: POST meal-plan {date, mealtime, dish, servings?}. Servings left out are the
// tribe size at the time of the addition (D9); 409 dish_already_served with the meal as it
// is now (Q21, D3).
func (a *api) serveDish(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Date     string `json:"date"`
		Mealtime string `json:"mealtime"`
		Dish     string `json:"dish"`
		Servings *int   `json:"servings"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	s, versions, err := planOf(r).Serve(r.Context(), mealplan.ServeInput{
		Date: body.Date, Mealtime: body.Mealtime, Dish: dish.ID(body.Dish), Servings: body.Servings,
	})
	if err != nil {
		a.writeMealPlanError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, servedResponse{ServedDish: newServedDishResponse(s), Versions: versions})
}

// setServings: PUT meal-plan/{id}/servings {servings}, the last save wins (D3); 404 for a
// served dish removed in between, unknown or of another tribe (ENF-02).
func (a *api) setServings(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Servings int `json:"servings"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	s, versions, err := planOf(r).SetServings(r.Context(), mealplan.ID(r.PathValue("id")), body.Servings)
	if err != nil {
		a.writeMealPlanError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, servedResponse{ServedDish: newServedDishResponse(s), Versions: versions})
}

// removeServedDish: DELETE meal-plan/{id}, 200 with the versions, for the copy to take
// them as after any write; 404 as for the servings (D19).
func (a *api) removeServedDish(w http.ResponseWriter, r *http.Request) {
	versions, err := planOf(r).Remove(r.Context(), mealplan.ID(r.PathValue("id")))
	if err != nil {
		a.writeMealPlanError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, versionsResponse{Versions: versions})
}

func (a *api) writeMealPlanError(w http.ResponseWriter, err error) {
	var already *mealplan.AlreadyServedError
	switch {
	case errors.Is(err, mealplan.ErrInvalidDate):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: errInvalidDate})
	case errors.Is(err, mealplan.ErrInvalidMealtime):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: errInvalidMealtime})
	case errors.Is(err, mealplan.ErrInvalidServings):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: errInvalidServings})
	case errors.Is(err, mealplan.ErrUnknownDish):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: errUnknownDish})
	case errors.As(err, &already):
		meal := make([]servedDishResponse, 0, len(already.Meal))
		for _, s := range already.Meal {
			meal = append(meal, newServedDishResponse(s))
		}
		writeJSON(w, http.StatusConflict, alreadyServedResponse{Error: errDishAlreadyServed, Meal: meal})
	case errors.Is(err, mealplan.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: errNotFound})
	default:
		a.server.fail(w, err)
	}
}
