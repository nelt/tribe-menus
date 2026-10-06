package acceptance

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/nelt/tribe-menus/internal/storage"
	"github.com/nelt/tribe-menus/internal/tribe"
)

const sessionCookie = "session"

var loginCodePattern = regexp.MustCompile(`\b[0-9]{8}\b`)

// apiError is the body of an error response of the API.
type apiError struct {
	Error        string `json:"error"`
	AttemptsLeft *int   `json:"attemptsLeft"`
}

// apiSession is the body of a session response of the API.
type apiSession struct {
	Tribe struct {
		Name string `json:"name"`
	} `json:"tribe"`
	Member struct {
		Email string `json:"email"`
	} `json:"member"`
}

// registerLoginSteps: authentification.feature (ENF-01), and the tribes and the login of
// the other features. The starting states are set by the domain code and the test clock;
// the actions and the checks go through the API.
func (w *world) registerLoginSteps(sc *godog.ScenarioContext) {
	// Tribes, members and starting states.
	sc.Step(`^la tribu "([^"]*)" d'identifiant "([^"]*)"$`, w.declareTribe)
	sc.Step(`^les tribus "([^"]*)" d'identifiant "([^"]*)" et "([^"]*)" d'identifiant "([^"]*)"$`, func(first, firstSlug, second, secondSlug string) {
		w.declareTribe(first, firstSlug)
		w.declareTribe(second, secondSlug)
	})
	sc.Step(`^"([^"]*)" est membre révoqué de la tribu "([^"]*)"$`, w.givenRevokedMember)
	sc.Step(`^"([^"]*)" a demandé (\d+) codes pour la tribu "([^"]*)" dans le dernier quart d'heure$`, w.givenRequestsByEmail)
	sc.Step(`^(\d+) demandes de code depuis la même adresse IP ont été faites dans la dernière heure$`, w.givenRequestsByIP)
	sc.Step(`^(\d+) demandes de code depuis l'adresse IP "([^"]*)" ont été faites dans la dernière heure$`, w.givenRequestsFromIP)
	sc.Step(`^(\d+) demandes de code pour des membres de la tribu "([^"]*)" ont été faites dans la dernière heure$`, w.givenRequestsForMembers)
	sc.Step(`^(\d+) demandes de code pour des adresses qui ne sont pas membres de la tribu "([^"]*)" ont été faites dans la dernière heure$`, w.givenRequestsForOthers)
	sc.Step(`^je suis connecté à la tribu "([^"]*)" en tant que "([^"]*)"$`, func(ctx context.Context, slug, email string) error {
		return w.signIn(ctx, w.newDevice(), slug, email)
	})
	sc.Step(`^je me suis connecté à la tribu "([^"]*)" en tant que "([^"]*)" avec un code$`, func(ctx context.Context, slug, email string) error {
		return w.signIn(ctx, w.newDevice(), slug, email)
	})
	sc.Step(`^"([^"]*)" est connecté à la tribu "([^"]*)" dans son navigateur$`, func(ctx context.Context, email, slug string) error {
		return w.signIn(ctx, w.namedDevice(email), slug, email)
	})
	sc.Step(`^"([^"]*)" s'est connecté à la tribu "([^"]*)"$`, func(ctx context.Context, email, slug string) error {
		return w.signIn(ctx, w.newDevice(), slug, email)
	})
	sc.Step(`^je me suis connecté il y a (\d+) jours$`, w.signedInDaysAgo)
	sc.Step(`^j'ai utilisé l'application hier$`, w.usedYesterday)
	sc.Step(`^je n'ai pas utilisé l'application depuis plus de (\d+) jours$`, w.unusedFor)
	sc.Step(`^je ne suis connecté à aucune tribu$`, func() { w.newDevice() })
	sc.Step(`^plus de (\d+) minutes se sont écoulées depuis la demande$`, func(minutes int) {
		w.now = w.now.Add(time.Duration(minutes)*time.Minute + time.Second)
	})

	// Requests.
	sc.Step(`^j'ouvre l'URL de la tribu "([^"]*)" sans être connecté$`, func(ctx context.Context, slug string) error {
		return w.openURL(ctx, w.newDevice(), slug)
	})
	sc.Step(`^j'ouvre l'URL de la tribu "([^"]*)"$`, func(ctx context.Context, slug string) error {
		return w.openURL(ctx, w.currentDevice(), slug)
	})
	sc.Step(`^(?:Alice|Bruno|Chloé|David) ouvre l'URL de la tribu "([^"]*)" dans le même navigateur$`, func(ctx context.Context, slug string) error {
		return w.openURL(ctx, w.currentDevice(), slug)
	})
	sc.Step(`^je saisis "([^"]*)"$`, func(ctx context.Context, email string) error {
		return w.requestCode(ctx, w.currentDevice(), string(w.tribe), email)
	})
	sc.Step(`^"([^"]*)" demande un code pour la tribu "([^"]*)"$`, func(ctx context.Context, email, slug string) error {
		return w.requestCode(ctx, w.newDevice(), slug, email)
	})
	sc.Step(`^"([^"]*)" demande un code pour la tribu "([^"]*)" depuis l'adresse IP "([^"]*)"$`, func(ctx context.Context, email, slug, ip string) error {
		d := w.newDevice()
		d.ip = ip
		return w.requestCode(ctx, d, slug, email)
	})
	sc.Step(`^(?:elle|il) demande un nouveau code$`, func(ctx context.Context) error {
		return w.requestCode(ctx, w.newDevice(), w.auth.home, string(w.auth.email))
	})
	sc.Step(`^une nouvelle demande de code est faite depuis cette adresse IP, pour une autre adresse e-mail$`, func(ctx context.Context) error {
		d := w.newDevice()
		d.ip = sharedIP
		return w.requestCode(ctx, d, w.auth.home, string(w.auth.me))
	})
	sc.Step(`^une nouvelle demande de code est faite pour la tribu "([^"]*)", depuis une autre adresse IP$`, func(ctx context.Context, slug string) error {
		return w.requestCode(ctx, w.newDevice(), slug, string(w.auth.me))
	})
	sc.Step(`^j'ai demandé un code pour "([^"]*)"$`, func(ctx context.Context, email string) error {
		return w.requestAndReceive(ctx, w.newDevice(), w.auth.home, email)
	})
	sc.Step(`^j'ai demandé un nouveau code pour "([^"]*)"$`, func(ctx context.Context, email string) error {
		return w.requestAndReceive(ctx, w.currentDevice(), w.auth.home, email)
	})
	sc.Step(`^"([^"]*)" a demandé un code pour la tribu "([^"]*)"$`, w.givenCodeRequested)
	sc.Step(`^"([^"]*)" a reçu un code de connexion pour la tribu "([^"]*)"$`, func(ctx context.Context, email, slug string) error {
		return w.requestAndReceive(ctx, w.newDevice(), slug, email)
	})

	// Codes entered.
	sc.Step(`^je saisis (?:ce|le bon) code$`, func(ctx context.Context) error { return w.enterCode(ctx, w.currentDevice(), w.lastCode()) })
	sc.Step(`^(?:elle|il) saisit ce code$`, func(ctx context.Context) error { return w.enterCode(ctx, w.currentDevice(), w.lastCode()) })
	sc.Step(`^je saisis le premier code$`, func(ctx context.Context) error { return w.enterCode(ctx, w.currentDevice(), w.auth.codes[0]) })
	sc.Step(`^je saisis un code erroné$`, func(ctx context.Context) error { return w.enterWrongCode(ctx, 1) })
	sc.Step(`^je saisis un code erroné (\d+) fois$`, w.enterWrongCode)
	sc.Step(`^un code erroné est saisi (\d+) fois pour "([^"]*)" depuis un autre navigateur$`, w.enterWrongCodeElsewhere)
	sc.Step(`^un code erroné est saisi depuis le navigateur de la demande$`, func(ctx context.Context) error {
		return w.enterCode(ctx, w.currentDevice(), w.wrongCode())
	})
	sc.Step(`^je saisis à nouveau ce code sur un autre appareil$`, func(ctx context.Context) error {
		return w.enterCode(ctx, w.newDevice(), w.lastCode())
	})
	sc.Step(`^j'utilise l'application aujourd'hui$`, func(ctx context.Context) error {
		w.now = today
		return w.getSession(ctx)
	})

	// Outcomes.
	sc.Step(`^un code à 8 chiffres est envoyé à "([^"]*)"$`, w.codeSentTo)
	sc.Step(`^je suis connecté à la tribu "([^"]*)"$`, w.connectedTo)
	sc.Step(`^une session est ouverte pour cet appareil$`, w.sessionOpenForDevice)
	sc.Step(`^l'application affiche le même message que pour une adresse membre$`, w.sameAnswerAsMember)
	sc.Step(`^aucun e-mail n'est envoyé$`, w.noMailSinceAction)
	sc.Step(`^un message m'indique que l'adresse n'est pas valide$`, func() error { return w.lastError(http.StatusBadRequest, "invalid_email") })
	sc.Step(`^je ne suis pas connecté(?: sur cet autre appareil)?$`, w.notConnected)
	sc.Step(`^un message m'indique que le code est incorrect$`, func() error { return w.lastError(http.StatusBadRequest, "incorrect_code") })
	sc.Step(`^un message m'invite à demander un nouveau code$`, func() error { return w.lastError(http.StatusBadRequest, "new_code_needed") })
	sc.Step(`^l'application affiche « Réessayez dans quelques minutes »$`, func() error {
		return w.lastError(http.StatusTooManyRequests, "too_many_requests")
	})
	sc.Step(`^le code est invalidé$`, w.codeInvalidated)
	sc.Step(`^ces essais reçoivent l'invitation à demander un nouveau code$`, w.foreignAttemptsRefused)
	sc.Step(`^un message indique que le code est incorrect et qu'il reste (\d+) essais$`, w.incorrectCodeWithAttempts)
	sc.Step(`^même le bon code n'est plus accepté$`, func(ctx context.Context) error {
		if err := w.enterCode(ctx, w.currentDevice(), w.lastCode()); err != nil {
			return err
		}
		if err := w.lastError(http.StatusBadRequest, "new_code_needed"); err != nil {
			return err
		}
		return w.notConnected(ctx)
	})
	sc.Step(`^le second code est accepté$`, func(ctx context.Context) error {
		if err := w.enterCode(ctx, w.currentDevice(), w.auth.codes[1]); err != nil {
			return err
		}
		return w.connectedTo(ctx, w.auth.home)
	})
	sc.Step(`^je suis toujours connecté$`, func() error { return w.lastStatus(http.StatusOK) })
	sc.Step(`^ma session expire (\d+) jours après aujourd'hui$`, w.sessionExpiresAfter)
	sc.Step(`^je vois l'écran de connexion$`, func() error { return w.loginScreen("") })
	sc.Step(`^le cookie de session est marqué HttpOnly, Secure et SameSite=Lax$`, w.cookieAttributes)
	sc.Step(`^le jeton de session n'est pas stocké en clair côté serveur$`, w.tokenHashed)
}

