package tribe

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/nelt/tribe-menus/internal/storage"
)

func newTribe(t *testing.T, slug, name string, first Email, displayName string, now time.Time) *Store {
	t.Helper()
	ctx := context.Background()
	migrations, err := storage.EmbeddedMigrations()
	if err != nil {
		t.Fatal(err)
	}
	st, err := storage.Open(ctx, t.TempDir(), migrations)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	err = st.CreateTribe(ctx, slug, name, func(ctx context.Context, db *sql.DB) error {
		return NewStore(db).Initialize(ctx, name, first, displayName, now)
	})
	if err != nil {
		t.Fatalf("CreateTribe: %v", err)
	}
	db, err := st.Tribe(ctx, slug)
	if err != nil {
		t.Fatal(err)
	}
	return NewStore(db)
}

func TestInitialize(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 3, 9, 30, 0, 0, time.FixedZone("CEST", 2*60*60))
	s := newTribe(t, "martin", "Les Martin", "alice@exemple.fr", "Alice", now)

	name, err := s.Name(ctx)
	if err != nil || name != "Les Martin" {
		t.Errorf("Name = %q, %v", name, err)
	}

	member, err := s.Member(ctx, "alice@exemple.fr")
	if err != nil {
		t.Fatal(err)
	}
	want := Member{ID: member.ID, Email: "alice@exemple.fr", DisplayName: "Alice", Status: Active}
	if member != want {
		t.Errorf("Member = %+v, want %+v", member, want)
	}

	if _, err := s.Member(ctx, "bruno@exemple.fr"); !errors.Is(err, ErrUnknownMember) {
		t.Errorf("Member(bruno) error = %v, want ErrUnknownMember", err)
	}

	entries, err := s.AuditLog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	wantEntry := AuditEntry{At: now.UTC(), Operation: TribeInitialized, MemberEmail: "alice@exemple.fr"}
	if len(entries) != 1 || entries[0] != wantEntry || !entries[0].ByAdminCommand() {
		t.Errorf("AuditLog = %+v, want [%+v]", entries, wantEntry)
	}
}

func TestInitializeWithoutDisplayName(t *testing.T) {
	s := newTribe(t, "martin", "Les Martin", "alice@exemple.fr", "", time.Now())
	member, err := s.Member(context.Background(), "alice@exemple.fr")
	if err != nil || member.DisplayName != "" {
		t.Errorf("Member = %+v, %v; want no display name", member, err)
	}
}
