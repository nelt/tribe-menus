package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// createDish creates a dish of this name and returns its identifier.
func (f *apiFixture) createDish(t *testing.T, slug, name string, cookie *http.Cookie) string {
	t.Helper()
	rec := f.do("POST", "/tribes/"+slug+"/api/dishes", `{"name":"`+name+`"}`, cookie)
	wantStatus(t, rec, http.StatusCreated)
	return decode[savedResponse](t, rec).Dish.ID
}

func serveBody(date, mealtime, dish string) string {
	return fmt.Sprintf(`{"date":%q,"mealtime":%q,"dish":%q}`, date, mealtime, dish)
}

func TestMealPlanRoutesNeedASession(t *testing.T) {
	f := newAPIFixture(t)
	routes := []struct{ method, path, body string }{
		{"GET", "/api/meal-plan", ""},
		{"POST", "/api/meal-plan", serveBody("2026-10-06", "dinner", "0123")},
		{"PUT", "/api/meal-plan/0123/servings", `{"servings":3}`},
		{"DELETE", "/api/meal-plan/0123", ""},
	}
	for _, slug := range []string{"martin", "dupont"} {
		for _, r := range routes {
			rec := f.do(r.method, "/tribes/"+slug+r.path, r.body)
			if rec.Code != http.StatusUnauthorized || strings.TrimSpace(rec.Body.String()) != `{"error":"no_session"}` {
				t.Errorf("%s %s %s: status %d, body %s", slug, r.method, r.path, rec.Code, rec.Body)
			}
		}
	}
}

func TestServeSetAndRemove(t *testing.T) {
	f := newAPIFixture(t)
	alice := f.signIn(t, "martin", "alice@exemple.fr")
	chili := f.createDish(t, "martin", "Chili con carne", alice)
	vege := f.createDish(t, "martin", "Chili végétarien", alice)
	wantStatus(t, f.do("PUT", "/tribes/martin/api/tribe/size", `{"size":5}`, alice), http.StatusOK)

	rec := f.do("POST", "/tribes/martin/api/meal-plan", serveBody("2026-10-06", "dinner", chili), alice)
	wantStatus(t, rec, http.StatusCreated)
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control %q", rec.Header().Get("Cache-Control"))
	}
	served := decode[servedResponse](t, rec)
	id := served.ServedDish.ID
	want := servedDishResponse{ID: id, Date: "2026-10-06", Mealtime: "dinner", Dish: chili, Servings: 5, Rank: 1}
	if served.ServedDish != want || len(id) != 32 || fmt.Sprint(served.Versions) != "map[meal_plan:1]" {
		t.Errorf("served %+v, want %+v", served, want)
	}

	// Servings given: kept.
	rec = f.do("POST", "/tribes/martin/api/meal-plan", `{"date":"2026-10-06","mealtime":"dinner","dish":"`+vege+`","servings":2}`, alice)
	wantStatus(t, rec, http.StatusCreated)
	second := decode[servedResponse](t, rec).ServedDish
	if second.Servings != 2 || second.Rank != 2 {
		t.Errorf("second %+v", second)
	}

	// Already served (Q21, D3): the meal as it is now, by rank.
	rec = f.do("POST", "/tribes/martin/api/meal-plan", serveBody("2026-10-06", "dinner", chili), alice)
	wantStatus(t, rec, http.StatusConflict)
	conflict := decode[alreadyServedResponse](t, rec)
	if conflict.Error != "dish_already_served" || len(conflict.Meal) != 2 || conflict.Meal[0] != want || conflict.Meal[1] != second {
		t.Errorf("conflict %+v", conflict)
	}

	rec = f.do("PUT", "/tribes/martin/api/meal-plan/"+id+"/servings", `{"servings":3}`, alice)
	wantStatus(t, rec, http.StatusOK)
	set := decode[servedResponse](t, rec)
	want.Servings = 3
	if set.ServedDish != want || set.Versions["meal_plan"] != 3 {
		t.Errorf("set %+v", set)
	}

	rec = f.do("DELETE", "/tribes/martin/api/meal-plan/"+id, "", alice)
	wantStatus(t, rec, http.StatusOK)
	if got := strings.TrimSpace(rec.Body.String()); got != `{"versions":{"meal_plan":4}}` {
		t.Errorf("removed: %s", got)
	}

	// Removed already, in the same tribe (D19; revue de la PR #87, point 3): 404, as for
	// its servings.
	for _, r := range []struct{ method, path, body string }{
		{"DELETE", id, ""},
		{"PUT", id + "/servings", `{"servings":3}`},
	} {
		rec := f.do(r.method, "/tribes/martin/api/meal-plan/"+r.path, r.body, alice)
		if rec.Code != http.StatusNotFound || strings.TrimSpace(rec.Body.String()) != `{"error":"not_found"}` {
			t.Errorf("%s of a removed served dish: %d %s", r.method, rec.Code, rec.Body)
		}
	}

	rec = f.do("GET", "/tribes/martin/api/meal-plan", "", alice)
	wantStatus(t, rec, http.StatusOK)
	if rec.Header().Get("ETag") != `"meal_plan-4"` {
		t.Errorf("ETag %q", rec.Header().Get("ETag"))
	}
	list := decode[struct {
		Items []servedDishResponse `json:"items"`
		Next  string               `json:"next"`
	}](t, rec)
	if len(list.Items) != 1 || list.Items[0] != second || list.Next != "" {
		t.Errorf("meal plan %s", rec.Body)
	}
}