// declareTribe names a tribe of the scenario; it is created with its first member.
func (w *world) declareTribe(name, slug string) {
	w.auth.declared[slug] = name
	if w.auth.home == "" {
		w.auth.home = slug
	}
	w.tribe = tribe.Slug(slug)
}

// ensureMember makes the address an active member of the tribe: the tribe is created with
// it as first member if needed, or it is added by the first member.
func (w *world) ensureMember(ctx context.Context, email, slug string) (tribe.Member, error) {
	store, err := w.openStore(ctx)
	if err != nil {
		return tribe.Member{}, err
	}
	if _, err := store.Tribe(ctx, slug); errors.Is(err, storage.ErrUnknownTribe) {
		name, ok := w.auth.declared[slug]
		if !ok {
			name = "Tribu " + slug
		}
		if err := w.initTribe(ctx, name, slug, email); err != nil {
			return tribe.Member{}, err
		}
	} else if err != nil {
		return tribe.Member{}, err
	}
	if w.auth.home == "" {
		w.auth.home = slug
	}
	if w.auth.me == "" && slug == w.auth.home {
		w.auth.me = tribe.Email(email)
	}

	t, err := w.tribeStore(ctx, slug)
	if err != nil {
		return tribe.Member{}, err
	}
	member, err := t.Member(ctx, tribe.Email(email))
	if !errors.Is(err, tribe.ErrUnknownMember) {
		return member, err
	}
	founder, err := w.founder(ctx, t)
	if err != nil {
		return tribe.Member{}, err
	}
	return t.AddMember(ctx, tribe.Email(email), "", founder.ID, w.now)
}

