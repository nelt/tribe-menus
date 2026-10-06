// Package server routes HTTP requests to the front end and the public site.
package server

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/nelt/tribe-menus/internal/tribe"
)

const (
	tribesPrefix  = "/tribes/"
	indexTemplate = "index.html"

	contentSecurityPolicy = "default-src 'self'; base-uri 'self'; object-src 'none'; frame-ancestors 'none'; form-action 'self'"

	// repositoryURL is the public repository of the source code (ADR 0011).
	repositoryURL = "https://github.com/nelt/tribe-menus"
)

var (
	releasePattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)
	commitPattern  = regexp.MustCompile(`^[0-9a-f]{7,40}$`)
)

// Config holds what the server serves.
type Config struct {
	// Web is the built front end (web/dist).
	Web fs.FS
	// Site is the public site, served at the root. Nil in production, where Caddy serves it.
	Site fs.FS
	// Dev re-reads the index template on every request, so that the front end can be rebuilt while the server runs.
	Dev    bool
	Logger *slog.Logger
	// Tribes and Login serve the API; without them, only the front end is served.
	Tribes Tribes
	Login  *tribe.Login
	// Version and Commit of the binary (ADR 0012) give the link to its source code.
	Version string
	Commit  string
	// BehindProxy is true when the server listens on the socket passed by systemd, which only
	// Caddy can open: the IP of the client is then read in X-Forwarded-For (ADR 0006, point
	// 7; plan production, D3).
	BehindProxy bool
}

type server struct {
	web       fs.FS
	dev       bool
	logger    *slog.Logger
	sourceURL string
	// behindProxy: see Config.BehindProxy.
	behindProxy bool
	// index is parsed once at startup in production, nil in development.
	index *template.Template
}

// New returns the HTTP handler of the application.
func New(cfg Config) (http.Handler, error) {
	if cfg.Web == nil {
		return nil, errors.New("server: no front end")
	}
	s := &server{web: cfg.Web, dev: cfg.Dev, logger: cfg.Logger, sourceURL: sourceURL(cfg.Version, cfg.Commit), behindProxy: cfg.BehindProxy}
	if s.logger == nil {
		s.logger = slog.Default()
	}
	if !cfg.Dev {
		index, err := parseIndex(cfg.Web)
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("server: front end not built, run make build: %w", err)
		}
		if err != nil {
			return nil, err
		}
		s.index = index
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /tribes/{tribe}", redirectToTribeRoot)
	mux.HandleFunc("GET /tribes/{tribe}/{rest...}", s.serveApp)
	if cfg.Tribes != nil && cfg.Login != nil {
		(&api{server: s, tribes: cfg.Tribes, login: cfg.Login}).routes(mux)
	}
	if cfg.Site != nil {
		mux.Handle("GET /", http.FileServerFS(cfg.Site))
	}
	return withCommonHeaders(mux), nil
}

// sourceURL is the address of the source code of this version (ADR 0011, point 2): its tag
// for a release vX.Y.Z, otherwise its commit if known, otherwise the repository.
func sourceURL(version, commit string) string {
	switch {
	case releasePattern.MatchString(version):
		return repositoryURL + "/tree/" + version
	case commitPattern.MatchString(commit):
		return repositoryURL + "/tree/" + commit
	default:
		return repositoryURL
	}
}

func parseIndex(web fs.FS) (*template.Template, error) {
	source, err := fs.ReadFile(web, indexTemplate)
	if err != nil {
		return nil, fmt.Errorf("server: read %s: %w", indexTemplate, err)
	}
	index, err := template.New(indexTemplate).Parse(string(source))
	if err != nil {
		return nil, fmt.Errorf("server: parse %s: %w", indexTemplate, err)
	}
	return index, nil
}

func withCommonHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", contentSecurityPolicy)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "same-origin")
		if strings.HasPrefix(r.URL.Path, tribesPrefix) {
			h.Set("X-Robots-Tag", "noindex")
		}
		next.ServeHTTP(w, r)
	})
}

// healthz answers once the server is up, which happens only after every database
// has been migrated (ADR 0016, point 1.6). Caddy does not forward it.
func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = io.WriteString(w, "ok\n")
}

func tribeBase(tribe string) string {
	return tribesPrefix + url.PathEscape(tribe) + "/"
}

func redirectToTribeRoot(w http.ResponseWriter, r *http.Request) {
	target := tribeBase(r.PathValue("tribe"))
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, target, http.StatusMovedPermanently)
}

// serveApp serves a file of the front end if it exists, and the index page otherwise,
// so that the client-side router handles the path.
func (s *server) serveApp(w http.ResponseWriter, r *http.Request) {
	rest := r.PathValue("rest")
	if rest != "" && rest != indexTemplate {
		if !fs.ValidPath(rest) {
			http.NotFound(w, r)
			return
		}
		if info, err := fs.Stat(s.web, rest); err == nil && info.Mode().IsRegular() {
			http.ServeFileFS(w, r, s.web, rest)
			return
		}
		if path.Ext(rest) != "" {
			http.NotFound(w, r)
			return
		}
	}
	s.renderIndex(w, r.PathValue("tribe"))
}

func (s *server) renderIndex(w http.ResponseWriter, tribe string) {
	index := s.index
	if s.dev {
		var err error
		index, err = parseIndex(s.web)
		if errors.Is(err, fs.ErrNotExist) {
			http.Error(w, "Front en cours de construction, rechargez la page dans un instant.", http.StatusServiceUnavailable)
			return
		}
		if err != nil {
			s.fail(w, err)
			return
		}
	}

	var body bytes.Buffer
	data := struct{ Base, SourceURL string }{Base: tribeBase(tribe), SourceURL: s.sourceURL}
	if err := index.Execute(&body, data); err != nil {
		s.fail(w, fmt.Errorf("server: render %s: %w", indexTemplate, err))
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(body.Bytes())
}

func (s *server) fail(w http.ResponseWriter, err error) {
	s.logger.Error("request failed", "error", err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}
