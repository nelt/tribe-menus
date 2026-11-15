package recetteconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const password = "secret-du-test"

func TestRead(t *testing.T) {
	const good = `{"tribeURL": "https://recette.example.org/tribes/x/", "address": "a@example.org", "pop3": {"host": "pop3.example.org", "port": 995, "user": "a@example.org"}}`
	tests := []struct {
		name     string
		config   string
		password string
		mode     os.FileMode
		wantErr  string
	}{
		{name: "complete", config: good, password: password + "\n", mode: 0o600},
		{name: "no configuration", password: password, mode: 0o600, wantErr: "read recette-auto.json"},
		{name: "missing port", config: `{"tribeURL": "https://h/tribes/x/", "address": "a@example.org", "pop3": {"host": "h", "user": "u"}}`, password: password, mode: 0o600, wantErr: "pop3.port"},
		{name: "tribe URL over http", config: strings.Replace(good, "https:", "http:", 1), password: password, mode: 0o600, wantErr: "tribeURL"},
		{name: "tribe URL without tribe", config: strings.Replace(good, "/tribes/x/", "/", 1), password: password, mode: 0o600, wantErr: "tribeURL"},
		{name: "tribe URL without final slash", config: strings.Replace(good, "/tribes/x/", "/tribes/x", 1), password: password, mode: 0o600, wantErr: "tribeURL"},
		{name: "no password", config: good, wantErr: "read recette-auto.pop3"},
		{name: "password readable by others", config: good, password: password, mode: 0o644, wantErr: "want mode 0600"},
		{name: "empty password", config: good, password: "\n", mode: 0o600, wantErr: "empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.config != "" {
				if err := os.WriteFile(filepath.Join(dir, ConfigFile), []byte(tt.config), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if tt.mode != 0 {
				path := filepath.Join(dir, PasswordFile)
				if err := os.WriteFile(path, []byte(tt.password), tt.mode); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(path, tt.mode); err != nil {
					t.Fatal(err)
				}
			}
			cfg, got, err := Read(dir)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) || strings.Contains(err.Error(), password) {
					t.Fatalf("Read: %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil || got != password || cfg.POP3.Port != 995 {
				t.Errorf("Read: %+v, %q, %v", cfg, got, err)
			}
			if origin, err := cfg.Origin(); origin != "https://recette.example.org" || err != nil {
				t.Errorf("Origin: %q, %v", origin, err)
			}
		})
	}
}

// TestExample checks that the model of deploy/ is read as a configuration (D10).
func TestExample(t *testing.T) {
	example, err := os.ReadFile(filepath.Join("..", "..", "..", "deploy", "recette-auto.example.json"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ConfigFile), example, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, PasswordFile), []byte(password), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Read(dir); err != nil {
		t.Errorf("Read of the example: %v", err)
	}
}