// founder returns the first member of the tribe, the one of its initialization.
func (w *world) founder(ctx context.Context, t *tribe.Store) (tribe.Member, error) {
	entries, err := t.AuditLog(ctx)
	if err != nil {
		return tribe.Member{}, err
	}
	if len(entries) == 0 || entries[0].Operation != tribe.TribeInitialized {
		return tribe.Member{}, fmt.Errorf("no initialization in the audit log %+v", entries)
	}
	return t.Member(ctx, entries[0].MemberEmail)
}

func (w *world) givenRevokedMember(ctx context.Context, email, slug string) error {
	member, err := w.ensureMember(ctx, email, slug)
	if err != nil {
		return err
	}
	t, err := w.tribeStore(ctx, slug)
	if err != nil {
		return err
	}
	return t.RevokeMember(ctx, member.ID, 0, w.now)
}

// sharedIP is the IP of the requests of the scenario on the limit by IP.
const sharedIP = "203.0.113.7"

// givenRequests makes n code requests through the domain, one a minute up to now.
func (w *world) givenRequests(ctx context.Context, n int, request func(i int) (slug, email, ip string)) error {
	if _, err := w.app(ctx); err != nil {
		return err
	}
	start := w.now
	for i := range n {
		w.now = start.Add(time.Duration(i-n) * time.Minute)
		slug, email, ip := request(i)
		var t *tribe.Store
		if found, err := w.tribeStore(ctx, slug); err == nil {
			t = found
		} else if !errors.Is(err, storage.ErrUnknownTribe) {
			return err
		}
		if _, err := w.login.RequestCode(ctx, slug, t, email, ip); err != nil {
			return fmt.Errorf("request %d: %w", i+1, err)
		}
	}
	w.now = start
	return nil
}

