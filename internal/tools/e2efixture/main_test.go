package main

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/nelt/tribe-menus/internal/storage"
	"github.com/nelt/tribe-menus/internal/tribe"
)

func TestPrepare(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	var out, stderr bytes.Buffer
	if code := run(ctx, []string{"-data", dir, "essai-abc", "alice@exemple.fr", "bruno@exemple.fr"}, &out, &stderr); code != 0 {
		t.Fatalf("exit code %d: %s", code, &stderr)
	}
	var tokens map[string]string
	if err := json.Unmarshal(out.Bytes(), &tokens); err != nil {
		t.Fatal(err)
	}

	migrations, err := storage.EmbeddedMigrations()
	if err != nil {
		t.Fatal(err)
	}
	store, err := storage.Open(ctx, dir, migrations)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	db, err := store.Tribe(ctx, "essai-abc")
	if err != nil {
		t.Fatal(err)
	}
	s := tribe.NewStore(db)
	for _, email := range []string{"alice@exemple.fr", "bruno@exemple.fr"} {
		session, err := s.ResumeSession(ctx, tokens[email], time.Now())
		if err != nil || string(session.Member.Email) != email {
			t.Errorf("%s: session %+v, %v", email, session, err)
		}
	}

	// The same tribe again: refused.
	if code := run(ctx, []string{"-data", dir, "essai-abc", "alice@exemple.fr"}, &out, &stderr); code != 1 {
		t.Errorf("tribe created twice: exit code %d", code)
	}
}

func TestRandomIP(t *testing.T) {
	a, err := randomIP()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := randomIP()
	if a == b || a[:9] != "2001:db8:" {
		t.Errorf("%s, %s", a, b)
	}
}
