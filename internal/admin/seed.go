package admin

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/nelt/tribe-menus/internal/dish"
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
// people of the Gherkin conventions and the dishes of the mock-up (plan bibliotheque, D21);
// the first member adds the others. It does nothing if the tribe exists.
func (c *Command) Seed(ctx context.Context, members []SeedMember) error {
	var dishes []dish.Input
	if len(members) == 0 {
		members, dishes = demoMembers, demoDishes
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
		for _, d := range dishes {
			if _, _, err := library.Create(ctx, d); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("create the demonstration tribe: %w", err)
	}
	fmt.Fprintf(c.Out, "Tribu de démonstration « %s » créée : %s\n", demoName, c.TribeURL(demoSlug))
	return nil
}