func (w *world) givenRequestsByEmail(ctx context.Context, email string, n int, slug string) error {
	w.auth.email = tribe.Email(email)
	return w.givenRequests(ctx, n, func(int) (string, string, string) { return slug, email, "198.51.100.1" })
}

func (w *world) givenRequestsByIP(ctx context.Context, n int) error {
	return w.givenRequestsFromIP(ctx, n, sharedIP)
}

func (w *world) givenRequestsFromIP(ctx context.Context, n int, ip string) error {
	return w.givenRequests(ctx, n, func(i int) (string, string, string) {
		return w.auth.home, fmt.Sprintf("personne-%d@exemple.fr", i), ip
	})
}

// givenRequestsForMembers makes n requests for n distinct members of the tribe, added for
// it, from a few IPs: no limit by address nor by IP is reached.
func (w *world) givenRequestsForMembers(ctx context.Context, n int, slug string) error {
	for i := range n {
		if _, err := w.ensureMember(ctx, fmt.Sprintf("membre-%d@exemple.fr", i), slug); err != nil {
			return err
		}
	}
	return w.givenRequests(ctx, n, func(i int) (string, string, string) {
		return slug, fmt.Sprintf("membre-%d@exemple.fr", i), fmt.Sprintf("198.51.100.%d", i%(tribe.IPRequestLimit-1)+1)
	})
}

// givenRequestsForOthers makes n requests for n distinct addresses that are not members of
// the tribe, from a few IPs.
func (w *world) givenRequestsForOthers(ctx context.Context, n int, slug string) error {
	return w.givenRequests(ctx, n, func(i int) (string, string, string) {
		return slug, fmt.Sprintf("personne-%d@exemple.fr", i), fmt.Sprintf("198.51.100.%d", i%(tribe.IPRequestLimit-1)+1)
	})
}

