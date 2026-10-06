package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"testing/fstest"
	"time"

	"github.com/nelt/tribe-menus/internal/tribe"
)

func TestSystemdSocketCount(t *testing.T) {
	const pid = 1234
	cases := []struct {
		name    string
		env     map[string]string
		want    int
		wantErr bool
	}{
		{name: "nothing passed", env: nil, want: 0},
		{name: "one socket", env: map[string]string{"LISTEN_PID": "1234", "LISTEN_FDS": "1"}, want: 1},
		{name: "two sockets", env: map[string]string{"LISTEN_PID": "1234", "LISTEN_FDS": "2"}, want: 2},
		{name: "for another process", env: map[string]string{"LISTEN_PID": "99", "LISTEN_FDS": "1"}, want: 0},
		{name: "invalid PID", env: map[string]string{"LISTEN_PID": "me", "LISTEN_FDS": "1"}, wantErr: true},
		{name: "count missing", env: map[string]string{"LISTEN_PID": "1234"}, wantErr: true},
		{name: "negative count", env: map[string]string{"LISTEN_PID": "1234", "LISTEN_FDS": "-1"}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := systemdSocketCount(func(k string) string { return tc.env[k] }, pid)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, want error: %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("count = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestListenRefusals(t *testing.T) {
	pid := os.Getpid()
	passed := func(n int) func(string) string {
		return func(k string) string {
			return map[string]string{"LISTEN_PID": strconv.Itoa(pid), "LISTEN_FDS": strconv.Itoa(n)}[k]
		}
	}
	none := func(string) string { return "" }
	cases := []struct {
		name    string
		addr    string
		getenv  func(string) string
		wantErr string
	}{
		{name: "TCP with a socket passed", addr: "localhost:0", getenv: passed(1), wantErr: "a socket is passed by systemd, but the config file listens on TCP"},
		{name: "systemd without socket", addr: "systemd", getenv: none, wantErr: "systemd socket: 0 passed, want 1"},
		{name: "systemd with two sockets", addr: "systemd", getenv: passed(2), wantErr: "systemd socket: 2 passed, want 1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l, err := listen(tc.addr, tc.getenv, pid)
			if err == nil {
				_ = l.Close()
				t.Fatal("listen succeeded")
			}
			if err.Error() != tc.wantErr {
				t.Errorf("error = %q, want %q", err, tc.wantErr)
			}
		})
	}
}

// passSocket passes the socket of the listener as systemd does, then closes the listener: the
// descriptor passed is a copy, whose only owner is listen, which closes it (review of PR #45,
// point 2).
func passSocket(t *testing.T, l interface {
	File() (*os.File, error)
	Close() error
}) {
	t.Helper()
	f, err := l.File()
	if err != nil {
		t.Fatal(err)
	}
	fd, err := syscall.Dup(int(f.Fd()))
	closeErr := f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	saved := systemdFirstFD
	systemdFirstFD = fd
	t.Cleanup(func() { systemdFirstFD = saved })
	t.Setenv("LISTEN_PID", strconv.Itoa(os.Getpid()))
	t.Setenv("LISTEN_FDS", "1")
}

// A unit that would run serve -dev is refused (plan revue-securite, finding 8).
func TestServeRefusesDevUnderSystemd(t *testing.T) {
	t.Setenv("LISTEN_PID", strconv.Itoa(os.Getpid()))
	t.Setenv("LISTEN_FDS", "1")
	var stderr bytes.Buffer
	code := run([]string{"serve", "-dev", "-data", t.TempDir(), "-addr", "localhost:0"}, strings.NewReader(""), io.Discard, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "-dev refused: a socket is passed by systemd") {
		t.Errorf("exit code %d, stderr %q", code, stderr.String())
	}
}

// fakeWeb replaces the embedded front end, not built when the Go tests run.
func fakeWeb(t *testing.T) {
	t.Helper()
	saved := embeddedWeb
	embeddedWeb = func() (fs.FS, error) {
		return fstest.MapFS{"index.html": {Data: []byte(`<base href="{{.Base}}">`)}}, nil
	}
	t.Cleanup(func() { embeddedWeb = saved })
}

// TestServeOnSystemdSocket: in server mode, on a Unix socket passed as systemd does, the
// server answers, logs JSON on the standard output, and leaves the file of the socket,
// which belongs to systemd, after it stops (ADR 0015, points 11 and 14).
func TestServeOnSystemdSocket(t *testing.T) {
	fakeWeb(t)
	dir := t.TempDir()
	socket := filepath.Join(dir, "tribe-menus.sock")
	unix, err := net.ListenUnix("unix", &net.UnixAddr{Name: socket, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	unix.SetUnlinkOnClose(false)
	passSocket(t, unix)
	t.Setenv("LISTEN_FDNAMES", "tribe-menus.socket")
	configFile := writeConfig(t, dir, "systemd")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stdout, stderr := &syncBuffer{}, &syncBuffer{}
	done := make(chan int, 1)
	go func() { done <- serve(ctx, []string{"-config", configFile}, stdout, stderr) }()

	started := waitForLog(t, stdout, "server started")
	for key, want := range map[string]any{"listen": "systemd", "addr": socket, "config": configFile, "version": "dev", "commit": "unknown", "dev": false} {
		if started[key] != want {
			t.Errorf("server started: %s = %v, want %v", key, started[key], want)
		}
	}
	for _, v := range systemdVariables {
		if _, ok := os.LookupEnv(v); ok {
			t.Errorf("%s still in the environment", v)
		}
	}

	client := &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}}
	resp, err := client.Get("http://tribes.example.org/healthz")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || string(body) != "ok\n" {
		t.Errorf("GET /healthz: status %d, body %q", resp.StatusCode, body)
	}

	// Behind the proxy, the IP of the client is the last of X-Forwarded-For (plan
	// production, D3): the limit by IP holds for it, not for the socket.
	requestCode := func(i int, forwarded string) int {
		req, err := http.NewRequest("POST", "http://tribes.example.org/tribes/demo/api/login-codes", strings.NewReader(`{"email":"personne-`+strconv.Itoa(i)+`@example.org"}`))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", forwarded)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		return resp.StatusCode
	}
	for i := range tribe.IPRequestLimit {
		if status := requestCode(i, "203.0.113.1, 198.51.100.9"); status != http.StatusAccepted {
			t.Fatalf("request %d: status %d", i+1, status)
		}
	}
	if status := requestCode(100, "198.51.100.9"); status != http.StatusTooManyRequests {
		t.Errorf("request over the limit of the client IP: status %d, want %d", status, http.StatusTooManyRequests)
	}
	if status := requestCode(101, "198.51.100.9, 198.51.100.10"); status != http.StatusAccepted {
		t.Errorf("request from another client IP: status %d, want %d", status, http.StatusAccepted)
	}
	client.CloseIdleConnections()

	cancel()
	select {
	case code := <-done:
		if code != 0 {
			t.Errorf("exit code = %d, want 0 (stdout: %s, stderr: %s)", code, stdout.String(), stderr.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop")
	}
	if _, err := os.Stat(socket); err != nil {
		t.Errorf("socket file removed at the shutdown: %v", err)
	}
	if stderr.String() != "" {
		t.Errorf("server mode wrote to the standard error: %s", stderr.String())
	}
	assertJSONLines(t, stdout.String())
}

// TestServeOnTCPFromConfig: in server mode, listen may be a TCP address, as on a station
// (plan production, step 14).
func TestServeOnTCPFromConfig(t *testing.T) {
	fakeWeb(t)
	dir := t.TempDir()
	configFile := writeConfig(t, dir, "localhost:0")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stdout := &syncBuffer{}
	done := make(chan int, 1)
	go func() { done <- serve(ctx, []string{"-config", configFile}, stdout, io.Discard) }()

	started := waitForLog(t, stdout, "server started")
	if started["listen"] != "tcp" {
		t.Errorf("listen = %v, want tcp", started["listen"])
	}
	addr, _ := started["addr"].(string)
	resp, err := http.Get(addr + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /healthz: status %d", resp.StatusCode)
	}
	if _, err := os.Stat(filepath.Join(dir, "data", "registry.db")); err != nil {
		t.Errorf("registry not created in the data directory of the config file: %v", err)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop")
	}
	assertJSONLines(t, stdout.String())
}

// waitForLog returns the first JSON record of logs with this message.
func waitForLog(t *testing.T, logs *syncBuffer, msg string) map[string]any {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		scanner := bufio.NewScanner(strings.NewReader(logs.String()))
		for scanner.Scan() {
			var record map[string]any
			if json.Unmarshal(scanner.Bytes(), &record) == nil && record["msg"] == msg {
				return record
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("no %q in the logs: %s", msg, logs.String())
	return nil
}

func assertJSONLines(t *testing.T, logs string) {
	t.Helper()
	for line := range strings.Lines(logs) {
		if !json.Valid([]byte(line)) {
			t.Errorf("log line is not JSON: %q", line)
		}
	}
}
