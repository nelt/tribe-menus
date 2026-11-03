// Command e2efixture prepares a tribe for a scenario @ui (ADR 0028): it creates the tribe
// in the data directory of the server of make e2e, with its members, and opens a session
// for each of them by the real login, in process. It prints the tokens in JSON, by address.
//
// Each scenario gets its own tribe: the scenarios do not see each other's dishes, and the
// limits of the code requests, by address and tribe, are not shared. The requests come
// from random IPv6 prefixes of the documentation range, so that the limit by IP, which all
// the browsers of the tests share on 127.0.0.1, is not touched (plan socle, D15). The
// sessions are used by the tests through a header, WebKit refusing the Secure cookie on
// http://localhost (plan socle, D14). Development only: the binary does not contain it.
//
// Usage: e2efixture -data dir slug email...
package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/netip"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/nelt/tribe-menus/internal/server"
	"github.com/nelt/tribe-menus/internal/storage"
	"github.com/nelt/tribe-menus/internal/tribe"
)

// userAgent is the device of the sessions: a phone, as the projects of Playwright.
const userAgent = "Mozilla/5.0 (Linux; Android 14; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0 Mobile Safari/537.36"

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("e2efixture", flag.ContinueOnError)
	flags.SetOutput(stderr)
	data := flags.String("data", "", "data directory of the server")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *data == "" || flags.NArg() < 2 {
		fmt.Fprintln(stderr, "usage: e2efixture -data dir slug email...")
		return 2
	}
	tokens, err := prepare(ctx, *data, flags.Arg(0), flags.Args()[1:])
	if err != nil {
		fmt.Fprintf(stderr, "e2efixture: %v\n", err)
		return 1
	}
	if err := json.NewEncoder(stdout).Encode(tokens); err != nil {
		fmt.Fprintf(stderr, "e2efixture: %v\n", err)
		return 1
	}
	return 0
}

// codes keeps the last code sent to each address.
type codes struct {
	mu   sync.Mutex
	sent map[string]string
}

func (c *codes) SendLoginCode(to, code string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sent[to] = code
}

func (c *codes) of(to string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sent[to]
}

type noAlerts struct{}

func (noAlerts) LogCodeRequestLimits(tribe.AlertReport)  {}
func (noAlerts) SendCodeRequestLimits(tribe.AlertReport) {}

var namePart = regexp.MustCompile(`^[^@]+`)

// prepare creates the tribe with its members, the first one first, and returns a session
// token for each.
func prepare(ctx context.Context, dir, slug string, emails []string) (map[string]string, error) {
	if _, err := tribe.ParseSlug(slug); err != nil {
		return nil, fmt.Errorf("slug %q: %w", slug, err)
	}
	migrations, err := storage.EmbeddedMigrations()
	if err != nil {
		return nil, err
	}
	store, err := storage.Open(ctx, dir, migrations)
	if err != nil {
		return nil, err
	}
	defer store.Close()

	now := time.Now()
	name := "Tribu " + slug
	err = store.CreateTribe(ctx, slug, name, func(ctx context.Context, db *sql.DB) error {
		s := tribe.NewStore(db)
		first, err := tribe.ParseEmail(emails[0])
		if err != nil {
			return err
		}
		if err := s.Initialize(ctx, name, first, namePart.FindString(emails[0]), now); err != nil {
			return err
		}
		founder, err := s.Member(ctx, first)
		if err != nil {
			return err
		}
		for _, e := range emails[1:] {
			email, err := tribe.ParseEmail(e)
			if err != nil {
				return err
			}
			if _, err := s.AddMember(ctx, email, namePart.FindString(e), founder.ID, now); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("create tribe %s: %w", slug, err)
	}

	sent := &codes{sent: map[string]string{}}
	login := &tribe.Login{
		RateLimit: tribe.NewRateLimitDB(store.RateLimit()),
		Mailer:    sent,
		Requests:  tribe.NewCodeRequests(len(emails)),
		Tribes:    server.TribeStores(store),
		Alerts:    noAlerts{},
		Logger:    slog.New(slog.DiscardHandler),
		Now:       time.Now,
	}
	processed := make(chan struct{})
	go func() {
		defer close(processed)
		login.ProcessCodeRequests(ctx)
	}()
	defer func() {
		login.Requests.Close()
		<-processed
	}()

	db, err := store.Tribe(ctx, slug)
	if err != nil {
		return nil, err
	}
	t := tribe.NewStore(db)
	tokens := map[string]string{}
	for _, email := range emails {
		ip, err := randomIP()
		if err != nil {
			return nil, err
		}
		request, err := login.RequestCode(ctx, slug, email, ip)
		if err != nil {
			return nil, fmt.Errorf("code request for %s: %w", email, err)
		}
		login.Requests.Wait()
		code := sent.of(email)
		if code == "" {
			return nil, errors.New("no code sent to " + email)
		}
		_, token, err := login.OpenSession(ctx, slug, t, email, code, request, tribe.DetectDevice(userAgent, false))
		if err != nil {
			return nil, fmt.Errorf("session of %s: %w", email, err)
		}
		tokens[email] = token
	}
	return tokens, nil
}

// randomIP returns an address of a random /64 of 2001:db8::/32, the range of the
// documentation: the limit by IP counts IPv6 addresses by /64.
func randomIP() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[4:8]); err != nil {
		return "", err
	}
	b[0], b[1], b[2], b[3], b[15] = 0x20, 0x01, 0x0d, 0xb8, 1
	return netip.AddrFrom16(b).String(), nil
}
