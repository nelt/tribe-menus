package tribe

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/nelt/tribe-menus/internal/storage/ratelimitdb"
)

// RateLimitStore keeps the code requests and the decoy codes of the instance (ADR 0021).
type RateLimitStore interface {
	// RecordCodeRequest records the request at now if the limits by address and by IP allow
	// it. Otherwise it counts the refusal for the alerts and returns it: EmailLimitRefusal or
	// IPLimitRefusal; "" when the request is allowed. Counting and recording are atomic.
	RecordCodeRequest(ctx context.Context, req CodeRequest, now time.Time) (AlertEvent, error)
	// ReplaceDecoyCode stores a decoy code for the address hash, replacing the previous one.
	ReplaceDecoyCode(ctx context.Context, emailHash []byte, code LoginCode) error
	// CheckDecoyCode records an attempt on the decoy code of the address hash, with the
	// token of its request: it returns ErrNewCodeNeeded or *IncorrectCodeError, as for a
	// real code. A decoy code invalidated by the attempt is counted for the alerts.
	CheckDecoyCode(ctx context.Context, emailHash []byte, entered, requestToken string, now time.Time) error
	// Purge deletes the rows out of their window.
	Purge(ctx context.Context, now time.Time) error
	// AlertCounts returns the counters of the window at now, and the number of address
	// hashes with RepeatedRequestsThreshold requests or more in it.
	AlertCounts(ctx context.Context, now time.Time) (AlertCounts, error)
	// AlertsSent returns the time of the last alert on each signal.
	AlertsSent(ctx context.Context) (map[AlertSignal]time.Time, error)
	// SetAlertsSent records alerts on these signals at now.
	SetAlertsSent(ctx context.Context, signals []AlertSignal, now time.Time) error
}

// RateLimitDB is the rate limit database: a thin adapter around ratelimitdb.
type RateLimitDB struct {
	db *sql.DB
}

// NewRateLimitDB returns the store of the rate limit database db.
func NewRateLimitDB(db *sql.DB) *RateLimitDB {
	return &RateLimitDB{db: db}
}

var _ RateLimitStore = (*RateLimitDB)(nil)

