// Package recetteconfig reads the values of the instance and the password of the box used by
// the automated acceptance run (plan recette-auto, D10): recette-auto.json and
// recette-auto.pop3, in a volume of the Dev Container, never in the repository. The same
// recette-auto.json is read by web/recette/config.ts. No error shows the password.
package recetteconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Files of the directory.
const (
	ConfigFile   = "recette-auto.json"
	PasswordFile = "recette-auto.pop3"
)

// Config is the content of recette-auto.json (model: deploy/recette-auto.example.json).
type Config struct {
	// TribeURL is the address of the tribe recette-auto on the staging instance,
	// https://<host>/tribes/<slug>/.
	TribeURL string `json:"tribeURL"`
	// Address is the only member of the tribe, a real box of the MX Plan.
	Address string `json:"address"`
	POP3    struct {
		Host string `json:"host"`
		Port int    `json:"port"`
		User string `json:"user"`
	} `json:"pop3"`
}

// DefaultDir is ~/.config/tribe-menus, where the volume tribe-menus-recette is mounted.
func DefaultDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "tribe-menus")
}

// Read reads and checks the configuration and the password of dir. The password file must be
// readable by its owner only.
func Read(dir string) (Config, string, error) {
	var cfg Config
	b, err := os.ReadFile(filepath.Join(dir, ConfigFile))
	if err != nil {
		return cfg, "", fmt.Errorf("read %s: %w", ConfigFile, err)
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, "", fmt.Errorf("read %s: %w", ConfigFile, err)
	}
	if cfg.Address == "" || cfg.POP3.Host == "" || cfg.POP3.Port == 0 || cfg.POP3.User == "" {
		return cfg, "", fmt.Errorf("%s: address, pop3.host, pop3.port and pop3.user needed", ConfigFile)
	}
	if _, err := cfg.Origin(); err != nil {
		return cfg, "", err
	}
	path := filepath.Join(dir, PasswordFile)
	info, err := os.Stat(path)
	if err != nil {
		return cfg, "", fmt.Errorf("read %s: %w", PasswordFile, err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return cfg, "", fmt.Errorf("%s: readable by others, want mode 0600", PasswordFile)
	}
	b, err = os.ReadFile(path)
	if err != nil {
		return cfg, "", fmt.Errorf("read %s: %w", PasswordFile, err)
	}
	password := strings.TrimRight(string(b), "\r\n")
	if password == "" {
		return cfg, "", fmt.Errorf("%s: empty", PasswordFile)
	}
	return cfg, password, nil
}

// Origin returns https://<host> of the tribe URL, checked to be https://<host>/tribes/<slug>/.
func (c Config) Origin() (string, error) {
	u, err := url.Parse(c.TribeURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New(ConfigFile + ": tribeURL: want https://<host>/tribes/<slug>/")
	}
	parts := strings.Split(u.Path, "/")
	if len(parts) != 4 || parts[1] != "tribes" || parts[2] == "" || parts[3] != "" {
		return "", errors.New(ConfigFile + ": tribeURL: want https://<host>/tribes/<slug>/")
	}
	return u.Scheme + "://" + u.Host, nil
}
