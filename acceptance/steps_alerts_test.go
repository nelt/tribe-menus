package acceptance

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/nelt/tribe-menus/internal/mail"
	"github.com/nelt/tribe-menus/internal/tribe"
)

// adminEmail is the address of the administrator of the instance of the scenarios.
const adminEmail = "admin@example.org"

// alertState is what the alert scenarios remember from step to step.
type alertState struct {
	// logs receives the log records of the alerts.
	logs bytes.Buffer
	// logMark and mailMark are the length of the logs and the number of emails before the
	// last check of the alerts.
	logMark, mailMark int
	// at is the time of the last alert received.
	at time.Time
	// next numbers the addresses and IPs made up by the steps.
	next int
}

// alertSignals are the signals named by the scenarios.
var alertSignals = map[string]tribe.AlertSignal{
	"les demandes de code refusées": tribe.RefusedRequests,
	"les codes épuisés":             tribe.ExhaustedCodes,
	"les demandes répétées":         tribe.RepeatedRequests,
}

// registerAlertSteps: the alerts on the code request limits of authentification.feature
// (ENF-01, ADR 0023). The events are made by the domain code at the test clock, and the
// check of the alerts is called by the step, as the server does every ten minutes.
func (w *world) registerAlertSteps(sc *godog.ScenarioContext) {
	sc.Step(`^(\d+) demandes de code ont été refusées par les limites dans la dernière heure$`, w.refuseRequests)
	sc.Step(`^(\d+) demandes de code sont refusées par les limites dans l'heure qui suit$`, func(ctx context.Context, n int) error {
		w.now = w.now.Add(30 * time.Minute)
		return w.refuseRequests(ctx, n)
	})
	sc.Step(`^(\d+) demandes de code sont refusées par les limites (\d+) heures après l'alerte$`, func(ctx context.Context, n, hours int) error {
		w.now = w.alerts.at.Add(time.Duration(hours) * time.Hour)
		return w.refuseRequests(ctx, n)
	})
	sc.Step(`^(\d+) codes ont été invalidés par des essais erronés dans la dernière heure$`, w.exhaustCodes)
	sc.Step(`^"([^"]*)" a demandé (\d+) codes pour la tribu "([^"]*)" dans la dernière heure$`, w.repeatRequests)
	sc.Step(`^l'administrateur a reçu une alerte sur (les demandes de code refusées)$`, func(ctx context.Context, signal string) error {
		if err := w.refuseRequests(ctx, tribe.RefusedRequestsThreshold); err != nil {
			return err
		}
		if err := w.checkAlerts(ctx); err != nil {
			return err
		}
		return w.alertReceived(signal)
	})
	sc.Step(`^l'application vérifie les limites atteintes$`, w.checkAlerts)
	sc.Step(`^l'administrateur reçoit une alerte sur (les demandes de code refusées|les codes épuisés|les demandes répétées)$`, w.alertReceived)
	sc.Step(`^aucune alerte n'est envoyée$`, w.noAlert)
	sc.Step(`^l'alerte ne contient ni adresse e-mail, ni adresse IP, ni le nom ou l'identifiant de la tribu$`, w.alertNamesNobody)
}

// madeUpIP returns a new IP, for each request of the steps: no limit by IP is reached.
func (w *world) madeUpIP() string {
	w.alerts.next++
	return fmt.Sprintf("198.51.100.%d", w.alerts.next%250+1)
}

// tribeOrNil returns the store of the tribe, nil for an unknown tribe.
func (w *world) tribeOrNil(ctx context.Context, slug string) (*tribe.Store, error) {
	t, err := w.tribeStore(ctx, slug)
	if err != nil {
		if _, ok := w.auth.declared[slug]; !ok {
			return nil, nil
		}
		return nil, err
	}
	return t, nil
}

// refuseRequests makes n requests refused by the limit by address, a minute apart up to now,
// for an address that is not a member: three accepted requests, then n refused.
func (w *world) refuseRequests(ctx context.Context, n int) error {
	if _, err := w.app(ctx); err != nil {
		return err
	}
	if _, err := w.ensureMember(ctx, string(w.auth.me), w.auth.home); err != nil {
		return err
	}
	t, err := w.tribeOrNil(ctx, w.auth.home)
	if err != nil {
		return err
	}
	w.alerts.next++
	email := fmt.Sprintf("refus-%d@exemple.fr", w.alerts.next)
	start := w.now
	defer func() { w.now = start }()
	total := tribe.EmailRequestLimit + n
	for i := range total {
		w.now = start.Add(time.Duration(i-total) * time.Minute)
		_, err := w.login.RequestCode(ctx, w.auth.home, t, email, w.madeUpIP())
		if refused := errors.Is(err, tribe.ErrTooManyRequests); refused != (i >= tribe.EmailRequestLimit) || err != nil && !refused {
			return fmt.Errorf("request %d for %s: %v", i+1, email, err)
		}
	}
	return nil
}

