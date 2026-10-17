package admin

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"

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
// people of the Gherkin conventions; the first member adds the others. It does nothing if
// the tribe exists.
func (c *Command) Seed(ctx context.Context, members []SeedMember) error {
	if len(members) == 0 {
		members = demoMembers
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
		return nil
	})
	if err != nil {
		return fmt.Errorf("create the demonstration tribe: %w", err)
	}
	fmt.Fprintf(c.Out, "Tribu de démonstration « %s » créée : %s\n", demoName, c.TribeURL(demoSlug))
	return nil
}
