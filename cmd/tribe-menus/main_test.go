package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	data := t.TempDir()
	configFile := writeConfig(t, data, "localhost:0")
	invalidConfigFile := filepath.Join(data, "invalid.json")
	writeFile(t, invalidConfigFile, `{"addr": "localhost:0"}`)
	cases := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{name: "version", args: []string{"version"}, wantCode: 0, wantStdout: "tribe-menus dev (unknown)\n"},
		{name: "help", args: []string{"help"}, wantCode: 0, wantStdout: usage},
		{name: "no command", args: nil, wantCode: 2, wantStderr: "Usage: tribe-menus"},
		{name: "unknown command", args: []string{"frobnicate"}, wantCode: 2, wantStderr: `unknown command "frobnicate"`},
		{name: "unknown serve flag", args: []string{"serve", "-nope"}, wantCode: 2, wantStderr: "flag provided but not defined: -nope"},
		{name: "serve without mode", args: []string{"serve", "-data", data}, wantCode: 2, wantStderr: "want -config <file> (server mode) or -dev (development mode)"},
		{name: "serve with both modes", args: []string{"serve", "-dev", "-config", configFile}, wantCode: 2, wantStderr: "-dev and -config exclude each other"},
		{name: "serve with an argument", args: []string{"serve", "-dev", "extra"}, wantCode: 2, wantStderr: `unexpected argument "extra"`},
		{name: "addr with config", args: []string{"serve", "-config", configFile, "-addr", "localhost:0"}, wantCode: 2, wantStderr: "-addr is a flag of the development mode, refused with -config"},
		{name: "data with config", args: []string{"serve", "-config", configFile, "-data", data}, wantCode: 2, wantStderr: "-data is a flag of the development mode, refused with -config"},
		{name: "root with config", args: []string{"serve", "-config", configFile, "-root", "."}, wantCode: 2, wantStderr: "-root is a flag of the development mode, refused with -config"},
		{name: "mail file with config", args: []string{"serve", "-config", configFile, "-mail-file", filepath.Join(data, "mails.jsonl")}, wantCode: 2, wantStderr: "-mail-file is a flag of the development mode, refused with -config"},
		{name: "missing config file", args: []string{"serve", "-config", filepath.Join(data, "missing.json")}, wantCode: 2, wantStderr: "tribe-menus serve: config: open"},
		{name: "invalid config file", args: []string{"serve", "-config", invalidConfigFile}, wantCode: 2, wantStderr: `tribe-menus serve: config: unknown key "addr"`},
		{name: "invalid address", args: []string{"serve", "-dev", "-root", "testdata-missing", "-data", data, "-addr", "localhost:-1"}, wantCode: 1, wantStderr: "listen"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(tc.args, strings.NewReader(""), &stdout, &stderr)
			if code != tc.wantCode {
				t.Errorf("code = %d, want %d (stderr: %s)", code, tc.wantCode, stderr.String())
			}
			if tc.wantStdout != "" && stdout.String() != tc.wantStdout {
				t.Errorf("stdout = %q, want %q", stdout.String(), tc.wantStdout)
			}
			if !strings.Contains(stderr.String(), tc.wantStderr) {
				t.Errorf("stderr = %q, want it to contain %q", stderr.String(), tc.wantStderr)
			}
		})
	}
}

