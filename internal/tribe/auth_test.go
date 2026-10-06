package tribe

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nelt/tribe-menus/internal/storage"
)

type sentCode struct{ to, code string }

type fakeMailer struct{ sent []sentCode }

func (m *fakeMailer) SendLoginCode(to, code string) { m.sent = append(m.sent, sentCode{to, code}) }

func (m *fakeMailer) last(t *testing.T, to string) string {
	t.Helper()
	for i := len(m.sent) - 1; i >= 0; i-- {
		if m.sent[i].to == to {
			return m.sent[i].code
		}
	}
	t.Fatalf("no code sent to %s", to)
	return ""
}

type loginFixture struct {
	store *storage.Store
	login *Login
	// token is the token of the last code request, as the browser keeps it in a cookie.
	token  string
	mailer *fakeMailer
	now    time.Time
	martin *Store
	alice  Member
}

func (f *loginFixture) advance(d time.Duration) { f.now = f.now.Add(d) }

// requestCode asks for a code as the browser of the fixture, which keeps the token of the
// request.
func (f *loginFixture) requestCode(ctx context.Context, slug string, t *Store, email, ip string) error {
	token, err := f.login.RequestCode(ctx, slug, t, email, ip)
	if err == nil {
		f.token = token
	}
	return err
}

// newLoginFixture creates the tribe "martin" with Alice, active, and Bruno, revoked.
func newLoginFixture(t *testing.T) *loginFixture {
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
	f := &loginFixture{store: st, mailer: &fakeMailer{}, now: time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC)}
	f.login = &Login{RateLimit: NewRateLimitDB(st.RateLimit()), Mailer: f.mailer, Now: func() time.Time { return f.now }}

	err = st.CreateTribe(ctx, "martin", "Les Martin", func(ctx context.Context, db *sql.DB) error {
		return NewStore(db).Initialize(ctx, "Les Martin", "alice@exemple.fr", "Alice", f.now)
	})
	if err != nil {
		t.Fatal(err)
	}
	db, err := st.Tribe(ctx, "martin")
	if err != nil {
		t.Fatal(err)
	}
	f.martin = NewStore(db)
	if f.alice, err = f.martin.Member(ctx, "alice@exemple.fr"); err != nil {
		t.Fatal(err)
	}
	bruno, err := f.martin.AddMember(ctx, "bruno@exemple.fr", "Bruno", f.alice.ID, f.now)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.martin.RevokeMember(ctx, bruno.ID, f.alice.ID, f.now); err != nil {
		t.Fatal(err)
	}
	return f
}

// tribeOf returns the store of the tribe "martin", nil for any other slug.
func (f *loginFixture) tribeOf(slug string) *Store {
	if slug == "martin" {
		return f.martin
	}
	return nil
}

// addMembers adds n active members to the tribe "martin": membre-0@exemple.fr…
func (f *loginFixture) addMembers(t *testing.T, n int) {
	t.Helper()
	for i := range n {
		if _, err := f.martin.AddMember(context.Background(), Email(fmt.Sprintf("membre-%d@exemple.fr", i)), "", f.alice.ID, f.now); err != nil {
			t.Fatal(err)
		}
	}
}

func (f *loginFixture) signIn(t *testing.T, email string) (Session, string) {
	t.Helper()
	ctx := context.Background()
	if err := f.requestCode(ctx, "martin", f.martin, email, "192.0.2.1"); err != nil {
		t.Fatal(err)
	}
	session, token, err := f.login.OpenSession(ctx, "martin", f.martin, email, f.mailer.last(t, email), f.token, Device{Type: Phone})
	if err != nil {
		t.Fatal(err)
	}
	return session, token
}

