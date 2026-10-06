// Package alert writes the alerts of the application to the log, and sends the alert on the
// code request limits to the administrator (ADR 0023). An alert is a log record of level
// error with an alert attribute, which the server relays by email (ADR 0015, point 15).
package alert

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/nelt/tribe-menus/internal/mail"
	"github.com/nelt/tribe-menus/internal/tribe"
)

// The values of the alert attribute of the log records.
const (
	CodeRequestLimits = "code_request_limits"
	SMTPFailures      = "smtp_failures"
)

// Sender sends an alert email in the background: mail.Outbox.
type Sender interface {
	SendAlert(msg mail.Message)
}

// Notifier writes the alert on the code request limits to the log and sends it to the
// administrator. Neither carries an address, an IP nor a tribe (ADR 0023, point 6).
type Notifier struct {
	Logger *slog.Logger
	Sender Sender
	// To is the address of the administrator, the alerts.to key of the config file.
	To string
	// Instance is the public address of the instance, its baseURL.
	Instance string
}

var _ tribe.AlertNotifier = (*Notifier)(nil)

// NotifyCodeRequestLimits implements tribe.AlertNotifier.
func (n *Notifier) NotifyCodeRequestLimits(r tribe.AlertReport) {
	signals := make([]string, len(r.Signals))
	for i, s := range r.Signals {
		signals[i] = string(s)
	}
	attrs := []any{"alert", CodeRequestLimits, "signals", strings.Join(signals, ",")}
	for _, e := range tribe.AlertEvents {
		attrs = append(attrs, string(e), r.Counts.Events[e])
	}
	attrs = append(attrs, "repeated_addresses", r.Counts.RepeatedAddresses)
	n.Logger.Error("code request limits reached repeatedly", attrs...)
	n.Sender.SendAlert(Message(n.To, n.Instance, r))
}

// signalLabels and eventLabels are the French wordings of the alert email.
var (
	signalLabels = map[tribe.AlertSignal]string{
		tribe.RefusedRequests:  fmt.Sprintf("demandes refusées par une limite : au moins %d dans l'heure", tribe.RefusedRequestsThreshold),
		tribe.ExhaustedCodes:   fmt.Sprintf("codes invalidés par leur dernier essai erroné : au moins %d dans l'heure", tribe.ExhaustedCodesThreshold),
		tribe.RepeatedRequests: fmt.Sprintf("demandes répétées pour une même adresse : au moins %d dans l'heure", tribe.RepeatedRequestsThreshold),
	}
	eventLabels = map[tribe.AlertEvent]string{
		tribe.EmailLimitRefusal:  "demandes refusées par la limite par adresse",
		tribe.IPLimitRefusal:     "demandes refusées par la limite par adresse IP",
		tribe.TribeLimitRefusal:  "demandes refusées par la limite par tribu",
		tribe.DecoyCodeExhausted: "codes fantômes épuisés (adresses qui ne sont pas membres)",
		tribe.LoginCodeExhausted: "codes de membres épuisés",
	}
)

// Message is the alert email on the code request limits: plain text, in French; the
// instance, the window, the signals and the counters; neither address, nor IP, nor tribe.
func Message(to, instance string, r tribe.AlertReport) mail.Message {
	var b strings.Builder
	fmt.Fprintf(&b, "Bonjour,\n\n")
	fmt.Fprintf(&b, "Sur l'instance %s, les limites de demandes de code de connexion ont été atteintes "+
		"de façon répétée dans l'heure qui précède le %s (UTC).\n\n", instance, r.At.UTC().Format("2006-01-02 à 15:04"))
	fmt.Fprintf(&b, "Signaux franchis :\n")
	for _, s := range r.Signals {
		fmt.Fprintf(&b, "- %s\n", signalLabels[s])
	}
	fmt.Fprintf(&b, "\nCompteurs de l'heure, pour toute l'instance :\n")
	for _, e := range tribe.AlertEvents {
		fmt.Fprintf(&b, "- %s : %d\n", eventLabels[e], r.Counts.Events[e])
	}
	fmt.Fprintf(&b, "- adresses à %d demandes ou plus : %d\n", tribe.RepeatedRequestsThreshold, r.Counts.RepeatedAddresses)
	fmt.Fprintf(&b, "\nCe message ne nomme ni adresse e-mail, ni adresse IP, ni tribu. "+
		"Les journaux d'accès de Caddy donnent l'heure, les adresses IP et les URL des requêtes.\n\n")
	fmt.Fprintf(&b, "Une même alerte n'est pas renvoyée avant %d heures, même si la situation dure.\n\n", int(tribe.AlertInterval.Hours()))
	fmt.Fprintf(&b, "Melting Tribe\n")
	return mail.Message{
		To:      to,
		Subject: "Alerte Melting Tribe : limites de demandes de code atteintes",
		Body:    b.String(),
	}
}
