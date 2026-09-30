package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func webFS() fstest.MapFS {
	return fstest.MapFS{
		"index.html":  {Data: []byte(`<base href="{{.Base}}">`)},
		"main.js":     {Data: []byte(`console.log("app");`)},
		"app.css":     {Data: []byte(`body{}`)},
		"assets/a.js": {Data: []byte(`// nested`)},
	}
}

func siteFS() fstest.MapFS {
	return fstest.MapFS{
		"index.html": {Data: []byte(`<title>Melting Tribe</title>`)},
		"robots.txt": {Data: []byte("Disallow: /tribes/\n")},
	}
}

func newHandler(t *testing.T, cfg Config) http.Handler {
	t.Helper()
	cfg.Logger = slog.New(slog.DiscardHandler)
	h, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return h
}

func get(h http.Handler, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func TestRoutes(t *testing.T) {
	cases := []struct {
		name         string
		target       string
		wantStatus   int
		wantLocation string
		wantBody     string
	}{
		{name: "tribe without trailing slash", target: "/tribes/demo", wantStatus: http.StatusMovedPermanently, wantLocation: "/tribes/demo/"},
		{name: "redirect keeps query", target: "/tribes/demo?x=1", wantStatus: http.StatusMovedPermanently, wantLocation: "/tribes/demo/?x=1"},
		{name: "redirect escapes identifier", target: "/tribes/les%20martin", wantStatus: http.StatusMovedPermanently, wantLocation: "/tribes/les%20martin/"},
		{name: "tribe root", target: "/tribes/demo/", wantStatus: http.StatusOK, wantBody: `<base href="/tribes/demo/">`},
		{name: "client-side route", target: "/tribes/demo/planning/2026-09-29", wantStatus: http.StatusOK, wantBody: `<base href="/tribes/demo/">`},
		{name: "index is rendered, not served raw", target: "/tribes/demo/index.html", wantStatus: http.StatusOK, wantBody: `<base href="/tribes/demo/">`},
		{name: "front-end file", target: "/tribes/demo/main.js", wantStatus: http.StatusOK, wantBody: `console.log("app");`},
		{name: "nested front-end file", target: "/tribes/demo/assets/a.js", wantStatus: http.StatusOK, wantBody: `// nested`},
		{name: "missing file with extension", target: "/tribes/demo/missing.js", wantStatus: http.StatusNotFound},
		{name: "encoded identifier", target: "/tribes/les%20martin/", wantStatus: http.StatusOK, wantBody: `<base href="/tribes/les%20martin/">`},
		{name: "identifier with markup", target: "/tribes/a%22%3E%3Cb/", wantStatus: http.StatusOK, wantBody: `<base href="/tribes/a%22%3E%3Cb/">`},
		{name: "identifier with slash", target: "/tribes/a%2Fb/", wantStatus: http.StatusOK, wantBody: `<base href="/tribes/a%2Fb/">`},
	}

	for _, dev := range []bool{false, true} {
		h := newHandler(t, Config{Web: webFS(), Dev: dev})
		for _, tc := range cases {
			t.Run(modeName(dev)+"/"+tc.name, func(t *testing.T) {
				rec := get(h, tc.target)
				if rec.Code != tc.wantStatus {
					t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
				}
				if got := rec.Header().Get("Location"); got != tc.wantLocation {
					t.Errorf("Location = %q, want %q", got, tc.wantLocation)
				}
				if body := rec.Body.String(); !strings.Contains(body, tc.wantBody) {
					t.Errorf("body = %q, want it to contain %q", body, tc.wantBody)
				}
				if got := rec.Header().Get("X-Robots-Tag"); got != "noindex" {
					t.Errorf("X-Robots-Tag = %q, want noindex", got)
				}
			})
		}
	}
}

func TestCommonHeaders(t *testing.T) {
	h := newHandler(t, Config{Web: webFS(), Site: siteFS(), Dev: true})
	want := map[string]string{
		"Content-Security-Policy": contentSecurityPolicy,
		"X-Content-Type-Options":  "nosniff",
		"Referrer-Policy":         "same-origin",
	}
	for _, target := range []string{"/", "/tribes/demo/", "/tribes/demo/main.js", "/tribes/demo/missing.js"} {
		t.Run(target, func(t *testing.T) {
			rec := get(h, target)
			for name, value := range want {
				if got := rec.Header().Get(name); got != value {
					t.Errorf("%s = %q, want %q", name, got, value)
				}
			}
		})
	}
}

func TestSite(t *testing.T) {
	cases := []struct {
		name       string
		cfg        Config
		target     string
		wantStatus int
		wantBody   string
	}{
		{name: "home page", cfg: Config{Web: webFS(), Site: siteFS(), Dev: true}, target: "/", wantStatus: http.StatusOK, wantBody: "<title>Melting Tribe</title>"},
		{name: "robots.txt", cfg: Config{Web: webFS(), Site: siteFS(), Dev: true}, target: "/robots.txt", wantStatus: http.StatusOK, wantBody: "Disallow: /tribes/"},
		{name: "no site in production", cfg: Config{Web: webFS()}, target: "/", wantStatus: http.StatusNotFound},
		{name: "front end not exposed at the root", cfg: Config{Web: webFS(), Site: siteFS(), Dev: true}, target: "/main.js", wantStatus: http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := get(newHandler(t, tc.cfg), tc.target)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if body := rec.Body.String(); !strings.Contains(body, tc.wantBody) {
				t.Errorf("body = %q, want it to contain %q", body, tc.wantBody)
			}
			if got := rec.Header().Get("X-Robots-Tag"); got != "" {
				t.Errorf("X-Robots-Tag = %q, want none outside /tribes/", got)
			}
		})
	}
}

func TestFrontEndNotBuilt(t *testing.T) {
	built := webFS()
	delete(built, "index.html")

	t.Run("production refuses to start", func(t *testing.T) {
		_, err := New(Config{Web: built, Logger: slog.New(slog.DiscardHandler)})
		if err == nil || !strings.Contains(err.Error(), "front end not built") {
			t.Fatalf("New error = %v, want front end not built", err)
		}
	})

	t.Run("development answers 503 until built", func(t *testing.T) {
		web := webFS()
		delete(web, "index.html")
		h := newHandler(t, Config{Web: web, Dev: true})

		rec := get(h, "/tribes/demo/")
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
		}
		if body := rec.Body.String(); !strings.Contains(body, "Front en cours de construction") {
			t.Errorf("body = %q", body)
		}

		web["index.html"] = &fstest.MapFile{Data: []byte(`rebuilt {{.Base}}`)}
		rec = get(h, "/tribes/demo/")
		if body, _ := io.ReadAll(rec.Body); rec.Code != http.StatusOK || string(body) != "rebuilt /tribes/demo/" {
			t.Errorf("after build: status = %d, body = %q", rec.Code, body)
		}
	})
}

func TestNoFrontEnd(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("New without front end: want an error")
	}
}

func modeName(dev bool) string {
	if dev {
		return "dev"
	}
	return "production"
}
