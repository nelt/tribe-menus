package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

// minHSTS is the max-age that HSTS has at least: a day to begin with (plan recette, step 12).
const minHSTS = 86400

func (p *prober) checkTLS(ctx context.Context) {
	resp, _, err := p.get(ctx, p.url("/"))
	if err != nil {
		p.fail("HTTPS: %s", p.errorText(err))
		return
	}
	state := resp.TLS
	p.check(state != nil && state.Version >= tls.VersionTLS12, "certificate valid for the host name, TLS 1.2 at least (%s)", tlsVersion(state))
	if state != nil && len(state.PeerCertificates) > 0 {
		cert := state.PeerCertificates[0]
		issuer := strings.Join(cert.Issuer.Organization, ", ")
		p.check(issuer == "Let's Encrypt", "certificate issued by Let's Encrypt (%s)", issuer)
		days := int(time.Until(cert.NotAfter).Hours() / 24)
		p.check(days >= 7, "certificate valid for %d more days", days)
	}
	p.info("protocol: %s", resp.Proto)
	p.check(resp.Header.Get("Alt-Svc") == "", "no HTTP/3 advertised (Alt-Svc)")
	p.check(resp.Header.Get("Server") == "", "no Server header")
	p.checkHSTS(resp.Header, "/")
}

func tlsVersion(state *tls.ConnectionState) string {
	if state == nil {
		return "none"
	}
	return tls.VersionName(state.Version)
}

// hstsMaxAge returns the max-age of a Strict-Transport-Security header, or -1.
func hstsMaxAge(header string) int {
	for directive := range strings.SplitSeq(header, ";") {
		name, value, _ := strings.Cut(strings.TrimSpace(directive), "=")
		if strings.EqualFold(name, "max-age") {
			n, err := strconv.Atoi(strings.Trim(value, `"`))
			if err == nil {
				return n
			}
		}
	}
	return -1
}

func (p *prober) checkHSTS(h http.Header, where string) {
	age := hstsMaxAge(h.Get("Strict-Transport-Security"))
	p.check(age >= minHSTS, "HSTS on %s, max-age %d (%d at least)", where, age, minHSTS)
}

func (p *prober) checkRedirect(ctx context.Context) {
	u := *p.base
	u.Scheme = "http"
	u.Path = "/tribes/"
	resp, _, err := p.get(ctx, u.String())
	if err != nil {
		p.fail("HTTP: %s", p.errorText(err))
		return
	}
	want := p.url("/tribes/")
	good := (resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusPermanentRedirect) &&
		resp.Header.Get("Location") == want
	p.check(good, "HTTP redirected to HTTPS, same path (%d)", resp.StatusCode)
}

// checkSite checks the headers of the public site, or of the waiting page when no site is
// installed.
func (p *prober) checkSite(ctx context.Context) {
	resp, _, err := p.get(ctx, p.url("/"))
	if err != nil {
		p.fail("site: %s", p.errorText(err))
		return
	}
	switch resp.StatusCode {
	case http.StatusOK:
		p.ok("site served (200)")
	case http.StatusServiceUnavailable:
		p.info("site not installed: waiting page (503)")
	default:
		p.fail("site: status %d", resp.StatusCode)
	}
	csp := resp.Header.Get("Content-Security-Policy")
	p.check(strings.Contains(csp, "default-src 'none'") && !strings.Contains(csp, "script-src"), "site: CSP without script (default-src 'none')")
	p.check(strings.Contains(csp, "frame-ancestors 'none'"), "site: CSP frame-ancestors 'none'")
	if resp.StatusCode == http.StatusOK {
		p.check(resp.Header.Get("X-Content-Type-Options") == "nosniff", "site: X-Content-Type-Options nosniff")
		p.check(resp.Header.Get("Referrer-Policy") != "", "site: Referrer-Policy")
	}
	p.check(resp.Header.Get("Set-Cookie") == "", "site: no cookie")
	robots := resp.Header.Get("X-Robots-Tag")
	if p.env == "recette" {
		p.check(strings.Contains(robots, "noindex"), "recette: X-Robots-Tag noindex on the site")
		resp, body, err := p.get(ctx, p.url("/robots.txt"))
		p.check(err == nil && resp.StatusCode == http.StatusOK && strings.Contains(body, "Disallow: /\n"), "recette: robots.txt forbids everything")
	} else {
		p.check(!strings.Contains(robots, "noindex"), "production: site indexable (no X-Robots-Tag noindex)")
	}
}

