package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// decode reads the JSON body of a response.
func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("body %q: %v", rec.Body, err)
	}
	return v
}

func wantStatus(t *testing.T, rec *httptest.ResponseRecorder, status int) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status %d, want %d; body %s", rec.Code, status, rec.Body)
	}
}

const chiliBody = `{"name":"Chili con carne","ingredients":[` +
	`{"name":"bœuf haché","quantity":"500","unit":"g"},` +
	`{"name":"oignon","quantity":"1,5","unit":"piece"},` +
	`{"name":"sel"}]}`

func TestDishRoutesNeedASession(t *testing.T) {
	f := newAPIFixture(t)
	routes := []struct{ method, path, body string }{
		{"GET", "/api/dishes", ""},
		{"POST", "/api/dishes", chiliBody},
		{"PUT", "/api/dishes/0123", `{"revision":1,"name":"Chili","servings":4}`},
		{"GET", "/api/ingredients", ""},
		{"GET", "/api/tribe", ""},
		{"PUT", "/api/tribe/size", `{"size":5}`},
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

func TestCreateAndUpdateDish(t *testing.T) {
	f := newAPIFixture(t)
	alice := f.signIn(t, "martin", "alice@exemple.fr")

	rec := f.do("POST", "/tribes/martin/api/dishes", chiliBody, alice)
	wantStatus(t, rec, http.StatusCreated)
	created := decode[savedResponse](t, rec)
	chili := created.Dish
	want := dishResponse{ID: chili.ID, Name: "Chili con carne", Color: "#C2502E", Servings: 4, Revision: 1, Ingredients: []lineResponse{
		{Name: "bœuf haché", Quantity: "500", Unit: "g", DefaultUnit: "g"},
		{Name: "oignon", Quantity: "1.5", Unit: "piece", DefaultUnit: "piece"},
		{Name: "sel"},
	}}
	if fmt.Sprint(chili) != fmt.Sprint(want) {
		t.Errorf("dish %+v, want %+v", chili, want)
	}
	if len(chili.ID) != 32 || created.Versions["dishes"] != 1 || created.Versions["ingredients"] != 1 {
		t.Errorf("id %q, versions %v", chili.ID, created.Versions)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control %q", rec.Header().Get("Cache-Control"))
	}

	update := `{"revision":1,"name":"Chili","servings":6,"ingredients":[{"name":"Bœuf haché","quantity":"0,6","unit":"kg"},{"name":"cumin","quantity":"1","unit":"teaspoon"}]}`
	rec = f.do("PUT", "/tribes/martin/api/dishes/"+chili.ID, update, alice)
	wantStatus(t, rec, http.StatusOK)
	updated := decode[savedResponse](t, rec)
	want = dishResponse{ID: chili.ID, Name: "Chili", Color: "#C2502E", Servings: 6, Revision: 2, Ingredients: []lineResponse{
		{Name: "bœuf haché", Quantity: "0.6", Unit: "kg", DefaultUnit: "g"},
		{Name: "cumin", Quantity: "1", Unit: "teaspoon", DefaultUnit: "teaspoon"},
	}}
	if fmt.Sprint(updated.Dish) != fmt.Sprint(want) {
		t.Errorf("dish %+v, want %+v", updated.Dish, want)
	}
	if updated.Versions["dishes"] != 2 || updated.Versions["ingredients"] != 2 {
		t.Errorf("versions %v", updated.Versions)
	}

	// The same revision again: changed in between (Q25), with the current dish.
	rec = f.do("PUT", "/tribes/martin/api/dishes/"+chili.ID, update, alice)
	wantStatus(t, rec, http.StatusConflict)
	conflict := decode[dishErrorResponse](t, rec)
	if conflict.Error != "dish_changed" || conflict.Dish == nil || fmt.Sprint(*conflict.Dish) != fmt.Sprint(want) {
		t.Errorf("conflict %+v", conflict)
	}

	// The library lists it without the default units, which the catalog gives.
	rec = f.do("GET", "/tribes/martin/api/dishes", "", alice)
	wantStatus(t, rec, http.StatusOK)
	list := decode[struct {
		Items []dishResponse `json:"items"`
		Next  string         `json:"next"`
	}](t, rec)
	if len(list.Items) != 1 || list.Items[0].Ingredients[0].DefaultUnit != "" || list.Items[0].Revision != 2 || list.Next != "" {
		t.Errorf("dishes %s", rec.Body)
	}
	rec = f.do("GET", "/tribes/martin/api/ingredients", "", alice)
	wantStatus(t, rec, http.StatusOK)
	if got := strings.TrimSpace(rec.Body.String()); got != `{"items":[{"name":"bœuf haché","defaultUnit":"g"},{"name":"oignon","defaultUnit":"piece"},{"name":"sel"},{"name":"cumin","defaultUnit":"teaspoon"}]}` {
		t.Errorf("ingredients %s", got)
	}
}

func TestDishErrors(t *testing.T) {
	f := newAPIFixture(t)
	alice := f.signIn(t, "martin", "alice@exemple.fr")
	wantStatus(t, f.do("POST", "/tribes/martin/api/dishes", chiliBody, alice), http.StatusCreated)

	var lines []string
	for i := range 40 {
		lines = append(lines, fmt.Sprintf(`{"name":"%s %d","quantity":"999999,999","unit":"tablespoon"}`, strings.Repeat("é", 34), i))
	}
	forty := `{"name":"` + strings.Repeat("é", 60) + `","ingredients":[` + strings.Join(lines, ",") + `]}`
	cases := []struct {
		name, method, path, body string
		status                   int
		want                     string
	}{
		{"name required", "POST", "dishes", `{"name":"  "}`, 400, `{"error":"name_required"}`},
		{"name taken", "POST", "dishes", `{"name":" chili CON carne"}`, 400, `{"error":"name_taken"}`},
		{"name with a control character", "POST", "dishes", `{"name":"Soupe\u0007"}`, 400, `{"error":"name_invalid"}`},
		{"ingredient name invisible", "POST", "dishes", `{"name":"Soupe","ingredients":[{"name":"\u200b"}]}`, 400, `{"error":"ingredient_name_required","line":0}`},
		{"same ingredient twice", "POST", "dishes", `{"name":"Ratatouille","ingredients":[{"name":"courgette","quantity":"3","unit":"piece"},{"name":"Courgette","quantity":"200","unit":"g"}]}`, 400, `{"error":"duplicate_ingredient","line":1,"ingredient":"courgette"}`},
		{"quantity too precise", "POST", "dishes", `{"name":"Soupe","ingredients":[{"name":"lait","quantity":"2,5","unit":"ml"}]}`, 400, `{"error":"quantity_too_precise","line":0}`},
		{"quantity as a number", "POST", "dishes", `{"name":"Soupe","ingredients":[{"name":"lait","quantity":2,"unit":"ml"}]}`, 400, `{"error":"bad_request"}`},
		{"servings out of range", "POST", "dishes", `{"name":"Soupe","servings":51}`, 400, `{"error":"invalid_servings"}`},
		{"40 lines of the longest names", "POST", "dishes", forty, 201, ""},
		{"body over 16 KiB", "POST", "dishes", `{"name":"Soupe","ingredients":[` + strings.Repeat(`{"name":"sel"},`, 1200) + `{"name":"sel"}]}`, 400, `{"error":"bad_request"}`},
		{"revision missing", "PUT", "dishes/0123", `{"name":"Chili","servings":4}`, 400, `{"error":"bad_request"}`},
		{"unknown dish", "PUT", "dishes/0123", `{"revision":1,"name":"Chili","servings":4}`, 404, `{"error":"not_found"}`},
		{"no reading route for a dish", "GET", "dishes/0123", "", 404, `{"error":"not_found"}`},
		{"no deletion (P2)", "DELETE", "dishes/0123", "", 405, ""},
		{"size out of range", "PUT", "tribe/size", `{"size":0}`, 400, `{"error":"invalid_size"}`},
		{"bad cursor", "GET", "ingredients?after=x", "", 400, `{"error":"bad_request"}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := f.do(c.method, "/tribes/martin/api/"+c.path, c.body, alice)
			if rec.Code != c.status || c.want != "" && strings.TrimSpace(rec.Body.String()) != c.want {
				t.Errorf("status %d, body %s; want %d, %s", rec.Code, rec.Body, c.status, c.want)
			}
		})
	}
	if len(forty) > maxDishBodyBytes || len(forty) < maxBodyBytes {
		t.Errorf("a dish of 40 lines weighs %d bytes", len(forty))
	}
}

// TestDishesConditional: ETag on each page and 304 for an unchanged collection (ADR 0029).
func TestDishesConditional(t *testing.T) {
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

	rec := get("dishes", "", "")
	if rec.Code != 200 || rec.Header().Get("ETag") != `"dishes-0"` || strings.TrimSpace(rec.Body.String()) != `{"items":[]}` {
		t.Errorf("empty library: %d %v %s", rec.Code, rec.Header(), rec.Body)
	}
	if rec := get("dishes", "If-None-Match", `"dishes-0"`); rec.Code != 304 || rec.Body.Len() != 0 {
		t.Errorf("unchanged: %d %s", rec.Code, rec.Body)
	}
	wantStatus(t, f.do("POST", "/tribes/martin/api/dishes", chiliBody, alice), http.StatusCreated)
	if rec := get("dishes", "If-None-Match", `"dishes-0"`); rec.Code != 200 || rec.Header().Get("ETag") != `"dishes-1"` {
		t.Errorf("changed: %d %v", rec.Code, rec.Header())
	}
	if rec := get("ingredients", "If-None-Match", `"ingredients-1"`); rec.Code != 304 {
		t.Errorf("catalog unchanged: %d", rec.Code)
	}
	if rec := get("dishes?after=0", "If-Match", `"dishes-0"`); rec.Code != 412 || strings.TrimSpace(rec.Body.String()) != `{"error":"collection_changed"}` {
		t.Errorf("changed during the loading: %d %s", rec.Code, rec.Body)
	}
}

// TestDishesOfAnotherTribe: neither listed, nor changed, and a dish of another tribe answers
// as an unknown one (ENF-02).
func TestDishesOfAnotherTribe(t *testing.T) {
	f := newAPIFixture(t)
	david := f.signIn(t, "durand", "david@exemple.fr")
	alice := f.signIn(t, "martin", "alice@exemple.fr")
	rec := f.do("POST", "/tribes/durand/api/dishes", `{"name":"Tartiflette","ingredients":[{"name":"reblochon"}]}`, david)
	wantStatus(t, rec, http.StatusCreated)
	tartiflette := decode[savedResponse](t, rec).Dish

	for _, path := range []string{"dishes", "ingredients"} {
		rec := f.do("GET", "/tribes/martin/api/"+path, "", alice)
		if strings.TrimSpace(rec.Body.String()) != `{"items":[]}` {
			t.Errorf("%s of martin: %s", path, rec.Body)
		}
	}
	// David's cookie on the URL of martin: no session there.
	if rec := f.do("GET", "/tribes/martin/api/dishes", "", david); rec.Code != 401 {
		t.Errorf("cookie of durand on martin: %d", rec.Code)
	}

	update := `{"revision":1,"name":"Tartiflette","servings":2}`
	foreign := f.do("PUT", "/tribes/martin/api/dishes/"+tartiflette.ID, update, alice)
	unknown := f.do("PUT", "/tribes/martin/api/dishes/"+strings.Repeat("0", 32), update, alice)
	if foreign.Code != 404 || foreign.Body.String() != unknown.Body.String() || fmt.Sprint(foreign.Header()) != fmt.Sprint(unknown.Header()) {
		t.Errorf("dish of durand: %d %v %s; unknown dish: %d %v %s", foreign.Code, foreign.Header(), foreign.Body, unknown.Code, unknown.Header(), unknown.Body)
	}
	rec = f.do("GET", "/tribes/durand/api/dishes", "", david)
	if !strings.Contains(rec.Body.String(), `"servings":4,"revision":1`) {
		t.Errorf("dish of durand changed: %s", rec.Body)
	}
}

func TestTribeSize(t *testing.T) {
	f := newAPIFixture(t)
	alice := f.signIn(t, "martin", "alice@exemple.fr")

	rec := f.do("GET", "/tribes/martin/api/tribe", "", alice)
	wantStatus(t, rec, http.StatusOK)
	if got := strings.TrimSpace(rec.Body.String()); got != `{"name":"Les Martin","size":4,"activeMembers":1}` {
		t.Errorf("tribe %s", got)
	}
	rec = f.do("PUT", "/tribes/martin/api/tribe/size", `{"size":5}`, alice)
	wantStatus(t, rec, http.StatusOK)
	if got := strings.TrimSpace(rec.Body.String()); got != `{"name":"Les Martin","size":5,"activeMembers":1}` {
		t.Errorf("tribe %s", got)
	}
	// A new dish is proposed for the tribe size (T3).
	rec = f.do("POST", "/tribes/martin/api/dishes", `{"name":"Curry de lentilles"}`, alice)
	wantStatus(t, rec, http.StatusCreated)
	if d := decode[savedResponse](t, rec).Dish; d.Servings != 5 {
		t.Errorf("servings %d, want 5", d.Servings)
	}
}
