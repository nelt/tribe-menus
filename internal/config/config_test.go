package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const smtp = `"smtp": {"host": "smtp.example.org", "port": 465, "username": "no-reply@example.org", "from": "no-reply@example.org"}`

var exampleSMTP = SMTP{Host: "smtp.example.org", Port: 465, Username: "no-reply@example.org", From: "no-reply@example.org"}

// withSMTP returns a config file with the keys data, listen and baseURL, and this smtp key.
func withSMTP(smtp string) string {
	return `{"data": "/srv", "listen": "systemd", "baseURL": "https://a.example.org", "smtp": ` + smtp + `}`
}

func TestRead(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    Config
		wantErr string
	}{
		{
			name:    "systemd",
			content: `{"data": "/var/lib/tribe-menus/recette", "listen": "systemd", "baseURL": "https://recette.example.org", ` + smtp + `}`,
			want:    Config{Data: "/var/lib/tribe-menus/recette", Listen: "systemd", BaseURL: "https://recette.example.org", SMTP: exampleSMTP},
		},
		{
			name:    "TCP",
			content: `{"data": "/srv/data", "listen": "127.0.0.1:8080", "baseURL": "https://tribes.example.org:8443", ` + smtp + `}`,
			want:    Config{Data: "/srv/data", Listen: "127.0.0.1:8080", BaseURL: "https://tribes.example.org:8443", SMTP: exampleSMTP},
		},
		{name: "empty file", content: "", wantErr: "config: empty file"},
		{name: "blank file", content: " \n", wantErr: "config: empty file"},
		{name: "not JSON", content: "data = /srv", wantErr: "invalid JSON at byte"},
		{name: "truncated", content: `{"data": "/srv"`, wantErr: "invalid JSON"},
		{name: "array", content: `["/srv"]`, wantErr: "invalid JSON"},
		{name: "two documents", content: `{"data": "/srv", "listen": "systemd", "baseURL": "https://a.example.org"} {}`, wantErr: "more than one JSON document"},
		{name: "unknown key", content: `{"data": "/srv", "listen": "systemd", "baseURL": "https://a.example.org", "secret": "hunter2"}`, wantErr: `unknown key "secret"`},
		{name: "wrong type", content: `{"data": 42}`, wantErr: `key "data": want a string`},
		{name: "missing data", content: `{"listen": "systemd", "baseURL": "https://a.example.org"}`, wantErr: `key "data": missing`},
		{name: "relative data", content: `{"data": "data", "listen": "systemd", "baseURL": "https://a.example.org"}`, wantErr: `key "data": want an absolute path`},
		{name: "missing listen", content: `{"data": "/srv", "baseURL": "https://a.example.org"}`, wantErr: `key "listen": missing`},
		{name: "listen without port", content: `{"data": "/srv", "listen": "localhost", "baseURL": "https://a.example.org"}`, wantErr: `key "listen": want`},
		{name: "listen without host", content: `{"data": "/srv", "listen": ":8080", "baseURL": "https://a.example.org"}`, wantErr: `key "listen": want`},
		{name: "listen port out of range", content: `{"data": "/srv", "listen": "localhost:70000", "baseURL": "https://a.example.org"}`, wantErr: `key "listen": want`},
		{name: "missing baseURL", content: `{"data": "/srv", "listen": "systemd"}`, wantErr: `key "baseURL": missing`},
		{name: "baseURL over HTTP", content: `{"data": "/srv", "listen": "systemd", "baseURL": "http://a.example.org"}`, wantErr: `key "baseURL": want https://host`},
		{name: "baseURL with path", content: `{"data": "/srv", "listen": "systemd", "baseURL": "https://a.example.org/tribes"}`, wantErr: `key "baseURL": want https://host`},
		{name: "baseURL with trailing slash", content: `{"data": "/srv", "listen": "systemd", "baseURL": "https://a.example.org/"}`, wantErr: `key "baseURL": want https://host`},
		{name: "baseURL with user", content: `{"data": "/srv", "listen": "systemd", "baseURL": "https://me@a.example.org"}`, wantErr: `key "baseURL": want https://host`},
		{name: "missing smtp", content: `{"data": "/srv", "listen": "systemd", "baseURL": "https://a.example.org"}`, wantErr: `key "smtp.host": missing`},
		{name: "null smtp", content: withSMTP(`null`), wantErr: `key "smtp.host": missing`},
		{name: "smtp not an object", content: withSMTP(`"smtp.example.org"`), wantErr: `key "smtp": want an object`},
		{name: "invalid smtp host", content: withSMTP(`{"host": "smtp.example.org:465"}`), wantErr: `key "smtp.host": want a host name`},
		{name: "missing smtp port", content: withSMTP(`{"host": "smtp.example.org"}`), wantErr: `key "smtp.port": missing`},
		{name: "smtp port as text", content: withSMTP(`{"host": "smtp.example.org", "port": "465"}`), wantErr: `key "smtp.port": want a number`},
		{name: "smtp port out of range", content: withSMTP(`{"host": "smtp.example.org", "port": 70000}`), wantErr: `key "smtp.port": want a port number`},
		{name: "missing smtp username", content: withSMTP(`{"host": "smtp.example.org", "port": 465}`), wantErr: `key "smtp.username": missing`},
		{name: "smtp username with a space", content: withSMTP(`{"host": "smtp.example.org", "port": 465, "username": "no reply"}`), wantErr: `key "smtp.username": want printable ASCII`},
		{name: "missing smtp from", content: withSMTP(`{"host": "smtp.example.org", "port": 465, "username": "u"}`), wantErr: `key "smtp.from": missing`},
		{name: "smtp from with a name", content: withSMTP(`{"host": "smtp.example.org", "port": 465, "username": "u", "from": "Melting Tribe <no-reply@example.org>"}`), wantErr: `key "smtp.from": want an email address`},
		{name: "smtp from in upper case", content: withSMTP(`{"host": "smtp.example.org", "port": 465, "username": "u", "from": "No-Reply@example.org"}`), wantErr: `key "smtp.from": want an email address`},
		{name: "smtp password", content: withSMTP(`{"host": "smtp.example.org", "port": 465, "username": "u", "from": "a@example.org", "password": "x"}`), wantErr: `unknown key "password"`},
		// Review of PR #45, point 3: what encoding/json would accept.
		{name: "key written twice", content: `{"data": "/srv", "listen": "systemd", "listen": "0.0.0.0:80", "baseURL": "https://a.example.org", ` + smtp + `}`, wantErr: `key "listen": written twice`},
		{name: "key in upper case", content: `{"data": "/srv", "LISTEN": "systemd", "baseURL": "https://a.example.org", ` + smtp + `}`, wantErr: `unknown key "LISTEN"`},
		{name: "key in another case", content: `{"data": "/srv", "listen": "systemd", "baseurl": "https://a.example.org", ` + smtp + `}`, wantErr: `unknown key "baseurl"`},
		{name: "nested key written twice", content: withSMTP(`{"host": "smtp.example.org", "host": "evil.example.org", "port": 465, "username": "u", "from": "a@example.org"}`), wantErr: `key "smtp.host": written twice`},
		{name: "nested key in another case", content: withSMTP(`{"Host": "smtp.example.org", "port": 465, "username": "u", "from": "a@example.org"}`), wantErr: `unknown key "smtp.Host"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Read(strings.NewReader(tc.content))
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want it to contain %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("config = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// The errors name the key, never its value, which may be a mistyped secret.
func TestReadErrorsDoNotQuoteValues(t *testing.T) {
	for _, content := range []string{
		`{"data": "hunter2", "listen": "systemd", "baseURL": "https://a.example.org"}`,
		`{"data": "/srv", "listen": "hunter2", "baseURL": "https://a.example.org"}`,
		`{"data": "/srv", "listen": "systemd", "baseURL": "hunter2"}`,
		`{"data": "/srv", "listen": "systemd", "baseURL": "https://a.example.org", "password": "hunter2"}`,
		`{"data": "/srv", "listen": hunter2}`,
		withSMTP(`{"host": "hunter2:1", "port": 465, "username": "u", "from": "a@example.org"}`),
		withSMTP(`{"host": "smtp.example.org", "port": 465, "username": "hunter2 x", "from": "a@example.org"}`),
		withSMTP(`{"host": "smtp.example.org", "port": 465, "username": "u", "from": "hunter2"}`),
		withSMTP(`{"host": "smtp.example.org", "port": 465, "username": "u", "from": "a@example.org", "password": "hunter2"}`),
	} {
		_, err := Read(strings.NewReader(content))
		if err == nil || strings.Contains(err.Error(), "hunter2") {
			t.Errorf("Read(%s): error %v, want one without the value", content, err)
		}
	}
}

func TestReadTooLarge(t *testing.T) {
	content := `{"data": "/srv", "listen": "systemd", "baseURL": "https://a.example.org"}` + strings.Repeat(" ", maxSize)
	if _, err := Read(strings.NewReader(content)); err == nil || !strings.Contains(err.Error(), "larger than") {
		t.Errorf("error = %v, want the size limit", err)
	}
}

func TestLoad(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "missing.json")); err == nil || !strings.HasPrefix(err.Error(), "config: ") {
		t.Errorf("missing file: error = %v", err)
	}
}

// The example of deploy/ stays readable by the real reader.
func TestExample(t *testing.T) {
	c, err := Load(filepath.Join("..", "..", "deploy", "config.example.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !c.Systemd() {
		t.Errorf("example listens on %q, want the socket passed by systemd", c.Listen)
	}
	content, err := os.ReadFile(filepath.Join("..", "..", "deploy", "config.example.json"))
	if err != nil {
		t.Fatal(err)
	}
	// Nothing specific to the instance in the repository (ADR 0011, point 5).
	if strings.Contains(string(content), "codingmatters") {
		t.Errorf("example names the real instance: %s", content)
	}
}
