package tribe

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"time"
)

// Login codes and sessions (ENF-01).
const (
	LoginCodeValidity = 10 * time.Minute
	LoginCodeAttempts = 3
	loginCodeDigits   = 8

	// SessionLifetime is the sliding expiry of a session: each use extends it.
	SessionLifetime = 90 * 24 * time.Hour
	tokenLen        = 32
)

var (
	// ErrNewCodeNeeded is returned when entering a code that expired, ran out of attempts,
	// was used or replaced, or was never requested: only a new code can succeed.
	ErrNewCodeNeeded = errors.New("login code expired or invalidated, request a new one")
	// ErrTooManyRequests is returned when a code request goes beyond a rate limit.
	ErrTooManyRequests = errors.New("too many code requests")
	// ErrNoSession is returned for a missing, unknown or expired session token.
	ErrNoSession = errors.New("no valid session")
)

// IncorrectCodeError is returned for a wrong code, with the attempts left on the code.
// With no attempt left, the code is invalidated.
type IncorrectCodeError struct {
	AttemptsLeft int
}

func (e *IncorrectCodeError) Error() string {
	return fmt.Sprintf("incorrect login code, %d attempts left", e.AttemptsLeft)
}

// loginCodeBound is the number of codes: 10^loginCodeDigits.
var loginCodeBound = new(big.Int).Exp(big.NewInt(10), big.NewInt(loginCodeDigits), nil)

// newLoginCode draws a code of loginCodeDigits digits with crypto/rand, among all of them.
func newLoginCode() (string, error) {
	n, err := rand.Int(rand.Reader, loginCodeBound)
	if err != nil {
		return "", fmt.Errorf("draw a login code: %w", err)
	}
	return fmt.Sprintf("%0*d", loginCodeDigits, n.Int64()), nil
}

// LoginCode is a login code waiting to be entered, real or decoy. Only the hash of a
// real code is kept; a decoy code has none, so that no input matches it. Both keep the
// hash of the token of their request, set in a cookie of the browser that requested the
// code: an attempt without it consumes nothing (ENF-01; plan revue-securite, D5).
type LoginCode struct {
	Hash         []byte
	RequestHash  []byte
	ExpiresAt    time.Time
	AttemptsLeft int
}

// newRealCode returns the state of a code issued now for the request of this token.
func newRealCode(code, requestToken string, now time.Time) LoginCode {
	return LoginCode{Hash: hash(code), RequestHash: hash(requestToken), ExpiresAt: now.Add(LoginCodeValidity), AttemptsLeft: LoginCodeAttempts}
}

// NewDecoyCode returns a decoy code issued now for the request of this token: the deadline
// and attempts of a real code, but nothing can match it.
func NewDecoyCode(requestToken string, now time.Time) LoginCode {
	return LoginCode{RequestHash: hash(requestToken), ExpiresAt: now.Add(LoginCodeValidity), AttemptsLeft: LoginCodeAttempts}
}

// Check tries the code entered at now, with the token of the request presented by the
// browser. It returns nil when the code is accepted; ErrNewCodeNeeded when it expired, has
// no attempt left, or when the token is not the one of the request, which consumes no
// attempt; *IncorrectCodeError otherwise. It also returns the state of the code after the
// attempt, with keep false when it must be deleted (accepted, expired, or invalidated by
// the last attempt).
func (c LoginCode) Check(entered, requestToken string, now time.Time) (after LoginCode, keep bool, err error) {
	if !now.Before(c.ExpiresAt) || c.AttemptsLeft <= 0 {
		return c, false, ErrNewCodeNeeded
	}
	if subtle.ConstantTimeCompare(hash(requestToken), c.RequestHash) != 1 {
		return c, true, ErrNewCodeNeeded
	}
	// A decoy code has no hash: compare with a hash of the same length, which never matches
	// a hash of the input, so that both cases take the same time.
	want := c.Hash
	if want == nil {
		want = make([]byte, sha256.Size)
	}
	if subtle.ConstantTimeCompare(hash(entered), want) == 1 && c.Hash != nil {
		return c, false, nil
	}
	c.AttemptsLeft--
	return c, c.AttemptsLeft > 0, &IncorrectCodeError{AttemptsLeft: c.AttemptsLeft}
}

// newToken returns an opaque token of 32 random bytes: a session token, for the session
// cookie, or the token of a code request, for its cookie.
func newToken() (string, error) {
	b := make([]byte, tokenLen)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("draw a token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// sessionExpiry returns the expiry of a session used at now.
func sessionExpiry(now time.Time) time.Time {
	return now.Add(SessionLifetime)
}
