package tribe

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// TestRequestCodeTouchesNoTribe: the response to a code request is written before any tribe
// store is opened, whatever the address (ADR 0026); the code of a member is sent by the
// processing of the queue.
func TestRequestCodeTouchesNoTribe(t *testing.T) {
	ctx := context.Background()
	for _, email := range []string{"alice@exemple.fr", "inconnu@exemple.fr", "bruno@exemple.fr"} {
		t.Run(email, func(t *testing.T) {
			f := newLoginFixture(t)
			opened := 0
			login := &Login{
				RateLimit: f.login.RateLimit,
				Mailer:    f.mailer,
				Requests:  NewCodeRequests(CodeRequestQueueSize),
				Tribes: func(context.Context, string) (*Store, error) {
					opened++
					return f.martin, nil
				},
				Logger: slog.New(slog.DiscardHandler),
				Now:    func() time.Time { return f.now },
			}
			if _, err := login.RequestCode(ctx, "martin", email, "192.0.2.1"); err != nil {
				t.Fatal(err)
			}
			if opened != 0 || len(f.mailer.sent) != 0 {
				t.Fatalf("before processing: %d tribe stores opened, %d codes sent, want none", opened, len(f.mailer.sent))
			}
			login.Requests.Close()
			login.ProcessCodeRequests(ctx)
			if opened != 1 {
				t.Errorf("tribe stores opened by the processing = %d, want 1", opened)
			}
			if want := email == "alice@exemple.fr"; (len(f.mailer.sent) == 1) != want {
				t.Errorf("codes sent = %v, want sent: %v", f.mailer.sent, want)
			}
		})
	}
}

// TestCodeRequestQueueFull: beyond the size of the queue, a request gets the same answer and
// is dropped, with a warning that names no address.
func TestCodeRequestQueueFull(t *testing.T) {
	ctx := context.Background()
	f := newLoginFixture(t)
	var logs bytes.Buffer
	login := &Login{
		RateLimit: f.login.RateLimit,
		Mailer:    f.mailer,
		Requests:  NewCodeRequests(1),
		Tribes:    storeOpener(f.store),
		Logger:    slog.New(slog.NewTextHandler(&logs, nil)),
		Now:       func() time.Time { return f.now },
	}
	for _, ip := range []string{"192.0.2.1", "192.0.2.2"} {
		if token, err := login.RequestCode(ctx, "martin", "alice@exemple.fr", ip); err != nil || token == "" {
			t.Fatalf("request from %s: token %q, error %v", ip, token, err)
		}
	}
	if !strings.Contains(logs.String(), "code request dropped, queue full") || strings.Contains(logs.String(), "alice") {
		t.Errorf("logs = %q, want a warning without the address", logs.String())
	}
	login.Requests.Close()
	login.ProcessCodeRequests(ctx)
	if len(f.mailer.sent) != 1 {
		t.Errorf("codes sent = %d, want 1", len(f.mailer.sent))
	}
}

// TestCodeRequestsClosed: once the queue is closed, a request is dropped, and the processing
// ends after the queued requests, even when its context is done.
func TestCodeRequestsClosed(t *testing.T) {
	f := newLoginFixture(t)
	login := &Login{
		RateLimit: f.login.RateLimit,
		Mailer:    f.mailer,
		Requests:  NewCodeRequests(CodeRequestQueueSize),
		Tribes:    storeOpener(f.store),
		Logger:    slog.New(slog.DiscardHandler),
		Now:       func() time.Time { return f.now },
	}
	ctx, cancel := context.WithCancel(context.Background())
	if _, err := login.RequestCode(ctx, "martin", "alice@exemple.fr", "192.0.2.1"); err != nil {
		t.Fatal(err)
	}
	login.Requests.Close()
	login.Requests.Close()
	if _, err := login.RequestCode(ctx, "martin", "alice@exemple.fr", "192.0.2.2"); err != nil {
		t.Fatal(err)
	}
	cancel()
	login.ProcessCodeRequests(ctx)
	login.Requests.Wait()
	if len(f.mailer.sent) != 1 {
		t.Errorf("codes sent = %d, want 1", len(f.mailer.sent))
	}
}
