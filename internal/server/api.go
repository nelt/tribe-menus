package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/nelt/tribe-menus/internal/storage"
	"github.com/nelt/tribe-menus/internal/tribe"
)

const (
	sessionCookie = "session"
	// codeRequestCookie holds the token of the last code request of the browser: a code is
	// entered only from the browser that requested it (ENF-01; plan revue-securite, D5).
	codeRequestCookie = "code_request"
	maxBodyBytes      = 4 << 10
)

// Tribes gives the database of a tribe by its slug, or storage.ErrUnknownTribe.
type Tribes interface {
	Tribe(ctx context.Context, slug string) (*sql.DB, error)
}

// Error codes of the API, in the "error" field of an error response.
const (
	errBadRequest      = "bad_request"
	errInvalidEmail    = "invalid_email"
	errTooManyRequests = "too_many_requests"
	errIncorrectCode   = "incorrect_code"
	errNewCodeNeeded   = "new_code_needed"
	errNoSession       = "no_session"
	errNotFound        = "not_found"
)

type errorResponse struct {
	Error        string `json:"error"`
	AttemptsLeft *int   `json:"attemptsLeft,omitempty"`
}

type sessionResponse struct {
	Tribe  tribeResponse  `json:"tribe"`
	Member memberResponse `json:"member"`
}

type tribeResponse struct {
	Name string `json:"name"`
}

type memberResponse struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
}

type contextKey int

const (
	tribeKey contextKey = iota
	sessionKey
)

// api serves /tribes/<slug>/api/: the login and the session (ENF-01).
type api struct {
	server *server
	tribes Tribes
	login  *tribe.Login
}

// routes registers the API. Each route resolves the tribe of the URL, then, past the login,
// the session in the database of this tribe (ADR 0003, point 3); requests other than GET
// must come from the same origin (ADR 0001).
func (a *api) routes(mux *http.ServeMux) {
	cop := http.NewCrossOriginProtection()
	handle := func(pattern string, h http.Handler) {
		mux.Handle(pattern, withNoStore(cop.Handler(a.withTribe(h))))
	}
	handle("POST /tribes/{tribe}/api/login-codes", http.HandlerFunc(a.requestCode))
	handle("POST /tribes/{tribe}/api/sessions", http.HandlerFunc(a.openSession))
	handle("GET /tribes/{tribe}/api/session", a.withSession(a.getSession))
	handle("DELETE /tribes/{tribe}/api/session", a.withSession(a.deleteSession))
	// Other methods on an API path get 405 from the mux.
	mux.Handle("GET /tribes/{tribe}/api/{rest...}", withNoStore(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: errNotFound})
	})))
}

func withNoStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// withTribe resolves the tribe of the URL: its store, or nil for an unknown tribe, which
// the handlers treat exactly as an existing tribe without session (ENF-02).
func (a *api) withTribe(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var store *tribe.Store
		db, err := a.tribes.Tribe(r.Context(), r.PathValue("tribe"))
		switch {
		case err == nil:
			store = tribe.NewStore(db)
		case !errors.Is(err, storage.ErrUnknownTribe):
			a.server.fail(w, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), tribeKey, store)))
	})
}

// withSession resolves the session of the cookie in the database of the tribe, and
// renews the cookie with the sliding expiry. A cookie without valid session is cleared.
func (a *api) withSession(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookie)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		session, err := a.login.Session(r.Context(), tribeOf(r), cookie.Value)
		if errors.Is(err, tribe.ErrNoSession) {
			clearSessionCookie(w, r)
			next.ServeHTTP(w, r)
			return
		}
		if err != nil {
			a.server.fail(w, err)
			return
		}
		setSessionCookie(w, r, cookie.Value, session.ExpiresAt)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), sessionKey, session)))
	})
}

func tribeOf(r *http.Request) *tribe.Store {
	store, _ := r.Context().Value(tribeKey).(*tribe.Store)
	return store
}

func sessionOf(r *http.Request) (tribe.Session, bool) {
	session, ok := r.Context().Value(sessionKey).(tribe.Session)
	return session, ok
}

// requestCode: POST login-codes {email}. Always the same answer, whether a code is sent or
// not, except for a malformed address or too many requests.
func (a *api) requestCode(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	token, err := a.login.RequestCode(r.Context(), r.PathValue("tribe"), tribeOf(r), body.Email, a.server.clientIP(r))
	switch {
	case err == nil:
		setCodeRequestCookie(w, r, token, tribe.LoginCodeValidity)
		w.WriteHeader(http.StatusAccepted)
	case errors.Is(err, tribe.ErrInvalidEmail):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: errInvalidEmail})
	case errors.Is(err, tribe.ErrTooManyRequests):
		writeJSON(w, http.StatusTooManyRequests, errorResponse{Error: errTooManyRequests})
	default:
		a.server.fail(w, err)
	}
}

