package alert

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/nelt/tribe-menus/internal/mail"
)

func TestSendFailures(t *testing.T) {
	start := time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		// failures are the times of the failures, from start.
		failures []time.Duration
		want     int
	}{
		{name: "two in an hour", failures: []time.Duration{0, time.Minute}, want: 0},
		{name: "three in an hour", failures: []time.Duration{0, time.Minute, 59 * time.Minute}, want: 1},
		{name: "three over more than an hour", failures: []time.Duration{0, 30 * time.Minute, time.Hour}, want: 0},
		{name: "more within six hours", failures: []time.Duration{0, 1, 2, time.Hour, time.Hour + 1, time.Hour + 2, 5 * time.Hour}, want: 1},
		{name: "again after six hours", failures: []time.Duration{0, 1, 2, 6 * time.Hour, 6*time.Hour + 1, 6*time.Hour + 2}, want: 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			now := start
			f := &SendFailures{Logger: slog.New(slog.NewJSONHandler(&logs, nil)), Now: func() time.Time { return now }}
			for _, at := range tc.failures {
				now = start.Add(at)
				f.Record()
			}
			if got := strings.Count(logs.String(), `"alert":"smtp_failures"`); got != tc.want {
				t.Errorf("%d alerts, want %d; logs: %s", got, tc.want, logs.String())
			}
			if tc.want > 0 && !strings.Contains(logs.String(), `"level":"ERROR"`) {
				t.Errorf("logs %s, want an error record", logs.String())
			}
		})
	}
}

type failingMailer struct{ err error }

func (m failingMailer) Send(context.Context, mail.Message) error { return m.err }

func TestCountingMailer(t *testing.T) {
	var logs bytes.Buffer
	f := &SendFailures{Logger: slog.New(slog.NewJSONHandler(&logs, nil)), Now: time.Now}
	ok := CountingMailer{Mailer: failingMailer{}, Failures: f}
	failing := CountingMailer{Mailer: failingMailer{err: errors.New("connection refused")}, Failures: f}
	for _, m := range []CountingMailer{ok, ok, failing, ok, failing} {
		_ = m.Send(context.Background(), mail.Message{To: "alice@exemple.fr"})
	}
	if strings.Contains(logs.String(), "smtp_failures") {
		t.Fatalf("alert after two failures: %s", logs.String())
	}
	if err := failing.Send(context.Background(), mail.Message{To: "alice@exemple.fr"}); err == nil {
		t.Error("the error of the mailer is not returned")
	}
	if !strings.Contains(logs.String(), `"alert":"smtp_failures"`) {
		t.Errorf("no alert after three failures: %s", logs.String())
	}
	if strings.Contains(logs.String(), "alice") {
		t.Errorf("the alert names the recipient: %s", logs.String())
	}
}
