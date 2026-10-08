package acceptance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing/fstest"
	"time"

	"github.com/cucumber/godog"
	"github.com/nelt/tribe-menus/internal/admin"
	"github.com/nelt/tribe-menus/internal/alert"
	"github.com/nelt/tribe-menus/internal/mail"
	"github.com/nelt/tribe-menus/internal/server"
	"github.com/nelt/tribe-menus/internal/storage"
	"github.com/nelt/tribe-menus/internal/tribe"
)

const (
	baseURL = "http://localhost:8080"
	// origin is the address of the instance for the API requests: https, so that the
	// cookie jars keep and send the Secure session cookie (D13).
	originHost = "meltingtribe.test"
	origin     = "https://" + originHost
	userAgent  = "Mozilla/5.0 (X11; Linux x86_64; rv:131.0) Gecko/20100101 Firefox/131.0"
	// indexSource is the template of the index page; the scenarios exercise the API only.
	indexSource = "../web/src/index.html"
)

// today is the date of the scenarios: the test clock starts there.
var today = time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC)

// world is the state of one scenario: its own data directory, hence its own databases,
// its own clock and its own recorded emails.
type world struct {
	dir   string
	store *storage.Store
	// now is the test clock, read by the admin command and the login.
	now time.Time
	// stepType tells a context step from an outcome step, for the wordings used as both.
	stepType string

	// admin is the admin command started by the scenario, if any.
	admin *adminSession
	// tribe is the slug of the last tribe the scenario named or created.
	tribe tribe.Slug

	// The application, called directly without network (D13), and its emails.
	handler  http.Handler
	login    *tribe.Login
	outbox   *mail.Outbox
	recorder *mail.Recorder

	auth   loginState
	alerts alertState
}

// loginState is what the login scenarios remember from step to step.
type loginState struct {
	// declared are the tribes named by the scenario, by slug, created with their first member.
	declared map[string]string
	// home is the tribe of "je", and me its member: the first ones the scenario declares.
	home string
	me   tribe.Email

	devices map[string]*device
	device  *device // the current device
	nextIP  int

	// email is the address of the last code request; codes, the codes received for it.
	email tribe.Email
	codes []string
	// last is the response to the last action; opened, those of the last opened URL.
	last   *httptest.ResponseRecorder
	opened []*httptest.ResponseRecorder
	// signIn is the response that opened the last session.
	signIn *httptest.ResponseRecorder
	// foreign are the responses to the last attempts made from another device than the
	// one of the code request.
	foreign []*httptest.ResponseRecorder
	// mailMark is the number of emails sent before the last action; ignored, the emails of
	// reference requests made by the outcome steps.
	mailMark int
	ignored  map[int]bool
}

// device is a browser or an app: its own cookies and IP.
type device struct {
	jar *cookiejar.Jar
	ip  string
}

// registerSteps registers the step definitions, grouped by feature in the steps_*_test.go files.
func (w *world) registerSteps(sc *godog.ScenarioContext) {
	sc.StepContext().Before(func(ctx context.Context, st *godog.Step) (context.Context, error) {
		w.stepType = string(st.Type)
		return ctx, nil
	})
	w.registerAdministrationSteps(sc)
	w.registerLoginSteps(sc)
	w.registerCompartmentSteps(sc)
	w.registerAlertSteps(sc)
}

// reset prepares the world of a new scenario.
func (w *world) reset(dir string) {
	*w = world{dir: dir, now: today}
	w.recorder = &mail.Recorder{}
	w.outbox = mail.NewOutbox(w.recorder, tribe.LoginCodeValidity, slog.New(slog.DiscardHandler))
	w.auth = loginState{declared: map[string]string{}, devices: map[string]*device{}, ignored: map[int]bool{}}
}

func (w *world) clock() time.Time { return w.now }

// isContext reports whether the current step states a context (Étant donné), rather than
// an outcome (Alors).
func (w *world) isContext() bool {
	return w.stepType == "Context"
}

// openStore returns the databases of the scenario, opened on first use.
func (w *world) openStore(ctx context.Context) (*storage.Store, error) {
	if w.store != nil {
		return w.store, nil
	}
	migrations, err := storage.EmbeddedMigrations()
	if err != nil {
		return nil, err
	}
	store, err := storage.Open(ctx, filepath.Join(w.dir, "data"), migrations)
	if err != nil {
		return nil, fmt.Errorf("open databases: %w", err)
	}
	w.store = store
	return store, nil
}

