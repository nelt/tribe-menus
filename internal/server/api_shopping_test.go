package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// lists returns the shopping lists of the tribe, as GET shopping-lists gives them.
func (f *apiFixture) lists(t *testing.T, slug string, cookie *http.Cookie) []listResponse {
	t.Helper()
	rec := f.do("GET", "/tribes/"+slug+"/api/shopping-lists", "", cookie)
	wantStatus(t, rec, http.StatusOK)
	return decode[struct {
		Items []listResponse `json:"items"`
	}](t, rec).Items
}

// mainList returns the main list of the tribe.
func (f *apiFixture) mainList(t *testing.T, slug string, cookie *http.Cookie) listResponse {
	t.Helper()
	for _, l := range f.lists(t, slug, cookie) {
		if l.Main {
			return l
		}
	}
	t.Fatal("no main list")
	return listResponse{}
}

func TestShoppingRoutesNeedASession(t *testing.T) {
	f := newAPIFixture(t)
	routes := []struct{ method, path, body string }{
		{"GET", "/api/shopping-lists", ""},
		{"POST", "/api/shopping-lists", `{"name":"Marché"}`},
		{"PUT", "/api/shopping-lists/main", `{"list":"0123"}`},
		{"PUT", "/api/shopping-lists/0123/name", `{"name":"Marché"}`},
		{"DELETE", "/api/shopping-lists/0123", ""},
		{"POST", "/api/shopping-lists/0123/items", `{"ingredient":"pain"}`},
		{"DELETE", "/api/shopping-lists/0123/items/0456", ""},
		{"PUT", "/api/shopping-lists/0123/items/0456/checked", `{"checked":true}`},
		{"POST", "/api/shopping-lists/0123/checkout", `{"items":[]}`},
		{"POST", "/api/shopping-lists/0123/meal-shopping", `{"from":"2026-10-06","to":"2026-10-12"}`},
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

func TestShoppingLists(t *testing.T) {
	f := newAPIFixture(t)
	alice := f.signIn(t, "martin", "alice@exemple.fr")

	// A new tribe: "Courses", main and empty (C1, D14).
	rec := f.do("GET", "/tribes/martin/api/shopping-lists", "", alice)
	wantStatus(t, rec, http.StatusOK)
	if rec.Header().Get("ETag") != `"shopping_lists-0"` || rec.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("headers %v", rec.Header())
	}
	courses := f.mainList(t, "martin", alice)
	want := fmt.Sprintf(`{"items":[{"id":%q,"name":"Courses","main":true,"rank":1,"items":[],"received":[]}]}`, courses.ID)
	if got := strings.TrimSpace(rec.Body.String()); got != want {
		t.Errorf("lists %s, want %s", got, want)
	}

	rec = f.do("POST", "/tribes/martin/api/shopping-lists", `{"name":" Marché  du samedi "}`, alice)
	wantStatus(t, rec, http.StatusCreated)
	created := decode[listWrittenResponse](t, rec)
	marche := created.List
	if marche.Name != "Marché du samedi" || marche.Main || marche.Rank != 2 || len(marche.ID) != 32 ||
		fmt.Sprint(created.Versions) != "map[shopping_lists:1]" {
		t.Errorf("created %s", rec.Body)
	}
	rec = f.do("PUT", "/tribes/martin/api/shopping-lists/"+marche.ID+"/name", `{"name":"Marché"}`, alice)
	wantStatus(t, rec, http.StatusOK)
	if l := decode[listWrittenResponse](t, rec).List; l.Name != "Marché" || l.ID != marche.ID {
		t.Errorf("renamed %s", rec.Body)
	}

	// Items: by hand, with or without quantity (C3).
	rec = f.do("POST", "/tribes/martin/api/shopping-lists/"+courses.ID+"/items", `{"ingredient":"pain","quantity":"2","unit":"piece"}`, alice)
	wantStatus(t, rec, http.StatusCreated)
	written := decode[listWrittenResponse](t, rec)
	if fmt.Sprint(written.Versions) != "map[ingredients:1 shopping_lists:3]" || len(written.List.Items) != 1 {
		t.Errorf("item added %s", rec.Body)
	}
	pain := written.List.Items[0].ID
	if got, _ := json.Marshal(written.List.Items[0]); string(got) != fmt.Sprintf(`{"id":%q,"ingredient":"pain","quantity":"2","unit":"piece","checked":false,"manual":true,"sources":[]}`, pain) {
		t.Errorf("item %s", got)
	}
	rec = f.do("POST", "/tribes/martin/api/shopping-lists/"+courses.ID+"/items", `{"ingredient":"lessive"}`, alice)
	wantStatus(t, rec, http.StatusCreated)
	lessive := decode[listWrittenResponse](t, rec).List.Items[0]
	if lessive.Ingredient != "lessive" || lessive.Quantity != "" || lessive.Unit != "" {
		t.Errorf("item without quantity %+v", lessive)
	}
	rec = f.do("GET", "/tribes/martin/api/ingredients", "", alice)
	if !strings.Contains(rec.Body.String(), `{"name":"lessive"}`) || !strings.Contains(rec.Body.String(), `{"name":"pain","defaultUnit":"piece"}`) {
		t.Errorf("catalog %s", rec.Body)
	}

	// Checked: at the bottom (Q40).
	rec = f.do("PUT", "/tribes/martin/api/shopping-lists/"+courses.ID+"/items/"+pain+"/checked", `{"checked":true}`, alice)
	wantStatus(t, rec, http.StatusOK)
	l := decode[listWrittenResponse](t, rec).List
	if len(l.Items) != 2 || l.Items[0].Ingredient != "lessive" || !l.Items[1].Checked {
		t.Errorf("checked %s", rec.Body)
	}

	// Main list (C1): the versions only.
	rec = f.do("PUT", "/tribes/martin/api/shopping-lists/main", fmt.Sprintf(`{"list":%q}`, marche.ID), alice)
	wantStatus(t, rec, http.StatusOK)
	if got := strings.TrimSpace(rec.Body.String()); got != `{"versions":{"shopping_lists":6}}` {
		t.Errorf("set main %s", got)
	}
	if f.mainList(t, "martin", alice).ID != marche.ID {
		t.Error("Marché is not the main list")
	}

	// The shopping done (C11, D9): pain seen checked leaves, lessive unchecked stays.
	rec = f.do("POST", "/tribes/martin/api/shopping-lists/"+courses.ID+"/checkout", fmt.Sprintf(`{"items":[%q,%q]}`, pain, lessive.ID), alice)
	wantStatus(t, rec, http.StatusOK)
	done := decode[checkedOutResponse](t, rec)
	if done.Removed != 1 || len(done.List.Items) != 1 || done.List.Items[0].ID != lessive.ID || done.Versions["shopping_lists"] != 7 {
		t.Errorf("checked out %s", rec.Body)
	}

	// Removal of an item added by hand, then again (D11).
	path := "/tribes/martin/api/shopping-lists/" + courses.ID + "/items/" + lessive.ID
	rec = f.do("DELETE", path, "", alice)
	wantStatus(t, rec, http.StatusOK)
	if l := decode[listWrittenResponse](t, rec).List; len(l.Items) != 0 {
		t.Errorf("removed %s", rec.Body)
	}
	if rec := f.do("DELETE", path, "", alice); rec.Code != 404 || strings.TrimSpace(rec.Body.String()) != `{"error":"not_found"}` {
		t.Errorf("removed again: %d %s", rec.Code, rec.Body)
	}

	// Deletion of an empty list (C12), but the main one.
	if rec := f.do("DELETE", "/tribes/martin/api/shopping-lists/"+marche.ID, "", alice); rec.Code != 409 || strings.TrimSpace(rec.Body.String()) != `{"error":"main_list"}` {
		t.Errorf("delete the main list: %d %s", rec.Code, rec.Body)
	}
	rec = f.do("DELETE", "/tribes/martin/api/shopping-lists/"+courses.ID, "", alice)
	wantStatus(t, rec, http.StatusOK)
	if got := strings.TrimSpace(rec.Body.String()); got != `{"versions":{"shopping_lists":9}}` {
		t.Errorf("deleted %s", got)
	}
	if lists := f.lists(t, "martin", alice); len(lists) != 1 || lists[0].ID != marche.ID {
		t.Errorf("lists %+v", lists)
	}
}

func TestShoppingErrors(t *testing.T) {
	f := newAPIFixture(t)
	alice := f.signIn(t, "martin", "alice@exemple.fr")
	courses := f.mainList(t, "martin", alice).ID
	rec := f.do("POST", "/tribes/martin/api/shopping-lists", `{"name":"Marché"}`, alice)
	wantStatus(t, rec, http.StatusCreated)
	marche := decode[listWrittenResponse](t, rec).List.ID
	rec = f.do("POST", "/tribes/martin/api/shopping-lists/"+marche+"/items", `{"ingredient":"pain"}`, alice)
	wantStatus(t, rec, http.StatusCreated)
	pain := decode[listWrittenResponse](t, rec).List.Items[0].ID
	unknown := strings.Repeat("0", 32)
	lists := "shopping-lists/"
	items := lists + courses + "/items"

	cases := []struct {
		name, method, path, body string
		status                   int
		want                     string
	}{
		{"list without name", "POST", "shopping-lists", `{"name":" "}`, 400, `{"error":"list_name_empty"}`},
		{"list name missing", "POST", "shopping-lists", `{}`, 400, `{"error":"list_name_empty"}`},
		{"list name with a control character", "POST", "shopping-lists", `{"name":"Marché\u0007"}`, 400, `{"error":"list_name_invalid"}`},
		{"list name of 41 characters", "POST", "shopping-lists", `{"name":"` + strings.Repeat("é", 41) + `"}`, 400, `{"error":"list_name_too_long"}`},
		{"rename without name", "PUT", lists + marche + "/name", `{"name":"\u200b"}`, 400, `{"error":"list_name_empty"}`},
		{"rename an unknown list", "PUT", lists + unknown + "/name", `{"name":"Drive"}`, 404, `{"error":"not_found"}`},
		{"main list unknown", "PUT", lists + "main", `{"list":"` + unknown + `"}`, 404, `{"error":"not_found"}`},
		{"main list missing", "PUT", lists + "main", `{}`, 404, `{"error":"not_found"}`},
		{"delete a list with items", "DELETE", lists + marche, "", 409, ""},
		{"delete an unknown list", "DELETE", lists + unknown, "", 404, `{"error":"not_found"}`},
		{"item without ingredient", "POST", items, `{"ingredient":""}`, 400, `{"error":"ingredient_name_required"}`},
		{"item with a control character", "POST", items, `{"ingredient":"sel\u0007"}`, 400, `{"error":"ingredient_name_invalid"}`},
		{"item name of 41 characters", "POST", items, `{"ingredient":"` + strings.Repeat("a", 41) + `"}`, 400, `{"error":"ingredient_name_too_long"}`},
		{"quantity without unit", "POST", items, `{"ingredient":"riz","quantity":"1"}`, 400, `{"error":"unit_required"}`},
		{"unknown unit", "POST", items, `{"ingredient":"riz","quantity":"1","unit":"cup"}`, 400, `{"error":"invalid_unit"}`},
		{"quantity 0", "POST", items, `{"ingredient":"riz","quantity":"0","unit":"g"}`, 400, `{"error":"invalid_quantity"}`},
		{"quantity as a number", "POST", items, `{"ingredient":"riz","quantity":1,"unit":"g"}`, 400, `{"error":"bad_request"}`},
		{"quantity too precise", "POST", items, `{"ingredient":"riz","quantity":"1.5","unit":"ml"}`, 400, `{"error":"quantity_too_precise"}`},
		{"item of an unknown list", "POST", lists + unknown + "/items", `{"ingredient":"pain"}`, 404, `{"error":"not_found"}`},
		{"body over 4 KiB", "POST", items, `{"ingredient":"` + strings.Repeat("a", 4096) + `"}`, 400, `{"error":"bad_request"}`},
		{"check an item of another list", "PUT", items + "/" + pain + "/checked", `{"checked":true}`, 404, `{"error":"not_found"}`},
		{"check an unknown item", "PUT", items + "/" + unknown + "/checked", `{"checked":true}`, 404, `{"error":"not_found"}`},
		{"check without value", "PUT", lists + marche + "/items/" + pain + "/checked", `{}`, 400, `{"error":"bad_request"}`},
		{"remove an item of another list", "DELETE", items + "/" + pain, "", 404, `{"error":"not_found"}`},
		{"check out an unknown list", "POST", lists + unknown + "/checkout", `{"items":[]}`, 404, `{"error":"not_found"}`},
		{"meal shopping of an unknown list", "POST", lists + unknown + "/meal-shopping", `{"from":"2026-10-06","to":"2026-10-12"}`, 404, `{"error":"not_found"}`},
		{"period ending before its start", "POST", lists + courses + "/meal-shopping", `{"from":"2026-10-08","to":"2026-10-06"}`, 400, `{"error":"invalid_period"}`},
		{"period of 32 days", "POST", lists + courses + "/meal-shopping", `{"from":"2026-10-06","to":"2026-11-06"}`, 400, `{"error":"period_too_long"}`},
		{"period before 2000", "POST", lists + courses + "/meal-shopping", `{"from":"1999-12-31","to":"2000-01-06"}`, 400, `{"error":"invalid_date"}`},
		{"period after 2099", "POST", lists + courses + "/meal-shopping", `{"from":"2099-12-28","to":"2100-01-01"}`, 400, `{"error":"invalid_date"}`},
		{"period without end", "POST", lists + courses + "/meal-shopping", `{"from":"2026-10-06"}`, 400, `{"error":"invalid_date"}`},
		{"period unreal", "POST", lists + courses + "/meal-shopping", `{"from":"2026-02-30","to":"2026-03-01"}`, 400, `{"error":"invalid_date"}`},
		{"no reading route for a list", "GET", lists + marche, "", 404, `{"error":"not_found"}`},
		{"no change of a list but its name", "PUT", lists + marche, `{"name":"Drive"}`, 405, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := f.do(c.method, "/tribes/martin/api/"+c.path, c.body, alice)
			if rec.Code != c.status || c.want != "" && strings.TrimSpace(rec.Body.String()) != c.want {
				t.Errorf("status %d, body %s; want %d, %s", rec.Code, rec.Body, c.status, c.want)
			}
		})
	}

	// A list with items: refused with the list up to date (D11).
	rec = f.do("DELETE", "/tribes/martin/api/shopping-lists/"+marche, "", alice)
	notEmpty := decode[listNotEmptyResponse](t, rec)
	if notEmpty.Error != "list_not_empty" || notEmpty.List.ID != marche || len(notEmpty.List.Items) != 1 {
		t.Errorf("not empty %s", rec.Body)
	}
	rec = f.do("GET", "/tribes/martin/api/shopping-lists", "", alice)
	if rec.Header().Get("ETag") != `"shopping_lists-2"` {
		t.Errorf("lists changed by refused writes: %v %s", rec.Header(), rec.Body)
	}
}

