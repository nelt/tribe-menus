package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"mime/quotedprintable"
	"net"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nelt/tribe-menus/internal/mail"
)

const (
	box      = "recette-auto@example.org"
	password = "secret-du-test"
)

// request is the time of the code request in the tests.
var request = time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC)

// loginCode returns a message carrying a login code as the application writes it
// (mail.LoginCodeMessage, quoted-printable, CRLF), dated at.
func loginCode(t *testing.T, to, code string, at time.Time) string {
	t.Helper()
	m := mail.LoginCodeMessage(to, code, 10*time.Minute)
	return message(t, to, at, m.Body)
}

func message(t *testing.T, to string, at time.Time, body string) string {
	t.Helper()
	var b bytes.Buffer
	fmt.Fprintf(&b, "From: \"Melting Tribe\" <no-reply@example.org>\r\nTo: <%s>\r\nSubject: Votre code\r\n", to)
	fmt.Fprintf(&b, "Date: %s\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n", at.Format(time.RFC1123Z))
	w := quotedprintable.NewWriter(&b)
	if _, err := w.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(strings.ReplaceAll(b.String(), "\r\n", "\n"), "\n", "\r\n")
}

// fakePOP3 is a POP3 server holding a box shared by its sessions: DELE takes effect at QUIT.
type fakePOP3 struct {
	mu       sync.Mutex
	messages []string
	// refusePass answers -ERR to PASS; cuts closes the connection after LIST for that many
	// sessions.
	refusePass bool
	cuts       int
	sessions   int
	passwords  []string
}

func (s *fakePOP3) add(m string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, m)
}

func (s *fakePOP3) box() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.messages)
}

// serve runs the server on l until it is closed.
func (s *fakePOP3) serve(l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		go s.session(conn)
	}
}

func (s *fakePOP3) session(conn net.Conn) {
	defer conn.Close()
	s.mu.Lock()
	s.sessions++
	cut := s.cuts > 0
	if cut {
		s.cuts--
	}
	messages := slices.Clone(s.messages)
	s.mu.Unlock()

	in := bufio.NewScanner(conn)
	say := func(line string) { fmt.Fprint(conn, line+"\r\n") }
	deleted := map[int]bool{}
	say("+OK POP3 ready")
	for in.Scan() {
		command, arg, _ := strings.Cut(in.Text(), " ")
		var n int
		fmt.Sscan(arg, &n)
		switch command {
		case "USER":
			say("+OK")
		case "PASS":
			s.mu.Lock()
			s.passwords = append(s.passwords, arg)
			s.mu.Unlock()
			if s.refusePass {
				say("-ERR [AUTH] invalid credentials for " + box)
				continue
			}
			say("+OK logged in")
		case "LIST":
			say(fmt.Sprintf("+OK %d messages", len(messages)))
			for i, m := range messages {
				say(fmt.Sprintf("%d %d", i+1, len(m)))
			}
			say(".")
			if cut {
				return
			}
		case "RETR":
			say("+OK")
			for line := range strings.SplitSeq(strings.TrimSuffix(messages[n-1], "\r\n"), "\r\n") {
				if strings.HasPrefix(line, ".") {
					line = "." + line
				}
				say(line)
			}
			say(".")
		case "DELE":
			deleted[n] = true
			say("+OK")
		case "QUIT":
			s.mu.Lock()
			var kept []string
			for _, m := range s.messages {
				if i := slices.Index(messages, m); i < 0 || !deleted[i+1] {
					kept = append(kept, m)
				}
			}
			s.messages = kept
			s.mu.Unlock()
			say("+OK bye")
			return
		default:
			say("-ERR unknown command")
		}
	}
}

// start runs s on a local listener, without TLS, and returns a fetcher for it.
func start(t *testing.T, s *fakePOP3) fetcher {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	go s.serve(l)
	return fetcher{
		address:  box,
		user:     box,
		password: password,
		dial: func(ctx context.Context) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "tcp", l.Addr().String())
		},
		interval: 10 * time.Millisecond,
	}
}

