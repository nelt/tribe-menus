package acceptance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/cucumber/godog"
	"github.com/nelt/tribe-menus/internal/tribe"
)

// unknownSlug is the slug of a tribe that does not exist.
const unknownSlug = "tribu-inexistante"

// registerCompartmentSteps: compartimentage-tribus.feature (ENF-02).
func (w *world) registerCompartmentSteps(sc *godog.ScenarioContext) {
	sc.Step(`^je vois l'écran de connexion de la tribu "([^"]*)"$`, w.loginScreen)
	sc.Step(`^aucune donnée de la tribu "([^"]*)" n'est transmise$`, w.nothingOfTribeSent)
	sc.Step(`^il doit se connecter à la tribu "([^"]*)" avec un code$`, w.loginScreen)
	sc.Step(`^une fois connecté, il reste connecté à la tribu "([^"]*)"$`, w.stillConnectedOnceSignedIn)
	sc.Step(`^le nom "([^"]*)" n'apparaît nulle part dans la réponse$`, w.nameNotSent)
	sc.Step(`^la réponse est la même que pour l'URL d'une tribu qui n'existe pas$`, w.sameAsUnknownTribe)
	sc.Step(`^(?:elle|il) voit le nom "([^"]*)"$`, w.seesName)
	sc.Step(`^cette connexion figure dans le journal d'audit de la tribu "([^"]*)"$`, w.signInAudited)
	sc.Step(`^elle ne figure pas dans le journal d'audit de la tribu "([^"]*)"$`, w.signInNotAudited)
}

// openedText is everything the opened URL sent: statuses, headers and bodies.
func (w *world) openedText() string {
	var b strings.Builder
	for _, rec := range w.auth.opened {
		fmt.Fprintf(&b, "%d %v %s\n", rec.Code, rec.Header(), rec.Body)
	}
	return b.String()
}

// nothingOfTribeSent checks that neither the name of the tribe nor the address of one of
// its members was sent.
func (w *world) nothingOfTribeSent(ctx context.Context, slug string) error {
	t, err := w.tribeStore(ctx, slug)
	if err != nil {
		return err
	}
	name, err := t.Name(ctx)
	if err != nil {
		return err
	}
	store, err := w.openStore(ctx)
	if err != nil {
		return err
	}
	db, err := store.Tribe(ctx, slug)
	if err != nil {
		return err
	}
	rows, err := db.QueryContext(ctx, "SELECT email FROM members")
	if err != nil {
		return err
	}
	secrets := []string{name}
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			_ = rows.Close()
			return err
		}
		secrets = append(secrets, email)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	text := w.openedText()
	for _, s := range secrets {
		if strings.Contains(text, s) {
			return fmt.Errorf("%q was sent: %s", s, text)
		}
	}
	return nil
}

func (w *world) nameNotSent(name string) error {
	if text := w.openedText(); strings.Contains(text, name) {
		return fmt.Errorf("%q was sent: %s", name, text)
	}
	return nil
}

// sameAsUnknownTribe opens the URL of a tribe that does not exist from a new device, and
// compares what both URLs sent, slugs aside.
func (w *world) sameAsUnknownTribe(ctx context.Context) error {
	slug, opened := string(w.tribe), w.auth.opened
	if err := w.openURL(ctx, w.newDevice(), unknownSlug); err != nil {
		return err
	}
	unknown := w.auth.opened
	if len(opened) != len(unknown) {
		return fmt.Errorf("%d responses, %d for an unknown tribe", len(opened), len(unknown))
	}
	for i := range opened {
		got := strings.ReplaceAll(answer(opened[i]), slug, "<slug>")
		want := strings.ReplaceAll(answer(unknown[i]), unknownSlug, "<slug>")
		if got != want {
			return fmt.Errorf("response %d: %q, for an unknown tribe %q", i+1, got, want)
		}
	}
	return nil
}

// stillConnectedOnceSignedIn signs in to the tribe of the opened URL on the same device,
// then checks that both sessions are valid.
func (w *world) stillConnectedOnceSignedIn(ctx context.Context, other string) error {
	d, slug := w.currentDevice(), string(w.tribe)
	if err := w.signIn(ctx, d, slug, string(w.auth.email)); err != nil {
		return err
	}
	for _, s := range []string{slug, other} {
		rec, err := w.do(ctx, d, http.MethodGet, tribePath(s, "api/session"), nil)
		if err != nil {
			return err
		}
		if rec.Code != http.StatusOK {
			return fmt.Errorf("tribe %s: status %d, body %s", s, rec.Code, rec.Body)
		}
	}
	return nil
}

func (w *world) seesName(name string) error {
	if err := w.lastStatus(http.StatusCreated); err != nil {
		return err
	}
	var got apiSession
	if err := json.Unmarshal(w.auth.last.Body.Bytes(), &got); err != nil {
		return err
	}
	if got.Tribe.Name != name {
		return fmt.Errorf("name %q, want %q", got.Tribe.Name, name)
	}
	return nil
}

// signInAudited checks the "session opened" entry of the last sign-in in the tribe.
func (w *world) signInAudited(ctx context.Context, slug string) error {
	t, err := w.tribeStore(ctx, slug)
	if err != nil {
		return err
	}
	entries, err := t.AuditLog(ctx)
	if err != nil {
		return err
	}
	member, err := t.Member(ctx, w.auth.email)
	if err != nil {
		return err
	}
	want := tribe.DetectDevice(userAgent, false)
	for _, e := range entries {
		if e.Operation == tribe.SessionOpened && e.MemberEmail == w.auth.email && e.AuthorID == member.ID &&
			e.SessionID != 0 && e.Device == want {
			return nil
		}
	}
	return fmt.Errorf("no session opened by %s on %+v in %+v", w.auth.email, want, entries)
}

func (w *world) signInNotAudited(ctx context.Context, slug string) error {
	t, err := w.tribeStore(ctx, slug)
	if err != nil {
		return err
	}
	entries, err := t.AuditLog(ctx)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.Operation == tribe.SessionOpened || e.MemberEmail == w.auth.email {
			return fmt.Errorf("entry %+v in the audit log of %s", e, slug)
		}
	}
	return nil
}
