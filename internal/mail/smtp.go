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

// SendError is a failed sending: the step of the conversation, and the SMTP reply code when
// the server gave one.
type SendError struct {
	Step string
	Code int
	err  error
}

func (e *SendError) Error() string {
	if e.Code != 0 {
		return fmt.Sprintf("smtp %s: %d: %v", e.Step, e.Code, e.err)
	}
	return fmt.Sprintf("smtp %s: %v", e.Step, e.err)
}

func (e *SendError) Unwrap() error { return e.err }

func stepError(step string, err error) error {
	se := &SendError{Step: step, err: err}
	var reply *textproto.Error
	if errors.As(err, &reply) {
		se.Code = reply.Code
	}
	return se
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
	return m.converse(ctx, func(c *smtp.Client) error {
		if err := c.Mail(m.From); err != nil {
			return stepError(StepFrom, err)
		}
		if err := c.Rcpt(msg.To); err != nil {
			return stepError(StepTo, err)
		}
		w, err := c.Data()
		if err != nil {
			return stepError(StepData, err)
		}
		if _, err := w.Write(data); err != nil {
			return stepError(StepData, err)
		}
		if err := w.Close(); err != nil {
			return stepError(StepData, err)
		}
		return nil
	})
}

// Check connects, authenticates and leaves, without sending anything: the check at startup
// (plan production, step 12).
func (m *SMTPMailer) Check(ctx context.Context) error {
	return m.converse(ctx, func(*smtp.Client) error { return nil })
}

// converse opens a connection, authenticates, runs do and quits.
func (m *SMTPMailer) converse(ctx context.Context, do func(*smtp.Client) error) error {
	dialer := &tls.Dialer{Config: &tls.Config{
		ServerName: m.Host,
		MinVersion: tls.VersionTLS12,
		RootCAs:    m.RootCAs,
	}}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(m.Host, strconv.Itoa(m.Port)))
	if err != nil {
		return stepError(StepConnect, err)
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return stepError(StepConnect, err)
		}
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()

	c, err := smtp.NewClient(conn, m.Host)
	if err != nil {
		return stepError(StepConnect, err)
	}
	defer c.Close()
	if err := c.Auth(smtp.PlainAuth("", m.Username, m.Password, m.Host)); err != nil {
		return stepError(StepAuth, err)
	}
	if err := do(c); err != nil {
		return err
	}
	// The message is accepted once DATA ends: a failing QUIT does not make it fail.
	_ = c.Quit()
	return nil
}