func TestWait(t *testing.T) {
	other := loginCode(t, "quelquun@example.org", "11111111", request.Add(time.Second))
	old := loginCode(t, box, "22222222", request.Add(-time.Hour))
	first := loginCode(t, box, "33333333", request.Add(time.Second))
	second := loginCode(t, box, "44444444", request.Add(5*time.Second))
	skewed := loginCode(t, box, "55555555", request.Add(-10*time.Second))
	noCode := message(t, box, request.Add(time.Second), "Bonjour,\n\nRien à voir.\n")

	tests := []struct {
		name     string
		server   *fakePOP3
		later    string // message added once the first session is over
		want     string
		wantErr  string
		wantBox  []string
		sessions int // at most, 0 for unchecked
	}{
		{name: "no message", wantErr: "no login code within"},
		{name: "message older than the request, deleted", server: &fakePOP3{messages: []string{old}}, wantErr: "no login code within", wantBox: []string{}},
		{name: "other recipient, kept", server: &fakePOP3{messages: []string{other}}, wantErr: "no login code within", wantBox: []string{other}},
		{name: "one message", server: &fakePOP3{messages: []string{first}}, want: "33333333", wantBox: []string{}, sessions: 1},
		{name: "several messages, the most recent one", server: &fakePOP3{messages: []string{old, second, other, first}}, want: "44444444", wantBox: []string{other}, sessions: 1},
		{name: "dated a little before the request", server: &fakePOP3{messages: []string{skewed}}, want: "55555555", wantBox: []string{}},
		{name: "message without code, deleted", server: &fakePOP3{messages: []string{noCode}}, wantErr: "no login code within", wantBox: []string{}},
		{name: "message arriving later", later: first, want: "33333333", wantBox: []string{}},
		{name: "connection cut, then the code", server: &fakePOP3{messages: []string{first}, cuts: 1}, want: "33333333", wantBox: []string{}, sessions: 2},
		{name: "connection always cut", server: &fakePOP3{messages: []string{first}, cuts: 1000}, wantErr: "last error: pop3 RETR", wantBox: []string{first}},
		{name: "password refused, no retry", server: &fakePOP3{messages: []string{first}, refusePass: true}, wantErr: "login refused: pop3 PASS: refused by the server", wantBox: []string{first}, sessions: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.server
			if s == nil {
				s = &fakePOP3{}
			}
			f := start(t, s)
			if tt.later != "" {
				go func() {
					for {
						s.mu.Lock()
						n := s.sessions
						s.mu.Unlock()
						if n > 0 {
							s.add(tt.later)
							return
						}
						time.Sleep(time.Millisecond)
					}
				}()
			}
			code, err := f.wait(context.Background(), request, 300*time.Millisecond)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("wait: %v", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("wait: error %v, want %q", err, tt.wantErr)
			}
			if err != nil && (strings.Contains(err.Error(), password) || strings.Contains(err.Error(), box) || strings.Contains(err.Error(), "33333333")) {
				t.Errorf("error %q shows the password, the account or the code", err)
			}
			if code != tt.want {
				t.Errorf("code %q, want %q", code, tt.want)
			}
			if tt.wantBox != nil && !slices.Equal(s.box(), tt.wantBox) {
				t.Errorf("box after the fetch: %d messages, want %d", len(s.box()), len(tt.wantBox))
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			if tt.sessions > 0 && s.sessions > tt.sessions {
				t.Errorf("%d sessions, want %d at most", s.sessions, tt.sessions)
			}
			for _, p := range s.passwords {
				if p != password {
					t.Errorf("password sent %q, want %q", p, password)
				}
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		wantCode string
		wantTo   string
		wantErr  bool
	}{
		{name: "login code", raw: loginCode(t, box, "01234567", request), wantCode: "01234567", wantTo: box},
		{name: "recipient in capitals", raw: loginCode(t, "Recette-Auto@Example.org", "01234567", request), wantCode: "01234567", wantTo: "Recette-Auto@Example.org"},
		{name: "not a login code", raw: message(t, box, request, "Votre commande 12345678 est partie.\n"), wantTo: box},
		{name: "two codes", raw: message(t, box, request, "Votre code de connexion à Melting Tribe : 11111111\nVotre code de connexion à Melting Tribe : 22222222\n"), wantTo: box},
		{name: "without Date", raw: "To: " + box + "\r\n\r\nVotre code de connexion à Melting Tribe : 11111111\r\n", wantErr: true},
		{name: "not a message", raw: "\x00", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := parse(1, []byte(strings.ReplaceAll(tt.raw, "\r\n", "\n")))
			if (err != nil) != tt.wantErr {
				t.Fatalf("parse: %v, want error %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if m.code != tt.wantCode || !m.addressedTo(box) || m.to[0] != tt.wantTo || !m.date.Equal(request) {
				t.Errorf("parse: %+v", m)
			}
		})
	}
}

// TestDialTLS checks that the connection is over TLS, with the certificate checked.
func TestDialTLS(t *testing.T) {
	srv := httptest.NewUnstartedServer(nil)
	srv.StartTLS()
	srv.Close()
	l, err := tls.Listen("tcp", "127.0.0.1:0", srv.TLS)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	go (&fakePOP3{messages: []string{loginCode(t, box, "01234567", request)}}).serve(l)
	trusted := x509.NewCertPool()
	trusted.AddCert(srv.Certificate())

	tests := []struct {
		name    string
		roots   *x509.CertPool
		want    string
		wantErr bool
	}{
		{name: "trusted certificate", roots: trusted, want: "01234567"},
		{name: "unknown certificate", roots: x509.NewCertPool(), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fetcher{address: box, user: box, password: password, interval: time.Millisecond,
				dial: func(ctx context.Context) (net.Conn, error) { return dialTLS(ctx, l.Addr().String(), tt.roots) }}
			code, err := f.fetch(context.Background(), request)
			if (err != nil) != tt.wantErr || code != tt.want {
				t.Errorf("fetch: %q, %v", code, err)
			}
			var unknown x509.UnknownAuthorityError
			if tt.wantErr && !errors.As(err, &unknown) {
				t.Errorf("fetch: %v, want an unknown authority", err)
			}
		})
	}
}

func TestReadConfig(t *testing.T) {
	const good = `{"tribeURL": "https://recette.example.org/tribes/x/", "address": "a@example.org", "pop3": {"host": "pop3.example.org", "port": 995, "user": "a@example.org"}}`
	tests := []struct {
		name     string
		config   string
		password string
		mode     os.FileMode
		wantErr  string
	}{
		{name: "complete", config: good, password: password + "\n", mode: 0o600},
		{name: "no configuration", password: password, mode: 0o600, wantErr: "read recette-auto.json"},
		{name: "missing port", config: `{"address": "a@example.org", "pop3": {"host": "h", "user": "u"}}`, password: password, mode: 0o600, wantErr: "pop3.port"},
		{name: "no password", config: good, wantErr: "read recette-auto.pop3"},
		{name: "password readable by others", config: good, password: password, mode: 0o644, wantErr: "want mode 0600"},
		{name: "empty password", config: good, password: "\n", mode: 0o600, wantErr: "empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.config != "" {
				if err := os.WriteFile(filepath.Join(dir, configFile), []byte(tt.config), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if tt.mode != 0 {
				path := filepath.Join(dir, passwordFile)
				if err := os.WriteFile(path, []byte(tt.password), tt.mode); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(path, tt.mode); err != nil {
					t.Fatal(err)
				}
			}
			cfg, got, err := readConfig(dir)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) || strings.Contains(err.Error(), password) {
					t.Fatalf("readConfig: %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil || got != password || cfg.POP3.Port != 995 {
				t.Errorf("readConfig: %+v, %q, %v", cfg, got, err)
			}
		})
	}
}

func TestRunArguments(t *testing.T) {
	for _, args := range [][]string{{}, {"-since", "hier"}, {"-since", request.Format(time.RFC3339), "plus"}} {
		var stdout, stderr bytes.Buffer
		if code := run(context.Background(), args, &stdout, &stderr); code != 2 || stdout.Len() != 0 {
			t.Errorf("run %q: exit %d, stdout %q", args, code, stdout.String())
		}
	}
}