// app returns the HTTP handler of the application, created on first use.
func (w *world) app(ctx context.Context) (http.Handler, error) {
	if w.handler != nil {
		return w.handler, nil
	}
	store, err := w.openStore(ctx)
	if err != nil {
		return nil, err
	}
	w.login = &tribe.Login{
		RateLimit: tribe.NewRateLimitDB(store.RateLimit()),
		Mailer:    w.outbox,
		Alerts:    &alert.Notifier{Logger: slog.New(slog.NewJSONHandler(&w.alerts.logs, nil)), Sender: w.outbox, To: adminEmail, Instance: origin},
		Now:       w.clock,
	}
	index, err := os.ReadFile(indexSource)
	if err != nil {
		return nil, fmt.Errorf("read the index page: %w", err)
	}
	handler, err := server.New(server.Config{
		// A front end as built, without files named after their content.
		Web:    fstest.MapFS{"index.html": {Data: index}, "files.json": {Data: []byte("[]")}},
		Logger: slog.New(slog.DiscardHandler),
		Tribes: store,
		Login:  w.login,
	})
	if err != nil {
		return nil, err
	}
	w.handler = handler
	return handler, nil
}

// adminCommand returns the admin command on the databases of the scenario.
func (w *world) adminCommand(ctx context.Context) (*admin.Command, error) {
	store, err := w.openStore(ctx)
	if err != nil {
		return nil, err
	}
	return &admin.Command{Store: store, BaseURL: baseURL, Now: w.clock}, nil
}

// tribeStore returns the store of the tribe with this slug.
func (w *world) tribeStore(ctx context.Context, slug string) (*tribe.Store, error) {
	store, err := w.openStore(ctx)
	if err != nil {
		return nil, err
	}
	db, err := store.Tribe(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("tribe %q: %w", slug, err)
	}
	return tribe.NewStore(db), nil
}

// newDevice returns a new device with its own IP, and makes it the current one.
func (w *world) newDevice() *device {
	w.auth.nextIP++
	d := &device{jar: newJar(), ip: fmt.Sprintf("192.0.2.%d", w.auth.nextIP)}
	w.auth.device = d
	return d
}

// otherDevice returns a new device with its own IP, without making it the current one.
func (w *world) otherDevice() *device {
	current := w.auth.device
	d := w.newDevice()
	w.auth.device = current
	return d
}

// currentDevice returns the current device, a new one if none.
func (w *world) currentDevice() *device {
	if w.auth.device == nil {
		return w.newDevice()
	}
	return w.auth.device
}

// namedDevice returns the device of a person ("le navigateur de Bruno"), and makes it current.
func (w *world) namedDevice(name string) *device {
	d, ok := w.auth.devices[name]
	if !ok {
		d = w.newDevice()
		w.auth.devices[name] = d
	}
	w.auth.device = d
	return d
}

// do sends a request from the device, as the front end would: same origin, JSON body,
// the cookies of the device. The cookies of the response go back to the device.
func (w *world) do(ctx context.Context, d *device, method, path string, body any) (*httptest.ResponseRecorder, error) {
	handler, err := w.app(ctx)
	if err != nil {
		return nil, err
	}
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = strings.NewReader(string(data))
	}
	req := httptest.NewRequestWithContext(ctx, method, origin+path, reader)
	req.RemoteAddr = net.JoinHostPort(d.ip, "49152")
	req.Header.Set("User-Agent", userAgent)
	if method != http.MethodGet {
		req.Header.Set("Origin", origin)
		req.Header.Set("Content-Type", "application/json")
	}
	for _, c := range d.jar.Cookies(req.URL) {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	d.jar.SetCookies(req.URL, rec.Result().Cookies())
	return rec, nil
}

func newJar() *cookiejar.Jar {
	jar, _ := cookiejar.New(nil) // never fails without options
	return jar
}

// originURL is the URL of a path of the instance, as the cookie jars see it.
func originURL(path string) *url.URL {
	return &url.URL{Scheme: "https", Host: originHost, Path: path}
}

// Kinds of cookies of a tribe, and the device of "je" when the scenario names it.
const (
	sessionCookie     = "session"
	codeRequestCookie = "code-request"
	myBrowser         = "mon navigateur"
)

// cookieName is the name of a cookie of the tribe (ADR 0024): the prefix __Host-, its kind,
// and the first 16 hexadecimal characters of the SHA-256 hash of the slug.
func cookieName(kind, slug string) string {
	sum := sha256.Sum256([]byte(slug))
	return "__Host-" + kind + "-" + hex.EncodeToString(sum[:])[:16]
}

func tribePath(slug, rest string) string {
	return "/tribes/" + url.PathEscape(slug) + "/" + rest
}

// sentMails waits for the emails being sent and returns those sent since the mark, but
// the reference ones.
func (w *world) sentMails(since int) []mail.Message {
	w.outbox.Wait()
	var sent []mail.Message
	for i, m := range w.recorder.Messages() {
		if i >= since && !w.auth.ignored[i] {
			sent = append(sent, m)
		}
	}
	return sent
}

// markMails marks the start of an action: the emails sent from now on are its own.
func (w *world) markMails() {
	w.outbox.Wait()
	w.auth.mailMark = len(w.recorder.Messages())
}

func (w *world) close() error {
	var errs []error
	if w.admin != nil {
		errs = append(errs, w.admin.close())
	}
	if w.outbox != nil {
		w.outbox.Wait()
	}
	if w.store != nil {
		errs = append(errs, w.store.Close())
	}
	return errors.Join(errs...)
}
