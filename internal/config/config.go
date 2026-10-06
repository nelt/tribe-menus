// Package config reads the config file of the server mode (plan production, D1): a JSON
// document kept on the server, outside of the repository (ADR 0011, point 5; ADR 0015,
// point 9).
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/nelt/tribe-menus/internal/tribe"
)

// ListenSystemd is the value of the listen key for the socket passed by systemd (ADR 0015,
// point 11). Any other value is a TCP address (ADR 0006, point 8).
const ListenSystemd = "systemd"

// maxSize bounds the config file: a few hundred bytes are expected.
const maxSize = 64 << 10

// Config is the content of the config file.
type Config struct {
	// Data is the directory of the databases, an absolute path.
	Data string `json:"data"`
	// Listen is ListenSystemd or a TCP address, host:port.
	Listen string `json:"listen"`
	// BaseURL is the public address of the instance, https://host without path.
	BaseURL string `json:"baseURL"`
	// SMTP is the server sending the emails (ADR 0014). Its password is not here: it is a
	// systemd credential (ADR 0015, point 10).
	SMTP SMTP `json:"smtp"`
	// Alerts are the alerts sent by email by the application (ADR 0023).
	Alerts Alerts `json:"alerts"`
}

// Alerts is the alerts key of the config file.
type Alerts struct {
	// To is the address of the administrator, who receives the alert on the code request
	// limits.
	To string `json:"to"`
}

// SMTP is the smtp key of the config file.
type SMTP struct {
	// Host is the name of the server, which its certificate must carry.
	Host string `json:"host"`
	// Port is the port of TLS from the start, 465 for the MX Plan.
	Port int `json:"port"`
	// Username is the account of the authentication.
	Username string `json:"username"`
	// From is the sending address.
	From string `json:"from"`
}

// keys are the keys of the config file, exactly: a nested map for an object.
var keys = map[string]map[string]bool{
	"data":    nil,
	"listen":  nil,
	"baseURL": nil,
	"smtp":    {"host": true, "port": true, "username": true, "from": true},
	"alerts":  {"to": true},
}

// Systemd reports whether the server listens on the socket passed by systemd, hence behind
// the proxy (plan production, D3).
func (c Config) Systemd() bool {
	return c.Listen == ListenSystemd
}

// Load reads and validates the config file at path.
func Load(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	defer f.Close()
	return Read(f)
}

// Read reads and validates a config file. Its errors name the faulty key, never a value.
func Read(r io.Reader) (Config, error) {
	content, err := io.ReadAll(io.LimitReader(r, maxSize+1))
	if err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	if len(content) > maxSize {
		return Config{}, fmt.Errorf("config: larger than %d bytes", maxSize)
	}
	if len(bytes.TrimSpace(content)) == 0 {
		return Config{}, errors.New("config: empty file")
	}

	dec := json.NewDecoder(bytes.NewReader(content))
	dec.DisallowUnknownFields()
	var c Config
	if err := dec.Decode(&c); err != nil {
		return Config{}, fmt.Errorf("config: %w", decodeError(err))
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return Config{}, errors.New("config: more than one JSON document")
	}
	if err := checkKeys(content); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	if err := c.validate(); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	return c, nil
}

// decodeError rewords the errors of encoding/json that would quote the content.
func decodeError(err error) error {
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		if typeErr.Field == "" {
			return errors.New("invalid JSON: not an object")
		}
		return fmt.Errorf("key %q: want %s", typeErr.Field, kindName(typeErr.Type.Kind()))
	}
	if field, ok := strings.CutPrefix(err.Error(), "json: unknown field "); ok {
		return fmt.Errorf("unknown key %s", field)
	}
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return fmt.Errorf("invalid JSON at byte %d", syntaxErr.Offset)
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return errors.New("invalid JSON: unexpected end")
	}
	return errors.New("invalid JSON: not an object")
}

