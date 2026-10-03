package admin

import (
	"bytes"
	"context"
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
		if err := cmd.Seed(ctx); err != nil {
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
		got, err := s.Member(ctx, m.email)
		if err != nil || got.Status != tribe.Active || got.DisplayName != m.displayName {
			t.Errorf("Member(%s) = %+v, %v", m.email, got, err)
		}
	}
	if entries, err := s.AuditLog(ctx); err != nil || len(entries) != len(demoMembers) {
		t.Errorf("AuditLog = %d entries, %v; want %d", len(entries), err, len(demoMembers))
	}
}
