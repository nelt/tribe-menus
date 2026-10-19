package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestHSTSMaxAge(t *testing.T) {
	tests := []struct {
		header string
		want   int
	}{
		{"max-age=86400", 86400},
		{"max-age=31536000; includeSubDomains", 31536000},
		{`includeSubDomains; Max-Age="600"`, 600},
		{"", -1},
		{"max-age=x", -1},
	}
	for _, tc := range tests {
		if got := hstsMaxAge(tc.header); got != tc.want {
			t.Errorf("hstsMaxAge(%q) = %d, want %d", tc.header, got, tc.want)
		}
	}
}

func TestCAALetsEncryptOnly(t *testing.T) {
	tests := []struct {
		name    string
		records []string
		want    bool
	}{
		{"one issue", []string{`0 issue "letsencrypt.org"`}, true},
		{"with parameters and iodef", []string{`0 issue "letsencrypt.org; validationmethods=dns-01"`, `0 iodef "mailto:a@example.org"`}, true},
		{"another authority", []string{`0 issue "letsencrypt.org"`, `0 issue "sectigo.com"`}, false},
		{"wildcard elsewhere", []string{`0 issue "letsencrypt.org"`, `0 issuewild "digicert.com"`}, false},
		{"no issue", []string{`0 iodef "mailto:a@example.org"`}, false},
		{"none", nil, false},
	}
	for _, tc := range tests {
		if got := caaLetsEncryptOnly(tc.records); got != tc.want {
			t.Errorf("%s: %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestJoinTXT(t *testing.T) {
	for in, want := range map[string]string{
		`v=spf1 ~all`:                `v=spf1 ~all`,
		`"v=spf1 include:a" " ~all"`: `v=spf1 include:a ~all`,
		`"v=DMARC1; p=none"`:         `v=DMARC1; p=none`,
	} {
		if got := joinTXT(in); got != want {
			t.Errorf("joinTXT(%q) = %q, want %q", in, got, want)
		}
	}
	if got := dmarcPolicy("v=DMARC1; p=none; rua=mailto:a@example.org"); got != "none" {
		t.Errorf("dmarcPolicy = %q", got)
	}
}

func ms3(ns ...int) []time.Duration {
	d := make([]time.Duration, len(ns))
	for i, n := range ns {
		d[i] = time.Duration(n) * time.Millisecond
	}
	return d
}

func TestSummarize(t *testing.T) {
	st := summarize(ms3(5, 1, 3, 2, 4))
	if st.n != 5 || st.median != 3*time.Millisecond || st.q1 != 2*time.Millisecond || st.q3 != 4*time.Millisecond {
		t.Errorf("summarize = %+v", st)
	}
	if st := summarize(ms3(1, 2)); st.median != 1500*time.Microsecond {
		t.Errorf("median of two = %v", st.median)
	}
	if st := summarize(nil); st.n != 0 {
		t.Errorf("summarize(nil) = %+v", st)
	}
}

func TestSuperiority(t *testing.T) {
	tests := []struct {
		name string
		x, y []time.Duration
		want float64
	}{
		{"always slower", ms3(10, 11, 12), ms3(1, 2, 3), 1},
		{"always faster", ms3(1, 2), ms3(5, 6), 0},
		{"same", ms3(1, 2, 3), ms3(1, 2, 3), 0.5},
		{"interleaved", ms3(1, 3), ms3(2, 4), 0.25},
		{"empty", nil, ms3(1), 0.5},
	}
	for _, tc := range tests {
		if got := superiority(tc.x, tc.y); got != tc.want {
			t.Errorf("%s: %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		args    []string
		env     string
		mail    string
		wantErr bool
	}{
		{args: []string{"-url", "https://recette.meltingtribe.example.org"}, env: "recette", mail: "example.org"},
		{args: []string{"-url", "https://meltingtribe.example.org/"}, env: "production", mail: "example.org"},
		{args: []string{"-url", "https://a.example.org", "-env", "recette", "-mail-domain", "mail.example.org"}, env: "recette", mail: "mail.example.org"},
		{args: []string{"-url", "http://meltingtribe.example.org"}, wantErr: true},
		{args: []string{"-url", "https://meltingtribe.example.org/tribes/"}, wantErr: true},
		{args: []string{"-url", "https://meltingtribe.example.org", "-env", "staging"}, wantErr: true},
		{args: []string{"-url", "https://meltingtribe.example.org", "-timing"}, wantErr: true},
		{args: []string{"-url", "https://meltingtribe.example.org", "-timing", "-tribe", "demo"}, wantErr: true},
		{args: []string{}, wantErr: true},
	}
	for _, tc := range tests {
		o, err := parse(tc.args, io.Discard)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parse(%q) accepted", tc.args)
			}
			continue
		}
		if err != nil {
			t.Errorf("parse(%q): %v", tc.args, err)
			continue
		}
		if o.env != tc.env || o.mailDomain != tc.mail {
			t.Errorf("parse(%q): env %q, mail domain %q", tc.args, o.env, o.mailDomain)
		}
	}
}

// caddy imitates what Caddy and the application answer, as the configuration of
// provision.sh wants it, and counts the code requests by client, ignoring X-Forwarded-For.
func caddy(trustXFF bool) http.Handler {
	var requests atomic.Int32
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Strict-Transport-Security", "max-age=86400")
		h.Set("X-Robots-Tag", "noindex, nofollow")
		switch {
		case r.URL.Path == "/robots.txt":
			_, _ = io.WriteString(w, "User-agent: *\nDisallow: /\n")
		case r.URL.Path == "/":
			h.Set("Content-Security-Policy", "default-src 'none'; style-src 'self'; frame-ancestors 'none'")
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("Referrer-Policy", "same-origin")
			_, _ = io.WriteString(w, "<!doctype html>")
		case strings.HasSuffix(r.URL.Path, "/api/login-codes"):
			body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4<<10))
			if err != nil || !bytes.HasPrefix(body, []byte(`{"email":`)) {
				http.Error(w, "too large", http.StatusRequestEntityTooLarge)
				return
			}
			if !trustXFF && requests.Add(1) > 10 {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			w.WriteHeader(http.StatusAccepted)
		case strings.HasPrefix(r.URL.Path, "/tribes/"):
			h.Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'")
			h.Set("X-Content-Type-Options", "nosniff")
			_, _ = io.WriteString(w, "<!doctype html>")
		default:
			http.NotFound(w, r)
		}
	})
}

