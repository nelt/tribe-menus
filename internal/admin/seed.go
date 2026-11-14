package admin

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/nelt/tribe-menus/internal/dish"
	"github.com/nelt/tribe-menus/internal/mealplan"
	"github.com/nelt/tribe-menus/internal/shopping"
	"github.com/nelt/tribe-menus/internal/storage"
	"github.com/nelt/tribe-menus/internal/tribe"
)

// The demonstration tribe, with the people of the Gherkin conventions.
const (
	demoSlug tribe.Slug = "demo"
	demoName string     = "Les Démo"
)

// SeedMember is a member of the demonstration tribe.
type SeedMember struct {
	Email       tribe.Email
	DisplayName string
}

// demoDishes are the eight dishes of the mock-up Doux-Bibliotheque (plan bibliotheque, D21),
// in its order, which gives them its hues (Q19); quantities to show the units.
var demoDishes = []dish.Input{
	{Name: "Chili con carne", Lines: []dish.LineInput{
		{Name: "bœuf haché", Quantity: "500", Unit: "g"}, {Name: "haricots rouges", Quantity: "400", Unit: "g"},
		{Name: "oignon", Quantity: "2", Unit: "piece"}, {Name: "tomate concassée", Quantity: "400", Unit: "g"},
	}},
	{Name: "Chili végétarien", Lines: []dish.LineInput{
		{Name: "haricots rouges", Quantity: "500", Unit: "g"}, {Name: "maïs", Quantity: "300", Unit: "g"},
		{Name: "oignon", Quantity: "2", Unit: "piece"}, {Name: "poivron", Quantity: "1", Unit: "piece"},
	}},
	{Name: "Gratin de courgettes", Lines: []dish.LineInput{
		{Name: "courgette", Quantity: "4", Unit: "piece"}, {Name: "crème", Quantity: "20", Unit: "cl"},
		{Name: "gruyère", Quantity: "100", Unit: "g"},
	}},
	{Name: "Gratin dauphinois", Servings: 6, Lines: []dish.LineInput{
		{Name: "pomme de terre", Quantity: "1,5", Unit: "kg"}, {Name: "crème", Quantity: "50", Unit: "cl"},
		{Name: "ail", Quantity: "1", Unit: "piece"},
	}},
	{Name: "Ratatouille", Lines: []dish.LineInput{
		{Name: "courgette", Quantity: "3", Unit: "piece"}, {Name: "aubergine", Quantity: "2", Unit: "piece"},
		{Name: "tomate", Quantity: "4", Unit: "piece"}, {Name: "poivron", Quantity: "1", Unit: "piece"},
		{Name: "oignon", Quantity: "1", Unit: "piece"},
	}},
	{Name: "Soupe à l'oignon", Lines: []dish.LineInput{
		{Name: "oignon", Quantity: "6", Unit: "piece"}, {Name: "bouillon", Quantity: "1", Unit: "l"},
		{Name: "pain", Quantity: "4", Unit: "piece"}, {Name: "gruyère", Quantity: "150", Unit: "g"},
	}},
	{Name: "Pâtes pesto", Lines: []dish.LineInput{
		{Name: "pâtes", Quantity: "500", Unit: "g"}, {Name: "basilic", Quantity: "1", Unit: "piece"},
		{Name: "parmesan", Quantity: "50", Unit: "g"}, {Name: "pignons", Quantity: "2", Unit: "tablespoon"},
	}},
	{Name: "Pizza maison", Lines: []dish.LineInput{
		{Name: "farine", Quantity: "300", Unit: "g"}, {Name: "mozzarella", Quantity: "1", Unit: "piece"},
		{Name: "tomate concassée", Quantity: "200", Unit: "g"}, {Name: "jambon", Quantity: "4", Unit: "piece"},
	}},
}

// demoServed is a dish served in the week of the demonstration tribe: day 0 is the day of
// the seed.
type demoServed struct {
	day      int
	mealtime mealplan.Mealtime
	dish     string
	servings int
}

// demoWeek is the week of the mock-up Doux-Main, Monday to Thursday, from the day of the
// seed (plan planning, D16); « Restes », absent from the library, is replaced by « Pizza
// maison ».
var demoWeek = []demoServed{
	{0, mealplan.Lunch, "Pizza maison", 4},
	{0, mealplan.Dinner, "Ratatouille", 4},
	{1, mealplan.Lunch, "Pâtes pesto", 4},
	{1, mealplan.Dinner, "Chili con carne", 3},
	{1, mealplan.Dinner, "Chili végétarien", 1},
	{2, mealplan.Dinner, "Soupe à l'oignon", 4},
	{3, mealplan.Lunch, "Chili con carne", 4},
	{3, mealplan.Dinner, "Gratin de courgettes", 4},
}

// demoItems are the items added by hand to the main list « Courses », after the shopping of
// the week; demoMarket, the item of the list « Marché » (plan courses, D20).
var (
	demoItems  = []dish.LineInput{{Name: "pain"}, {Name: "lessive"}}
	demoMarket = dish.LineInput{Name: "fraises", Quantity: "500", Unit: "g"}
)

var demoMembers = []SeedMember{
	{"alice@exemple.fr", "Alice"},
	{"bruno@exemple.fr", "Bruno"},
	{"chloe@exemple.fr", "Chloé"},
	{"david@exemple.fr", "David"},
}