func TestRequestCode(t *testing.T) {
	cases := []struct {
		name     string
		slug     string
		unknown  bool // the tribe does not exist
		email    string
		wantErr  error
		wantSent bool
	}{
		{name: "active member", slug: "martin", email: " Alice@Exemple.fr", wantSent: true},
		{name: "unknown address", slug: "martin", email: "inconnu@exemple.fr"},
		{name: "revoked member", slug: "martin", email: "bruno@exemple.fr"},
		{name: "unknown tribe", slug: "dupont", unknown: true, email: "alice@exemple.fr"},
		{name: "malformed address", slug: "martin", email: "alice", wantErr: ErrInvalidEmail},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newLoginFixture(t)
			tribe := f.martin
			if tc.unknown {
				tribe = nil
			}
			err := f.requestCode(context.Background(), tc.slug, tribe, tc.email, "192.0.2.1")
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
			if sent := len(f.mailer.sent) == 1; sent != tc.wantSent {
				t.Errorf("sent = %v, want %v", f.mailer.sent, tc.wantSent)
			}
		})
	}
}

func TestRateLimits(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name  string
		prior func(i int) (slug, email, ip string)
		n     int
		slug  string
		email string
		ip    string
		// members are added to the tribe "martin" first: membre-0@exemple.fr…
		members int
	}{
		{
			name:  "by address in a tribe",
			prior: func(int) (string, string, string) { return "martin", "alice@exemple.fr", "192.0.2.1" },
			n:     EmailRequestLimit, slug: "martin", email: "alice@exemple.fr", ip: "192.0.2.2",
		},
		{
			name:  "by unknown address",
			prior: func(int) (string, string, string) { return "martin", "inconnu@exemple.fr", "192.0.2.1" },
			n:     EmailRequestLimit, slug: "martin", email: "inconnu@exemple.fr", ip: "192.0.2.2",
		},
		{
			name: "by IP",
			prior: func(i int) (string, string, string) {
				return "dupont", string(rune('a'+i)) + "@exemple.fr", "192.0.2.1"
			},
			n: IPRequestLimit, slug: "martin", email: "alice@exemple.fr", ip: "192.0.2.1",
		},
		{
			name: "by tribe, for its members",
			prior: func(i int) (string, string, string) {
				return "martin", fmt.Sprintf("membre-%d@exemple.fr", i), fmt.Sprintf("198.51.100.%d", i%10)
			},
			members: TribeRequestLimit,
			n:       TribeRequestLimit, slug: "martin", email: "alice@exemple.fr", ip: "192.0.2.9",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newLoginFixture(t)
			f.addMembers(t, tc.members)
			tribeOf := f.tribeOf
			for i := range tc.n {
				slug, email, ip := tc.prior(i)
				if err := f.requestCode(ctx, slug, tribeOf(slug), email, ip); err != nil {
					t.Fatalf("request %d: %v", i+1, err)
				}
				f.advance(time.Second)
			}
			sent := len(f.mailer.sent)
			err := f.requestCode(ctx, tc.slug, tribeOf(tc.slug), tc.email, tc.ip)
			if !errors.Is(err, ErrTooManyRequests) {
				t.Fatalf("error = %v, want ErrTooManyRequests", err)
			}
			if len(f.mailer.sent) != sent {
				t.Error("a code was sent beyond the limit")
			}

			// The windows end: a new request is accepted.
			f.advance(time.Hour)
			if err := f.requestCode(ctx, tc.slug, tribeOf(tc.slug), tc.email, tc.ip); err != nil {
				t.Errorf("after the window: %v", err)
			}
		})
	}
}

// TestTribeLimitCountsMembersOnly: requests for addresses that are not active members of the
// tribe never reach the limit by tribe, whether the tribe exists or not (D2).
func TestTribeLimitCountsMembersOnly(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name  string
		prior string // slug of the prior requests, for addresses that are not members
		slug  string
		email string
	}{
		{name: "addresses that are not members", prior: "martin", slug: "martin", email: "alice@exemple.fr"},
		{name: "revoked member", prior: "martin", slug: "martin", email: "alice@exemple.fr"},
		{name: "unknown tribe", prior: "dupont", slug: "dupont", email: "alice@exemple.fr"},
		{name: "malformed slug", prior: "Pas un identifiant", slug: "Pas un identifiant", email: "alice@exemple.fr"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newLoginFixture(t)
			for i := range 2 * TribeRequestLimit {
				email := fmt.Sprintf("personne-%d@exemple.fr", i)
				if tc.name == "revoked member" {
					email = "bruno@exemple.fr"
					f.advance(5 * time.Minute) // 3 requests a quarter of an hour
				}
				ip := fmt.Sprintf("198.51.100.%d", i%(IPRequestLimit-1))
				if err := f.requestCode(ctx, tc.prior, f.tribeOf(tc.prior), email, ip); err != nil {
					t.Fatalf("request %d: %v", i+1, err)
				}
				f.advance(time.Second)
			}
			if err := f.requestCode(ctx, tc.slug, f.tribeOf(tc.slug), tc.email, "192.0.2.9"); err != nil {
				t.Errorf("request for %s: %v", tc.email, err)
			}
		})
	}
}

