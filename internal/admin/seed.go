package admin

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/nelt/tribe-menus/internal/tribe"
)

// The demonstration tribe, with the people of the Gherkin conventions.
const (
	demoSlug tribe.Slug = "demo"
	demoName string     = "Les Démo"
)

var demoMembers = []struct {
	email       tribe.Email
	displayName string
}{
	{"alice@exemple.fr", "Alice"},
	{"bruno@exemple.fr", "Bruno"},
	{"chloe@exemple.fr", "Chloé"},
	{"david@exemple.fr", "David"},
}

// Seed creates the demonstration tribe, initialized with Alice, who adds the others.
// It does nothing if the tribe exists.
func (c *Command) Seed(ctx context.Context) error {
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
		first := demoMembers[0]
		if err := s.Initialize(ctx, demoName, first.email, first.displayName, now); err != nil {
			return err
		}
		alice, err := s.Member(ctx, first.email)
		if err != nil {
			return err
		}
		for _, m := range demoMembers[1:] {
			if _, err := s.AddMember(ctx, m.email, m.displayName, alice.ID, now); err != nil {
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
