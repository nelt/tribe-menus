package mail

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"net/textproto"
	"regexp"
	"strconv"
	"time"
)

// Steps of an SMTP conversation, named in a SendError.
const (
	StepConnect = "connect"
	StepAuth    = "auth"
	StepFrom    = "from"
	StepTo      = "to"
	StepData    = "data"
)

// SMTPMailer sends each message over its own SMTP connection, in TLS from the start (port
// 465), with PLAIN authentication: the MX Plan of OVHcloud (ADR 0014, points 1 and 4).
type SMTPMailer struct {
	Host     string
	Port     int
	Username string
	Password string
	// From is the sending address, in the From header and the envelope.
	From string
	// RootCAs verify the certificate of the server; nil for the roots of the system. For the
	// tests.
	RootCAs *x509.CertPool
	// Now gives the Date header; nil for time.Now.
	Now func() time.Time
}

var _ Mailer = (*SMTPMailer)(nil)

// SendError is a failed sending: the step of the conversation, the SMTP reply code when the
// server gave one, and the text of the error, where the address of the recipient and the
// password are replaced: a server readily quotes the recipient in its reply (plan
// production, step 13). Error never gives more than this text.
type SendError struct {
	Step string
	Code int
	// Reply is the text of the error, redacted.
	Reply string
	err   error
}

func (e *SendError) Error() string {
	return fmt.Sprintf("smtp %s: %s", e.Step, e.Reply)
}

// Unwrap gives the error unredacted, for errors.Is: never log it.
func (e *SendError) Unwrap() error { return e.err }

// Placeholders of what a SendError never quotes.
const (
	redactedRecipient = "[recipient]"
	redactedPassword  = "[password]"
)

// stepError returns the error of the step, its text redacted of the recipient and of the
// password.
func (m *SMTPMailer) stepError(step string, err error, recipient string) error {
	se := &SendError{Step: step, err: err}
	text := err.Error()
	var reply *textproto.Error
	if errors.As(err, &reply) {
		se.Code = reply.Code
		text = reply.Msg
	}
	if recipient != "" {
		text = replaceFold(text, recipient, redactedRecipient)
	}
	if m.Password != "" {
		text = replaceFold(text, m.Password, redactedPassword)
	}
	se.Reply = text
	return se
}

// replaceFold replaces every occurrence of old in s, whatever the case.
func replaceFold(s, old, replacement string) string {
	return regexp.MustCompile("(?i)"+regexp.QuoteMeta(old)).ReplaceAllLiteralString(s, replacement)
}

// ErrorAttrs are the attributes of a log record for a failed sending: the error, and for a
// SendError its step and SMTP code.
func ErrorAttrs(err error) []any {
	attrs := []any{"error", err.Error()}
	var se *SendError
	if errors.As(err, &se) {
		attrs = append(attrs, "step", se.Step, "code", se.Code)
	}
	return attrs
}

// Send implements Mailer. The deadline of ctx bounds the whole conversation, connection
// included.
func (m *SMTPMailer) Send(ctx context.Context, msg Message) error {
	now := time.Now
	if m.Now != nil {
		now = m.Now
	}
	data, err := compose(m.From, msg, now(), newMessageID(m.From))
	if err != nil {
		return err
	}
	return m.converse(ctx, msg.To, func(c *smtp.Client) error {
		if err := c.Mail(m.From); err != nil {
			return m.stepError(StepFrom, err, msg.To)
		}
		if err := c.Rcpt(msg.To); err != nil {
			return m.stepError(StepTo, err, msg.To)
		}
		w, err := c.Data()
		if err != nil {
			return m.stepError(StepData, err, msg.To)
		}
		if _, err := w.Write(data); err != nil {
			return m.stepError(StepData, err, msg.To)
		}
		if err := w.Close(); err != nil {
			return m.stepError(StepData, err, msg.To)
		}
		return nil
	})
}

// Check connects, authenticates and leaves, without sending anything: the check at startup
// (plan production, step 12).
func (m *SMTPMailer) Check(ctx context.Context) error {
	return m.converse(ctx, "", func(*smtp.Client) error { return nil })
}

// converse opens a connection, authenticates, runs do and quits. recipient is redacted
// from the errors.
func (m *SMTPMailer) converse(ctx context.Context, recipient string, do func(*smtp.Client) error) error {
	dialer := &tls.Dialer{Config: &tls.Config{
		ServerName: m.Host,
		MinVersion: tls.VersionTLS12,
		RootCAs:    m.RootCAs,
	}}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(m.Host, strconv.Itoa(m.Port)))
	if err != nil {
		return m.stepError(StepConnect, err, recipient)
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return m.stepError(StepConnect, err, recipient)
		}
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()

	c, err := smtp.NewClient(conn, m.Host)
	if err != nil {
		return m.stepError(StepConnect, err, recipient)
	}
	defer c.Close()
	if err := c.Auth(smtp.PlainAuth("", m.Username, m.Password, m.Host)); err != nil {
		return m.stepError(StepAuth, err, recipient)
	}
	if err := do(c); err != nil {
		return err
	}
	// The message is accepted once DATA ends: a failing QUIT does not make it fail.
	_ = c.Quit()
	return nil
}
