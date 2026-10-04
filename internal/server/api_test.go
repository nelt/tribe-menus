package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/nelt/tribe-menus/internal/mail"
	"github.com/nelt/tribe-menus/internal/storage"
	"github.com/nelt/tribe-menus/internal/tribe"
)

type apiFixture struct {
	handler  http.Handler
	outbox   *mail.Outbox
	recorder *mail.Recorder
}

// newAPIFixture serves the tribes "martin" (Alice) and "durand" (David).
func newAPIFixture(t *testing.T) *apiFixture {
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
	now := time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC)
	for _, tr := range []struct{ slug, name, email string }{
		{"martin", "Les Martin", "alice@exemple.fr"},
		{"durand", "Les Durand", "david@exemple.fr"},
	} {
		err := st.CreateTribe(ctx, tr.slug, tr.name, func(ctx context.Context, db *sql.DB) error {
			return tribe.NewStore(db).Initialize(ctx, tr.name, tribe.Email(tr.email), "", now)
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	f := &apiFixture{recorder: &mail.Recorder{}}
	logger := slog.New(slog.DiscardHandler)
	f.outbox = mail.NewOutbox(f.recorder, tribe.LoginCodeValidity, logger)
	login := &tribe.Login{RateLimit: tribe.NewRateLimitDB(st.RateLimit()), Mailer: f.outbox, Now: func() time.Time { return now }}
	f.handler = newHandler(t, Config{Web: webFS(), Tribes: st, Login: login})
	return f
}

// do sends a request as the browser of the application: same origin, with the cookies given.
func (f *apiFixture) do(method, target, body string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "https://localhost"+target, strings.NewReader(body))
	if method != http.MethodGet {
		req.Header.Set("Origin", "https://localhost")
		req.Header.Set("Content-Type", "application/json")
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, req)
	return rec
}

var sixDigits = regexp.MustCompile(`\b[0-9]{6}\b`)

// signIn signs in to the tribe and returns the session cookie.
func (f *apiFixture) signIn(t *testing.T, slug, email string) *http.Cookie {
	t.Helper()
	if rec := f.do("POST", "/tribes/"+slug+"/api/login-codes", `{"email":"`+email+`"}`); rec.Code != http.StatusAccepted {
		t.Fatalf("login-codes: status %d", rec.Code)
	}
	f.outbox.Wait()
	msgs := f.recorder.Messages()
	code := sixDigits.FindString(msgs[len(msgs)-1].Body)
	rec := f.do("POST", "/tribes/"+slug+"/api/sessions", `{"email":"`+email+`","code":"`+code+`"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("sessions: status %d, body %s", rec.Code, rec.Body)
	}
	return sessionCookieOf(t, rec)
}

func sessionCookieOf(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookie {
			return c
		}
	}
	t.Fatalf("no session cookie in %v", rec.Header())
	return nil
}

func TestSignInAndOut(t *testing.T) {
	f := newAPIFixture(t)
	cookie := f.signIn(t, "martin", "alice@exemple.fr")

	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode ||
		cookie.Path != "/tribes/martin/" || cookie.MaxAge != 90*24*60*60 || len(cookie.Value) < 40 {
		t.Errorf("cookie = %+v", cookie)
	}

	rec := f.do("GET", "/tribes/martin/api/session", "", cookie)
	var got sessionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("GET session: status %d, body %s", rec.Code, rec.Body)
	}
	if got.Tribe.Name != "Les Martin" || got.Member.Email != "alice@exemple.fr" {
		t.Errorf("GET session = %+v", got)
	}
	if renewed := sessionCookieOf(t, rec); renewed.Value != cookie.Value || renewed.MaxAge != cookie.MaxAge {
		t.Errorf("renewed cookie = %+v", renewed)
	}

	// The session of martin opens nothing in durand.
	if rec := f.do("GET", "/tribes/durand/api/session", "", cookie); rec.Code != http.StatusUnauthorized || strings.Contains(rec.Body.String(), "Durand") {
		t.Errorf("GET durand session with martin's cookie: status %d, body %s", rec.Code, rec.Body)
	}

	if rec := f.do("DELETE", "/tribes/martin/api/session", "", cookie); rec.Code != http.StatusNoContent || sessionCookieOf(t, rec).MaxAge >= 0 {
		t.Errorf("DELETE session: status %d, cookies %v", rec.Code, rec.Result().Cookies())
	}
	if rec := f.do("GET", "/tribes/martin/api/session", "", cookie); rec.Code != http.StatusUnauthorized {
		t.Errorf("GET session after sign out: status %d", rec.Code)
	}
}

func TestAPIErrors(t *testing.T) {
	f := newAPIFixture(t)
	cases := []struct {
		name       string
		method     string
		target     string
		body       string
		wantStatus int
		wantBody   string
	}{
		{name: "malformed address", method: "POST", target: "/tribes/martin/api/login-codes", body: `{"email":"alice"}`, wantStatus: 400, wantBody: `{"error":"invalid_email"}`},
		{name: "malformed body", method: "POST", target: "/tribes/martin/api/login-codes", body: `{`, wantStatus: 400, wantBody: `{"error":"bad_request"}`},
		{name: "code never requested", method: "POST", target: "/tribes/martin/api/sessions", body: `{"email":"alice@exemple.fr","code":"123456"}`, wantStatus: 400, wantBody: `{"error":"new_code_needed"}`},
		{name: "no session", method: "GET", target: "/tribes/martin/api/session", wantStatus: 401, wantBody: `{"error":"no_session"}`},
		{name: "sign out without session", method: "DELETE", target: "/tribes/martin/api/session", wantStatus: 401, wantBody: `{"error":"no_session"}`},
		{name: "unknown API path", method: "GET", target: "/tribes/martin/api/nope", wantStatus: 404, wantBody: `{"error":"not_found"}`},
		{name: "code request for a slug no tribe can have", method: "POST", target: "/tribes/" + strings.Repeat("X", 5000) + "/api/login-codes", body: `{"email":"alice@exemple.fr"}`, wantStatus: 202},
		{name: "wrong method", method: "PUT", target: "/tribes/martin/api/session", wantStatus: 405},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := f.do(tc.method, tc.target, tc.body)
			if rec.Code != tc.wantStatus || strings.TrimSpace(rec.Body.String()) != tc.wantBody && tc.wantBody != "" {
				t.Errorf("status %d, body %s; want %d, %s", rec.Code, rec.Body, tc.wantStatus, tc.wantBody)
			}
			if tc.wantStatus != 405 && rec.Header().Get("Cache-Control") != "no-store" {
				t.Errorf("Cache-Control = %q", rec.Header().Get("Cache-Control"))
			}
		})
	}
}

func TestInvalidCookieIsCleared(t *testing.T) {
	f := newAPIFixture(t)
	rec := f.do("GET", "/tribes/martin/api/session", "", &http.Cookie{Name: sessionCookie, Value: "forged"})
	if rec.Code != http.StatusUnauthorized || sessionCookieOf(t, rec).MaxAge >= 0 {
		t.Errorf("status %d, cookies %v; want 401 and the cookie cleared", rec.Code, rec.Result().Cookies())
	}
}

func TestCrossOriginRefused(t *testing.T) {
	f := newAPIFixture(t)
	for _, origin := range []string{"https://evil.example", "null"} {
		req := httptest.NewRequest("POST", "https://localhost/tribes/martin/api/login-codes", strings.NewReader(`{"email":"alice@exemple.fr"}`))
		req.Header.Set("Origin", origin)
		rec := httptest.NewRecorder()
		f.handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("Origin %s: status %d, want 403", origin, rec.Code)
		}
	}
	f.outbox.Wait()
	if n := len(f.recorder.Messages()); n != 0 {
		t.Errorf("%d messages sent", n)
	}
}

// TestUnknownTribeAnswersAlike: without session, an unknown tribe answers exactly as an
// existing one, request after request (ENF-02).
func TestUnknownTribeAnswersAlike(t *testing.T) {
	f := newAPIFixture(t)
	steps := []struct{ method, path, body string }{
		{"GET", "/", ""},
		{"GET", "/planning", ""},
		{"GET", "/api/session", ""},
		{"POST", "/api/sessions", `{"email":"alice@exemple.fr","code":"000000"}`},
		{"POST", "/api/login-codes", `{"email":"alice@exemple.fr"}`},
		{"POST", "/api/sessions", `{"email":"alice@exemple.fr","code":"000000"}`},
		{"POST", "/api/sessions", `{"email":"alice@exemple.fr","code":"000000"}`},
		{"POST", "/api/sessions", `{"email":"alice@exemple.fr","code":"000000"}`},
		{"POST", "/api/sessions", `{"email":"alice@exemple.fr","code":"000000"}`},
		{"DELETE", "/api/session", ""},
	}
	answer := func(slug string, i int) string {
		s := steps[i]
		rec := f.do(s.method, "/tribes/"+slug+s.path, s.body)
		body, _ := io.ReadAll(rec.Body)
		h := rec.Header()
		return strings.Join([]string{
			http.StatusText(rec.Code), h.Get("Content-Type"), h.Get("Cache-Control"),
			strings.ReplaceAll(string(body), slug, "<slug>"),
		}, " | ")
	}
	for i, s := range steps {
		existing, unknown := answer("martin", i), answer("dupont", i)
		if existing != unknown {
			t.Errorf("%s %s: martin %q, unknown tribe %q", s.method, s.path, existing, unknown)
		}
		if strings.Contains(existing, "Martin") {
			t.Errorf("%s %s reveals the tribe name: %q", s.method, s.path, existing)
		}
	}
}