// requestCode asks for a code from the device, as the login screen does.
func (w *world) requestCode(ctx context.Context, d *device, slug, email string) error {
	w.markMails()
	rec, err := w.do(ctx, d, http.MethodPost, tribePath(slug, "api/login-codes"), map[string]string{"email": email})
	if err != nil {
		return err
	}
	w.auth.last = rec
	w.auth.email = tribe.Email(email)
	w.tribe = tribe.Slug(slug)
	return nil
}

// receiveCode reads the code sent to the address by the last request.
func (w *world) receiveCode(email string) error {
	var codes []string
	for _, m := range w.sentMails(w.auth.mailMark) {
		if m.To == email {
			codes = append(codes, loginCodePattern.FindAllString(m.Body, -1)...)
		}
	}
	if len(codes) != 1 {
		return fmt.Errorf("want one code of 8 digits sent to %s, got %q", email, codes)
	}
	w.auth.codes = append(w.auth.codes, codes[0])
	return nil
}

func (w *world) requestAndReceive(ctx context.Context, d *device, slug, email string) error {
	if err := w.requestCode(ctx, d, slug, email); err != nil {
		return err
	}
	if err := w.lastStatus(http.StatusAccepted); err != nil {
		return err
	}
	return w.receiveCode(email)
}

func (w *world) lastCode() string {
	if len(w.auth.codes) == 0 {
		return ""
	}
	return w.auth.codes[len(w.auth.codes)-1]
}

// enterCode enters the code for the address of the last request, in its tribe.
func (w *world) enterCode(ctx context.Context, d *device, code string) error {
	if code == "" {
		return errors.New("no code received")
	}
	rec, err := w.do(ctx, d, http.MethodPost, tribePath(string(w.tribe), "api/sessions"), map[string]any{
		"email": w.auth.email, "code": code, "installedApp": false,
	})
	if err != nil {
		return err
	}
	w.auth.last = rec
	if rec.Code == http.StatusCreated {
		w.auth.signIn = rec
	}
	return nil
}

// wrongCode returns a code that differs from the last one received, if any.
func (w *world) wrongCode() string {
	right, err := strconv.Atoi(w.lastCode())
	if err != nil {
		return "12345678"
	}
	return fmt.Sprintf("%08d", (right+1)%100_000_000)
}

// enterWrongCode enters n times a code that differs from the one received.
func (w *world) enterWrongCode(ctx context.Context, n int) error {
	if w.lastCode() == "" {
		return errors.New("no code received")
	}
	wrong := w.wrongCode()
	for range n {
		if err := w.enterCode(ctx, w.currentDevice(), wrong); err != nil {
			return err
		}
		if err := w.lastError(http.StatusBadRequest, "incorrect_code"); err != nil {
			return err
		}
	}
	return nil
}

// givenCodeRequested asks for a code from a new device, which becomes the current one,
// and keeps the code if one was sent: the address may be a member's or not.
func (w *world) givenCodeRequested(ctx context.Context, email, slug string) error {
	if err := w.requestCode(ctx, w.newDevice(), slug, email); err != nil {
		return err
	}
	if err := w.lastStatus(http.StatusAccepted); err != nil {
		return err
	}
	for _, m := range w.sentMails(w.auth.mailMark) {
		if m.To == email {
			return w.receiveCode(email)
		}
	}
	return nil
}

// enterWrongCodeElsewhere enters n times a wrong code for the address, from a device that
// did not request the code, as anyone knowing the address could.
func (w *world) enterWrongCodeElsewhere(ctx context.Context, n int, email string) error {
	other := w.otherDevice()
	w.auth.foreign = nil
	for range n {
		rec, err := w.do(ctx, other, http.MethodPost, tribePath(string(w.tribe), "api/sessions"), map[string]any{
			"email": email, "code": w.wrongCode(), "installedApp": false,
		})
		if err != nil {
			return err
		}
		w.auth.foreign = append(w.auth.foreign, rec)
	}
	return nil
}

func (w *world) foreignAttemptsRefused() error {
	if len(w.auth.foreign) == 0 {
		return errors.New("no attempt from another device")
	}
	for i, rec := range w.auth.foreign {
		if rec.Code != http.StatusBadRequest || strings.TrimSpace(rec.Body.String()) != `{"error":"new_code_needed"}` {
			return fmt.Errorf("attempt %d: status %d, body %s; want new_code_needed", i+1, rec.Code, rec.Body)
		}
	}
	return nil
}