func TestMealPlanErrors(t *testing.T) {
	f := newAPIFixture(t)
	alice := f.signIn(t, "martin", "alice@exemple.fr")
	chili := f.createDish(t, "martin", "Chili con carne", alice)
	rec := f.do("POST", "/tribes/martin/api/meal-plan", serveBody("2026-10-06", "dinner", chili), alice)
	wantStatus(t, rec, http.StatusCreated)
	id := decode[servedResponse](t, rec).ServedDish.ID

	cases := []struct {
		name, method, path, body string
		status                   int
		want                     string
	}{
		{"date out of the bounds", "POST", "meal-plan", serveBody("2100-01-01", "dinner", chili), 400, `{"error":"invalid_date"}`},
		{"date unreal", "POST", "meal-plan", serveBody("2026-02-30", "dinner", chili), 400, `{"error":"invalid_date"}`},
		{"date missing", "POST", "meal-plan", `{"mealtime":"dinner","dish":"` + chili + `"}`, 400, `{"error":"invalid_date"}`},
		{"mealtime unknown", "POST", "meal-plan", serveBody("2026-10-06", "breakfast", chili), 400, `{"error":"invalid_mealtime"}`},
		{"servings 0", "POST", "meal-plan", `{"date":"2026-10-07","mealtime":"dinner","dish":"` + chili + `","servings":0}`, 400, `{"error":"invalid_servings"}`},
		{"servings 51", "POST", "meal-plan", `{"date":"2026-10-07","mealtime":"dinner","dish":"` + chili + `","servings":51}`, 400, `{"error":"invalid_servings"}`},
		{"servings as text", "POST", "meal-plan", `{"date":"2026-10-07","mealtime":"dinner","dish":"` + chili + `","servings":"4"}`, 400, `{"error":"bad_request"}`},
		{"unknown dish", "POST", "meal-plan", serveBody("2026-10-07", "dinner", strings.Repeat("0", 32)), 400, `{"error":"unknown_dish"}`},
		{"dish missing", "POST", "meal-plan", `{"date":"2026-10-07","mealtime":"dinner"}`, 400, `{"error":"unknown_dish"}`},
		{"body over 4 KiB", "POST", "meal-plan", `{"date":"2026-10-07","mealtime":"dinner","dish":"` + strings.Repeat("0", 4096) + `"}`, 400, `{"error":"bad_request"}`},
		{"servings of a served dish 0", "PUT", "meal-plan/" + id + "/servings", `{"servings":0}`, 400, `{"error":"invalid_servings"}`},
		{"servings missing", "PUT", "meal-plan/" + id + "/servings", `{}`, 400, `{"error":"invalid_servings"}`},
		{"servings of an unknown served dish", "PUT", "meal-plan/0123/servings", `{"servings":3}`, 404, `{"error":"not_found"}`},
		{"removal of an unknown served dish", "DELETE", "meal-plan/0123", "", 404, `{"error":"not_found"}`},
		{"no reading route for a served dish", "GET", "meal-plan/" + id, "", 404, `{"error":"not_found"}`},
		{"no change of a served dish but its servings", "PUT", "meal-plan/" + id, `{"servings":3}`, 405, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := f.do(c.method, "/tribes/martin/api/"+c.path, c.body, alice)
			if rec.Code != c.status || c.want != "" && strings.TrimSpace(rec.Body.String()) != c.want {
				t.Errorf("status %d, body %s; want %d, %s", rec.Code, rec.Body, c.status, c.want)
			}
		})
	}
	rec = f.do("GET", "/tribes/martin/api/meal-plan", "", alice)
	if !strings.Contains(rec.Body.String(), `"servings":4,"rank":1}]`) || rec.Header().Get("ETag") != `"meal_plan-1"` {
		t.Errorf("meal plan changed by refused writes: %v %s", rec.Header(), rec.Body)
	}
}