// TestMalformedSlugIsNotStored: a slug no tribe can have gets the answers of an unknown
// tribe, and the rate limit database keeps neither its text nor its length (ADR 0021).
func TestMalformedSlugIsNotStored(t *testing.T) {
	ctx := context.Background()
	f := newLoginFixture(t)
	slug := "alice@exemple.fr " + strings.Repeat("x", 10_000)

	if err := f.requestCode(ctx, slug, nil, "alice@exemple.fr", "192.0.2.1"); err != nil {
		t.Fatalf("request: %v", err)
	}
	if len(f.mailer.sent) != 0 {
		t.Errorf("sent = %v, want nothing", f.mailer.sent)
	}
	var incorrect *IncorrectCodeError
	if _, _, err := f.login.OpenSession(ctx, slug, nil, "alice@exemple.fr", "12345678", f.token, Device{}); !errors.As(err, &incorrect) || incorrect.AttemptsLeft != LoginCodeAttempts-1 {
		t.Errorf("open session: %v, want an incorrect code with %d attempts left", err, LoginCodeAttempts-1)
	}

	for _, table := range []string{"code_requests", "decoy_codes"} {
		rows, err := f.store.RateLimit().QueryContext(ctx, "SELECT * FROM "+table)
		if err != nil {
			t.Fatal(err)
		}
		columns, err := rows.Columns()
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for rows.Next() {
			n++
			values := make([]any, len(columns))
			pointers := make([]any, len(columns))
			for i := range values {
				pointers[i] = &values[i]
			}
			if err := rows.Scan(pointers...); err != nil {
				t.Fatal(err)
			}
			// Hashes of 32 bytes and dates only.
			for i, v := range values {
				b, isBytes := v.([]byte)
				text, isText := v.(string)
				if isBytes && len(b) != 32 || isText && len(text) != len(timeFormat) {
					t.Errorf("%s.%s = %q: neither a hash nor a date", table, columns[i], v)
				}
			}
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		_ = rows.Close()
		if n != 1 {
			t.Errorf("%d rows in %s, want 1", n, table)
		}
	}
}

func TestOpenSession(t *testing.T) {
	ctx := context.Background()
	f := newLoginFixture(t)
	device := Device{Type: Phone, OS: "iOS", Browser: "Safari", InstalledApp: true}

	if err := f.requestCode(ctx, "martin", f.martin, "alice@exemple.fr", "192.0.2.1"); err != nil {
		t.Fatal(err)
	}
	code := f.mailer.last(t, "alice@exemple.fr")
	wrong := "00000000"
	if code == wrong {
		wrong = "111111"
	}
	_, _, err := f.login.OpenSession(ctx, "martin", f.martin, "alice@exemple.fr", wrong, f.token, device)
	var incorrect *IncorrectCodeError
	if !errors.As(err, &incorrect) || incorrect.AttemptsLeft != 2 {
		t.Fatalf("wrong code: error = %v, want 2 attempts left", err)
	}

	session, token, err := f.login.OpenSession(ctx, "martin", f.martin, "ALICE@exemple.fr", code, f.token, device)
	if err != nil {
		t.Fatal(err)
	}
	if session.Member.ID != f.alice.ID || session.Device != device || !session.ExpiresAt.Equal(f.now.Add(SessionLifetime)) {
		t.Errorf("session = %+v", session)
	}

	// The code is used once.
	if _, _, err := f.login.OpenSession(ctx, "martin", f.martin, "alice@exemple.fr", code, f.token, device); !errors.Is(err, ErrNewCodeNeeded) {
		t.Errorf("code used twice: error = %v, want ErrNewCodeNeeded", err)
	}

	// The token resumes the session, which slides.
	f.advance(80 * 24 * time.Hour)
	resumed, err := f.login.Session(ctx, f.martin, token)
	if err != nil || resumed.ID != session.ID || !resumed.ExpiresAt.Equal(f.now.Add(SessionLifetime)) {
		t.Errorf("Session = %+v, %v; want expiry 90 days from now", resumed, err)
	}

	entries, err := f.martin.AuditLog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	last := entries[len(entries)-1]
	if last.Operation != SessionOpened || last.SessionID != session.ID || last.Device != device || last.AuthorID != f.alice.ID {
		t.Errorf("last audit entry = %+v, want the session opened", last)
	}

	// Sign out.
	if err := f.login.SignOut(ctx, f.martin, resumed); err != nil {
		t.Fatal(err)
	}
	if _, err := f.login.Session(ctx, f.martin, token); !errors.Is(err, ErrNoSession) {
		t.Errorf("after sign out: error = %v, want ErrNoSession", err)
	}
	entries, _ = f.martin.AuditLog(ctx)
	if last := entries[len(entries)-1]; last.Operation != SignedOut || last.SessionID != session.ID || last.Device != device {
		t.Errorf("last audit entry = %+v, want signed out", last)
	}
}

func TestSessionExpires(t *testing.T) {
	ctx := context.Background()
	f := newLoginFixture(t)
	_, token := f.signIn(t, "alice@exemple.fr")

	f.advance(SessionLifetime - time.Millisecond)
	if _, err := f.login.Session(ctx, f.martin, token); err != nil {
		t.Fatalf("just before expiry: %v", err)
	}
	f.advance(SessionLifetime)
	if _, err := f.login.Session(ctx, f.martin, token); !errors.Is(err, ErrNoSession) {
		t.Errorf("90 days unused: error = %v, want ErrNoSession", err)
	}
	for _, tc := range []struct {
		name  string
		tribe *Store
		token string
	}{
		{name: "unknown tribe", token: token},
		{name: "no token", tribe: f.martin},
		{name: "unknown token", tribe: f.martin, token: "x"},
	} {
		if _, err := f.login.Session(ctx, tc.tribe, tc.token); !errors.Is(err, ErrNoSession) {
			t.Errorf("%s: error = %v, want ErrNoSession", tc.name, err)
		}
	}
}

// TestSameAnswers: whatever the address or the tribe, the answers are those of an active
// member, attempt after attempt (ENF-02).
func TestSameAnswers(t *testing.T) {
	ctx := context.Background()
	f := newLoginFixture(t)
	targets := []struct {
		name  string
		slug  string
		tribe *Store
		email string
	}{
		{name: "active member", slug: "martin", tribe: f.martin, email: "alice@exemple.fr"},
		{name: "unknown address", slug: "martin", tribe: f.martin, email: "inconnu@exemple.fr"},
		{name: "revoked member", slug: "martin", tribe: f.martin, email: "bruno@exemple.fr"},
		{name: "unknown tribe", slug: "dupont", email: "alice@exemple.fr"},
	}
	var answers [][]string
	for _, target := range targets {
		var got []string
		_, _, err := f.login.OpenSession(ctx, target.slug, target.tribe, target.email, "000001", f.token, Device{})
		got = append(got, errorString(err))
		if err := f.requestCode(ctx, target.slug, target.tribe, target.email, "192.0.2.1"); err != nil {
			t.Fatal(err)
		}
		for range LoginCodeAttempts + 1 {
			_, _, err := f.login.OpenSession(ctx, target.slug, target.tribe, target.email, "000001", f.token, Device{})
			got = append(got, errorString(err))
		}
		answers = append(answers, got)
	}
	for i, got := range answers[1:] {
		for j := range got {
			if got[j] != answers[0][j] {
				t.Errorf("%s: answers %q, want those of an active member %q", targets[i+1].name, got, answers[0])
				break
			}
		}
	}
}

// TestAttemptsFromAnotherBrowser: an attempt without the token of the code request, or with
// the token of another request, consumes nothing, and gets the same answer whatever the
// address or the tribe (D5).
func TestAttemptsFromAnotherBrowser(t *testing.T) {
	ctx := context.Background()
	targets := []struct {
		name    string
		slug    string
		unknown bool
		email   string
	}{
		{name: "active member", slug: "martin", email: "alice@exemple.fr"},
		{name: "unknown address", slug: "martin", email: "inconnu@exemple.fr"},
		{name: "revoked member", slug: "martin", email: "bruno@exemple.fr"},
		{name: "unknown tribe", slug: "dupont", unknown: true, email: "alice@exemple.fr"},
	}
	for _, target := range targets {
		t.Run(target.name, func(t *testing.T) {
			f := newLoginFixture(t)
			store := f.martin
			if target.unknown {
				store = nil
			}
			// Another browser requests a code for chloe: its token is valid, for another code.
			if err := f.requestCode(ctx, target.slug, store, "chloe@exemple.fr", "192.0.2.2"); err != nil {
				t.Fatal(err)
			}
			other := f.token
			if err := f.requestCode(ctx, target.slug, store, target.email, "192.0.2.1"); err != nil {
				t.Fatal(err)
			}
			for _, token := range []string{"", other, f.token + "x", other, ""} {
				if _, _, err := f.login.OpenSession(ctx, target.slug, store, target.email, "00000001", token, Device{}); !errors.Is(err, ErrNewCodeNeeded) {
					t.Fatalf("attempt with the token %q: error = %v, want ErrNewCodeNeeded", token, err)
				}
			}
			var incorrect *IncorrectCodeError
			if _, _, err := f.login.OpenSession(ctx, target.slug, store, target.email, "00000001", f.token, Device{}); !errors.As(err, &incorrect) || incorrect.AttemptsLeft != LoginCodeAttempts-1 {
				t.Errorf("attempt from the browser of the request: error = %v, want %d attempts left", err, LoginCodeAttempts-1)
			}
		})
	}
}

func errorString(err error) string {
	if err == nil {
		return "accepted"
	}
	return err.Error()
}

// TestDecoyCodeForEveryRequest: the rate limit database keeps the same rows for a request
// for a member and for another address, decoy code included, so that nothing it keeps
// tells a member's address (ADR 0021, point 4; plan production, D11).
func TestDecoyCodeForEveryRequest(t *testing.T) {
	ctx := context.Background()
	f := newLoginFixture(t)
	tokens := map[string]string{}
	for _, email := range []string{"alice@exemple.fr", "inconnu@exemple.fr"} {
		if err := f.requestCode(ctx, "martin", f.martin, email, "192.0.2.1"); err != nil {
			t.Fatal(err)
		}
		tokens[email] = f.token
	}
	if len(f.mailer.sent) != 1 {
		t.Fatalf("sent = %v, want the code of Alice", f.mailer.sent)
	}

	rows := map[string][]string{}
	for _, email := range []string{"alice@exemple.fr", "inconnu@exemple.fr"} {
		h := emailHash("martin", Email(email))
		var requests int
		if err := f.store.RateLimit().QueryRowContext(ctx, "SELECT count(*) FROM code_requests WHERE email_hash = ?", h).Scan(&requests); err != nil {
			t.Fatal(err)
		}
		var expiresAt string
		var attempts int
		var requestHash []byte
		err := f.store.RateLimit().QueryRowContext(ctx, "SELECT expires_at, attempts_left, request_hash FROM decoy_codes WHERE email_hash = ?", h).Scan(&expiresAt, &attempts, &requestHash)
		if err != nil {
			t.Fatalf("%s: decoy code: %v", email, err)
		}
		rows[email] = []string{fmt.Sprint(requests), expiresAt, fmt.Sprint(attempts), fmt.Sprint(len(requestHash))}
	}
	if a, b := strings.Join(rows["alice@exemple.fr"], " "), strings.Join(rows["inconnu@exemple.fr"], " "); a != b {
		t.Errorf("rate limit rows of a member %q, of another address %q: want the same", a, b)
	}

	// The decoy code of a member is never checked: its real code is.
	if _, _, err := f.login.OpenSession(ctx, "martin", f.martin, "alice@exemple.fr", f.mailer.last(t, "alice@exemple.fr"), tokens["alice@exemple.fr"], Device{}); err != nil {
		t.Errorf("Alice's code: %v", err)
	}
}

// TestRevokedAfterRequest: a member revoked after the request enters codes against its decoy
// code, as an unknown address does.
func TestRevokedAfterRequest(t *testing.T) {
	ctx := context.Background()
	f := newLoginFixture(t)
	chloe, err := f.martin.AddMember(ctx, "chloe@exemple.fr", "Chloé", f.alice.ID, f.now)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.requestCode(ctx, "martin", f.martin, "chloe@exemple.fr", "192.0.2.1"); err != nil {
		t.Fatal(err)
	}
	if err := f.martin.RevokeMember(ctx, chloe.ID, f.alice.ID, f.now); err != nil {
		t.Fatal(err)
	}
	_, _, err = f.login.OpenSession(ctx, "martin", f.martin, "chloe@exemple.fr", f.mailer.last(t, "chloe@exemple.fr"), f.token, Device{})
	var incorrect *IncorrectCodeError
	if !errors.As(err, &incorrect) || incorrect.AttemptsLeft != LoginCodeAttempts-1 {
		t.Errorf("code of a member revoked after the request: %v, want an incorrect code with %d attempts left", err, LoginCodeAttempts-1)
	}
}

func TestRevokeMemberClosesSessions(t *testing.T) {
	ctx := context.Background()
	f := newLoginFixture(t)
	chloe, err := f.martin.AddMember(ctx, "chloe@exemple.fr", "Chloé", f.alice.ID, f.now)
	if err != nil {
		t.Fatal(err)
	}
	_, token := f.signIn(t, "chloe@exemple.fr")
	if err := f.requestCode(ctx, "martin", f.martin, "chloe@exemple.fr", "192.0.2.1"); err != nil {
		t.Fatal(err)
	}
	code := f.mailer.last(t, "chloe@exemple.fr")

	if err := f.martin.RevokeMember(ctx, chloe.ID, f.alice.ID, f.now); err != nil {
		t.Fatal(err)
	}
	if _, err := f.login.Session(ctx, f.martin, token); !errors.Is(err, ErrNoSession) {
		t.Errorf("session of a revoked member: error = %v, want ErrNoSession", err)
	}
	if _, _, err := f.login.OpenSession(ctx, "martin", f.martin, "chloe@exemple.fr", code, f.token, Device{}); err == nil {
		t.Error("the pending code of a revoked member is accepted")
	}
	if err := f.martin.RevokeMember(ctx, chloe.ID, f.alice.ID, f.now); !errors.Is(err, ErrUnknownMember) {
		t.Errorf("revoked twice: error = %v, want ErrUnknownMember", err)
	}
	entries, _ := f.martin.AuditLog(ctx)
	var ops []AuditOperation
	for _, e := range entries[len(entries)-2:] {
		ops = append(ops, e.Operation)
	}
	if ops[0] != SessionClosed || ops[1] != MemberRevoked {
		t.Errorf("last audit operations = %v, want session closed then member revoked", ops)
	}
}

func TestPurge(t *testing.T) {
	ctx := context.Background()
	f := newLoginFixture(t)
	f.signIn(t, "alice@exemple.fr")
	if err := f.requestCode(ctx, "martin", f.martin, "alice@exemple.fr", "192.0.2.1"); err != nil {
		t.Fatal(err)
	}
	if err := f.requestCode(ctx, "martin", f.martin, "inconnu@exemple.fr", "192.0.2.1"); err != nil {
		t.Fatal(err)
	}
	counts := func() (requests, decoys, codes, sessions int) {
		t.Helper()
		for _, c := range []struct {
			db    *sql.DB
			table string
			n     *int
		}{
			{f.store.RateLimit(), "code_requests", &requests},
			{f.store.RateLimit(), "decoy_codes", &decoys},
			{f.martin.db, "login_codes", &codes},
			{f.martin.db, "sessions", &sessions},
		} {
			if err := c.db.QueryRowContext(ctx, "SELECT count(*) FROM "+c.table).Scan(c.n); err != nil {
				t.Fatal(err)
			}
		}
		return
	}

	if err := f.login.Purge(ctx, f.store); err != nil {
		t.Fatal(err)
	}
	if r, d, c, s := counts(); r != 3 || d != 2 || c != 1 || s != 1 {
		t.Errorf("nothing to purge yet: %d requests, %d decoys, %d codes, %d sessions", r, d, c, s)
	}

	f.advance(time.Hour)
	if err := f.login.Purge(ctx, f.store); err != nil {
		t.Fatal(err)
	}
	if r, d, c, s := counts(); r != 0 || d != 0 || c != 0 || s != 1 {
		t.Errorf("after an hour: %d requests, %d decoys, %d codes, %d sessions; want only the session", r, d, c, s)
	}

	f.advance(SessionLifetime)
	if err := f.login.Purge(ctx, f.store); err != nil {
		t.Fatal(err)
	}
	if _, _, _, s := counts(); s != 0 {
		t.Errorf("after 90 days: %d sessions, want 0", s)
	}
}

// TestLimitsSurviveRestart: after the databases are closed and opened again, as at a
// restart of the server, the requests already made and the attempts already used still
// count, for a real code as for a decoy code (ADR 0021).
func TestLimitsSurviveRestart(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	now := time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC)
	mailer := &fakeMailer{}
	migrations, err := storage.EmbeddedMigrations()
	if err != nil {
		t.Fatal(err)
	}
	open := func() (*storage.Store, *Login, *Store) {
		t.Helper()
		st, err := storage.Open(ctx, dir, migrations)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = st.Close() })
		login := &Login{RateLimit: NewRateLimitDB(st.RateLimit()), Mailer: mailer, Now: func() time.Time { return now }}
		db, err := st.Tribe(ctx, "martin")
		if errors.Is(err, storage.ErrUnknownTribe) {
			return st, login, nil
		}
		if err != nil {
			t.Fatal(err)
		}
		return st, login, NewStore(db)
	}

	st, login, _ := open()
	err = st.CreateTribe(ctx, "martin", "Les Martin", func(ctx context.Context, db *sql.DB) error {
		return NewStore(db).Initialize(ctx, "Les Martin", "alice@exemple.fr", "Alice", now)
	})
	if err != nil {
		t.Fatal(err)
	}
	db, err := st.Tribe(ctx, "martin")
	if err != nil {
		t.Fatal(err)
	}
	martin := NewStore(db)
	tokens := map[string]string{}
	for _, email := range []string{"alice@exemple.fr", "inconnu@exemple.fr"} {
		for range EmailRequestLimit {
			if tokens[email], err = login.RequestCode(ctx, "martin", martin, email, "192.0.2.1"); err != nil {
				t.Fatal(err)
			}
		}
		if _, _, err := login.OpenSession(ctx, "martin", martin, email, "wrong", tokens[email], Device{}); err == nil {
			t.Fatal("a wrong code was accepted")
		}
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	_, login, martin = open()
	for _, email := range []string{"alice@exemple.fr", "inconnu@exemple.fr"} {
		if _, err := login.RequestCode(ctx, "martin", martin, email, "192.0.2.1"); !errors.Is(err, ErrTooManyRequests) {
			t.Errorf("%s: request after restart: error = %v, want ErrTooManyRequests", email, err)
		}
		_, _, err := login.OpenSession(ctx, "martin", martin, email, "wrong", tokens[email], Device{})
		var incorrect *IncorrectCodeError
		if !errors.As(err, &incorrect) || incorrect.AttemptsLeft != LoginCodeAttempts-2 {
			t.Errorf("%s: attempt after restart: error = %v, want %d attempts left", email, err, LoginCodeAttempts-2)
		}
	}
}
