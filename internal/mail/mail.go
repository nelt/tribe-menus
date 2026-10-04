// Package mail sends the only email of the application, the login code (ENF-01, ADR 0014),
// through a Mailer: in development to the logs (and to a file for the end-to-end tests), in
// tests to memory. Sending happens in the
// background (Outbox), outside of the request.
package mail

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"
)

const (
	appName = "Melting Tribe"
	// sendTimeout bounds each sending.
	sendTimeout = 30 * time.Second
)

// Message is a plain text email.
type Message struct {
	To      string
	Subject string
	Body    string
}

// Mailer sends a message.
type Mailer interface {
	Send(ctx context.Context, m Message) error
}

// LoginCodeMessage is the message carrying a login code: plain text, in French, with the
// code, its validity and the name of the application; neither link nor tribe name (ADR 0014).
func LoginCodeMessage(to, code string, validity time.Duration) Message {
	return Message{
		To:      to,
		Subject: "Votre code de connexion " + appName,
		Body: fmt.Sprintf("Bonjour,\n\n"+
			"Votre code de connexion à %s : %s\n\n"+
			"Il est valable %d minutes. Si vous n'avez pas demandé ce code, ignorez ce message.\n\n"+
			"%s\n", appName, code, int(validity.Minutes()), appName),
	}
}

// LogMailer writes each message to the logs, code included: development only (ADR 0009).
type LogMailer struct {
	Logger *slog.Logger
}

// Send implements Mailer.
func (m LogMailer) Send(_ context.Context, msg Message) error {
	m.Logger.Info("email (development: not sent)", "to", msg.To, "subject", msg.Subject, "body", msg.Body)
	return nil
}

// FileMailer appends each message to a file, one JSON object per line, so that the
// end-to-end tests read the code (D6): development only, code included.
type FileMailer struct {
	Path string
	mu   sync.Mutex
}

// Send implements Mailer.
func (m *FileMailer) Send(_ context.Context, msg Message) (err error) {
	line, err := json.Marshal(struct {
		To      string `json:"to"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}{msg.To, msg.Subject, msg.Body})
	if err != nil {
		return fmt.Errorf("mail file: %w", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	f, err := os.OpenFile(m.Path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return fmt.Errorf("mail file: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("mail file: %w", closeErr)
		}
	}()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("mail file: %w", err)
	}
	return nil
}

// Mailers sends each message with every mailer, in order, and returns their errors joined.
type Mailers []Mailer

// Send implements Mailer.
func (ms Mailers) Send(ctx context.Context, msg Message) error {
	var errs []error
	for _, m := range ms {
		errs = append(errs, m.Send(ctx, msg))
	}
	return errors.Join(errs...)
}

// ErrNotConfigured is returned by Unconfigured.
var ErrNotConfigured = errors.New("no SMTP server configured")

// Unconfigured refuses every message: the server outside development until SMTP sending
// comes with the deployment (ADR 0014). The code never reaches the logs.
type Unconfigured struct{}

// Send implements Mailer.
func (Unconfigured) Send(context.Context, Message) error {
	return ErrNotConfigured
}

// Recorder keeps the messages in memory, for the tests.
type Recorder struct {
	mu       sync.Mutex
	messages []Message
}

// Send implements Mailer.
func (r *Recorder) Send(_ context.Context, msg Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messages = append(r.messages, msg)
	return nil
}

// Messages returns the messages sent so far, oldest first.
func (r *Recorder) Messages() []Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Message(nil), r.messages...)
}

// Outbox sends the login codes in the background, so that the response time does not
// depend on whether a code is sent (ENF-02). Wait waits for the sendings in progress: at
// shutdown, before closing the databases, and in the tests, before looking at what was sent.
type Outbox struct {
	mailer   Mailer
	validity time.Duration
	logger   *slog.Logger
	wg       sync.WaitGroup
}

// NewOutbox returns an outbox sending with mailer codes valid for validity.
func NewOutbox(mailer Mailer, validity time.Duration, logger *slog.Logger) *Outbox {
	return &Outbox{mailer: mailer, validity: validity, logger: logger}
}

// SendLoginCode sends the code in the background. A failure is logged, without the code.
func (o *Outbox) SendLoginCode(to, code string) {
	msg := LoginCodeMessage(to, code, o.validity)
	o.wg.Go(func() {
		ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
		defer cancel()
		if err := o.mailer.Send(ctx, msg); err != nil {
			o.logger.Error("login code not sent", "error", err)
		}
	})
}

// Wait waits for the sendings in progress.
func (o *Outbox) Wait() {
	o.wg.Wait()
}
