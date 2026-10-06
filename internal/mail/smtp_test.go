package mail

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"math/big"
	"net"
	"net/textproto"
	"strings"
	"sync"
	"testing"
	"time"
)

// newCertificate returns a self-signed certificate for the name, and the pool that trusts it.
func newCertificate(t *testing.T, name string) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: name},
		DNSNames:     []string{name},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IsCA:         true,

		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, pool
}

// received is what the fake server received in a conversation.
type received struct {
	auth     string
	from, to string
	data     string
}

// fakeSMTP is an SMTP server in TLS from the start, in the process, with a certificate for
// "localhost". Its fields choose its answers.
type fakeSMTP struct {
	port    int
	rootCAs *x509.CertPool
	// password accepted by AUTH PLAIN for the user "no-reply@example.org".
	password string
	// rcptReply, if set, refuses RCPT TO with this reply ("%s" is the recipient, "%S" the
	// recipient in upper case).
	rcptReply string
	// authReply, if set, refuses AUTH with this reply.
	authReply string
	// authEcho: the server refuses AUTH, quoting the command it received.
	authEcho bool
	// silent: the server accepts the connection and says nothing.
	silent bool

	mu       sync.Mutex
	received []received
}

// newFakeSMTP starts a fake server, set by configure before it accepts connections.
func newFakeSMTP(t *testing.T, configure func(*fakeSMTP)) *fakeSMTP {
	t.Helper()
	cert, pool := newCertificate(t, "localhost")
	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{cert}})
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeSMTP{port: listener.Addr().(*net.TCPAddr).Port, rootCAs: pool, password: "s3cret-pass"}
	if configure != nil {
		configure(f)
	}
	var wg sync.WaitGroup
	t.Cleanup(func() {
		_ = listener.Close()
		wg.Wait()
	})
	wg.Go(func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			wg.Go(func() {
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
				f.serve(conn)
			})
		}
	})
	return f
}

func (f *fakeSMTP) serve(conn net.Conn) {
	if f.silent {
		_, _ = bufio.NewReader(conn).ReadString('\n')
		return
	}
	tp := textproto.NewConn(conn)
	var r received
	reply := func(s string) { _ = tp.PrintfLine("%s", s) }
	reply("220 localhost ESMTP")
	for {
		line, err := tp.ReadLine()
		if err != nil {
			return
		}
		verb, arg, _ := strings.Cut(line, " ")
		switch strings.ToUpper(verb) {
		case "EHLO":
			reply("250-localhost")
			reply("250 AUTH PLAIN")
		case "AUTH":
			r.auth = arg
			want := "PLAIN " + base64.StdEncoding.EncodeToString([]byte("\x00no-reply@example.org\x00"+f.password))
			switch {
			case f.authEcho:
				reply("535 5.7.8 bad AUTH " + arg)
			case f.authReply != "":
				reply(f.authReply)
			case arg != want:
				reply("535 5.7.8 authentication failed")
			default:
				reply("235 2.7.0 authenticated")
			}
		case "MAIL":
			r.from = arg
			reply("250 2.1.0 ok")
		case "RCPT":
			r.to = arg
			if f.rcptReply != "" {
				rcpt := strings.TrimSuffix(strings.TrimPrefix(arg, "TO:<"), ">")
				reply(strings.NewReplacer("%s", rcpt, "%S", strings.ToUpper(rcpt)).Replace(f.rcptReply))
				continue
			}
			reply("250 2.1.5 ok")
		case "DATA":
			reply("354 go ahead")
			lines, err := tp.ReadDotLines()
			if err != nil {
				return
			}
			r.data = strings.Join(lines, "\r\n") + "\r\n"
			reply("250 2.0.0 queued")
		case "QUIT":
			f.mu.Lock()
			f.received = append(f.received, r)
			f.mu.Unlock()
			reply("221 2.0.0 bye")
			return
		default:
			reply("502 5.5.2 unknown command")
		}
	}
}

func (f *fakeSMTP) messages() []received {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]received(nil), f.received...)
}

func (f *fakeSMTP) mailer() *SMTPMailer {
	return &SMTPMailer{
		Host: "localhost", Port: f.port, Username: "no-reply@example.org", Password: f.password,
		From: "no-reply@example.org", RootCAs: f.rootCAs,
		Now: func() time.Time { return time.Date(2026, 10, 6, 9, 30, 0, 0, time.UTC) },
	}
}

func testContext(t *testing.T, d time.Duration) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	t.Cleanup(cancel)
	return ctx
}

func TestSMTPMailerSends(t *testing.T) {
	f := newFakeSMTP(t, nil)
	msg := LoginCodeMessage("alice@exemple.fr", "12345678", 10*time.Minute)
	if err := f.mailer().Send(testContext(t, 5*time.Second), msg); err != nil {
		t.Fatal(err)
	}
	got := f.messages()
	if len(got) != 1 {
		t.Fatalf("%d conversations, want 1", len(got))
	}
	if got[0].from != "FROM:<no-reply@example.org>" || got[0].to != "TO:<alice@exemple.fr>" {
		t.Errorf("envelope %q, %q", got[0].from, got[0].to)
	}
	data := got[0].data
	// The message of step 9, but its random Message-ID.
	want, err := compose("no-reply@example.org", msg, time.Date(2026, 10, 6, 9, 30, 0, 0, time.UTC), "ID@example.org")
	if err != nil {
		t.Fatal(err)
	}
	gotLines, wantLines := strings.Split(data, "\r\n"), strings.Split(string(want), "\r\n")
	if len(gotLines) != len(wantLines) {
		t.Fatalf("message:\n%q\nwant:\n%q", data, want)
	}
	for i := range gotLines {
		if strings.HasPrefix(wantLines[i], "Message-ID: ") {
			if !strings.HasPrefix(gotLines[i], "Message-ID: <") || !strings.HasSuffix(gotLines[i], "@example.org>") {
				t.Errorf("line %d = %q, want a Message-ID under example.org", i, gotLines[i])
			}
			continue
		}
		if gotLines[i] != wantLines[i] {
			t.Errorf("line %d = %q, want %q", i, gotLines[i], wantLines[i])
		}
	}
}

