package tribe

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/nelt/tribe-menus/internal/storage/tribedb"
)

// Session is a session of a member on a device (ENF-01, EF-04).
type Session struct {
	ID           int64
	Member       Member
	OpenedAt     time.Time
	LastActiveAt time.Time
	ExpiresAt    time.Time
	Device       Device
	Label        string
}

// IssueLoginCode stores the code of a member, replacing the previous one, unless the limit
// by tribe is reached: it then returns ErrTooManyRequests. Only the requests for active
// members are counted here (ENF-01; plan revue-securite, D2).
func (s *Store) IssueLoginCode(ctx context.Context, memberID int64, code LoginCode, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("issue login code: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := tribedb.New(tx)

	n, err := q.CountCodeRequests(ctx, formatTime(TribeRequestWindowStart(now)))
	if err != nil {
		return fmt.Errorf("issue login code: count requests: %w", err)
	}
	if !TribeRequestAllowed(int(n)) {
		return ErrTooManyRequests
	}
	if err := q.InsertCodeRequest(ctx, formatTime(now)); err != nil {
		return fmt.Errorf("issue login code: record request: %w", err)
	}
	if err := q.ReplaceLoginCode(ctx, tribedb.ReplaceLoginCodeParams{
		MemberID:     memberID,
		CodeHash:     code.Hash,
		RequestHash:  code.RequestHash,
		ExpiresAt:    formatTime(code.ExpiresAt),
		AttemptsLeft: int64(code.AttemptsLeft),
	}); err != nil {
		return fmt.Errorf("issue login code: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("issue login code: %w", err)
	}
	return nil
}

// OpenSession checks the code entered by the member, with the token of its request, and,
// when it is accepted, opens a session for the device and traces it. It returns the session
// and its token, or ErrNewCodeNeeded, or *IncorrectCodeError; the attempt is recorded in
// every case.
func (s *Store) OpenSession(ctx context.Context, memberID int64, entered, requestToken string, device Device, now time.Time) (Session, string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Session{}, "", fmt.Errorf("open session: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := tribedb.New(tx)

	row, err := q.LoginCode(ctx, memberID)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, "", ErrNewCodeNeeded
	}
	if err != nil {
		return Session{}, "", fmt.Errorf("open session: login code: %w", err)
	}
	expiresAt, err := parseTime(row.ExpiresAt)
	if err != nil {
		return Session{}, "", fmt.Errorf("open session: %w", err)
	}
	code := LoginCode{Hash: row.CodeHash, RequestHash: row.RequestHash, ExpiresAt: expiresAt, AttemptsLeft: int(row.AttemptsLeft)}
	after, keep, checkErr := code.Check(entered, requestToken, now)
	if keep {
		err = q.SetLoginCodeAttempts(ctx, tribedb.SetLoginCodeAttemptsParams{AttemptsLeft: int64(after.AttemptsLeft), MemberID: memberID})
	} else {
		err = q.DeleteLoginCode(ctx, memberID)
	}
	if err != nil {
		return Session{}, "", fmt.Errorf("open session: record attempt: %w", err)
	}
	if checkErr != nil {
		if err := tx.Commit(); err != nil {
			return Session{}, "", fmt.Errorf("open session: record attempt: %w", err)
		}
		return Session{}, "", checkErr
	}

	token, err := newToken()
	if err != nil {
		return Session{}, "", fmt.Errorf("open session: %w", err)
	}
	at := formatTime(now)
	id, err := q.InsertSession(ctx, tribedb.InsertSessionParams{
		TokenHash:    hash(token),
		MemberID:     memberID,
		OpenedAt:     at,
		LastActiveAt: at,
		ExpiresAt:    formatTime(sessionExpiry(now)),
		DeviceType:   string(device.Type),
		Os:           device.OS,
		Browser:      device.Browser,
		InstalledApp: boolInt(device.InstalledApp),
	})
	if err != nil {
		return Session{}, "", fmt.Errorf("open session: %w", err)
	}
	if err := q.InsertAuditEntry(ctx, sessionAudit(SessionOpened, memberID, memberID, id, device, now)); err != nil {
		return Session{}, "", fmt.Errorf("open session: audit: %w", err)
	}
	session, err := readSession(ctx, q, id)
	if err != nil {
		return Session{}, "", fmt.Errorf("open session: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Session{}, "", fmt.Errorf("open session: %w", err)
	}
	return session, token, nil
}

// ResumeSession returns the session of this token, extending its expiry (sliding expiry),
// or ErrNoSession for an unknown or expired token, or a member no longer active.
func (s *Store) ResumeSession(ctx context.Context, token string, now time.Time) (Session, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Session{}, fmt.Errorf("resume session: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := tribedb.New(tx)

	row, err := q.SessionByTokenHash(ctx, tribedb.SessionByTokenHashParams{TokenHash: hash(token), ExpiresAt: formatTime(now)})
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrNoSession
	}
	if err != nil {
		return Session{}, fmt.Errorf("resume session: %w", err)
	}
	if err := q.TouchSession(ctx, tribedb.TouchSessionParams{
		LastActiveAt: formatTime(now),
		ExpiresAt:    formatTime(sessionExpiry(now)),
		ID:           row.ID,
	}); err != nil {
		return Session{}, fmt.Errorf("resume session: %w", err)
	}
	session, err := readSession(ctx, q, row.ID)
	if err != nil {
		return Session{}, fmt.Errorf("resume session: %w", err)
	}
	if session.Member.Status != Active {
		return Session{}, ErrNoSession
	}
	if err := tx.Commit(); err != nil {
		return Session{}, fmt.Errorf("resume session: %w", err)
	}
	return session, nil
}

// SignOut closes the session at the request of its member, and traces it.
func (s *Store) SignOut(ctx context.Context, session Session, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sign out: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := tribedb.New(tx)

	if err := q.DeleteSession(ctx, session.ID); err != nil {
		return fmt.Errorf("sign out: %w", err)
	}
	memberID := session.Member.ID
	if err := q.InsertAuditEntry(ctx, sessionAudit(SignedOut, memberID, memberID, session.ID, session.Device, now)); err != nil {
		return fmt.Errorf("sign out: audit: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sign out: %w", err)
	}
	return nil
}

// Sessions returns the sessions of a member, expired ones included until purged.
func (s *Store) Sessions(ctx context.Context, memberID int64) ([]Session, error) {
	q := tribedb.New(s.db)
	rows, err := q.MemberSessions(ctx, memberID)
	if err != nil {
		return nil, fmt.Errorf("sessions: %w", err)
	}
	member, err := memberByID(ctx, q, memberID)
	if err != nil {
		return nil, fmt.Errorf("sessions: %w", err)
	}
	sessions := make([]Session, 0, len(rows))
	for _, row := range rows {
		session, err := sessionFromRow(row, member)
		if err != nil {
			return nil, fmt.Errorf("sessions: %w", err)
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}

// RevokeMember revokes an active member on behalf of the member revokedBy, or of the admin
// command when revokedBy is 0 (EF-03): its sessions are closed and its login code invalidated.
func (s *Store) RevokeMember(ctx context.Context, memberID, revokedBy int64, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("revoke member: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := tribedb.New(tx)

	at := formatTime(now)
	author := sql.NullInt64{Int64: revokedBy, Valid: revokedBy != 0}
	n, err := q.RevokeMember(ctx, tribedb.RevokeMemberParams{RevokedAt: sql.NullString{String: at, Valid: true}, RevokedBy: author, ID: memberID})
	if err != nil {
		return fmt.Errorf("revoke member: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("revoke member %d: %w", memberID, ErrUnknownMember)
	}
	if err := q.DeleteLoginCode(ctx, memberID); err != nil {
		return fmt.Errorf("revoke member: login code: %w", err)
	}
	sessions, err := q.MemberSessions(ctx, memberID)
	if err != nil {
		return fmt.Errorf("revoke member: sessions: %w", err)
	}
	for _, row := range sessions {
		if err := q.DeleteSession(ctx, row.ID); err != nil {
			return fmt.Errorf("revoke member: close session: %w", err)
		}
		entry := sessionAudit(SessionClosed, memberID, revokedBy, row.ID, deviceFromRow(row), now)
		if err := q.InsertAuditEntry(ctx, entry); err != nil {
			return fmt.Errorf("revoke member: audit: %w", err)
		}
	}
	if err := q.InsertAuditEntry(ctx, tribedb.InsertAuditEntryParams{
		At:        at,
		Operation: string(MemberRevoked),
		MemberID:  memberID,
		AuthorID:  author,
	}); err != nil {
		return fmt.Errorf("revoke member: audit: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("revoke member: %w", err)
	}
	return nil
}

// Purge deletes the login codes expired or out of attempts, the expired sessions (PT-07),
// and the code requests out of the window of the limit by tribe.
func (s *Store) Purge(ctx context.Context, now time.Time) error {
	q := tribedb.New(s.db)
	at := formatTime(now)
	if _, err := q.PurgeLoginCodes(ctx, at); err != nil {
		return fmt.Errorf("purge login codes: %w", err)
	}
	if _, err := q.PurgeCodeRequests(ctx, formatTime(TribeRequestWindowStart(now))); err != nil {
		return fmt.Errorf("purge code requests: %w", err)
	}
	if _, err := q.PurgeSessions(ctx, at); err != nil {
		return fmt.Errorf("purge sessions: %w", err)
	}
	return nil
}

func sessionAudit(op AuditOperation, memberID, authorID, sessionID int64, device Device, now time.Time) tribedb.InsertAuditEntryParams {
	return tribedb.InsertAuditEntryParams{
		At:           formatTime(now),
		Operation:    string(op),
		MemberID:     memberID,
		AuthorID:     sql.NullInt64{Int64: authorID, Valid: authorID != 0},
		SessionID:    sql.NullInt64{Int64: sessionID, Valid: true},
		DeviceType:   sql.NullString{String: string(device.Type), Valid: true},
		Os:           sql.NullString{String: device.OS, Valid: true},
		Browser:      sql.NullString{String: device.Browser, Valid: true},
		InstalledApp: sql.NullInt64{Int64: boolInt(device.InstalledApp), Valid: true},
	}
}

func readSession(ctx context.Context, q *tribedb.Queries, id int64) (Session, error) {
	row, err := q.SessionByID(ctx, id)
	if err != nil {
		return Session{}, fmt.Errorf("session %d: %w", id, err)
	}
	member, err := memberByID(ctx, q, row.MemberID)
	if err != nil {
		return Session{}, err
	}
	return sessionFromRow(row, member)
}

func memberByID(ctx context.Context, q *tribedb.Queries, id int64) (Member, error) {
	row, err := q.MemberByID(ctx, id)
	if err != nil {
		return Member{}, fmt.Errorf("member %d: %w", id, err)
	}
	return memberFromRow(row), nil
}

func sessionFromRow(row tribedb.Session, member Member) (Session, error) {
	var times [3]time.Time
	for i, s := range []string{row.OpenedAt, row.LastActiveAt, row.ExpiresAt} {
		t, err := parseTime(s)
		if err != nil {
			return Session{}, fmt.Errorf("session %d: %w", row.ID, err)
		}
		times[i] = t
	}
	return Session{
		ID:           row.ID,
		Member:       member,
		OpenedAt:     times[0],
		LastActiveAt: times[1],
		ExpiresAt:    times[2],
		Device:       deviceFromRow(row),
		Label:        row.Label.String,
	}, nil
}

func deviceFromRow(row tribedb.Session) Device {
	return Device{Type: DeviceType(row.DeviceType), OS: row.Os, Browser: row.Browser, InstalledApp: row.InstalledApp == 1}
}

func boolInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