func (p *prober) checkApp(ctx context.Context) {
	resp, _, err := p.get(ctx, p.url("/tribes/"+p.tribe+"/"))
	if err != nil {
		p.fail("application: %s", p.errorText(err))
		return
	}
	if resp.StatusCode == http.StatusServiceUnavailable {
		p.info("application not in service: waiting page (503)")
		p.check(resp.Header.Get("Retry-After") != "", "waiting page: Retry-After")
		return
	}
	p.check(resp.StatusCode == http.StatusOK, "application served (%d)", resp.StatusCode)
	csp := resp.Header.Get("Content-Security-Policy")
	p.check(strings.Contains(csp, "default-src 'self'") && strings.Contains(csp, "frame-ancestors 'none'"), "application: CSP of the application (default-src 'self')")
	p.check(resp.Header.Get("X-Content-Type-Options") == "nosniff", "application: X-Content-Type-Options nosniff")
	p.check(strings.Contains(resp.Header.Get("X-Robots-Tag"), "noindex"), "application: X-Robots-Tag noindex")
	p.checkHSTS(resp.Header, "the application")
}

func (p *prober) checkHealthz(ctx context.Context) {
	resp, body, err := p.get(ctx, p.url("/healthz"))
	if err != nil {
		p.fail("/healthz: %s", p.errorText(err))
		return
	}
	p.check(resp.StatusCode == http.StatusNotFound && !strings.HasPrefix(body, "ok"), "/healthz not forwarded (%d)", resp.StatusCode)
}

// checkBodyLimit sends a body too large, which Caddy or the application refuses (plan
// recette, step 12).
func (p *prober) checkBodyLimit(ctx context.Context) {
	body := bytes.Repeat([]byte("a"), 128<<10)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url("/tribes/"+p.tribe+"/api/login-codes"), bytes.NewReader(body))
	if err != nil {
		p.fail("body too large: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, _, err := p.do(req)
	switch {
	case err != nil:
		p.fail("body too large: %s", p.errorText(err))
	case resp.StatusCode == http.StatusServiceUnavailable:
		p.info("body too large: application not in service, not checked")
	default:
		p.check(resp.StatusCode == http.StatusRequestEntityTooLarge || resp.StatusCode == http.StatusBadRequest,
			"body of 128 KiB refused (%d)", resp.StatusCode)
	}
}

// checkSlowRequests sends two requests that stay incomplete, headers then body: Caddy
// closes them after its read timeouts (plan recette, step 12).
func (p *prober) checkSlowRequests(ctx context.Context) {
	var wg sync.WaitGroup
	var header, slowBody time.Duration
	var headerErr, bodyErr error
	wg.Go(func() {
		header, headerErr = p.slow(ctx, "GET /tribes/ HTTP/1.1\r\nHost: "+p.base.Host+"\r\nX-Slow: ")
	})
	wg.Go(func() {
		slowBody, bodyErr = p.slow(ctx, "POST /tribes/"+p.tribe+"/api/login-codes HTTP/1.1\r\nHost: "+p.base.Host+
			"\r\nContent-Type: application/json\r\nContent-Length: 1000\r\n\r\n{\"email\":")
	})
	wg.Wait()
	p.checkSlow("request headers never finished", header, headerErr, 15*time.Second)
	p.checkSlow("request body never finished", slowBody, bodyErr, 30*time.Second)
}

// slow sends the beginning of a request, then nothing, and returns the time the server
// takes to answer or close the connection.
func (p *prober) slow(ctx context.Context, start string) (time.Duration, error) {
	dialer := &tls.Dialer{Config: &tls.Config{ServerName: p.base.Hostname(), NextProtos: []string{"http/1.1"}}}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(p.base.Hostname(), port(p.base.Port(), "443")))
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	begin := time.Now()
	if _, err := io.WriteString(conn, start); err != nil {
		return 0, err
	}
	if err := conn.SetReadDeadline(time.Now().Add(55 * time.Second)); err != nil {
		return 0, err
	}
	// The answer, or the end of the connection.
	_, _ = io.Copy(io.Discard, conn)
	return time.Since(begin), nil
}

func (p *prober) checkSlow(what string, took time.Duration, err error, limit time.Duration) {
	if err != nil {
		p.fail("%s: %s", what, p.errorText(err))
		return
	}
	p.check(took < limit, "%s: connection ended after %ds (%ds at most)", what, int(took.Seconds()), int(limit.Seconds()))
}

