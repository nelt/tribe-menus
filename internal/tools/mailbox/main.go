// Command mailbox fetches the login code sent to the box of the tribe recette-auto, by POP3
// over TLS (plan recette-auto, D7; ADR 0032, point 8): the code of the most recent message
// addressed to the address of the box and dated after the request, polled every 5 seconds
// for 2 minutes at most. The messages fetched for the address are deleted.
//
// Usage:
//
//	go run ./internal/tools/mailbox -since <RFC 3339 time of the request> [-config <dir>]
//
// The directory (default: ~/.config/tribe-menus, the volume tribe-menus-recette of the Dev
// Container, D10) holds recette-auto.json (address of the box, server, port and account
// POP3) and recette-auto.pop3 (password, mode 0600). Nothing but the code is written to the
// standard output; neither the password nor anything of a message reaches an error.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}

// Files of the configuration directory (D10), also read by web/playwright.recette.config.ts.
const (
	configFile   = "recette-auto.json"
	passwordFile = "recette-auto.pop3"
)

// config is what the fetch reads from recette-auto.json; the URL of the tribe, in the same
// file, is for Playwright.
type config struct {
	Address string `json:"address"`
	POP3    struct {
		Host string `json:"host"`
		Port int    `json:"port"`
		User string `json:"user"`
	} `json:"pop3"`
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("mailbox", flag.ContinueOnError)
	flags.SetOutput(stderr)
	home, _ := os.UserHomeDir()
	dir := flags.String("config", filepath.Join(home, ".config", "tribe-menus"), "directory of "+configFile+" and "+passwordFile)
	rawSince := flags.String("since", "", "time of the code request, RFC 3339")
	timeout := flags.Duration("timeout", 2*time.Minute, "how long to wait for the code")
	interval := flags.Duration("interval", 5*time.Second, "time between two fetches")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	since, err := time.Parse(time.RFC3339Nano, *rawSince)
	if flags.NArg() != 0 || err != nil || *timeout <= 0 || *interval <= 0 {
		fmt.Fprintln(stderr, "mailbox: want -since <RFC 3339 time>, without other arguments")
		return 2
	}
	cfg, password, err := readConfig(*dir)
	if err != nil {
		fmt.Fprintln(stderr, "mailbox:", err)
		return 1
	}
	f := fetcher{
		address:  cfg.Address,
		user:     cfg.POP3.User,
		password: password,
		dial: func(ctx context.Context) (net.Conn, error) {
			return dialTLS(ctx, net.JoinHostPort(cfg.POP3.Host, strconv.Itoa(cfg.POP3.Port)), nil)
		},
		interval: *interval,
	}
	code, err := f.wait(ctx, since, *timeout)
	if err != nil {
		fmt.Fprintln(stderr, "mailbox:", err)
		return 1
	}
	fmt.Fprintln(stdout, code)
	return 0
}

// readConfig reads the configuration and the password, refused when readable by others.
func readConfig(dir string) (config, string, error) {
	var cfg config
	b, err := os.ReadFile(filepath.Join(dir, configFile))
	if err != nil {
		return cfg, "", fmt.Errorf("read %s: %w", configFile, err)
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, "", fmt.Errorf("read %s: %w", configFile, err)
	}
	if cfg.Address == "" || cfg.POP3.Host == "" || cfg.POP3.Port == 0 || cfg.POP3.User == "" {
		return cfg, "", fmt.Errorf("%s: address, pop3.host, pop3.port and pop3.user needed", configFile)
	}
	path := filepath.Join(dir, passwordFile)
	info, err := os.Stat(path)
	if err != nil {
		return cfg, "", fmt.Errorf("read %s: %w", passwordFile, err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return cfg, "", fmt.Errorf("%s: readable by others, want mode 0600", passwordFile)
	}
	b, err = os.ReadFile(path)
	if err != nil {
		return cfg, "", fmt.Errorf("read %s: %w", passwordFile, err)
	}
	password := strings.TrimRight(string(b), "\r\n")
	if password == "" {
		return cfg, "", fmt.Errorf("%s: empty", passwordFile)
	}
	return cfg, password, nil
}

// fetcher fetches the login code from a box.
type fetcher struct {
	address  string
	user     string
	password string
	dial     func(ctx context.Context) (net.Conn, error)
	interval time.Duration
}

// errLogin stops the polling: a refused password stays refused.
var errLogin = errors.New("login refused")

// wait fetches until a code dated after since is found, or timeout. A session that fails
// otherwise than by the login (connection cut, server unavailable) is tried again.
func (f fetcher) wait(ctx context.Context, since time.Time, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var last error
	for {
		code, err := f.fetch(ctx, since)
		switch {
		case err == nil:
			return code, nil
		case errors.Is(err, errLogin):
			return "", err
		case !errors.Is(err, errNoCode):
			last = err
		}
		select {
		case <-ctx.Done():
			if last != nil {
				return "", fmt.Errorf("no login code within %v, last error: %w", timeout, last)
			}
			return "", fmt.Errorf("no login code within %v", timeout)
		case <-time.After(f.interval):
		}
	}
}

// fetch runs one session: reads every message, deletes those addressed to the box, and
// returns the code of the most recent one dated after since, or errNoCode.
func (f fetcher) fetch(ctx context.Context, since time.Time) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, pollTimeout)
	defer cancel()
	conn, err := f.dial(ctx)
	if err != nil {
		return "", err
	}
	c, err := newPOP3(ctx, conn)
	if err != nil {
		_ = conn.Close() // the error of newPOP3 says what failed
		return "", err
	}
	// After QUIT, closing changes nothing; before, it leaves the box as it was.
	defer func() { _ = c.close() }()
	if err := c.login(f.user, f.password); errors.Is(err, errRefused) {
		return "", fmt.Errorf("%w: %w", errLogin, err)
	} else if err != nil {
		return "", err
	}
	numbers, err := c.list()
	if err != nil {
		return "", err
	}
	var messages []received
	for _, n := range numbers {
		raw, err := c.retr(n)
		if err != nil {
			return "", err
		}
		// A message that cannot be read is not a login code: it stays in the box.
		if m, err := parse(n, raw); err == nil {
			messages = append(messages, m)
		}
	}
	code, remove, found := pick(messages, f.address, since)
	for _, n := range remove {
		if err := c.dele(n); err != nil {
			return "", err
		}
	}
	if err := c.quit(); err != nil {
		return "", err
	}
	return code, found
}