// checkKeys refuses what encoding/json accepts: a key written twice, of which it keeps the
// last value, and a key in another case, which it matches anyway (review of PR #45, point
// 3). content is a JSON object already decoded once.
func checkKeys(content []byte) error {
	dec := json.NewDecoder(bytes.NewReader(content))
	if _, err := dec.Token(); err != nil { // {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	seen := map[string]bool{}
	for dec.More() {
		key, err := nextKey(dec, seen, "")
		if err != nil {
			return err
		}
		nested, known := keys[key]
		if !known {
			return fmt.Errorf("unknown key %q", key)
		}
		if nested == nil {
			if err := skipValue(dec); err != nil {
				return err
			}
			continue
		}
		if err := checkNestedKeys(dec, key, nested); err != nil {
			return err
		}
	}
	return nil
}

// checkNestedKeys checks the keys of the object of the key parent.
func checkNestedKeys(dec *json.Decoder, parent string, nested map[string]bool) error {
	tok, err := dec.Token()
	if err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if tok != json.Delim('{') {
		return nil // null: the key is missing, which validate says
	}
	seen := map[string]bool{}
	for dec.More() {
		key, err := nextKey(dec, seen, parent+".")
		if err != nil {
			return err
		}
		if !nested[key] {
			return fmt.Errorf("unknown key %q", parent+"."+key)
		}
		if err := skipValue(dec); err != nil {
			return err
		}
	}
	_, err = dec.Token() // }
	return err
}

func nextKey(dec *json.Decoder, seen map[string]bool, prefix string) (string, error) {
	tok, err := dec.Token()
	if err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}
	key, _ := tok.(string)
	if seen[key] {
		return "", fmt.Errorf("key %q: written twice", prefix+key)
	}
	seen[key] = true
	return key, nil
}

func skipValue(dec *json.Decoder) error {
	var v json.RawMessage
	return dec.Decode(&v)
}

// kindName names the JSON type of a Go kind of the config.
func kindName(k reflect.Kind) string {
	switch k {
	case reflect.String:
		return "a string"
	case reflect.Int:
		return "a number"
	default:
		return "an object"
	}
}

func (c Config) validate() error {
	switch {
	case c.Data == "":
		return missing("data")
	case !filepath.IsAbs(c.Data):
		return errors.New(`key "data": want an absolute path`)
	case c.Listen == "":
		return missing("listen")
	case c.Listen != ListenSystemd && !validTCPAddress(c.Listen):
		return fmt.Errorf(`key "listen": want %q or host:port`, ListenSystemd)
	case c.BaseURL == "":
		return missing("baseURL")
	case !validBaseURL(c.BaseURL):
		return errors.New(`key "baseURL": want https://host, without path`)
	}
	if err := c.SMTP.validate(); err != nil {
		return err
	}
	return c.Alerts.validate()
}

func (a Alerts) validate() error {
	if a.To == "" {
		return missing("alerts.to")
	}
	if !lowerCaseEmail(a.To) {
		return errors.New(`key "alerts.to": want an email address, in lower case`)
	}
	return nil
}

// lowerCaseEmail reports whether s is an address that ParseEmail accepts as is.
func lowerCaseEmail(s string) bool {
	email, err := tribe.ParseEmail(s)
	return err == nil && string(email) == s
}

func (s SMTP) validate() error {
	switch {
	case s.Host == "":
		return missing("smtp.host")
	case !hostPattern.MatchString(s.Host):
		return errors.New(`key "smtp.host": want a host name`)
	case s.Port == 0:
		return missing("smtp.port")
	case s.Port < 0 || s.Port > 65535:
		return errors.New(`key "smtp.port": want a port number`)
	case s.Username == "":
		return missing("smtp.username")
	case strings.ContainsFunc(s.Username, func(r rune) bool { return r < 0x21 || r > 0x7e }):
		return errors.New(`key "smtp.username": want printable ASCII`)
	case s.From == "":
		return missing("smtp.from")
	}
	if !lowerCaseEmail(s.From) {
		return errors.New(`key "smtp.from": want an email address, in lower case`)
	}
	return nil
}

// hostPattern: labels of letters, digits and hyphens.
var hostPattern = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?(?:\.[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?)*$`)

func missing(key string) error {
	return fmt.Errorf("key %q: missing", key)
}

func validTCPAddress(addr string) bool {
	host, port, err := net.SplitHostPort(addr)
	if err != nil || host == "" {
		return false
	}
	n, err := strconv.Atoi(port)
	return err == nil && n >= 0 && n <= 65535
}

func validBaseURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && u.Scheme == "https" && u.Host != "" && u.User == nil &&
		u.Path == "" && u.RawQuery == "" && u.Fragment == "" && !u.ForceQuery && u.Opaque == ""
}