// TestMealShopping: the shopping of the meals of a period (C4, C5), its quantities in the
// smallest unit of their family, the served dishes added and ignored (Q22).
func TestMealShopping(t *testing.T) {
	f := newAPIFixture(t)
	alice := f.signIn(t, "martin", "alice@exemple.fr")
	courses := f.mainList(t, "martin", alice).ID
	rec := f.do("POST", "/tribes/martin/api/dishes", `{"name":"Chili con carne","servings":4,"ingredients":[`+
		`{"name":"bœuf haché","quantity":"0.5","unit":"kg"},{"name":"oignon","quantity":"1","unit":"piece"},{"name":"sel"}]}`, alice)
	wantStatus(t, rec, http.StatusCreated)
	chili := decode[savedResponse](t, rec).Dish.ID
	var served []string
	for _, b := range []string{
		`{"date":"2026-10-06","mealtime":"dinner","dish":"` + chili + `","servings":3}`,
		`{"date":"2026-10-08","mealtime":"lunch","dish":"` + chili + `","servings":4}`,
		`{"date":"2026-10-13","mealtime":"dinner","dish":"` + chili + `","servings":4}`,
	} {
		rec := f.do("POST", "/tribes/martin/api/meal-plan", b, alice)
		wantStatus(t, rec, http.StatusCreated)
		served = append(served, decode[struct {
			ServedDish struct {
				ID string `json:"id"`
			} `json:"servedDish"`
		}](t, rec).ServedDish.ID)
	}

	rec = f.do("POST", "/tribes/martin/api/shopping-lists/"+courses+"/meal-shopping", `{"from":"2026-10-06","to":"2026-10-12"}`, alice)
	wantStatus(t, rec, http.StatusOK)
	got := decode[mealShoppingResponse](t, rec)
	if got.Added != 2 || got.Ignored != 0 || got.Versions["shopping_lists"] != 1 || len(got.List.Received) != 2 {
		t.Errorf("response %s", rec.Body)
	}
	var lines []string
	for _, it := range got.List.Items {
		line := fmt.Sprintf("%s %s %s manual=%v", it.Ingredient, it.Quantity, it.Unit, it.Manual)
		for _, s := range it.Sources {
			line += fmt.Sprintf(" | %s %s %d %s %s", s.Date, s.Mealtime, s.Servings, s.Quantity, s.Unit)
		}
		lines = append(lines, line)
	}
	want := []string{
		"bœuf haché 875 g manual=false | 2026-10-06 dinner 3 375 g | 2026-10-08 lunch 4 500 g",
		"oignon 2 piece manual=false | 2026-10-06 dinner 3 0.75 piece | 2026-10-08 lunch 4 1 piece",
		"sel   manual=false | 2026-10-06 dinner 3   | 2026-10-08 lunch 4  ",
	}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Errorf("items\n%s\nwant\n%s", strings.Join(lines, "\n"), strings.Join(want, "\n"))
	}
	if s := got.List.Items[0].Sources[0]; s.ServedDish != served[0] || s.Dish != chili {
		t.Errorf("source %+v", s)
	}
	if lists := f.lists(t, "martin", alice); fmt.Sprint(lists[0]) != fmt.Sprint(got.List) {
		t.Errorf("read back %+v\nreturned %+v", lists[0], got.List)
	}

	// Again, with the next week: the two served dishes received are ignored.
	rec = f.do("POST", "/tribes/martin/api/shopping-lists/"+courses+"/meal-shopping", `{"from":"2026-10-06","to":"2026-10-13"}`, alice)
	wantStatus(t, rec, http.StatusOK)
	if got := decode[mealShoppingResponse](t, rec); got.Added != 1 || got.Ignored != 2 || got.List.Items[0].Quantity != "1375" {
		t.Errorf("again %s", rec.Body)
	}
}

