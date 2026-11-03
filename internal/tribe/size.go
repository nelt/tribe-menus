package tribe

import (
	"context"
	"errors"
	"fmt"

	"github.com/nelt/tribe-menus/internal/storage/tribedb"
)

// Tribe size (T3): the servings proposed for a new dish, then for a dish served (Q31).
const (
	DefaultSize = 4
	MinSize     = 1
	MaxSize     = 50
)

// ErrInvalidSize is returned for a size out of MinSize to MaxSize.
var ErrInvalidSize = errors.New("invalid tribe size")

// Size returns the size of the tribe.
func (s *Store) Size(ctx context.Context) (int, error) {
	size, err := tribedb.New(s.db).TribeSize(ctx)
	if err != nil {
		return 0, fmt.Errorf("tribe size: %w", err)
	}
	return int(size), nil
}

// SetSize sets the size of the tribe, or returns ErrInvalidSize. The dishes keep their
// servings (T3).
func (s *Store) SetSize(ctx context.Context, size int) error {
	if size < MinSize || size > MaxSize {
		return ErrInvalidSize
	}
	if err := tribedb.New(s.db).SetTribeSize(ctx, int64(size)); err != nil {
		return fmt.Errorf("set tribe size: %w", err)
	}
	return nil
}

// ActiveMembers counts the active members of the tribe.
func (s *Store) ActiveMembers(ctx context.Context) (int, error) {
	n, err := tribedb.New(s.db).CountActiveMembers(ctx)
	if err != nil {
		return 0, fmt.Errorf("count active members: %w", err)
	}
	return int(n), nil
}
