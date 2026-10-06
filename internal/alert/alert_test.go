package alert

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/nelt/tribe-menus/internal/mail"
	"github.com/nelt/tribe-menus/internal/tribe"
)

type recorder struct{ sent []mail.Message }

func (r *recorder) SendAlert(msg mail.Message) { r.sent = append(r.sent, msg) }

var report = tribe.AlertReport{
	At:      time.Date(2026, 10, 5, 9, 20, 0, 0, time.UTC),
	Signals: []tribe.AlertSignal{tribe.RefusedRequests, tribe.RepeatedRequests},
	Counts: tribe.AlertCounts{
		Events:            map[tribe.AlertEvent]int{tribe.EmailLimitRefusal: 7, tribe.IPLimitRefusal: 4, tribe.LoginCodeExhausted: 1},
		RepeatedAddresses: 2,
	},
}

func TestMessage(t *testing.T) {
	m := Message("admin@example.org", "https://recette.example.org", report)
	if m.To != "admin@example.org" || !strings.Contains(m.Subject, "Melting Tribe") {
		t.Errorf("To = %q, Subject = %q", m.To, m.Subject)
	}
	for _, want := range []string{
		"https://recette.example.org",
		"2026-10-05 à 09:20",
		"demandes refusées par une limite : au moins 10",
		"demandes répétées pour une même adresse : au moins 9",
		"limite par adresse : 7",
		"limite par adresse IP : 4",
		"limite par tribu : 0",
		"codes de membres épuisés : 1",
		"adresses à 9 demandes ou plus : 2",
		"journaux d'accès de Caddy",
	} {
		if !strings.Contains(m.Body, want) {
			t.Errorf("body does not contain %q:\n%s", want, m.Body)
		}
	}
	if strings.Contains(m.Body, "codes invalidés") {
		t.Errorf("body names a signal not alerted on:\n%s", m.Body)
	}
	if strings.Contains(m.Body, "@") {
		t.Errorf("body contains an address:\n%s", m.Body)
	}
}

func TestNotifier(t *testing.T) {
	var logs bytes.Buffer
	r := &recorder{}
	n := &Notifier{Logger: slog.New(slog.NewJSONHandler(&logs, nil)), Sender: r, To: "admin@example.org", Instance: "https://recette.example.org"}
	n.LogCodeRequestLimits(report)
	if len(r.sent) != 0 {
		t.Fatalf("sent = %+v, want nothing before SendCodeRequestLimits", r.sent)
	}
	n.SendCodeRequestLimits(report)
	if len(r.sent) != 1 || r.sent[0].To != "admin@example.org" {
		t.Fatalf("sent = %+v, want one email to the administrator", r.sent)
	}
	var record map[string]any
	if err := json.Unmarshal(logs.Bytes(), &record); err != nil {
		t.Fatalf("logs %q: %v", logs.String(), err)
	}
	want := map[string]any{
		"level": "ERROR", "alert": "code_request_limits", "signals": "refused_requests,repeated_requests",
		"email_limit": 7.0, "ip_limit": 4.0, "tribe_limit": 0.0, "decoy_code_exhausted": 0.0, "login_code_exhausted": 1.0,
		"repeated_addresses": 2.0,
	}
	for key, value := range want {
		if record[key] != value {
			t.Errorf("record[%q] = %v, want %v", key, record[key], value)
		}
	}
}