func (w *world) incorrectCodeWithAttempts(attempts int) error {
	if err := w.lastError(http.StatusBadRequest, "incorrect_code"); err != nil {
		return err
	}
	e, err := w.lastAPIError()
	if err != nil {
		return err
	}
	if e.AttemptsLeft == nil || *e.AttemptsLeft != attempts {
		return fmt.Errorf("answer %+v, want %d attempts left", e, attempts)
	}
	return nil
}

// signIn signs in from the device: code requested, received and entered.
func (w *world) signIn(ctx context.Context, d *device, slug, email string) error {
	if err := w.requestAndReceive(ctx, d, slug, email); err != nil {
		return err
	}
	if err := w.enterCode(ctx, d, w.lastCode()); err != nil {
		return err
	}
	return w.lastStatus(http.StatusCreated)
}

func (w *world) signedInDaysAgo(ctx context.Context, days int) error {
	w.now = today.AddDate(0, 0, -days)
	defer func() { w.now = today }()
	return w.signIn(ctx, w.newDevice(), w.auth.home, string(w.auth.me))
}

func (w *world) usedYesterday(ctx context.Context) error {
	w.now = today.AddDate(0, 0, -1)
	defer func() { w.now = today }()
	if err := w.getSession(ctx); err != nil {
		return err
	}
	return w.lastStatus(http.StatusOK)
}

func (w *world) unusedFor(ctx context.Context, days int) error {
	w.now = today.AddDate(0, 0, -days).Add(-time.Minute)
	defer func() { w.now = today }()
	return w.signIn(ctx, w.newDevice(), w.auth.home, string(w.auth.me))
}

// getSession asks the API for the session of the current device, as the front end does
// when it starts.
func (w *world) getSession(ctx context.Context) error {
	rec, err := w.do(ctx, w.currentDevice(), http.MethodGet, tribePath(string(w.tribe), "api/session"), nil)
	if err != nil {
		return err
	}
	w.auth.last = rec
	return nil
}

// openURL opens the URL of the tribe from the device: the page, then the session, as the
// front end does when it starts. The cookies of the device's other tribes are also sent to
// this tribe, as a client ignoring their Path would: the server must find nothing with them.
func (w *world) openURL(ctx context.Context, d *device, slug string) error {
	w.tribe = tribe.Slug(slug)
	w.auth.opened = nil
	for _, rest := range []string{"", "api/session"} {
		rec, err := w.do(ctx, d, http.MethodGet, tribePath(slug, rest), nil)
		if err != nil {
			return err
		}
		w.auth.opened = append(w.auth.opened, rec)
	}
	w.auth.last = w.auth.opened[len(w.auth.opened)-1]

	for otherSlug := range w.auth.declared {
		if otherSlug == slug {
			continue
		}
		cookies := d.jar.Cookies(originURL(tribePath(otherSlug, "")))
		if len(cookies) == 0 {
			continue
		}
		foreign := &device{jar: newJar(), ip: d.ip}
		foreign.jar.SetCookies(originURL(tribePath(slug, "")), cookies)
		rec, err := w.do(ctx, foreign, http.MethodGet, tribePath(slug, "api/session"), nil)
		if err != nil {
			return err
		}
		w.auth.opened = append(w.auth.opened, rec)
	}
	return nil
}

func (w *world) codeSentTo(email string) error {
	if err := w.lastStatus(http.StatusAccepted); err != nil {
		return err
	}
	return w.receiveCode(email)
}

// connectedTo checks that the current device has a session in the tribe, which gives the
// name of the tribe as stored in its database.
func (w *world) connectedTo(ctx context.Context, slug string) error {
	if err := w.lastStatus(http.StatusCreated); err != nil {
		return err
	}
	rec, err := w.do(ctx, w.currentDevice(), http.MethodGet, tribePath(slug, "api/session"), nil)
	if err != nil {
		return err
	}
	if rec.Code != http.StatusOK {
		return fmt.Errorf("GET session: status %d, body %s", rec.Code, rec.Body)
	}
	var got apiSession
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		return err
	}
	t, err := w.tribeStore(ctx, slug)
	if err != nil {
		return err
	}
	name, err := t.Name(ctx)
	if err != nil {
		return err
	}
	if got.Tribe.Name != name || got.Member.Email != string(w.auth.email) {
		return fmt.Errorf("session of %s in %q, want %s in %q", got.Member.Email, got.Tribe.Name, w.auth.email, name)
	}
	return nil
}

