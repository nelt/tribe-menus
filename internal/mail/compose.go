package mail

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"time"
)

// compose returns the message as sent by SMTP, with CRLF line endings: headers written by
// net/mail and mime, body in quoted-printable (plan production, step 9). from is the
// sending address, id the Message-ID without its angle brackets. Both addresses must be
// bare addresses, read back unchanged by net/mail: nothing written by a member reaches a
// header other than as an address.
func compose(from string, msg Message, date time.Time, id string) ([]byte, error) {
	if !bareAddress(from) {
		return nil, errors.New("compose: invalid sending address")
	}
	if !bareAddress(msg.To) {
		return nil, errors.New("compose: invalid recipient address")
	}
	if strings.ContainsAny(id, "<>\r\n ") || strings.ContainsAny(msg.Subject, "\r\n") {
		return nil, errors.New("compose: invalid header")
	}

	var b bytes.Buffer
	header := func(name, value string) {
		b.WriteString(name + ": " + value + "\r\n")
	}
	header("From", (&mail.Address{Name: appName, Address: from}).String())
	header("To", (&mail.Address{Address: msg.To}).String())
	header("Subject", mime.QEncoding.Encode("utf-8", msg.Subject))
	header("Date", date.Format(time.RFC1123Z))
	header("Message-ID", "<"+id+">")
	header("MIME-Version", "1.0")
	header("Content-Type", "text/plain; charset=utf-8")
	header("Content-Transfer-Encoding", "quoted-printable")
	header("Auto-Submitted", "auto-generated")
	b.WriteString("\r\n")

	body := quotedprintable.NewWriter(&b)
	if _, err := body.Write([]byte(msg.Body)); err != nil {
		return nil, fmt.Errorf("compose: %w", err)
	}
	if err := body.Close(); err != nil {
		return nil, fmt.Errorf("compose: %w", err)
	}
	return b.Bytes(), nil
}

// bareAddress reports whether s is an address alone, read back unchanged by net/mail.
func bareAddress(s string) bool {
	if strings.ContainsAny(s, "\r\n<>\" ") {
		return false
	}
	a, err := mail.ParseAddress(s)
	return err == nil && a.Name == "" && a.Address == s
}

// newMessageID returns a random Message-ID under the domain of the sending address.
func newMessageID(from string) string {
	_, domain, _ := strings.Cut(from, "@")
	return rand.Text() + "@" + domain
}