// exhaustCodes invalidates n codes by wrong attempts, a minute apart up to now: those of
// members of the tribe, added for it, and decoy ones of other addresses, alternately.
func (w *world) exhaustCodes(ctx context.Context, n int) error {
	if _, err := w.app(ctx); err != nil {
		return err
	}
	start := w.now
	defer func() { w.now = start }()
	for i := range n {
		w.now = start.Add(time.Duration(i-n) * time.Minute)
		w.alerts.next++
		email := fmt.Sprintf("inconnu-%d@exemple.fr", w.alerts.next)
		if i%2 == 0 {
			email = fmt.Sprintf("membre-%d@exemple.fr", w.alerts.next)
			if _, err := w.ensureMember(ctx, email, w.auth.home); err != nil {
				return err
			}
		}
		t, err := w.tribeOrNil(ctx, w.auth.home)
		if err != nil {
			return err
		}
		token, err := w.login.RequestCode(ctx, w.auth.home, t, email, w.madeUpIP())
		if err != nil {
			return fmt.Errorf("request for %s: %w", email, err)
		}
		for range tribe.LoginCodeAttempts {
			_, _, err := w.login.OpenSession(ctx, w.auth.home, t, email, "00000000", token, tribe.Device{})
			var incorrect *tribe.IncorrectCodeError
			if !errors.As(err, &incorrect) {
				return fmt.Errorf("wrong code for %s: %v, want an incorrect code", email, err)
			}
		}
	}
	return nil
}

// repeatRequests makes n requests for the address in the tribe, at the pace of the limit by
// address (one every five minutes), up to now: none is refused.
func (w *world) repeatRequests(ctx context.Context, email string, n int, slug string) error {
	if _, err := w.app(ctx); err != nil {
		return err
	}
	if _, err := w.ensureMember(ctx, string(w.auth.me), w.auth.home); err != nil {
		return err
	}
	t, err := w.tribeOrNil(ctx, slug)
	if err != nil {
		return err
	}
	pace := tribe.EmailRequestWindow / tribe.EmailRequestLimit
	start := w.now
	defer func() { w.now = start }()
	for i := range n {
		w.now = start.Add(time.Duration(i-n) * pace)
		if _, err := w.login.RequestCode(ctx, slug, t, email, w.madeUpIP()); err != nil {
			return fmt.Errorf("request %d for %s: %w", i+1, email, err)
		}
	}
	return nil
}

// checkAlerts checks the alerts at the test clock, as the server does every ten minutes.
func (w *world) checkAlerts(ctx context.Context) error {
	if _, err := w.app(ctx); err != nil {
		return err
	}
	store, err := w.openStore(ctx)
	if err != nil {
		return err
	}
	w.outbox.Wait()
	w.alerts.logMark = w.alerts.logs.Len()
	w.alerts.mailMark = len(w.recorder.Messages())
	return w.login.CheckAlerts(ctx, store)
}

// newAlerts returns the alert records and the emails to the administrator since the last
// check.
func (w *world) newAlerts() ([]map[string]any, []mail.Message, error) {
	var records []map[string]any
	scanner := bufio.NewScanner(bytes.NewReader(w.alerts.logs.Bytes()[w.alerts.logMark:]))
	for scanner.Scan() {
		var record map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return nil, nil, fmt.Errorf("log record %q: %w", scanner.Text(), err)
		}
		if record["alert"] != nil {
			records = append(records, record)
		}
	}
	var emails []mail.Message
	for _, m := range w.sentMails(w.alerts.mailMark) {
		if m.To == adminEmail {
			emails = append(emails, m)
		}
	}
	return records, emails, nil
}

func (w *world) alertReceived(signal string) error {
	records, emails, err := w.newAlerts()
	if err != nil {
		return err
	}
	if len(records) != 1 || len(emails) != 1 {
		return fmt.Errorf("%d alert records and %d emails to the administrator, want one of each", len(records), len(emails))
	}
	record := records[0]
	signals, _ := record["signals"].(string)
	if record["alert"] != "code_request_limits" || record["level"] != "ERROR" || !slices.Contains(strings.Split(signals, ","), string(alertSignals[signal])) {
		return fmt.Errorf("alert record %v, want one on %s", record, alertSignals[signal])
	}
	w.alerts.at = w.now
	return nil
}

func (w *world) noAlert() error {
	records, emails, err := w.newAlerts()
	if err != nil {
		return err
	}
	if len(records) != 0 || len(emails) != 0 {
		return fmt.Errorf("alert records %v, %d emails to the administrator; want none", records, len(emails))
	}
	return nil
}

// alertNamesNobody checks the alert email and record of the last check: no address, no IP,
// neither the name nor the slug of a tribe of the scenario.
func (w *world) alertNamesNobody() error {
	records, emails, err := w.newAlerts()
	if err != nil {
		return err
	}
	if len(records) == 0 || len(emails) == 0 {
		return errors.New("no alert")
	}
	var texts []string
	for _, m := range emails {
		texts = append(texts, m.Subject, m.Body)
	}
	for _, r := range records {
		data, err := json.Marshal(r)
		if err != nil {
			return err
		}
		texts = append(texts, string(data))
	}
	unwanted := []string{"@", "198.51.100.", "192.0.2.", "203.0.113."}
	for slug, name := range w.auth.declared {
		unwanted = append(unwanted, slug, name)
	}
	for _, text := range texts {
		for _, u := range unwanted {
			if strings.Contains(text, u) {
				return fmt.Errorf("alert contains %q: %s", u, text)
			}
		}
	}
	return nil
}