// TestShoppingLimits: 20 lists, 300 items (D12); a checkout of 300 identifiers fits in its
// body (revue #92, point 1).
func TestShoppingLimits(t *testing.T) {
	f := newAPIFixture(t)
	alice := f.signIn(t, "martin", "alice@exemple.fr")
	courses := f.mainList(t, "martin", alice).ID
	for i := 2; i <= 20; i++ {
		wantStatus(t, f.do("POST", "/tribes/martin/api/shopping-lists", fmt.Sprintf(`{"name":"Liste %d"}`, i), alice), http.StatusCreated)
	}
	if rec := f.do("POST", "/tribes/martin/api/shopping-lists", `{"name":"Marché"}`, alice); rec.Code != 409 || strings.TrimSpace(rec.Body.String()) != `{"error":"too_many_lists"}` {
		t.Errorf("21st list: %d %s", rec.Code, rec.Body)
	}
	if lists := f.lists(t, "martin", alice); len(lists) != 20 {
		t.Errorf("%d lists on the first page (D4)", len(lists))
	}

	// 299 items checked, written in the database; the 300th by the API.
	db, err := f.store.Tribe(context.Background(), "martin")
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for i := range 299 {
		ids = append(ids, fmt.Sprintf("%032x", i+1))
		for _, s := range []string{
			fmt.Sprintf("INSERT INTO ingredients (id, name, normalized_name) VALUES (%d, 'article %d', 'article %d')", i+1, i, i),
			fmt.Sprintf("INSERT INTO shopping_items VALUES ('%s', '%s', %d, NULL, NULL, 1, 1)", ids[i], courses, i+1),
		} {
			if _, err := db.Exec(s); err != nil {
				t.Fatalf("%s: %v", s, err)
			}
		}
	}
	rec := f.do("POST", "/tribes/martin/api/shopping-lists/"+courses+"/items", `{"ingredient":"article 299"}`, alice)
	wantStatus(t, rec, http.StatusCreated)
	for _, it := range decode[listWrittenResponse](t, rec).List.Items {
		if !it.Checked {
			ids = append(ids, it.ID)
			wantStatus(t, f.do("PUT", "/tribes/martin/api/shopping-lists/"+courses+"/items/"+it.ID+"/checked", `{"checked":true}`, alice), http.StatusOK)
		}
	}
	if rec := f.do("POST", "/tribes/martin/api/shopping-lists/"+courses+"/items", `{"ingredient":"pain"}`, alice); rec.Code != 409 || strings.TrimSpace(rec.Body.String()) != `{"error":"too_many_items"}` {
		t.Errorf("301st item: %d %s", rec.Code, rec.Body)
	}
	body, err := json.Marshal(map[string][]string{"items": ids})
	if err != nil {
		t.Fatal(err)
	}
	if len(body) <= maxBodyBytes {
		t.Fatalf("checkout of %d bytes, within the 4 KiB", len(body))
	}
	rec = f.do("POST", "/tribes/martin/api/shopping-lists/"+courses+"/checkout", string(body), alice)
	wantStatus(t, rec, http.StatusOK)
	if done := decode[checkedOutResponse](t, rec); done.Removed != 300 || len(done.List.Items) != 0 {
		t.Errorf("checked out %d, %d left", done.Removed, len(done.List.Items))
	}
	over := `{"items":["` + strings.Repeat("0", maxCheckoutBodyBytes) + `"]}`
	// 299 items, then a dish of two ingredients served: refused in full (D12).
	for _, id := range ids[:299] {
		if _, err := db.Exec("INSERT INTO shopping_items VALUES (?, ?, 1, NULL, NULL, 0, 1)", id, courses); err != nil {
			t.Fatal(err)
		}
	}
	chili := f.createDish(t, "martin", "Chili con carne", alice)
	wantStatus(t, f.do("PUT", "/tribes/martin/api/dishes/"+chili, `{"revision":1,"name":"Chili con carne","servings":4,"ingredients":[{"name":"oignon"},{"name":"sel"}]}`, alice), http.StatusOK)
	wantStatus(t, f.do("POST", "/tribes/martin/api/meal-plan", serveBody("2026-10-06", "dinner", chili), alice), http.StatusCreated)
	before := f.lists(t, "martin", alice)
	if rec := f.do("POST", "/tribes/martin/api/shopping-lists/"+courses+"/meal-shopping", `{"from":"2026-10-06","to":"2026-10-06"}`, alice); rec.Code != 409 || strings.TrimSpace(rec.Body.String()) != `{"error":"too_many_items"}` {
		t.Errorf("meal shopping to 301 items: %d %s", rec.Code, rec.Body)
	}
	if after := f.lists(t, "martin", alice); fmt.Sprint(after) != fmt.Sprint(before) {
		t.Errorf("lists changed by a refused meal shopping")
	}
	if rec := f.do("POST", "/tribes/martin/api/shopping-lists/"+courses+"/checkout", over, alice); rec.Code != 400 {
		t.Errorf("checkout over 16 KiB: %d", rec.Code)
	}
}

