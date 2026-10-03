package tribe

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/nelt/tribe-menus/internal/storage/tribedb"
)

// ErrUnknownMember is returned for an address that is not a member of the tribe.
var ErrUnknownMember = errors.New("unknown member")

// Member is a member of the tribe.
type Member struct {
	ID          int64
	Email       Email
	DisplayName string
	Status      Status
}

// AuditEntry is an entry of the audit log.
type AuditEntry struct {
	At          time.Time
	Operation   AuditOperation
	MemberEmail Email
	// AuthorID is the member who made the operation, 0 for the admin command.
	AuthorID int64
}

// ByAdminCommand reports whether the operation was made by the admin command.
func (e AuditEntry) ByAdminCommand() bool {
	return e.AuthorID == 0
}

// Store reads and writes the database of one tribe: a thin adapter around tribedb.
type Store struct {
	db *sql.DB
}

// NewStore returns the store of the tribe whose database is db.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Initialize fills a new tribe database: its name, its first member, added by the
// admin command, and the "tribe initialized" audit entry (EF-08).
func (s *Store) Initialize(ctx context.Context, name string, first Email, displayName string, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("initialize tribe: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := tribedb.New(tx)

	if err := q.InsertTribe(ctx, name); err != nil {
		return fmt.Errorf("initialize tribe: name: %w", err)
	}
	at := formatTime(now)
	memberID, err := q.InsertMember(ctx, tribedb.InsertMemberParams{
		Email:       nullString(string(first)),
		DisplayName: nullString(displayName),
		AddedAt:     at,
	})
	if err != nil {
		return fmt.Errorf("initialize tribe: first member: %w", err)
	}
	if err := q.InsertAuditEntry(ctx, tribedb.InsertAuditEntryParams{
		At:        at,
		Operation: string(TribeInitialized),
		MemberID:  memberID,
	}); err != nil {
		return fmt.Errorf("initialize tribe: audit: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("initialize tribe: %w", err)
	}
	return nil
}

// Name returns the name of the tribe.
func (s *Store) Name(ctx context.Context) (string, error) {
	name, err := tribedb.New(s.db).TribeName(ctx)
	if err != nil {
		return "", fmt.Errorf("tribe name: %w", err)
	}
	return name, nil
}

// Member returns the member with this address, or ErrUnknownMember.
func (s *Store) Member(ctx context.Context, email Email) (Member, error) {
	row, err := tribedb.New(s.db).MemberByEmail(ctx, nullString(string(email)))
	if errors.Is(err, sql.ErrNoRows) {
		return Member{}, ErrUnknownMember
	}
	if err != nil {
		return Member{}, fmt.Errorf("member: %w", err)
	}
	return Member{
		ID:          row.ID,
		Email:       Email(row.Email.String),
		DisplayName: row.DisplayName.String,
		Status:      Status(row.Status),
	}, nil
}

// AuditLog returns the audit log, oldest entry first.
func (s *Store) AuditLog(ctx context.Context) ([]AuditEntry, error) {
	rows, err := tribedb.New(s.db).AuditLog(ctx)
	if err != nil {
		return nil, fmt.Errorf("audit log: %w", err)
	}
	entries := make([]AuditEntry, 0, len(rows))
	for _, row := range rows {
		at, err := time.Parse(time.RFC3339Nano, row.At)
		if err != nil {
			return nil, fmt.Errorf("audit log: entry %d: %w", row.ID, err)
		}
		entries = append(entries, AuditEntry{
			At:          at,
			Operation:   AuditOperation(row.Operation),
			MemberEmail: Email(row.MemberEmail.String),
			AuthorID:    row.AuthorID.Int64,
		})
	}
	return entries, nil
}

// formatTime is the storage format of dates: UTC, RFC 3339.
func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

// nullString stores an empty string as NULL.
func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
