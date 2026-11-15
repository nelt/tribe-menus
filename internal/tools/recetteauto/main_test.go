package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nelt/tribe-menus/internal/tools/recetteconfig"
)

var now = time.Date(2026, 11, 15, 10, 30, 0, 0, time.UTC)

// setup writes a volume of the configuration for the server and a repository with the plan
// of v0.6.0, and returns their paths.
func setup(t *testing.T, srv *httptest.Server) (configDir, root string) {
	t.Helper()
	configDir, root = t.TempDir(), t.TempDir()
	config := `{"tribeURL": "` + srv.URL + `/tribes/recette-test/", "address": "a@example.org", "pop3": {"host": "h", "port": 995, "user": "u"}}`
	write(t, filepath.Join(configDir, recetteconfig.ConfigFile), config)
	write(t, filepath.Join(configDir, recetteconfig.PasswordFile), "secret")
	write(t, filepath.Join(root, "docs", "recettes", "v0.6.0.md"), "# Plan\n")
	return configDir, root
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		served     string
		status     int
		failing    bool
		noConfig   bool
		wantCode   int
		wantErr    string
		wantRun    bool
		wantExtras []string
		wantReuse  bool
	}{
		{name: "pull request served", args: []string{"-version", "v0.6.0", "-pr", "131"}, served: "pr-131", wantRun: true},
		{name: "version served", args: []string{"-version", "v0.6.0"}, served: "v0.6.0", wantRun: true},
		{name: "session reused", args: []string{"-version", "v0.6.0", "-pr", "131", "-reuse-session"}, served: "pr-131", wantRun: true, wantReuse: true},
		{name: "arguments for playwright", args: []string{"-version", "v0.6.0", "-pr", "131", "--", "--project", "phone-chromium"}, served: "pr-131", wantRun: true, wantExtras: []string{"--project", "phone-chromium"}},
		{name: "another pull request served", args: []string{"-version", "v0.6.0", "-pr", "131"}, served: "pr-130", wantCode: 1, wantErr: "serves pr-130, not pr-131"},
		{name: "pull request served, version asked", args: []string{"-version", "v0.6.0"}, served: "pr-131", wantCode: 1, wantErr: "serves pr-131, not v0.6.0"},
		{name: "site unavailable", args: []string{"-version", "v0.6.0"}, status: http.StatusServiceUnavailable, wantCode: 1, wantErr: "status 503"},
		{name: "no version on the site", args: []string{"-version", "v0.6.0"}, served: "", wantCode: 1, wantErr: "not found"},
		{name: "no plan of the acceptance run", args: []string{"-version", "v0.7.0"}, served: "v0.7.0", wantCode: 1, wantErr: "docs/recettes/v0.7.0.md needed"},
		{name: "volume not filled", args: []string{"-version", "v0.6.0"}, served: "v0.6.0", noConfig: true, wantCode: 1, wantErr: "volume tribe-menus-recette"},
		{name: "journey failed", args: []string{"-version", "v0.6.0"}, served: "v0.6.0", failing: true, wantRun: true, wantCode: 1, wantErr: "playwright: exit status 1"},
		{name: "version dev", args: []string{"-version", "dev"}, wantCode: 2},
		{name: "no version", args: []string{"-pr", "131"}, wantCode: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/" {
					http.NotFound(w, r)
					return
				}
				if tt.status != 0 {
					w.WriteHeader(tt.status)
				}
				io.WriteString(w, `<footer><a href="https://example.org/src">Code source de cette version (`+tt.served+`)</a></footer>`)
			}))
			defer srv.Close()
			configDir, root := setup(t, srv)
			if tt.noConfig {
				configDir = t.TempDir()
			}
			videos := t.TempDir()
			var ran bool
			var gotEnv, gotArgs []string
			var gotDir string
			e := env{client: srv.Client(), now: func() time.Time { return now }, playwright: func(_ context.Context, dir string, environ, args []string, stdout, _ io.Writer) error {
				ran, gotDir, gotEnv, gotArgs = true, dir, environ, args
				out := strings.TrimPrefix(environ[1], "RECETTE_OUTPUT_DIR=")
				write(t, filepath.Join(out, "controle.html"), "<!doctype html>")
				if tt.failing {
					return errors.New("exit status 1")
				}
				return nil
			}}
			var stdout, stderr bytes.Buffer
			args := append([]string{"run", "-config", configDir, "-videos", videos, "-root", root}, tt.args...)
			code := e.main(context.Background(), args, &stdout, &stderr)
			if code != tt.wantCode || !strings.Contains(stderr.String(), tt.wantErr) {
				t.Fatalf("exit %d, stderr %q; want %d, %q", code, stderr.String(), tt.wantCode, tt.wantErr)
			}
			if strings.Contains(stderr.String()+stdout.String(), "secret") {
				t.Errorf("output shows the password")
			}
			if ran != tt.wantRun {
				t.Fatalf("playwright ran: %v, want %v", ran, tt.wantRun)
			}
			if !ran {
				return
			}
			out := filepath.Join(videos, "v0.6.0", "20261115-103000")
			wantEnv := []string{"RECETTE_CONFIG_DIR=" + configDir, "RECETTE_OUTPUT_DIR=" + out, "RECETTE_PLAN=" + filepath.Join(root, "docs", "recettes", "v0.6.0.md")}
			if tt.wantReuse {
				wantEnv = append(wantEnv, "RECETTE_REUSE_SESSION=1")
			}
			if !slices.Equal(gotEnv, wantEnv) || gotDir != filepath.Join(root, "web") || !slices.Equal(gotArgs, tt.wantExtras) {
				t.Errorf("playwright in %s with %q, %q", gotDir, gotEnv, gotArgs)
			}
			if info, err := os.Stat(out); err != nil || info.Mode().Perm() != 0o700 {
				t.Errorf("directory of the run: %v, %v", info, err)
			}
			if !strings.Contains(stdout.String(), "Control page: "+filepath.Join(out, "controle.html")) {
				t.Errorf("stdout %q without the control page", stdout.String())
			}
		})
	}
}

func TestClean(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    []string
		wantErr int
	}{
		{name: "one version", args: []string{"-version", "v0.6.0"}, want: []string{"v0.5.0"}},
		{name: "a version without run", args: []string{"-version", "v0.7.0"}, want: []string{"v0.5.0", "v0.6.0"}},
		{name: "everything", want: []string{}},
		{name: "not a version", args: []string{"-version", "../x"}, want: []string{"v0.5.0", "v0.6.0"}, wantErr: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			videos := t.TempDir()
			write(t, filepath.Join(videos, "v0.5.0", "20261109-100000", "video.webm"), "x")
			write(t, filepath.Join(videos, "v0.6.0", "20261115-103000", "video.webm"), "x")
			var stdout, stderr bytes.Buffer
			if code := (env{}).main(context.Background(), append([]string{"clean", "-videos", videos}, tt.args...), &stdout, &stderr); code != tt.wantErr {
				t.Fatalf("exit %d: %s", code, stderr.String())
			}
			entries, err := os.ReadDir(videos)
			if err != nil {
				t.Fatalf("the volume itself is removed: %v", err)
			}
			got := []string{}
			for _, e := range entries {
				got = append(got, e.Name())
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("left %q, want %q", got, tt.want)
			}
		})
	}
}