// RecordCodeRequest implements RateLimitStore.
func (r *RateLimitDB) RecordCodeRequest(ctx context.Context, req CodeRequest, now time.Time) (AlertEvent, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("record code request: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := ratelimitdb.New(tx)

	emailSince, ipSince := RequestWindows(now)
	byEmail, err := q.CountCodeRequestsByEmail(ctx, ratelimitdb.CountCodeRequestsByEmailParams{EmailHash: req.EmailHash, At: formatTime(emailSince)})
	if err != nil {
		return "", fmt.Errorf("record code request: count by address: %w", err)
	}
	byIP, err := q.CountCodeRequestsByIP(ctx, ratelimitdb.CountCodeRequestsByIPParams{IpHash: req.IPHash, At: formatTime(ipSince)})
	if err != nil {
		return "", fmt.Errorf("record code request: count by IP: %w", err)
	}
	counts := CodeRequestCounts{Email: int(byEmail), IP: int(byIP)}
	refusal := counts.Refusal()
	if refusal != "" {
		err = q.IncrementAlertCounter(ctx, ratelimitdb.IncrementAlertCounterParams{Slot: alertSlot(now), Event: string(refusal)})
	} else {
		err = q.InsertCodeRequest(ctx, ratelimitdb.InsertCodeRequestParams{
			At:        formatTime(now),
			EmailHash: req.EmailHash,
			IpHash:    req.IPHash,
		})
	}
	if err != nil {
		return "", fmt.Errorf("record code request: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("record code request: %w", err)
	}
	return refusal, nil
}

// ReplaceDecoyCode implements RateLimitStore.
func (r *RateLimitDB) ReplaceDecoyCode(ctx context.Context, emailHash []byte, code LoginCode) error {
	err := ratelimitdb.New(r.db).ReplaceDecoyCode(ctx, ratelimitdb.ReplaceDecoyCodeParams{
		EmailHash:    emailHash,
		RequestHash:  code.RequestHash,
		ExpiresAt:    formatTime(code.ExpiresAt),
		AttemptsLeft: int64(code.AttemptsLeft),
	})
	if err != nil {
		return fmt.Errorf("replace decoy code: %w", err)
	}
	return nil
}

// CheckDecoyCode implements RateLimitStore.
func (r *RateLimitDB) CheckDecoyCode(ctx context.Context, emailHash []byte, entered, requestToken string, now time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("check decoy code: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := ratelimitdb.New(tx)

	row, err := q.DecoyCode(ctx, emailHash)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNewCodeNeeded
	}
	if err != nil {
		return fmt.Errorf("check decoy code: %w", err)
	}
	expiresAt, err := parseTime(row.ExpiresAt)
	if err != nil {
		return fmt.Errorf("check decoy code: %w", err)
	}
	decoy := LoginCode{RequestHash: row.RequestHash, ExpiresAt: expiresAt, AttemptsLeft: int(row.AttemptsLeft)}
	after, keep, checkErr := decoy.Check(entered, requestToken, now)
	if checkErr == nil {
		return errors.New("check decoy code: a decoy code was accepted")
	}
	if keep {
		err = q.SetDecoyCodeAttempts(ctx, ratelimitdb.SetDecoyCodeAttemptsParams{AttemptsLeft: int64(after.AttemptsLeft), EmailHash: emailHash})
	} else {
		err = q.DeleteDecoyCode(ctx, emailHash)
	}
	if err == nil && exhausted(checkErr) {
		err = q.IncrementAlertCounter(ctx, ratelimitdb.IncrementAlertCounterParams{Slot: alertSlot(now), Event: string(DecoyCodeExhausted)})
	}
	if err != nil {
		return fmt.Errorf("check decoy code: record attempt: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("check decoy code: %w", err)
	}
	return checkErr
}

// Purge implements RateLimitStore: requests older than the longest window, decoy codes
// expired or out of attempts, alert counters out of their window.
func (r *RateLimitDB) Purge(ctx context.Context, now time.Time) error {
	q := ratelimitdb.New(r.db)
	if _, err := q.PurgeCodeRequests(ctx, formatTime(now.Add(-RateLimitRetention))); err != nil {
		return fmt.Errorf("purge code requests: %w", err)
	}
	if _, err := q.PurgeDecoyCodes(ctx, formatTime(now)); err != nil {
		return fmt.Errorf("purge decoy codes: %w", err)
	}
	if _, err := q.PurgeAlertCounters(ctx, formatTime(alertWindowStart(now))); err != nil {
		return fmt.Errorf("purge alert counters: %w", err)
	}
	return nil
}

// AlertCounts implements RateLimitStore.
func (r *RateLimitDB) AlertCounts(ctx context.Context, now time.Time) (AlertCounts, error) {
	q := ratelimitdb.New(r.db)
	since := formatTime(alertWindowStart(now))
	rows, err := q.AlertCounts(ctx, since)
	if err != nil {
		return AlertCounts{}, fmt.Errorf("alert counts: %w", err)
	}
	var counts AlertCounts
	events := map[AlertEvent]int{}
	for _, row := range rows {
		events[AlertEvent(row.Event)] = int(row.N)
	}
	counts.add(events)
	repeated, err := q.CountRepeatedRequests(ctx, ratelimitdb.CountRepeatedRequestsParams{Since: since, MinRequests: RepeatedRequestsThreshold})
	if err != nil {
		return AlertCounts{}, fmt.Errorf("alert counts: repeated requests: %w", err)
	}
	counts.RepeatedAddresses = int(repeated)
	return counts, nil
}

// AlertsSent implements RateLimitStore.
func (r *RateLimitDB) AlertsSent(ctx context.Context) (map[AlertSignal]time.Time, error) {
	rows, err := ratelimitdb.New(r.db).AlertsSent(ctx)
	if err != nil {
		return nil, fmt.Errorf("alerts sent: %w", err)
	}
	sent := map[AlertSignal]time.Time{}
	for _, row := range rows {
		at, err := parseTime(row.SentAt)
		if err != nil {
			return nil, fmt.Errorf("alerts sent: %w", err)
		}
		sent[AlertSignal(row.Signal)] = at
	}
	return sent, nil
}

// SetAlertsSent implements RateLimitStore.
func (r *RateLimitDB) SetAlertsSent(ctx context.Context, signals []AlertSignal, now time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("set alerts sent: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := ratelimitdb.New(tx)
	for _, s := range signals {
		if err := q.SetAlertSent(ctx, ratelimitdb.SetAlertSentParams{Signal: string(s), SentAt: formatTime(now)}); err != nil {
			return fmt.Errorf("set alerts sent: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("set alerts sent: %w", err)
	}
	return nil
}
