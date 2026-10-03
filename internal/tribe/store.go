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
	// SessionID and DetectedDevice are set for an operation on a session.
	SessionID      int64
	DetectedDevice string
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

// AddMember adds an active member, on behalf of the member addedBy, and traces it (EF-01).
func (s *Store) AddMember(ctx context.Context, email Email, displayName string, addedBy int64, now time.Time) (Member, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Member{}, fmt.Errorf("add member: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := tribedb.New(tx)

	at := formatTime(now)
	id, err := q.InsertMember(ctx, tribedb.InsertMemberParams{
		Email:       nullString(string(email)),
		DisplayName: nullString(displayName),
		AddedAt:     at,
		AddedBy:     sql.NullInt64{Int64: addedBy, Valid: true},
	})
	if err != nil {
		return Member{}, fmt.Errorf("add member: %w", err)
	}
	if err := q.InsertAuditEntry(ctx, tribedb.InsertAuditEntryParams{
		At:        at,
		Operation: string(MemberAdded),
		MemberID:  id,
		AuthorID:  sql.NullInt64{Int64: addedBy, Valid: true},
	}); err != nil {
		return Member{}, fmt.Errorf("add member: audit: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Member{}, fmt.Errorf("add member: %w", err)
	}
	return Member{ID: id, Email: email, DisplayName: displayName, Status: Active}, nil
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
	return memberFromRow(row), nil
}

func memberFromRow(row tribedb.Member) Member {
	return Member{
		ID:          row.ID,
		Email:       Email(row.Email.String),
		DisplayName: row.DisplayName.String,
		Status:      Status(row.Status),
	}
}

// AuditLog returns the audit log, oldest entry first.
func (s *Store) AuditLog(ctx context.Context) ([]AuditEntry, error) {
	rows, err := tribedb.New(s.db).AuditLog(ctx)
	if err != nil {
		return nil, fmt.Errorf("audit log: %w", err)
	}
	entries := make([]AuditEntry, 0, len(rows))
	for _, row := range rows {
		at, err := parseTime(row.At)
		if err != nil {
			return nil, fmt.Errorf("audit log: entry %d: %w", row.ID, err)
		}
		entries = append(entries, AuditEntry{
			At:             at,
			Operation:      AuditOperation(row.Operation),
			MemberEmail:    Email(row.MemberEmail.String),
			AuthorID:       row.AuthorID.Int64,
			SessionID:      row.SessionID.Int64,
			DetectedDevice: row.DetectedDevice.String,
		})
	}
	return entries, nil
}

// timeFormat is the storage format of dates: UTC, fixed width, milliseconds always
// present, so that dates sort as text in chronological order.
const timeFormat = "2006-01-02T15:04:05.000Z"

func formatTime(t time.Time) string {
	return t.UTC().Format(timeFormat)
}

func parseTime(s string) (time.Time, error) {
	t, err := time.Parse(timeFormat, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("stored date %q: %w", s, err)
	}
	return t, nil
}

// nullString stores an empty string as NULL.
func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
