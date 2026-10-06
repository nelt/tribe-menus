package mail

import (
	"strings"
	"testing"
	"time"
)

func TestCompose(t *testing.T) {
	date := time.Date(2026, 10, 6, 9, 30, 0, 0, time.FixedZone("CEST", 2*60*60))
	msg := LoginCodeMessage("alice@exemple.fr", "12345678", 10*time.Minute)
	got, err := compose("no-reply@example.org", msg, date, "ABC123@example.org")
	if err != nil {
		t.Fatal(err)
	}
	want := "From: \"Melting Tribe\" <no-reply@example.org>\r\n" +
		"To: <alice@exemple.fr>\r\n" +
		"Subject: Votre code de connexion Melting Tribe\r\n" +
		"Date: Tue, 06 Oct 2026 09:30:00 +0200\r\n" +
		"Message-ID: <ABC123@example.org>\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n" +
		"Content-Transfer-Encoding: quoted-printable\r\n" +
		"Auto-Submitted: auto-generated\r\n" +
		"\r\n" +
		"Bonjour,\r\n" +
		"\r\n" +
		"Votre code de connexion =C3=A0 Melting Tribe : 12345678\r\n" +
		"\r\n" +
		"Il est valable 10 minutes. Si vous n'avez pas demand=C3=A9 ce code, ignorez=\r\n" +
		" ce message.\r\n" +
		"\r\n" +
		"Melting Tribe\r\n"
	if string(got) != want {
		t.Errorf("message:\n%q\nwant:\n%q", got, want)
	}
}

func TestComposeEncodesSubject(t *testing.T) {
	got, err := compose("no-reply@example.org", Message{To: "alice@exemple.fr", Subject: "Code à saisir", Body: "x"}, time.Unix(0, 0), "id@example.org")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "\r\nSubject: =?utf-8?q?Code_=C3=A0_saisir?=\r\n") {
		t.Errorf("subject not encoded: %q", got)
	}
}

// Nothing can add a header or a recipient: compose refuses anything but bare addresses.
func TestComposeRefusesInjection(t *testing.T) {
	cases := []struct {
		name, from, to, subject, id string
	}{
		{name: "line feed in recipient", to: "alice@exemple.fr\r\nBcc: x@example.org"},
		{name: "two recipients", to: "alice@exemple.fr,bruno@exemple.fr"},
		{name: "display name", to: "Alice <alice@exemple.fr>"},
		{name: "quoted local part", to: `"a b"@exemple.fr`},
		{name: "empty recipient", to: ""},
		{name: "invalid sender", from: "Melting Tribe <no-reply@example.org>"},
		{name: "line feed in subject", subject: "Code\r\nBcc: x@example.org"},
		{name: "invalid id", id: "a>\r\nBcc: x@example.org"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			from, to, subject, id := "no-reply@example.org", "alice@exemple.fr", "Code", "id@example.org"
			if tc.from != "" {
				from = tc.from
			}
			if tc.to != "" || tc.name == "empty recipient" {
				to = tc.to
			}
			if tc.subject != "" {
				subject = tc.subject
			}
			if tc.id != "" {
				id = tc.id
			}
			if _, err := compose(from, Message{To: to, Subject: subject, Body: "x"}, time.Unix(0, 0), id); err == nil {
				t.Error("compose accepted it")
			}
		})
	}
}

func TestNewMessageID(t *testing.T) {
	a, b := newMessageID("no-reply@example.org"), newMessageID("no-reply@example.org")
	if a == b || !strings.HasSuffix(a, "@example.org") || len(a) < 20 {
		t.Errorf("message IDs %q and %q", a, b)
	}
}