func (w *world) sessionOpenForDevice(ctx context.Context) error {
	t, err := w.tribeStore(ctx, string(w.tribe))
	if err != nil {
		return err
	}
	member, err := t.Member(ctx, w.auth.email)
	if err != nil {
		return err
	}
	sessions, err := t.Sessions(ctx, member.ID)
	if err != nil {
		return err
	}
	if len(sessions) != 1 {
		return fmt.Errorf("%d sessions for %s, want 1", len(sessions), w.auth.email)
	}
	cookies := w.currentDevice().jar.Cookies(originURL(tribePath(string(w.tribe), "")))
	if len(cookies) != 1 || cookies[0].Name != sessionCookie {
		return fmt.Errorf("cookies of the device: %v, want the session cookie", cookies)
	}
	return nil
}

// sameAnswerAsMember compares the answer to the last request with the answer to the same
// request for an active member of the tribe, made from another device. The code sent to
// the member is not counted by the next steps.
func (w *world) sameAnswerAsMember(ctx context.Context) error {
	got := w.auth.last
	mark := len(w.sentMails(0))
	member := string(w.auth.me)
	if _, err := w.ensureMember(ctx, member, string(w.tribe)); err != nil {
		return err
	}
	reference, err := w.do(ctx, w.newDevice(), http.MethodPost, tribePath(string(w.tribe), "api/login-codes"), map[string]string{"email": member})
	if err != nil {
		return err
	}
	w.outbox.Wait()
	for i := mark; i < len(w.recorder.Messages()); i++ {
		w.auth.ignored[i] = true
	}
	if a, b := answer(got), answer(reference); a != b {
		return fmt.Errorf("answer %q, want the answer for a member %q", a, b)
	}
	return nil
}

// answer is what the client sees of a response: status, headers that matter, cookies but
// their random value, body.
func answer(rec *httptest.ResponseRecorder) string {
	h := rec.Header()
	var cookies []string
	for _, c := range rec.Result().Cookies() {
		c.Value = strings.Repeat("v", len(c.Value))
		cookies = append(cookies, c.String())
	}
	return strings.Join([]string{strconv.Itoa(rec.Code), h.Get("Content-Type"), h.Get("Cache-Control"), strings.Join(cookies, ", "), rec.Body.String()}, " | ")
}

func (w *world) noMailSinceAction() error {
	if sent := w.sentMails(w.auth.mailMark); len(sent) != 0 {
		return fmt.Errorf("%d emails sent, to %s first", len(sent), sent[0].To)
	}
	return nil
}

// notConnected checks that the last action opened no session, and that the device has none.
func (w *world) notConnected(ctx context.Context) error {
	if w.auth.last != nil && w.auth.last.Code == http.StatusCreated {
		return fmt.Errorf("a session was opened: %s", w.auth.last.Body)
	}
	last := w.auth.last
	defer func() { w.auth.last = last }()
	if err := w.getSession(ctx); err != nil {
		return err
	}
	return w.lastStatus(http.StatusUnauthorized)
}

func (w *world) lastStatus(status int) error {
	if w.auth.last == nil {
		return errors.New("no request made")
	}
	if w.auth.last.Code != status {
		return fmt.Errorf("status %d, want %d; body %s", w.auth.last.Code, status, w.auth.last.Body)
	}
	return nil
}

func (w *world) lastAPIError() (apiError, error) {
	var e apiError
	if w.auth.last == nil {
		return e, errors.New("no request made")
	}
	if err := json.Unmarshal(w.auth.last.Body.Bytes(), &e); err != nil {
		return e, fmt.Errorf("body %q: %w", w.auth.last.Body, err)
	}
	return e, nil
}

