// Command probe checks an instance from the outside (plan recette, D10, step 21): TLS and
// the redirection from HTTP, HSTS, the headers of the site and of the application,
// noindex in recette, /healthz, the limits of Caddy, the other ports, a forged
// X-Forwarded-For, and DNS through a public resolver. With -timing, it measures instead the
// response times of the login for a member and another address, an existing tribe and
// another.
//
// Usage:
//
//	go run ./internal/tools/probe -url https://<host> [-tribe <slug>] [-dkim-selector <s>]
//	go run ./internal/tools/probe -url https://<host> -timing -tribe <slug> -members <a,b>
//
// One line per point, ok, FAIL or info, without any value of the instance (host name,
// address): the report is written to be copied into a pull request. It exits in error
// when a point fails.
//
// The X-Forwarded-For check sends eleven code requests, which use up the budget of the
// hour of the connection it runs from (ENF-01: ten by hour and by IP); -timing uses it
// too.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}

type options struct {
	base         *url.URL
	env          string
	tribe        string
	mailDomain   string
	dkimSelector string
	skipXFF      bool
	timing       bool
	members      []string
	samples      int
	rounds       int
}

func parse(args []string, stderr io.Writer) (options, error) {
	flags := flag.NewFlagSet("probe", flag.ContinueOnError)
	flags.SetOutput(stderr)
	rawURL := flags.String("url", "", "address of the instance, https://<host>")
	env := flags.String("env", "", "recette or production (default: recette when the host starts with recette.)")
	slug := flags.String("tribe", "sonde", "tribe of the URL used by the checks of the application")
	mailDomain := flags.String("mail-domain", "", "domain of the email addresses (default: the last two labels of the host)")
	dkim := flags.String("dkim-selector", "", "DKIM selector of the mail domain (s= in the DKIM-Signature of a received email)")
	skipXFF := flags.Bool("skip-xff", false, "skip the X-Forwarded-For check, which uses up the code requests of the hour")
	timing := flags.Bool("timing", false, "measure the response times of the login instead of the other checks")
	members := flags.String("members", "", "with -timing: addresses of members of the tribe, comma separated")
	samples := flags.Int("samples", 60, "with -timing: requests by case on POST sessions")
	rounds := flags.Int("rounds", 1, "with -timing: hours of code requests on POST login-codes, nine an hour")
	if err := flags.Parse(args); err != nil {
		return options{}, err
	}
	if flags.NArg() != 0 {
		return options{}, errors.New("unexpected arguments")
	}
	base, err := url.Parse(*rawURL)
	if err != nil || base.Scheme != "https" || base.Host == "" || (base.Path != "" && base.Path != "/") || base.RawQuery != "" {
		return options{}, errors.New("-url: want https://<host>")
	}
	base.Path = ""
	o := options{
		base:         base,
		env:          *env,
		tribe:        *slug,
		mailDomain:   *mailDomain,
		dkimSelector: *dkim,
		skipXFF:      *skipXFF,
		timing:       *timing,
		samples:      *samples,
		rounds:       *rounds,
	}
	if o.env == "" {
		o.env = "production"
		if strings.HasPrefix(base.Hostname(), "recette.") {
			o.env = "recette"
		}
	}
	if o.env != "production" && o.env != "recette" {
		return options{}, errors.New("-env: recette or production")
	}
	if o.mailDomain == "" {
		o.mailDomain = lastLabels(base.Hostname(), 2)
	}
	for m := range strings.SplitSeq(*members, ",") {
		if m = strings.TrimSpace(m); m != "" {
			o.members = append(o.members, m)
		}
	}
	if o.timing && (len(o.members) == 0 || *slug == "sonde") {
		return options{}, errors.New("-timing: -tribe <existing tribe> and -members <addresses> needed")
	}
	if o.samples < 1 || o.rounds < 1 {
		return options{}, errors.New("-samples and -rounds: 1 at least")
	}
	return o, nil
}

// lastLabels returns the last n labels of a host name.
func lastLabels(host string, n int) string {
	labels := strings.Split(host, ".")
	if len(labels) <= n {
		return host
	}
	return strings.Join(labels[len(labels)-n:], ".")
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	o, err := parse(args, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "probe: %v\n", err)
		return 2
	}
	p := &prober{
		options: o,
		out:     stdout,
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		dns: googleDNS,
	}
	if o.timing {
		fmt.Fprintf(stdout, "probe -timing, %s, %s\n", o.env, time.Now().Format("2006-01-02T15:04-07:00"))
		p.timing(ctx)
	} else {
		fmt.Fprintf(stdout, "probe, %s, %s\n", o.env, time.Now().Format("2006-01-02T15:04-07:00"))
		p.all(ctx)
	}
	fmt.Fprintf(stdout, "\n%d point(s) failed\n", p.failures)
	if p.failures > 0 {
		return 1
	}
	return 0
}

// prober runs the checks and writes their lines.
type prober struct {
	options
	out      io.Writer
	client   *http.Client
	dns      string
	failures int
}

func (p *prober) ok(format string, args ...any) {
	fmt.Fprintf(p.out, "ok    "+format+"\n", args...)
}

func (p *prober) fail(format string, args ...any) {
	fmt.Fprintf(p.out, "FAIL  "+format+"\n", args...)
	p.failures++
}

func (p *prober) info(format string, args ...any) {
	fmt.Fprintf(p.out, "info  "+format+"\n", args...)
}

// check writes ok or FAIL for a point.
func (p *prober) check(good bool, format string, args ...any) {
	if good {
		p.ok(format, args...)
	} else {
		p.fail(format, args...)
	}
}

func (p *prober) section(title string) {
	fmt.Fprintf(p.out, "\n%s\n", title)
}

func (p *prober) all(ctx context.Context) {
	p.section("TLS and HTTP")
	p.checkTLS(ctx)
	p.checkRedirect(ctx)
	p.section("Headers")
	p.checkSite(ctx)
	p.checkApp(ctx)
	p.checkHealthz(ctx)
	p.section("Limits of Caddy")
	p.checkBodyLimit(ctx)
	p.checkSlowRequests(ctx)
	p.section("Ports")
	p.checkPorts(ctx)
	if p.skipXFF {
		p.section("X-Forwarded-For")
		p.info("skipped (-skip-xff)")
	} else {
		p.section("X-Forwarded-For")
		p.checkXFF(ctx)
	}
	p.section("DNS, through a public resolver")
	p.checkDNS(ctx)
}

// url returns the address of a path of the instance.
func (p *prober) url(path string) string {
	u := *p.base
	u.Path = path
	return u.String()
}

// get sends a GET and returns the response with its body read, at most 1 MiB.
func (p *prober) get(ctx context.Context, rawURL string) (*http.Response, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", err
	}
	return p.do(req)
}

func (p *prober) do(req *http.Request) (*http.Response, string, error) {
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, "", err
	}
	return resp, string(body), nil
}

// errorText describes an error without the host name or an address of the instance.
func (p *prober) errorText(err error) string {
	text := err.Error()
	text = strings.ReplaceAll(text, p.base.Hostname(), "<host>")
	var ue *url.Error
	if errors.As(err, &ue) {
		text = strings.ReplaceAll(ue.Err.Error(), p.base.Hostname(), "<host>")
	}
	if strings.Contains(text, "dial tcp") {
		return "connection failed"
	}
	return text
}