// TestShoppingConditional: ETag and 304 for unchanged lists (ADR 0029); a new dish does not
// change them.
func TestShoppingConditional(t *testing.T) {
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
	f.createDish(t, "martin", "Chili con carne", alice)
	if rec := get("shopping-lists", "If-None-Match", `"shopping_lists-0"`); rec.Code != 304 {
		t.Errorf("a new dish changed the lists: %d", rec.Code)
	}
	wantStatus(t, f.do("POST", "/tribes/martin/api/shopping-lists", `{"name":"Marché"}`, alice), http.StatusCreated)
	if rec := get("shopping-lists", "If-None-Match", `"shopping_lists-0"`); rec.Code != 200 || rec.Header().Get("ETag") != `"shopping_lists-1"` {
		t.Errorf("changed: %d %v", rec.Code, rec.Header())
	}
	if rec := get("shopping-lists?after=0", "If-Match", `"shopping_lists-0"`); rec.Code != 412 {
		t.Errorf("changed during the loading: %d %s", rec.Code, rec.Body)
	}
}

// TestShoppingListsOfAnotherTribe: neither listed, nor changed or deleted; the answers are
// those for an unknown identifier (ENF-02).
func TestShoppingListsOfAnotherTribe(t *testing.T) {
	f := newAPIFixture(t)
	david := f.signIn(t, "durand", "david@exemple.fr")
	alice := f.signIn(t, "martin", "alice@exemple.fr")
	rec := f.do("POST", "/tribes/durand/api/shopping-lists", `{"name":"Marché"}`, david)
	wantStatus(t, rec, http.StatusCreated)
	marche := decode[listWrittenResponse](t, rec).List.ID
	courses := f.mainList(t, "durand", david).ID
	rec = f.do("POST", "/tribes/durand/api/shopping-lists/"+courses+"/items", `{"ingredient":"reblochon"}`, david)
	wantStatus(t, rec, http.StatusCreated)
	item := decode[listWrittenResponse](t, rec).List.Items[0].ID

	if lists := f.lists(t, "martin", alice); len(lists) != 1 || lists[0].Name != "Courses" || len(lists[0].Items) != 0 {
		t.Errorf("lists of martin %+v", lists)
	}
	if rec := f.do("GET", "/tribes/martin/api/shopping-lists", "", david); rec.Code != 401 {
		t.Errorf("cookie of durand on martin: %d", rec.Code)
	}
	unknown := strings.Repeat("0", 32)
	attempts := []struct{ method, path, body string }{
		{"PUT", "shopping-lists/%s/name", `{"name":"Drive"}`},
		{"PUT", "shopping-lists/main", `{"list":"%s"}`},
		{"DELETE", "shopping-lists/%s", ""},
		{"POST", "shopping-lists/%s/items", `{"ingredient":"pain"}`},
		{"POST", "shopping-lists/%s/checkout", `{"items":[]}`},
		{"POST", "shopping-lists/%s/meal-shopping", `{"from":"2026-10-06","to":"2026-10-12"}`},
		{"PUT", "shopping-lists/" + courses + "/items/%s/checked", `{"checked":true}`},
		{"DELETE", "shopping-lists/" + courses + "/items/%s", ""},
	}
	for _, a := range attempts {
		target := marche
		if strings.Contains(a.path, "/items/") {
			target = item
		}
		do := func(id string) *httptest.ResponseRecorder {
			path, body := strings.Replace(a.path, "%s", id, 1), strings.Replace(a.body, "%s", id, 1)
			return f.do(a.method, "/tribes/martin/api/"+path, body, alice)
		}
		foreign, other := do(target), do(unknown)
		if foreign.Code != 404 || foreign.Body.String() != other.Body.String() || fmt.Sprint(foreign.Header()) != fmt.Sprint(other.Header()) {
			t.Errorf("%s %s of durand: %d %v %s; unknown: %d %v %s", a.method, a.path, foreign.Code, foreign.Header(), foreign.Body, other.Code, other.Header(), other.Body)
		}
	}
	rec = f.do("GET", "/tribes/durand/api/shopping-lists", "", david)
	if rec.Header().Get("ETag") != `"shopping_lists-2"` || !strings.Contains(rec.Body.String(), `"ingredient":"reblochon","checked":false`) {
		t.Errorf("lists of durand changed: %v %s", rec.Header(), rec.Body)
	}
}