func port(p, fallback string) string {
	if p == "" {
		return fallback
	}
	return p
}

// expectedPorts are open on the server (ADR 0015, point 4); otherPorts, among the most
// common, must not answer.
var (
	expectedPorts = []int{22, 80, 443}
	otherPorts    = []int{21, 23, 25, 53, 110, 111, 143, 445, 465, 587, 993, 995, 2019, 2375, 3000, 3306, 5432, 6379, 8000, 8080, 8443, 9000, 9090, 27017}
)

// checkPorts dials the ports on every address of the host, IPv4 and IPv6. A port filtered
// on the way from here counts as closed: what answers is open.
func (p *prober) checkPorts(ctx context.Context) {
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, p.base.Hostname())
	if err != nil {
		p.fail("addresses of the host: %s", p.errorText(err))
		return
	}
	for _, family := range []string{"IPv4", "IPv6"} {
		var addrs []string
		for _, ip := range ips {
			if (ip.IP.To4() != nil) == (family == "IPv4") {
				addrs = append(addrs, ip.IP.String())
			}
		}
		if len(addrs) == 0 {
			p.fail("%s: no address", family)
			continue
		}
		if !reachable(ctx, addrs[0], 443) {
			p.info("%s: unreachable from here, not checked", family)
			continue
		}
		var open []int
		for _, addr := range addrs {
			open = append(open, openPorts(ctx, addr, append(slices.Clone(expectedPorts), otherPorts...))...)
		}
		slices.Sort(open)
		open = slices.Compact(open)
		p.check(slices.Equal(open, expectedPorts), "%s: only TCP 22, 80 and 443 answer among %d common ports (%s)",
			family, len(expectedPorts)+len(otherPorts), joinInts(open))
	}
}

func reachable(ctx context.Context, addr string, port int) bool {
	d := net.Dialer{Timeout: 5 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(addr, strconv.Itoa(port)))
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func openPorts(ctx context.Context, addr string, ports []int) []int {
	var mu sync.Mutex
	var wg sync.WaitGroup
	var open []int
	for _, port := range ports {
		wg.Go(func() {
			if reachable(ctx, addr, port) {
				mu.Lock()
				open = append(open, port)
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	return open
}

func joinInts(ns []int) string {
	if len(ns) == 0 {
		return "none"
	}
	s := make([]string, len(ns))
	for i, n := range ns {
		s[i] = strconv.Itoa(n)
	}
	return strings.Join(s, " ")
}

// xffRequests is the number of code requests of the check: one more than the limit by IP
// (ENF-01).
const xffRequests = 11

// checkXFF sends code requests that each claim another client in X-Forwarded-For, for
// addresses that are not members: Caddy replaces the header, so the limit by IP still
// refuses them (ADR 0006, point 7).
func (p *prober) checkXFF(ctx context.Context) {
	run := randomHex(4)
	var statuses []int
	refusedAt := 0
	for i := 1; i <= xffRequests; i++ {
		status, err := p.requestCode(ctx, p.tribe, fmt.Sprintf("sonde-%s-%d@example.org", run, i),
			fmt.Sprintf("203.0.113.%d, 198.51.100.%d", i, i))
		if err != nil {
			p.fail("code request %d: %s", i, p.errorText(err))
			return
		}
		statuses = append(statuses, status)
		if status == http.StatusServiceUnavailable {
			p.info("application not in service, not checked")
			return
		}
		if status == http.StatusTooManyRequests {
			refusedAt = i
			break
		}
	}
	switch {
	case refusedAt == 1:
		p.ok("forged X-Forwarded-For without effect: refused from the first request (budget of the hour already used)")
	case refusedAt > 0:
		p.ok("forged X-Forwarded-For without effect: request %d refused (429) after %d accepted", refusedAt, refusedAt-1)
	default:
		p.fail("forged X-Forwarded-For: %d requests, none refused (%s)", xffRequests, joinInts(statuses))
	}
}

// requestCode sends POST login-codes and returns its status.
func (p *prober) requestCode(ctx context.Context, slug, email, forwardedFor string) (int, error) {
	body := fmt.Sprintf(`{"email":%q}`, email)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url("/tribes/"+slug+"/api/login-codes"), strings.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if forwardedFor != "" {
		req.Header.Set("X-Forwarded-For", forwardedFor)
	}
	resp, _, err := p.do(req)
	if err != nil {
		return 0, err
	}
	return resp.StatusCode, nil
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
