package tribe

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSize(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 3, 9, 30, 0, 0, time.UTC)
	s := newTribe(t, "martin", "Les Martin", "alice@exemple.fr", "Alice", now)

	if size, err := s.Size(ctx); err != nil || size != DefaultSize {
		t.Errorf("Size = %d, %v, want %d (T3)", size, err, DefaultSize)
	}
	cases := []struct {
		size    int
		wantErr error
	}{
		{1, nil}, {50, nil}, {5, nil},
		{0, ErrInvalidSize}, {51, ErrInvalidSize}, {-3, ErrInvalidSize},
	}
	for _, c := range cases {
		if err := s.SetSize(ctx, c.size); !errors.Is(err, c.wantErr) {
			t.Errorf("SetSize(%d) = %v, want %v", c.size, err, c.wantErr)
		}
	}
	if size, err := s.Size(ctx); err != nil || size != 5 {
		t.Errorf("Size = %d, %v, want 5", size, err)
	}
}

func TestActiveMembers(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 3, 9, 30, 0, 0, time.UTC)
	s := newTribe(t, "martin", "Les Martin", "alice@exemple.fr", "Alice", now)
	alice, err := s.Member(ctx, "alice@exemple.fr")
	if err != nil {
		t.Fatal(err)
	}
	bruno, err := s.AddMember(ctx, "bruno@exemple.fr", "Bruno", alice.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddMember(ctx, "chloe@exemple.fr", "Chloé", alice.ID, now); err != nil {
		t.Fatal(err)
	}
	if err := s.RevokeMember(ctx, bruno.ID, alice.ID, now); err != nil {
		t.Fatal(err)
	}
	if n, err := s.ActiveMembers(ctx); err != nil || n != 2 {
		t.Errorf("ActiveMembers = %d, %v, want 2", n, err)
	}
}
