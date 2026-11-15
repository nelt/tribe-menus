package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/quotedprintable"
	"net/mail"
	"regexp"
	"strings"
	"time"
)

// clockSkew is how much earlier than the request a message may be dated: its Date header,
// written by the server of the instance, has a resolution of a second, and the two clocks
// are not exactly the same.
const clockSkew = 30 * time.Second

// codeLine finds the code in the text of the message (internal/mail.LoginCodeMessage;
// ADR 0014, point 4): the line that announces it, ending with eight digits.
var codeLine = regexp.MustCompile(`(?m)^Votre code de connexion à .+ : (\d{8})\s*$`)

// received is a message of the box, as far as the fetch is concerned.
type received struct {
	number int
	to     []string
	date   time.Time
	code   string
}

// parse reads the recipients, the date and the code of a message. A message without a code
// is not an error: it is not a login code, and the fetch passes it over.
func parse(number int, raw []byte) (received, error) {
	m, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return received{}, fmt.Errorf("message %d: unreadable", number)
	}
	r := received{number: number}
	if r.date, err = m.Header.Date(); err != nil {
		return received{}, fmt.Errorf("message %d: no valid Date", number)
	}
	if to, err := m.Header.AddressList("To"); err == nil {
		for _, a := range to {
			r.to = append(r.to, a.Address)
		}
	}
	var body io.Reader = m.Body
	if strings.EqualFold(m.Header.Get("Content-Transfer-Encoding"), "quoted-printable") {
		body = quotedprintable.NewReader(m.Body)
	}
	text, err := io.ReadAll(body)
	if err != nil {
		return received{}, fmt.Errorf("message %d: unreadable body", number)
	}
	if found := codeLine.FindAllSubmatch(bytes.ReplaceAll(text, []byte("\r\n"), []byte("\n")), -1); len(found) == 1 {
		r.code = string(found[0][1])
	}
	return r, nil
}

// addressedTo reports whether the message is addressed to address, case aside.
func (r received) addressedTo(address string) bool {
	for _, to := range r.to {
		if strings.EqualFold(to, address) {
			return true
		}
	}
	return false
}

// errNoCode is the outcome of a fetch that found no code yet.
var errNoCode = errors.New("no login code yet")

// pick returns the code of the most recent message addressed to address, dated after since
// (clockSkew aside), and the numbers of the messages to delete: those addressed to address,
// older codes included; the others stay in the box.
func pick(messages []received, address string, since time.Time) (string, []int, error) {
	var code string
	var latest time.Time
	var remove []int
	for _, m := range messages {
		if !m.addressedTo(address) {
			continue
		}
		remove = append(remove, m.number)
		if m.code == "" || m.date.Before(since.Add(-clockSkew)) {
			continue
		}
		if code == "" || !m.date.Before(latest) {
			code, latest = m.code, m.date
		}
	}
	if code == "" {
		return "", remove, errNoCode
	}
	return code, remove, nil
}
