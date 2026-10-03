package mail

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestLoginCodeMessage(t *testing.T) {
	m := LoginCodeMessage("alice@exemple.fr", "042517", 10*time.Minute)
	if m.To != "alice@exemple.fr" {
		t.Errorf("To = %q", m.To)
	}
	for _, want := range []string{"042517", "10 minutes", "Melting Tribe"} {
		if !strings.Contains(m.Body, want) {
			t.Errorf("body %q does not contain %q", m.Body, want)
		}
	}
	if !strings.Contains(m.Subject, "Melting Tribe") {
		t.Errorf("Subject = %q", m.Subject)
	}
	for _, unwanted := range []string{"http", "www", "tribu "} {
		if strings.Contains(m.Body, unwanted) {
			t.Errorf("body %q contains %q", m.Body, unwanted)
		}
	}
}

func TestOutbox(t *testing.T) {
	r := &Recorder{}
	o := NewOutbox(r, 10*time.Minute, slog.New(slog.DiscardHandler))
	o.SendLoginCode("alice@exemple.fr", "123456")
	o.SendLoginCode("bruno@exemple.fr", "654321")
	o.Wait()
	got := r.Messages()
	if len(got) != 2 {
		t.Fatalf("messages = %+v, want 2", got)
	}
	recipients := got[0].To + " " + got[1].To
	if !strings.Contains(recipients, "alice@exemple.fr") || !strings.Contains(recipients, "bruno@exemple.fr") {
		t.Errorf("recipients = %s", recipients)
	}
}

func TestOutboxFailureDoesNotLogTheCode(t *testing.T) {
	var logs bytes.Buffer
	o := NewOutbox(Unconfigured{}, 10*time.Minute, slog.New(slog.NewTextHandler(&logs, nil)))
	o.SendLoginCode("alice@exemple.fr", "123456")
	o.Wait()
	if !strings.Contains(logs.String(), "login code not sent") || strings.Contains(logs.String(), "123456") {
		t.Errorf("logs = %q, want the failure without the code", logs.String())
	}
}

func TestLogMailer(t *testing.T) {
	var logs bytes.Buffer
	m := LogMailer{Logger: slog.New(slog.NewTextHandler(&logs, nil))}
	if err := m.Send(context.Background(), LoginCodeMessage("alice@exemple.fr", "123456", 10*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(logs.String(), "123456") || !strings.Contains(logs.String(), "alice@exemple.fr") {
		t.Errorf("logs = %q, want the address and the code", logs.String())
	}
}
