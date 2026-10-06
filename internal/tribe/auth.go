package tribe

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// CodeMailer sends a login code by email. Sending happens outside of the request, so
// that the response time does not tell a member's address from another (ENF-02).
type CodeMailer interface {
	SendLoginCode(to, code string)
}

// Login is the login by code and the sessions of every tribe (ENF-01). Nothing it answers
// before a session is open tells an active member from another address, nor an existing
// tribe from an unknown one (ENF-02): the same errors, attempts and limits apply to both,
// with decoy codes for the latter.
type Login struct {
	RateLimit RateLimitStore
	Mailer    CodeMailer
	Now       func() time.Time
}

// RequestCode asks for a code for the address in the tribe of this slug, from ip. The
// tribe store t is nil for an unknown tribe. It returns the token of the request, for a
// cookie of the browser, whether a code was sent or not; or ErrInvalidEmail or
// ErrTooManyRequests.
func (l *Login) RequestCode(ctx context.Context, slug string, t *Store, rawEmail, ip string) (string, error) {
	email, err := ParseEmail(rawEmail)
	if err != nil {
		return "", err
	}
	now := l.Now()
	req := NewCodeRequest(slug, email, ip)
	refusal, err := l.RateLimit.RecordCodeRequest(ctx, req, now)
	if err != nil {
		return "", err
	}
	if refusal != "" {
		return "", ErrTooManyRequests
	}
	token, err := newToken()
	if err != nil {
		return "", fmt.Errorf("request code: %w", err)
	}
	// A decoy code for every request, before looking for the member: the rate limit
	// database keeps the same rows for a member and for any other address (ADR 0021,
	// point 4; plan production, D11). The decoy code of a member is never checked.
	if err := l.RateLimit.ReplaceDecoyCode(ctx, req.EmailHash, NewDecoyCode(token, now)); err != nil {
		return "", err
	}

	member, ok, err := activeMember(ctx, t, email)
	if err != nil {
		return "", err
	}
	if !ok {
		return token, nil
	}
	code, err := newLoginCode()
	if err != nil {
		return "", err
	}
	// Counted by the rate limit database first, like any other request: what it records
	// does not depend on membership (ADR 0021).
	if err := t.IssueLoginCode(ctx, member.ID, newRealCode(code, token, now), now); err != nil {
		return "", err
	}
	l.Mailer.SendLoginCode(string(email), code)
	return token, nil
}

// OpenSession checks the code entered for the address in the tribe of this slug (t nil for
// an unknown tribe), with the token of the request presented by the browser, and opens a
// session for the device. It returns the session and its token, or ErrInvalidEmail,
// ErrNewCodeNeeded or *IncorrectCodeError.
func (l *Login) OpenSession(ctx context.Context, slug string, t *Store, rawEmail, code, requestToken string, device Device) (Session, string, error) {
	email, err := ParseEmail(rawEmail)
	if err != nil {
		return Session{}, "", err
	}
	now := l.Now()
	member, ok, err := activeMember(ctx, t, email)
	if err != nil {
		return Session{}, "", err
	}
	if !ok {
		return Session{}, "", l.RateLimit.CheckDecoyCode(ctx, emailHash(slug, email), code, requestToken, now)
	}
	return t.OpenSession(ctx, member.ID, code, requestToken, device, now)
}

// Session returns the session of the token in the tribe store t, extending it, or
// ErrNoSession. t is nil for an unknown tribe.
func (l *Login) Session(ctx context.Context, t *Store, token string) (Session, error) {
	if t == nil || token == "" {
		return Session{}, ErrNoSession
	}
	return t.ResumeSession(ctx, token, l.Now())
}

// SignOut closes the session at the request of its member.
func (l *Login) SignOut(ctx context.Context, t *Store, session Session) error {
	return t.SignOut(ctx, session, l.Now())
}

// activeMember returns the member of this address if it is active in the tribe store t.
func activeMember(ctx context.Context, t *Store, email Email) (Member, bool, error) {
	if t == nil {
		return Member{}, false, nil
	}
	member, err := t.Member(ctx, email)
	if errors.Is(err, ErrUnknownMember) {
		return Member{}, false, nil
	}
	if err != nil {
		return Member{}, false, err
	}
	return member, member.Status == Active, nil
}

// Databases gives access to the database of each tribe: storage.Store.
type Databases interface {
	Tribes(ctx context.Context) ([]string, error)
	Tribe(ctx context.Context, slug string) (*sql.DB, error)
}

// Purge deletes what is no longer needed (PT-07): in every tribe, the login codes expired
// or out of attempts and the expired sessions; in the rate limit database, every row out of
// its window. It goes on after a failing tribe and returns every error.
func (l *Login) Purge(ctx context.Context, dbs Databases) error {
	now := l.Now()
	errs := []error{l.RateLimit.Purge(ctx, now)}
	slugs, err := dbs.Tribes(ctx)
	if err != nil {
		return errors.Join(append(errs, err)...)
	}
	for _, slug := range slugs {
		db, err := dbs.Tribe(ctx, slug)
		if err == nil {
			err = NewStore(db).Purge(ctx, now)
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("tribe %s: %w", slug, err))
		}
	}
	return errors.Join(errs...)
}
