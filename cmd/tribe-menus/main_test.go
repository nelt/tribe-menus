package main

import (
	"bytes"
	"context"
	"io"
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
		{name: "invalid address", args: []string{"serve", "-dev", "-root", "testdata-missing", "-addr", "localhost:-1"}, wantCode: 1, wantStderr: "listen"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(tc.args, &stdout, &stderr)
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
	go func() { done <- serve(ctx, []string{"-dev", "-root", root, "-addr", "localhost:0"}, logs) }()

	base := waitForAddress(t, logs)
	for path, want := range map[string]string{
		"/":             "<title>Melting Tribe</title>",
		"/tribes/demo/": `<base href="/tribes/demo/">`,
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