// TestMealPlanConditional: ETag and 304 for an unchanged meal plan (ADR 0029); a change of a
// dish does not change the meal plan.
func TestMealPlanConditional(t *testing.T) {
	f := newAPIFixture(t)
	alice := f.signIn(t, "martin", "alice@exemple.fr")
	get := func(path, header, value string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "https://localhost/tribes/martin/api/"+path, nil)
		req.AddCookie(alice)
		if header != "" {
			req.Header.Set(header, value)
		}
		rec := httptest.NewRecorder()
		f.handler.ServeHTTP(rec, req)
		return rec
	}

	rec := get("meal-plan", "", "")
	if rec.Code != 200 || rec.Header().Get("ETag") != `"meal_plan-0"` || strings.TrimSpace(rec.Body.String()) != `{"items":[]}` {
		t.Errorf("empty meal plan: %d %v %s", rec.Code, rec.Header(), rec.Body)
	}
	chili := f.createDish(t, "martin", "Chili con carne", alice)
	if rec := get("meal-plan", "If-None-Match", `"meal_plan-0"`); rec.Code != 304 {
		t.Errorf("a new dish changed the meal plan: %d", rec.Code)
	}
	wantStatus(t, f.do("POST", "/tribes/martin/api/meal-plan", serveBody("2026-10-06", "dinner", chili), alice), http.StatusCreated)
	if rec := get("meal-plan", "If-None-Match", `"meal_plan-0"`); rec.Code != 200 || rec.Header().Get("ETag") != `"meal_plan-1"` {
		t.Errorf("changed: %d %v", rec.Code, rec.Header())
	}
	wantStatus(t, f.do("PUT", "/tribes/martin/api/dishes/"+chili, `{"revision":1,"name":"Chili","servings":6}`, alice), http.StatusOK)
	if rec := get("meal-plan", "If-None-Match", `"meal_plan-1"`); rec.Code != 304 {
		t.Errorf("a change of a dish changed the meal plan: %d", rec.Code)
	}
	if rec := get("meal-plan?after=0", "If-Match", `"meal_plan-0"`); rec.Code != 412 {
		t.Errorf("changed during the loading: %d %s", rec.Code, rec.Body)
	}
}

// TestMealPlanOfAnotherTribe: neither listed, nor served, changed or removed; the answers
// are those for an unknown identifier (ENF-02).
func TestMealPlanOfAnotherTribe(t *testing.T) {
	f := newAPIFixture(t)
	david := f.signIn(t, "durand", "david@exemple.fr")
	alice := f.signIn(t, "martin", "alice@exemple.fr")
	tartiflette := f.createDish(t, "durand", "Tartiflette", david)
	rec := f.do("POST", "/tribes/durand/api/meal-plan", serveBody("2026-10-06", "dinner", tartiflette), david)
	wantStatus(t, rec, http.StatusCreated)
	id := decode[servedResponse](t, rec).ServedDish.ID

	if rec := f.do("GET", "/tribes/martin/api/meal-plan", "", alice); strings.TrimSpace(rec.Body.String()) != `{"items":[]}` {
		t.Errorf("meal plan of martin: %s", rec.Body)
	}
	if rec := f.do("GET", "/tribes/martin/api/meal-plan", "", david); rec.Code != 401 {
		t.Errorf("cookie of durand on martin: %d", rec.Code)
	}
	unknown := strings.Repeat("0", 32)
	attempts := []struct{ method, foreign, unknown, body string }{
		{"POST", "meal-plan", "meal-plan", ""},
		{"PUT", "meal-plan/" + id + "/servings", "meal-plan/" + unknown + "/servings", `{"servings":2}`},
		{"DELETE", "meal-plan/" + id, "meal-plan/" + unknown, ""},
	}
	for _, a := range attempts {
		foreignBody, unknownBody := a.body, a.body
		if a.method == "POST" {
			foreignBody, unknownBody = serveBody("2026-10-06", "dinner", tartiflette), serveBody("2026-10-06", "dinner", unknown)
		}
		foreign := f.do(a.method, "/tribes/martin/api/"+a.foreign, foreignBody, alice)
		other := f.do(a.method, "/tribes/martin/api/"+a.unknown, unknownBody, alice)
		if foreign.Code < 400 || foreign.Body.String() != other.Body.String() || fmt.Sprint(foreign.Header()) != fmt.Sprint(other.Header()) {
			t.Errorf("%s of durand: %d %v %s; unknown: %d %v %s", a.method, foreign.Code, foreign.Header(), foreign.Body, other.Code, other.Header(), other.Body)
		}
	}
	rec = f.do("GET", "/tribes/durand/api/meal-plan", "", david)
	if rec.Header().Get("ETag") != `"meal_plan-1"` || !strings.Contains(rec.Body.String(), `"servings":4`) {
		t.Errorf("meal plan of durand changed: %v %s", rec.Header(), rec.Body)
	}
}