func (w *world) lastError(status int, code string) error {
	if err := w.lastStatus(status); err != nil {
		return err
	}
	e, err := w.lastAPIError()
	if err != nil {
		return err
	}
	if e.Error != code {
		return fmt.Errorf("error %q, want %q", e.Error, code)
	}
	return nil
}

// codeInvalidated: the last attempt left none, and the code is gone from the database.
func (w *world) codeInvalidated(ctx context.Context) error {
	e, err := w.lastAPIError()
	if err != nil {
		return err
	}
	if e.Error != "incorrect_code" || e.AttemptsLeft == nil || *e.AttemptsLeft != 0 {
		return fmt.Errorf("answer %+v, want an incorrect code with no attempt left", e)
	}
	store, err := w.openStore(ctx)
	if err != nil {
		return err
	}
	db, err := store.Tribe(ctx, string(w.tribe))
	if err != nil {
		return err
	}
	var n int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM login_codes").Scan(&n); err != nil {
		return err
	}
	if n != 0 {
		return fmt.Errorf("%d login codes left", n)
	}
	return nil
}

// sessionExpiresAfter checks the renewed cookie and the session in the database.
func (w *world) sessionExpiresAfter(ctx context.Context, days int) error {
	want := w.now.AddDate(0, 0, days)
	cookie := findCookie(w.auth.last)
	if cookie == nil || cookie.MaxAge != days*24*60*60 || !cookie.Expires.Equal(want) {
		return fmt.Errorf("cookie %+v, want an expiry %d days after %v", cookie, days, w.now)
	}
	t, err := w.tribeStore(ctx, string(w.tribe))
	if err != nil {
		return err
	}
	member, err := t.Member(ctx, w.auth.me)
	if err != nil {
		return err
	}
	sessions, err := t.Sessions(ctx, member.ID)
	if err != nil {
		return err
	}
	if len(sessions) != 1 || !sessions[0].ExpiresAt.Equal(want) {
		return fmt.Errorf("sessions %+v, want one expiring on %v", sessions, want)
	}
	return nil
}

// loginScreen checks that the opened URL shows the login screen: the page of the
// application, and no session. With a slug, the page is the one of this tribe.
func (w *world) loginScreen(slug string) error {
	if len(w.auth.opened) < 2 {
		return errors.New("no URL opened")
	}
	page := w.auth.opened[0]
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "<mt-app>") {
		return fmt.Errorf("page: status %d, body %s", page.Code, page.Body)
	}
	if slug != "" && !strings.Contains(page.Body.String(), `<base href="`+tribePath(slug, "")+`">`) {
		return fmt.Errorf("page is not the one of the tribe %s: %s", slug, page.Body)
	}
	for _, rec := range w.auth.opened[1:] {
		if rec.Code != http.StatusUnauthorized {
			return fmt.Errorf("session: status %d, body %s; want none", rec.Code, rec.Body)
		}
	}
	return nil
}

func (w *world) cookieAttributes() error {
	cookie := findCookie(w.auth.signIn)
	if cookie == nil {
		return errors.New("no session cookie set")
	}
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
		return fmt.Errorf("cookie %s", cookie)
	}
	return nil
}

// tokenHashed checks that no file of the data directory contains the token, and that the
// sessions hold its SHA-256 hash.
func (w *world) tokenHashed(ctx context.Context) error {
	cookie := findCookie(w.auth.signIn)
	if cookie == nil {
		return errors.New("no session cookie set")
	}
	token := []byte(cookie.Value)
	err := filepath.WalkDir(w.dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(data, token) {
			return fmt.Errorf("%s contains the session token in clear", path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	store, err := w.openStore(ctx)
	if err != nil {
		return err
	}
	db, err := store.Tribe(ctx, string(w.tribe))
	if err != nil {
		return err
	}
	sum := sha256.Sum256(token)
	var n int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM sessions WHERE token_hash = ?", sum[:]).Scan(&n); err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("%d sessions with the hash of the token, want 1", n)
	}
	return nil
}

func findCookie(rec *httptest.ResponseRecorder) *http.Cookie {
	if rec == nil {
		return nil
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookie {
			return c
		}
	}
	return nil
}