func TestSMTPMailerCheck(t *testing.T) {
	f := newFakeSMTP(t, nil)
	if err := f.mailer().Check(testContext(t, 5*time.Second)); err != nil {
		t.Fatal(err)
	}
	if got := f.messages(); len(got) != 1 || got[0].from != "" || got[0].data != "" {
		t.Errorf("check sent something: %+v", got)
	}
}

func TestSMTPMailerFailures(t *testing.T) {
	cases := []struct {
		name     string
		server   func(*fakeSMTP)
		client   func(*SMTPMailer)
		timeout  time.Duration
		wantStep string
		wantCode int
	}{
		{name: "authentication refused", client: func(m *SMTPMailer) { m.Password = "wrong" }, wantStep: StepAuth, wantCode: 535},
		{name: "recipient refused", server: func(f *fakeSMTP) { f.rcptReply = "550 5.1.1 <%s>: no such user" }, wantStep: StepTo, wantCode: 550},
		{name: "silent server", server: func(f *fakeSMTP) { f.silent = true }, timeout: 300 * time.Millisecond, wantStep: StepConnect},
		{name: "certificate of another name", client: func(m *SMTPMailer) { m.Host = "127.0.0.1" }, wantStep: StepConnect},
		{name: "certificate not trusted", client: func(m *SMTPMailer) { m.RootCAs = x509.NewCertPool() }, wantStep: StepConnect},
		{name: "nobody listening", client: func(m *SMTPMailer) { m.Port = closedPort(t) }, wantStep: StepConnect},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeSMTP(t, tc.server)
			m := f.mailer()
			if tc.client != nil {
				tc.client(m)
			}
			timeout := tc.timeout
			if timeout == 0 {
				timeout = 5 * time.Second
			}
			start := time.Now()
			err := m.Send(testContext(t, timeout), LoginCodeMessage("alice@exemple.fr", "12345678", 10*time.Minute))
			var se *SendError
			if !errors.As(err, &se) {
				t.Fatalf("error = %v, want a SendError", err)
			}
			if se.Step != tc.wantStep || se.Code != tc.wantCode {
				t.Errorf("step %q, code %d; want %q, %d (%v)", se.Step, se.Code, tc.wantStep, tc.wantCode, err)
			}
			if elapsed := time.Since(start); elapsed > timeout+time.Second {
				t.Errorf("took %v, beyond the deadline %v", elapsed, timeout)
			}
			if strings.Contains(err.Error(), "s3cret-pass") || strings.Contains(err.Error(), "wrong") {
				t.Errorf("error quotes the password: %v", err)
			}
		})
	}
}

// closedPort returns a local port nobody listens on.
func closedPort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	return port
}

// TestSendFailureLogs: an email not sent leaves one error record with the step and the SMTP
// code, but neither the address of the recipient, which the server quotes, nor the
// password, nor the code (plan production, step 13).
func TestSendFailureLogs(t *testing.T) {
	cases := []struct {
		name     string
		server   func(*fakeSMTP)
		wantStep string
		wantCode int
	}{
		{name: "recipient refused", server: func(f *fakeSMTP) { f.rcptReply = "550 5.1.1 <%s>: no such user (%S)" }, wantStep: StepTo, wantCode: 550},
		{name: "authentication refused", server: func(f *fakeSMTP) { f.authReply = "535 5.7.8 bad credentials for s3cret-pass" }, wantStep: StepAuth, wantCode: 535},
		// Review of PR #46, point 1: the password, encoded in the command quoted.
		{name: "authentication refused, command quoted", server: func(f *fakeSMTP) { f.authEcho = true }, wantStep: StepAuth, wantCode: 535},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeSMTP(t, tc.server)
			var logs bytes.Buffer
			o := NewOutbox(f.mailer(), 10*time.Minute, slog.New(slog.NewJSONHandler(&logs, nil)))
			o.SendLoginCode("alice@exemple.fr", "12345678")
			o.Wait()

			var record map[string]any
			if err := json.Unmarshal(logs.Bytes(), &record); err != nil {
				t.Fatalf("logs = %q, want one JSON record: %v", logs.String(), err)
			}
			if record["level"] != "ERROR" || record["msg"] != "login code not sent" || record["step"] != tc.wantStep || record["code"] != float64(tc.wantCode) {
				t.Errorf("record = %v, want an error with step %q and code %d", record, tc.wantStep, tc.wantCode)
			}
			lower := strings.ToLower(logs.String())
			encoded := strings.ToLower(base64.StdEncoding.EncodeToString([]byte("\x00no-reply@example.org\x00s3cret-pass")))
			for _, secret := range []string{"alice@exemple.fr", "s3cret-pass", "12345678", encoded, encoded[:24]} {
				if strings.Contains(lower, secret) {
					t.Errorf("logs contain %q: %s", secret, logs.String())
				}
			}
		})
	}
}