func TestServeDevAndShutdown(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "site", "index.html"), "<title>Melting Tribe</title>")
	writeFile(t, filepath.Join(root, "web", "dist", "index.html"), `<base href="{{.Base}}">`)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logs := &syncBuffer{}
	done := make(chan int, 1)
	data := filepath.Join(root, "data")
	go func() {
		done <- serve(ctx, []string{"-dev", "-root", root, "-data", data, "-addr", "localhost:0"}, io.Discard, logs)
	}()

	base := waitForAddress(t, logs)

	// The API answers, and the development mailer writes nothing for an unknown tribe.
	resp, err := http.Post(base+"/tribes/demo/api/login-codes", "application/json", strings.NewReader(`{"email":"alice@exemple.fr"}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("POST login-codes: status %d, want %d", resp.StatusCode, http.StatusAccepted)
	}

	for path, want := range map[string]string{
		"/":             "<title>Melting Tribe</title>",
		"/tribes/demo/": `<base href="/tribes/demo/">`,
		"/healthz":      "ok",
	} {
		resp, err := http.Get(base + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), want) {
			t.Errorf("GET %s: status %d, body %q", path, resp.StatusCode, body)
		}
	}

	if _, err := os.Stat(filepath.Join(data, "registry.db")); err != nil {
		t.Errorf("registry not created: %v", err)
	}

	cancel()
	select {
	case code := <-done:
		if code != 0 {
			t.Errorf("exit code = %d, want 0 (logs: %s)", code, logs.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop")
	}
}

// With -mail-file, the development mailer also writes each email to the file, which the
// end-to-end tests read (D6); the file is complete once the server has stopped (D11).
func TestServeDevMailFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "web", "dist", "index.html"), `<base href="{{.Base}}">`)
	data := filepath.Join(root, "data")
	if code := run([]string{"admin", "seed", "-data", data}, strings.NewReader(""), io.Discard, io.Discard); code != 0 {
		t.Fatalf("admin seed: exit code %d", code)
	}
	mailFile := filepath.Join(root, "mails.jsonl")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logs := &syncBuffer{}
	done := make(chan int, 1)
	go func() {
		done <- serve(ctx, []string{"-dev", "-root", root, "-data", data, "-addr", "localhost:0", "-mail-file", mailFile}, io.Discard, logs)
	}()
	base := waitForAddress(t, logs)

	resp, err := http.Post(base+"/tribes/demo/api/login-codes", "application/json", strings.NewReader(`{"email":"alice@exemple.fr"}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("POST login-codes: status %d, want %d", resp.StatusCode, http.StatusAccepted)
	}

	cancel()
	select {
	case code := <-done:
		if code != 0 {
			t.Errorf("exit code = %d, want 0 (logs: %s)", code, logs.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop")
	}
	content, err := os.ReadFile(mailFile)
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^\{"to":"alice@exemple.fr",.*\b\d{8}\b.*\}\n$`).Match(content) {
		t.Errorf("mail file = %q, want one message to alice@exemple.fr with the code", content)
	}
}

func TestAdmin(t *testing.T) {
	data := t.TempDir()
	cases := []struct {
		name       string
		args       []string
		stdin      string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{name: "no subcommand", args: []string{"admin"}, wantCode: 2, wantStderr: "Usage: tribe-menus admin init|seed"},
		{name: "unknown subcommand", args: []string{"admin", "frobnicate"}, wantCode: 2, wantStderr: "Usage: tribe-menus admin init|seed"},
		{name: "unknown flag", args: []string{"admin", "init", "-nope"}, wantCode: 2, wantStderr: "flag provided but not defined: -nope"},
		{
			name:       "init",
			args:       []string{"admin", "init", "-data", data, "-base-url", "https://meltingtribe.example"},
			stdin:      "Les Martin\nmartin\nalice@exemple.fr\n\n",
			wantStdout: "https://meltingtribe.example/tribes/martin/",
		},
		{
			name:       "init with default base URL",
			args:       []string{"admin", "init", "-data", data},
			stdin:      "Les Durand\ndurand\nalice@exemple.fr\n\n",
			wantStdout: "http://localhost:8080/tribes/durand/",
		},
		{name: "seed", args: []string{"admin", "seed", "-data", data}, wantStdout: "http://localhost:8080/tribes/demo/"},
		{name: "seed again", args: []string{"admin", "seed", "-data", data}, wantStdout: "existe déjà"},
		{name: "input closed", args: []string{"admin", "init", "-data", data}, stdin: "Les Leroy\n", wantCode: 1, wantStderr: "tribe-menus admin init: input closed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(tc.args, strings.NewReader(tc.stdin), &stdout, &stderr)
			if code != tc.wantCode {
				t.Errorf("code = %d, want %d (stderr: %s)", code, tc.wantCode, stderr.String())
			}
			if !strings.Contains(stdout.String(), tc.wantStdout) {
				t.Errorf("stdout = %q, want it to contain %q", stdout.String(), tc.wantStdout)
			}
			if !strings.Contains(stderr.String(), tc.wantStderr) {
				t.Errorf("stderr = %q, want it to contain %q", stderr.String(), tc.wantStderr)
			}
		})
	}
}

// A database newer than the binary cannot be migrated: the server must not start.
func TestServeRefusesUnmigratedDatabases(t *testing.T) {
	data := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(data, "registry.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("PRAGMA user_version = 999"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	code := run([]string{"serve", "-dev", "-data", data, "-addr", "localhost:0"}, strings.NewReader(""), io.Discard, &stderr)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	logs := stderr.String()
	if !strings.Contains(logs, "newer than this binary") {
		t.Errorf("logs = %q, want the migration error", logs)
	}
	if strings.Contains(logs, "server started") {
		t.Errorf("server started despite the failed migration: %q", logs)
	}
}

var startedPattern = regexp.MustCompile(`addr=(http://\S+)`)

func waitForAddress(t *testing.T, logs *syncBuffer) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if m := startedPattern.FindStringSubmatch(logs.String()); m != nil {
			return m[1]
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server did not start, logs: %s", logs.String())
	return ""
}

// writeConfig writes a config file of the server mode in dir, with its data directory
// in dir, and returns its path.
func writeConfig(t *testing.T, dir, listen string) string {
	t.Helper()
	content, err := json.Marshal(map[string]string{"data": filepath.Join(dir, "data"), "listen": listen, "baseURL": "https://tribes.example.org"})
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(dir, "config.json")
	writeFile(t, name, string(content))
	return name
}

func writeFile(t *testing.T, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// syncBuffer is a bytes.Buffer safe for concurrent use by the server and the test.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// TestServerTimeouts: a connection that stops sending its body, or stays idle after a
// response, is closed at the timeout, instead of holding a goroutine and a file descriptor
// until the shutdown fails.
func TestServerTimeouts(t *testing.T) {
	short := timeouts{readHeader: time.Second, read: 300 * time.Millisecond, write: time.Second, idle: 300 * time.Millisecond}
	cases := []struct {
		name    string
		request string
	}{
		{name: "body never sent", request: "POST / HTTP/1.1\r\nHost: localhost\r\nContent-Length: 100\r\n\r\n"},
		{name: "idle after a response", request: "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.ReadAll(r.Body) })
			srv := newHTTPServer(handler, nil, short)
			go func() { _ = srv.Serve(listener) }()
			t.Cleanup(func() { _ = srv.Close() })

			conn, err := net.Dial("tcp", listener.Addr().String())
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			if _, err := io.WriteString(conn, tc.request); err != nil {
				t.Fatal(err)
			}
			if err := conn.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
				t.Fatal(err)
			}
			// The server answers or not, then closes: reading ends with EOF, not the deadline.
			if _, err := io.Copy(io.Discard, conn); err != nil {
				t.Errorf("connection still open after the timeout: %v", err)
			}
		})
	}
}

func TestDefaultTimeouts(t *testing.T) {
	d := defaultTimeouts
	srv := newHTTPServer(http.NotFoundHandler(), nil, d)
	if srv.ReadHeaderTimeout != d.readHeader || srv.ReadTimeout != d.read || srv.WriteTimeout != d.write || srv.IdleTimeout != d.idle {
		t.Errorf("server timeouts %v, %v, %v, %v, want %+v", srv.ReadHeaderTimeout, srv.ReadTimeout, srv.WriteTimeout, srv.IdleTimeout, d)
	}
	for _, v := range []time.Duration{d.readHeader, d.read, d.write, d.idle} {
		if v <= 0 || v > time.Minute {
			t.Errorf("timeouts %+v: each one between 0 and a minute", d)
		}
	}
}

// TestShutdownWithSlowClient: a connection still reading its body at the end of the
// shutdown wait is closed, and the shutdown succeeds: a slow client cannot make a
// deployment fail (review of PR #40, point 1).
func TestShutdownWithSlowClient(t *testing.T) {
	defer func(d time.Duration) { shutdownTimeout = d }(shutdownTimeout)
	shutdownTimeout = 300 * time.Millisecond

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "web", "dist", "index.html"), `<base href="{{.Base}}">`)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logs := &syncBuffer{}
	done := make(chan int, 1)
	go func() {
		done <- serve(ctx, []string{"-dev", "-root", root, "-data", filepath.Join(root, "data"), "-addr", "localhost:0"}, io.Discard, logs)
	}()
	base := waitForAddress(t, logs)

	conn, err := net.Dial("tcp", strings.TrimPrefix(base, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := io.WriteString(conn, "POST /tribes/demo/api/login-codes HTTP/1.1\r\nHost: localhost\r\nContent-Length: 100\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond) // the request reaches its handler

	cancel()
	select {
	case code := <-done:
		if code != 0 {
			t.Errorf("exit code = %d, want 0 (logs: %s)", code, logs.String())
		}
		if !strings.Contains(logs.String(), "connections closed at the end of the shutdown wait") {
			t.Errorf("no log of the closed connections: %s", logs.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop")
	}
}
