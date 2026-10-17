package admin

import (
	"bytes"
	"context"
	"slices"
	"strings"
	"testing"
	"time"

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
}

// TestSeedWithMembers: the members of the file, the first one first member, who adds the
// others (recette, D3).
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