// ParseMembers reads a members file (plan recette, D3): one member per line, the address
// then the display name, optional, separated by spaces or tabs; the first member is the
// first member of the tribe. Blank lines and lines starting with # are skipped. Each address
// follows the rule of tribe.ParseEmail; the file is refused as a whole, at its first
// invalid line, so that nothing is created from half a file.
func ParseMembers(r io.Reader) ([]SeedMember, error) {
	var members []SeedMember
	seen := map[tribe.Email]int{}
	scanner := bufio.NewScanner(r)
	for n := 1; scanner.Scan(); n++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		address, displayName, _ := strings.Cut(strings.ReplaceAll(line, "\t", " "), " ")
		email, err := tribe.ParseEmail(address)
		if err != nil {
			return nil, fmt.Errorf("members file, line %d: %w", n, err)
		}
		if first, ok := seen[email]; ok {
			return nil, fmt.Errorf("members file, line %d: address of line %d again", n, first)
		}
		seen[email] = n
		members = append(members, SeedMember{Email: email, DisplayName: strings.TrimSpace(displayName)})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read members file: %w", err)
	}
	if len(members) == 0 {
		return nil, errors.New("members file: no member")
	}
	return members, nil
}

// Seed creates the demonstration tribe with the members given, or without them with the
// people of the Gherkin conventions, the dishes of the mock-up (plan bibliotheque, D21) and
// its week from the day of the seed (plan planning, D16); the first member adds the others.
// It does nothing if the tribe exists.
func (c *Command) Seed(ctx context.Context, members []SeedMember) error {
	var dishes []dish.Input
	var week []demoServed
	demo := len(members) == 0
	if demo {
		members, dishes, week = demoMembers, demoDishes, demoWeek
	}
	taken, err := c.Store.SlugTaken(ctx, string(demoSlug))
	if err != nil {
		return err
	}
	if taken {
		fmt.Fprintf(c.Out, "La tribu de démonstration existe déjà : %s\n", c.TribeURL(demoSlug))
		return nil
	}

	now := c.Now()
	err = c.Store.CreateTribe(ctx, string(demoSlug), demoName, func(ctx context.Context, db *sql.DB) error {
		s := tribe.NewStore(db)
		first := members[0]
		if err := s.Initialize(ctx, demoName, first.Email, first.DisplayName, now); err != nil {
			return err
		}
		firstMember, err := s.Member(ctx, first.Email)
		if err != nil {
			return err
		}
		for _, m := range members[1:] {
			if _, err := s.AddMember(ctx, m.Email, m.DisplayName, firstMember.ID, now); err != nil {
				return err
			}
		}
		library := dish.NewLibrary(storage.NewDishStore(db))
		ids := map[string]dish.ID{}
		for _, d := range dishes {
			created, _, err := library.Create(ctx, d)
			if err != nil {
				return err
			}
			ids[created.Name] = created.ID
		}
		plan := mealplan.NewPlan(storage.NewMealPlanStore(db))
		var served []mealplan.ServedDish
		for _, s := range week {
			sd, _, err := plan.Serve(ctx, mealplan.ServeInput{
				Date: now.AddDate(0, 0, s.day).Format(time.DateOnly), Mealtime: string(s.mealtime),
				Dish: ids[s.dish], Servings: &s.servings,
			})
			if err != nil {
				return err
			}
			served = append(served, sd)
		}
		if !demo {
			return nil
		}
		return seedShopping(ctx, shopping.NewLists(storage.NewShoppingStore(db)), plan, served, now)
	})
	if err != nil {
		return fmt.Errorf("create the demonstration tribe: %w", err)
	}
	fmt.Fprintf(c.Out, "Tribu de démonstration « %s » créée : %s\n", demoName, c.TribeURL(demoSlug))
	return nil
}

// demoTakenServed and demoStaleServed are the served dishes of demoWeek whose servings change
// after the shopping of the week (plan courses-suite, D20): the Tuesday chili, the list
// recomputed, then the onion soup, the list left stale.
const (
	demoTakenServed, demoTakenServings = 3, 5
	demoStaleServed, demoStaleServings = 5, 6
)

// seedShopping fills the lists of the demonstration tribe (plan courses, D20): the shopping
// of the week of the seed and the items added by hand in « Courses », the main list, and a
// list « Marché » with one item. Then (plan courses-suite, D20) « bœuf haché » is checked and
// the Tuesday chili served for 5: the recomputation gives it a taken quantity; the onion soup
// served for 6 leaves « Courses » stale.
func seedShopping(ctx context.Context, lists *shopping.Lists, plan *mealplan.Plan, served []mealplan.ServedDish, now time.Time) error {
	_, page, err := lists.Page(ctx, "")
	if err != nil {
		return err
	}
	var main shopping.ListID
	for _, l := range page.Items {
		if l.Main {
			main = l.ID
		}
	}
	from, to := now.Format(time.DateOnly), now.AddDate(0, 0, 6).Format(time.DateOnly)
	if _, _, _, _, err := lists.AddMealShopping(ctx, main, from, to); err != nil {
		return err
	}
	for _, in := range demoItems {
		if _, _, err := lists.AddItem(ctx, main, in); err != nil {
			return err
		}
	}
	market, _, err := lists.Create(ctx, "Marché")
	if err != nil {
		return err
	}
	if _, _, err = lists.AddItem(ctx, market.ID, demoMarket); err != nil {
		return err
	}

	l, _, err := lists.Recompute(ctx, main)
	if err != nil {
		return err
	}
	for _, it := range l.Items {
		if it.Ingredient == "bœuf haché" {
			if _, _, err := lists.SetChecked(ctx, main, it.ID, true); err != nil {
				return err
			}
		}
	}
	if _, _, err := plan.SetServings(ctx, served[demoTakenServed].ID, demoTakenServings); err != nil {
		return err
	}
	if _, _, err := lists.Recompute(ctx, main); err != nil {
		return err
	}
	_, _, err = plan.SetServings(ctx, served[demoStaleServed].ID, demoStaleServings)
	return err
}
