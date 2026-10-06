package mail

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
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
	o := NewOutbox(failingMailer{}, 10*time.Minute, slog.New(slog.NewTextHandler(&logs, nil)))
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

func TestFileMailer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mails.jsonl")
	m := &FileMailer{Path: path}
	for _, to := range []string{"alice@exemple.fr", "bruno@exemple.fr"} {
		if err := m.Send(context.Background(), LoginCodeMessage(to, "123456", 10*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(content), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines = %q, want 2", lines)
	}
	for i, want := range []string{"alice@exemple.fr", "bruno@exemple.fr"} {
		var got struct{ To, Subject, Body string }
		if err := json.Unmarshal([]byte(lines[i]), &got); err != nil {
			t.Fatalf("line %d: %v", i, err)
		}
		if got.To != want || !strings.Contains(got.Body, "123456") || got.Subject == "" {
			t.Errorf("line %d = %+v, want a message to %s with the code", i, got, want)
		}
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("file mode = %v, %v; want 0600", info, err)
	}
}

func TestFileMailerError(t *testing.T) {
	m := &FileMailer{Path: filepath.Join(t.TempDir(), "missing", "mails.jsonl")}
	if err := m.Send(context.Background(), LoginCodeMessage("alice@exemple.fr", "123456", 10*time.Minute)); err == nil {
		t.Error("Send succeeded in a missing directory")
	}
}

func TestMailers(t *testing.T) {
	first, second := &Recorder{}, &Recorder{}
	err := Mailers{first, failingMailer{}, second}.Send(context.Background(), LoginCodeMessage("alice@exemple.fr", "123456", 10*time.Minute))
	if !errors.Is(err, errFailing) {
		t.Errorf("error = %v, want errFailing", err)
	}
	if len(first.Messages()) != 1 || len(second.Messages()) != 1 {
		t.Errorf("messages = %d and %d, want 1 each", len(first.Messages()), len(second.Messages()))
	}
}

var errFailing = errors.New("sending failed")

// failingMailer refuses every message.
type failingMailer struct{}

func (failingMailer) Send(context.Context, Message) error { return errFailing }
