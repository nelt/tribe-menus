package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"time"
)

// pop3 is a client of the few commands of POP3 (RFC 1939) that a fetch needs. An error
// never carries the text of an answer of the server, which may echo the account, nor
// anything of a message.
type pop3 struct {
	conn net.Conn
	text *textproto.Conn
}

// dialTLS opens a connection to a POP3 server over TLS from the start (port 995): TLS 1.2 at
// least, certificate checked for the host name; roots nil means the roots of the system.
func dialTLS(ctx context.Context, addr string, roots *x509.CertPool) (net.Conn, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("pop3 address: %w", err)
	}
	d := tls.Dialer{Config: &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12, RootCAs: roots}}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("pop3 connection: %w", err)
	}
	return conn, nil
}

// newPOP3 reads the greeting of the server on conn. Every exchange must end before the
// deadline, if any, of ctx.
func newPOP3(ctx context.Context, conn net.Conn) (*pop3, error) {
	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return nil, fmt.Errorf("pop3 deadline: %w", err)
		}
	}
	c := &pop3{conn: conn, text: textproto.NewConn(conn)}
	if _, err := c.status("greeting"); err != nil {
		return nil, err
	}
	return c, nil
}

// errRefused is an answer -ERR of the server.
var errRefused = errors.New("refused by the server")

// status reads a status line, +OK or -ERR, and returns what follows +OK.
func (c *pop3) status(command string) (string, error) {
	line, err := c.text.ReadLine()
	if err != nil {
		return "", fmt.Errorf("pop3 %s: %w", command, err)
	}
	switch {
	case line == "+OK" || strings.HasPrefix(line, "+OK "):
		return strings.TrimPrefix(strings.TrimPrefix(line, "+OK"), " "), nil
	case line == "-ERR" || strings.HasPrefix(line, "-ERR "):
		return "", fmt.Errorf("pop3 %s: %w", command, errRefused)
	default:
		return "", fmt.Errorf("pop3 %s: unexpected answer", command)
	}
}

// cmd sends a command and reads its status line. name is the command alone, for the
// errors: the arguments (the password) never reach them.
func (c *pop3) cmd(name, line string) (string, error) {
	if err := c.text.PrintfLine("%s", line); err != nil {
		return "", fmt.Errorf("pop3 %s: %w", name, err)
	}
	return c.status(name)
}

// login authenticates with USER and PASS.
func (c *pop3) login(user, password string) error {
	if strings.ContainsAny(user, "\r\n") || strings.ContainsAny(password, "\r\n") {
		return errors.New("pop3 login: line break in the account or the password")
	}
	if _, err := c.cmd("USER", "USER "+user); err != nil {
		return err
	}
	_, err := c.cmd("PASS", "PASS "+password)
	return err
}

// list returns the numbers of the messages of the box.
func (c *pop3) list() ([]int, error) {
	if _, err := c.cmd("LIST", "LIST"); err != nil {
		return nil, err
	}
	lines, err := c.text.ReadDotLines()
	if err != nil {
		return nil, fmt.Errorf("pop3 LIST: %w", err)
	}
	numbers := make([]int, 0, len(lines))
	for _, line := range lines {
		field, _, _ := strings.Cut(line, " ")
		n, err := strconv.Atoi(field)
		if err != nil || n < 1 {
			return nil, errors.New("pop3 LIST: unexpected line")
		}
		numbers = append(numbers, n)
	}
	return numbers, nil
}

// retr returns a message, dot-unstuffed, with LF line endings.
func (c *pop3) retr(n int) ([]byte, error) {
	if _, err := c.cmd("RETR", "RETR "+strconv.Itoa(n)); err != nil {
		return nil, err
	}
	b, err := c.text.ReadDotBytes()
	if err != nil {
		return nil, fmt.Errorf("pop3 RETR: %w", err)
	}
	return b, nil
}

// dele marks a message for deletion, done by QUIT.
func (c *pop3) dele(n int) error {
	_, err := c.cmd("DELE", "DELE "+strconv.Itoa(n))
	return err
}

// quit ends the session: the server deletes then the messages marked by DELE.
func (c *pop3) quit() error {
	_, err := c.cmd("QUIT", "QUIT")
	return err
}

// close closes the connection, without QUIT: nothing marked is deleted.
func (c *pop3) close() error {
	return c.conn.Close()
}

// pollTimeout bounds a session of the fetch, from the connection to QUIT.
const pollTimeout = 30 * time.Second