// openSession: POST sessions {email, code, installedApp}, with the cookie of the code
// request. Opens the session and sets its cookie, or tells an incorrect code with the
// attempts left, or a new code to request: without the cookie of the request, an attempt
// consumes nothing.
func (a *api) openSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email        string `json:"email"`
		Code         string `json:"code"`
		InstalledApp bool   `json:"installedApp"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	store := tribeOf(r)
	device := tribe.DetectDevice(r.UserAgent(), body.InstalledApp)
	var requestToken string
	if cookie, err := r.Cookie(codeRequestCookie); err == nil {
		requestToken = cookie.Value
	}
	session, token, err := a.login.OpenSession(r.Context(), r.PathValue("tribe"), store, body.Email, body.Code, requestToken, device)
	var incorrect *tribe.IncorrectCodeError
	switch {
	case err == nil:
		setSessionCookie(w, r, token, session.ExpiresAt)
		setCodeRequestCookie(w, r, "", -1)
		a.writeSession(w, r, http.StatusCreated, session)
	case errors.Is(err, tribe.ErrInvalidEmail):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: errInvalidEmail})
	case errors.As(err, &incorrect):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: errIncorrectCode, AttemptsLeft: &incorrect.AttemptsLeft})
	case errors.Is(err, tribe.ErrNewCodeNeeded):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: errNewCodeNeeded})
	default:
		a.server.fail(w, err)
	}
}

// getSession: GET session. The tribe and the member of the session, or 401.
func (a *api) getSession(w http.ResponseWriter, r *http.Request) {
	session, ok := sessionOf(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: errNoSession})
		return
	}
	a.writeSession(w, r, http.StatusOK, session)
}

// deleteSession: DELETE session. Signs out.
func (a *api) deleteSession(w http.ResponseWriter, r *http.Request) {
	session, ok := sessionOf(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: errNoSession})
		return
	}
	if err := a.login.SignOut(r.Context(), tribeOf(r), session); err != nil {
		a.server.fail(w, err)
		return
	}
	clearSessionCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

// writeSession answers with the tribe and the member of a session. The name of the tribe
// is read in its own database, never in the registry (D9).
func (a *api) writeSession(w http.ResponseWriter, r *http.Request, status int, session tribe.Session) {
	name, err := tribeOf(r).Name(r.Context())
	if err != nil {
		a.server.fail(w, err)
		return
	}
	writeJSON(w, status, sessionResponse{
		Tribe:  tribeResponse{Name: name},
		Member: memberResponse{Email: string(session.Member.Email), DisplayName: session.Member.DisplayName},
	})
}

// setSessionCookie sets the session cookie of the tribe of the request (ADR 0001, 0006).
func setSessionCookie(w http.ResponseWriter, r *http.Request, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     tribeBase(r.PathValue("tribe")),
		MaxAge:   int(tribe.SessionLifetime / time.Second),
		Expires:  expires,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

// setCodeRequestCookie sets the cookie of a code request of the tribe, valid as long as the
// code; a negative validity deletes it.
func setCodeRequestCookie(w http.ResponseWriter, r *http.Request, token string, validity time.Duration) {
	maxAge := int(validity / time.Second)
	if validity < 0 {
		maxAge = -1
	}
	http.SetCookie(w, &http.Cookie{
		Name:     codeRequestCookie,
		Value:    token,
		Path:     tribeBase(r.PathValue("tribe")),
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

// clearSessionCookie asks the browser to delete the cookie, replacing a renewal already set.
func clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	w.Header().Del("Set-Cookie")
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Path:     tribeBase(r.PathValue("tribe")),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

// unknownClientIP is the IP of every request behind the proxy without a readable
// X-Forwarded-For: they share one limit by IP, stricter, never looser.
const unknownClientIP = "unknown"

// clientIP is the IP of the client. Behind the proxy, it is the last address of
// X-Forwarded-For, the one Caddy writes: what comes before comes from the client (ADR 0006,
// point 7; plan production, D3). Otherwise, the header is ignored and the IP is the one of
// the TCP connection (point 8).
func (s *server) clientIP(r *http.Request) string {
	if !s.behindProxy {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return r.RemoteAddr
		}
		return host
	}
	values := r.Header.Values("X-Forwarded-For")
	if len(values) > 0 {
		last := values[len(values)-1]
		if i := strings.LastIndexByte(last, ','); i >= 0 {
			last = last[i+1:]
		}
		if addr, err := netip.ParseAddr(strings.TrimSpace(last)); err == nil {
			return addr.String()
		}
	}
	s.logger.Warn("client IP unknown behind the proxy: X-Forwarded-For missing or unreadable")
	return unknownClientIP
}

// readJSON decodes the body of the request, or answers 400 and returns false.
func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	if err := dec.Decode(v); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: errBadRequest})
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
