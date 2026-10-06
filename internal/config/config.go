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
	"strconv"
	"strings"
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
		return fmt.Errorf("key %q: want a %s", typeErr.Field, typeErr.Type)
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
	return nil
}

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
