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
		"index.html":               {Data: []byte(`<base href="{{.Base}}"><meta name="source-url" content="{{.SourceURL}}">`)},
		"main.js":                  {Data: []byte(`console.log("app");`)},
		"app.css":                  {Data: []byte(`body{}`)},
		"assets/a.js":              {Data: []byte(`// nested`)},
		"app-K3JQ2X7A.css":         {Data: []byte(`body{}`)},
		"app-K3JQ2X7A.css.map":     {Data: []byte(`{}`)},
		"Figtree[wght]-YQNG.woff2": {Data: []byte(`font`)},
		"OFL-Figtree.txt":          {Data: []byte(`SIL Open Font License`)},
		"files.json":               {Data: []byte(`["Figtree[wght]-YQNG.woff2", "app-K3JQ2X7A.css"]`)},
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

func TestCacheHeaders(t *testing.T) {
	cases := []struct {
		name       string
		dev        bool
		target     string
		wantStatus int
		want       string
	}{
		{name: "listed file", target: "/tribes/demo/app-K3JQ2X7A.css", wantStatus: http.StatusOK, want: immutableCache},
		{name: "listed file with brackets", target: "/tribes/demo/Figtree%5Bwght%5D-YQNG.woff2", wantStatus: http.StatusOK, want: immutableCache},
		{name: "sourcemap", target: "/tribes/demo/app-K3JQ2X7A.css.map", wantStatus: http.StatusOK, want: revalidateCache},
		{name: "unlisted file", target: "/tribes/demo/OFL-Figtree.txt", wantStatus: http.StatusOK, want: revalidateCache},
		{name: "tribe root", target: "/tribes/demo/", wantStatus: http.StatusOK, want: revalidateCache},
		{name: "index page", target: "/tribes/demo/index.html", wantStatus: http.StatusOK, want: revalidateCache},
		{name: "client-side route", target: "/tribes/demo/planning", wantStatus: http.StatusOK, want: revalidateCache},
		{name: "list not served", target: "/tribes/demo/files.json", wantStatus: http.StatusNotFound},
		{name: "development: listed file", dev: true, target: "/tribes/demo/app-K3JQ2X7A.css", wantStatus: http.StatusOK, want: revalidateCache},
		{name: "development: list not served", dev: true, target: "/tribes/demo/files.json", wantStatus: http.StatusNotFound},
		{name: "health check", target: "/healthz", wantStatus: http.StatusOK, want: "no-store"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := get(newHandler(t, Config{Web: webFS(), Dev: tc.dev}), tc.target)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if got := rec.Header().Get("Cache-Control"); got != tc.want {
				t.Errorf("Cache-Control = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSourceURL(t *testing.T) {
	cases := []struct {
		name, version, commit, want string
	}{
		{name: "release", version: "v1.2.3", commit: "0123abc", want: repositoryURL + "/tree/v1.2.3"},
		{name: "pull request build", version: "pr-31", commit: "0123abc", want: repositoryURL + "/tree/0123abc"},
		{name: "full commit", version: "dev", commit: "84745e5b96261ae5f8c6c856e262fe78d1d6efdd", want: repositoryURL + "/tree/84745e5b96261ae5f8c6c856e262fe78d1d6efdd"},
		{name: "unknown commit", version: "dev", commit: "unknown", want: repositoryURL},
		{name: "no commit", version: "dev", commit: "", want: repositoryURL},
		{name: "not a release tag", version: "v1.2", commit: "unknown", want: repositoryURL},
		{name: "release with suffix", version: "v1.2.3-rc1", commit: "unknown", want: repositoryURL},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sourceURL(tc.version, tc.commit); got != tc.want {
				t.Errorf("sourceURL(%q, %q) = %q, want %q", tc.version, tc.commit, got, tc.want)
			}
		})
	}
}

func TestIndexHasSourceURL(t *testing.T) {
	for _, dev := range []bool{false, true} {
		t.Run(modeName(dev), func(t *testing.T) {
			h := newHandler(t, Config{Web: webFS(), Dev: dev, Version: "v1.2.3", Commit: "0123abc"})
			want := `<meta name="source-url" content="` + repositoryURL + `/tree/v1.2.3">`
			if body := get(h, "/tribes/demo/").Body.String(); !strings.Contains(body, want) {
				t.Errorf("body = %q, want it to contain %q", body, want)
			}
		})
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
	for _, missing := range []string{"index.html", "files.json"} {
		t.Run("production refuses to start without "+missing, func(t *testing.T) {
			web := webFS()
			delete(web, missing)
			_, err := New(Config{Web: web, Logger: slog.New(slog.DiscardHandler)})
			if err == nil || !strings.Contains(err.Error(), "front end not built") {
				t.Fatalf("New error = %v, want front end not built", err)
			}
		})
	}

	t.Run("production refuses an unreadable list", func(t *testing.T) {
		web := webFS()
		web["files.json"] = &fstest.MapFile{Data: []byte(`{"files": []}`)}
		if _, err := New(Config{Web: web, Logger: slog.New(slog.DiscardHandler)}); err == nil || !strings.Contains(err.Error(), "parse files.json") {
			t.Fatalf("New error = %v, want parse files.json", err)
		}
	})

	t.Run("development starts without the list", func(t *testing.T) {
		web := webFS()
		delete(web, "files.json")
		if rec := get(newHandler(t, Config{Web: web, Dev: true}), "/tribes/demo/main.js"); rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
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

func TestHealthz(t *testing.T) {
	for _, dev := range []bool{false, true} {
		t.Run(modeName(dev), func(t *testing.T) {
			rec := get(newHandler(t, Config{Web: webFS(), Dev: dev}), "/healthz")
			if rec.Code != http.StatusOK || rec.Body.String() != "ok\n" {
				t.Errorf("status = %d, body = %q", rec.Code, rec.Body.String())
			}
			if got := rec.Header().Get("Cache-Control"); got != "no-store" {
				t.Errorf("Cache-Control = %q, want no-store", got)
			}
		})
	}
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