func testProber(t *testing.T, handler http.Handler) (*prober, *bytes.Buffer) {
	t.Helper()
	srv := httptest.NewTLSServer(handler)
	t.Cleanup(srv.Close)
	base, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	client := srv.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	var out bytes.Buffer
	return &prober{
		options: options{base: base, env: "recette", tribe: "sonde"},
		out:     &out,
		client:  client,
	}, &out
}

func TestHeaders(t *testing.T) {
	p, out := testProber(t, caddy(false))
	ctx := context.Background()
	p.checkSite(ctx)
	p.checkApp(ctx)
	p.checkHealthz(ctx)
	if p.failures != 0 {
		t.Errorf("%d failure(s):\n%s", p.failures, out)
	}
	if strings.Contains(out.String(), p.base.Host) {
		t.Errorf("report names the host:\n%s", out)
	}
}

func TestHeadersMissing(t *testing.T) {
	p, out := testProber(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok\n")
	}))
	ctx := context.Background()
	p.checkSite(ctx)
	p.checkApp(ctx)
	p.checkHealthz(ctx)
	// CSP twice, robots, robots.txt, nosniff, Referrer-Policy, application CSP, nosniff,
	// noindex, HSTS, /healthz.
	if p.failures < 10 {
		t.Errorf("%d failure(s) only:\n%s", p.failures, out)
	}
}

func TestXFF(t *testing.T) {
	for _, tc := range []struct {
		name     string
		trustXFF bool
		failures int
	}{
		{"header replaced", false, 0},
		{"header trusted", true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, out := testProber(t, caddy(tc.trustXFF))
			p.checkXFF(context.Background())
			if p.failures != tc.failures {
				t.Errorf("%d failure(s), want %d:\n%s", p.failures, tc.failures, out)
			}
		})
	}
}

func TestBodyTooLarge(t *testing.T) {
	p, out := testProber(t, caddy(false))
	p.checkBodyLimit(context.Background())
	if p.failures != 0 {
		t.Errorf("%d failure(s):\n%s", p.failures, out)
	}
}
