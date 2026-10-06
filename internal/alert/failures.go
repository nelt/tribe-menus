package alert

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/nelt/tribe-menus/internal/mail"
	"github.com/nelt/tribe-menus/internal/tribe"
)

// Repeated SMTP failures (ADR 0015, point 15; ADR 0023, point 3): constants of the code.
const (
	SMTPFailuresThreshold = 3
	SMTPFailuresWindow    = time.Hour
)

// SendFailures counts the failed sendings in memory (plan production, D9): a state of
// supervision, not of security, and a count kept in a database would date sendings, hence
// requests for members. At the SMTPFailuresThreshold-th failure within SMTPFailuresWindow,
// it writes an alert to the log, at most once per tribe.AlertInterval. No email: the sending
// is what fails.
type SendFailures struct {
	Logger *slog.Logger
	Now    func() time.Time

	mu        sync.Mutex
	failures  []time.Time
	lastAlert time.Time
}

// Record counts a failed sending, or a failed check of the SMTP account at startup.
func (f *SendFailures) Record() {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := f.Now()
	start := now.Add(-SMTPFailuresWindow)
	kept := f.failures[:0]
	for _, at := range f.failures {
		if at.After(start) {
			kept = append(kept, at)
		}
	}
	f.failures = append(kept, now)
	if len(f.failures) < SMTPFailuresThreshold {
		return
	}
	if !f.lastAlert.IsZero() && now.Before(f.lastAlert.Add(tribe.AlertInterval)) {
		return
	}
	f.lastAlert = now
	f.Logger.Error("repeated SMTP failures", "alert", SMTPFailures, "failures", len(f.failures), "window", SMTPFailuresWindow.String())
}

// CountingMailer counts the failures of its mailer.
type CountingMailer struct {
	Mailer   mail.Mailer
	Failures *SendFailures
}

// Send implements mail.Mailer.
func (m CountingMailer) Send(ctx context.Context, msg mail.Message) error {
	err := m.Mailer.Send(ctx, msg)
	if err != nil {
		m.Failures.Record()
	}
	return err
}
