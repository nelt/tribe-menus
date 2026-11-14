package admin

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nelt/tribe-menus/internal/dish"
	"github.com/nelt/tribe-menus/internal/mealplan"
	"github.com/nelt/tribe-menus/internal/shopping"
	"github.com/nelt/tribe-menus/internal/storage"
	"github.com/nelt/tribe-menus/internal/tribe"
)

func TestSeed(t *testing.T) {
	ctx := context.Background()
	store := openStore(t, t.TempDir())
	var out bytes.Buffer
	cmd := &Command{Store: store, Out: &out, BaseURL: "http://localhost:8080", Now: func() time.Time { return now }}

	for _, want := range []string{"créée : http://localhost:8080/tribes/demo/", "existe déjà"} {
		out.Reset()
		if err := cmd.Seed(ctx, nil); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), want) {
			t.Errorf("output = %q, want it to contain %q", out.String(), want)
		}
	}

	db, err := store.Tribe(ctx, "demo")
	if err != nil {
		t.Fatal(err)
	}
	s := tribe.NewStore(db)
	if name, err := s.Name(ctx); err != nil || name != "Les Démo" {
		t.Errorf("Name = %q, %v", name, err)
	}
	for _, m := range demoMembers {
		got, err := s.Member(ctx, m.Email)
		if err != nil || got.Status != tribe.Active || got.DisplayName != m.DisplayName {
			t.Errorf("Member(%s) = %+v, %v", m.Email, got, err)
		}
	}
	if entries, err := s.AuditLog(ctx); err != nil || len(entries) != len(demoMembers) {
		t.Errorf("AuditLog = %d entries, %v; want %d", len(entries), err, len(demoMembers))
	}

	// The eight dishes of the mock-up Doux-Bibliotheque (plan bibliotheque, D21), once.
	_, page, err := dish.NewLibrary(storage.NewDishStore(db)).Dishes(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != len(demoDishes) || len(demoDishes) != 8 {
		t.Fatalf("%d dishes, want the %d of the mock-up", len(page.Items), len(demoDishes))
	}
	colors := map[dish.Color]string{}
	for _, d := range page.Items {
		colors[d.Color] = d.Name
	}
	for i, d := range demoDishes {
		if colors[dish.Palette[i]] != d.Name {
			t.Errorf("%s: want the hue %s of the mock-up, held by %q", d.Name, dish.Palette[i], colors[dish.Palette[i]])
		}
	}

	// The week of the mock-up Doux-Main, from the day of the seed (plan planning, D16).
	names := map[dish.ID]string{}
	for _, d := range page.Items {
		names[d.ID] = d.Name
	}
	_, served, err := mealplan.NewPlan(storage.NewMealPlanStore(db)).ServedDishes(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	slices.SortFunc(served.Items, func(a, b mealplan.ServedDish) int { return int(a.Rank - b.Rank) })
	var got []string
	for _, s := range served.Items {
		got = append(got, fmt.Sprintf("%s %s %s %d", s.Date, s.Mealtime, names[s.Dish], s.Servings))
	}
	day := func(n int) string { return now.AddDate(0, 0, n).Format(time.DateOnly) }
	want := []string{
		day(0) + " lunch Pizza maison 4",
		day(0) + " dinner Ratatouille 4",
		day(1) + " lunch Pâtes pesto 4",
		day(1) + " dinner Chili con carne 5", // 3, then 5 after the shopping (plan courses-suite, D20)
		day(1) + " dinner Chili végétarien 1",
		day(2) + " dinner Soupe à l'oignon 6", // 4, then 6
		day(3) + " lunch Chili con carne 4",
		day(3) + " dinner Gratin de courgettes 4",
	}
	if !slices.Equal(got, want) {
		t.Errorf("week %q, want %q", got, want)
	}

	// The shopping of the week in « Courses », « pain » and « lessive » added by hand, a list
	// « Marché » (plan courses, D20).
	_, lists, err := shopping.NewLists(storage.NewShoppingStore(db)).Page(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	slices.SortFunc(lists.Items, func(a, b shopping.List) int { return int(a.Rank - b.Rank) })
	if len(lists.Items) != 2 || !lists.Items[0].Main || lists.Items[0].Name != "Courses" || lists.Items[1].Name != "Marché" {
		t.Fatalf("lists %+v", lists.Items)
	}
	courses, market := lists.Items[0], lists.Items[1]
	if len(courses.Received) != len(demoWeek) {
		t.Errorf("%d served dishes received, want the %d of the week", len(courses.Received), len(demoWeek))
	}
	items := map[string]string{}
	for _, it := range courses.Items {
		q := "-"
		if it.Quantity != nil {
			q = it.Quantity.Text() + " " + string(it.Quantity.Unit)
		}
		if it.Taken != nil {
			q += " taken " + it.Taken.Text()
		}
		if it.Manual {
			q += " manual"
		}
		items[it.Ingredient] += q + ";"
	}
	// Chili con carne for 3 then 4 servings of 4: 875 g of beef, checked, then the Tuesday
	// chili for 5 servings and the list recomputed: 1125 g, 875 g taken (plan courses-suite,
	// D20); onions 2.5 and 2 of it, 0.5 of the vegetarian chili, 1 of the ratatouille, 6 of
	// the soup, whose 6 servings since leave the list stale.
	for ingredient, want := range map[string]string{"bœuf haché": "1125 g taken 875;", "oignon": "12 piece;", "pain": "4 piece;- manual;", "lessive": "- manual;"} {
		if items[ingredient] != want {
			t.Errorf("%s: %q, want %q; items %v", ingredient, items[ingredient], want, items)
		}
	}
	if len(market.Items) != 1 || market.Items[0].Ingredient != "fraises" {
		t.Errorf("Marché %+v", market)
	}
	if stale, err := shopping.NewLists(storage.NewShoppingStore(db)).Stale(ctx); err != nil || !slices.Equal(stale, []shopping.ListID{courses.ID}) {
		t.Errorf("stale lists %v, %v; want « Courses »", stale, err)
	}
}

func TestSeedWithMembers(t *testing.T) {
	ctx := context.Background()
	store := openStore(t, t.TempDir())
	cmd := &Command{Store: store, Out: &bytes.Buffer{}, BaseURL: "https://recette.example.org", Now: func() time.Time { return now }}
	members := []SeedMember{
		{Email: "recette-alice@example.org", DisplayName: "Alice"},
		{Email: "recette-bruno@example.org"},
	}
	if err := cmd.Seed(ctx, members); err != nil {
		t.Fatal(err)
	}
	db, err := store.Tribe(ctx, "demo")
	if err != nil {
		t.Fatal(err)
	}
	s := tribe.NewStore(db)
	for _, m := range members {
		got, err := s.Member(ctx, m.Email)
		if err != nil || got.Status != tribe.Active || got.DisplayName != m.DisplayName {
			t.Errorf("Member(%s) = %+v, %v", m.Email, got, err)
		}
	}
	if _, err := s.Member(ctx, demoMembers[0].Email); err == nil {
		t.Errorf("%s is a member, want only the members of the file", demoMembers[0].Email)
	}
	if _, page, err := dish.NewLibrary(storage.NewDishStore(db)).Dishes(ctx, ""); err != nil || len(page.Items) != 0 {
		t.Errorf("dishes %+v, %v; want none in the tribe of a members file", page.Items, err)
	}
	if _, page, err := mealplan.NewPlan(storage.NewMealPlanStore(db)).ServedDishes(ctx, ""); err != nil || len(page.Items) != 0 {
		t.Errorf("meal plan %+v, %v; want none in the tribe of a members file", page.Items, err)
	}
	if _, page, err := shopping.NewLists(storage.NewShoppingStore(db)).Page(ctx, ""); err != nil || len(page.Items) != 1 || len(page.Items[0].Items) != 0 {
		t.Errorf("lists %+v, %v; want « Courses » alone and empty in the tribe of a members file", page.Items, err)
	}
	entries, err := s.AuditLog(ctx)
	if err != nil || len(entries) != 2 || entries[0].Operation != tribe.TribeInitialized || entries[1].Operation != tribe.MemberAdded {
		t.Errorf("AuditLog = %+v, %v", entries, err)
	}
}

func TestParseMembers(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    []SeedMember
		wantErr string
	}{
		{
			name:  "address then display name",
			input: "recette-alice@example.org Alice\nRecette-Bruno@Example.org   Bruno Martin  \n",
			want:  []SeedMember{{"recette-alice@example.org", "Alice"}, {"recette-bruno@example.org", "Bruno Martin"}},
		},
		{
			name:  "display name optional, tabs, blank lines and comments",
			input: "# Tribu de recette\n\nrecette-alice@example.org\n\trecette-chloe@example.org\tChloé\n",
			want:  []SeedMember{{"recette-alice@example.org", ""}, {"recette-chloe@example.org", "Chloé"}},
		},
		{name: "no line ending", input: "a@example.org A", want: []SeedMember{{"a@example.org", "A"}}},
		{name: "empty", input: "", wantErr: "no member"},
		{name: "comments only", input: "# rien\n\n", wantErr: "no member"},
		{name: "invalid address", input: "a@example.org A\nbruno B\n", wantErr: "line 2: invalid email address"},
		{name: "address with comment", input: "\"Alice\" <a@example.org>\n", wantErr: "line 1: invalid email address"},
		{name: "duplicate address", input: "a@example.org A\nb@example.org B\nA@example.org C\n", wantErr: "line 3: address of line 1 again"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseMembers(strings.NewReader(tc.input))
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("ParseMembers = %v, %v; want error %q", got, err, tc.wantErr)
				}
				return
			}
			if err != nil || !slices.Equal(got, tc.want) {
				t.Errorf("ParseMembers = %+v, %v; want %+v", got, err, tc.want)
			}
		})
	}
}
