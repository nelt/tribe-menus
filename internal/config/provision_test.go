package config

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// provision runs deploy/provision.sh --render with the instance file of this content, for
// the environments, and returns the directory of the files made, its standard error and
// the error of the run.
func provision(t *testing.T, instance string, envs ...string) (string, string, error) {
	t.Helper()
	dir := t.TempDir()
	file := filepath.Join(dir, "instance.conf")
	if err := os.WriteFile(file, []byte(instance), 0o600); err != nil {
		t.Fatal(err)
	}
	script, err := filepath.Abs(filepath.Join("..", "..", "deploy", "provision.sh"))
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	args := append([]string{script, "--render", out, "--instance", file}, envs...)
	cmd := exec.Command("bash", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err = cmd.Run()
	return out, stderr.String(), err
}

func exampleInstance(t *testing.T) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", "deploy", "instance.example.conf"))
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

// The config.json that provision.sh makes from the instance file is read by this package
// (plan recette, D4).
func TestProvisionConfig(t *testing.T) {
	out, stderr, err := provision(t, exampleInstance(t), "production", "recette")
	if err != nil {
		t.Fatalf("provision.sh: %v\n%s", err, stderr)
	}
	for env, baseURL := range map[string]string{
		"production": "https://meltingtribe.example.org",
		"recette":    "https://recette.meltingtribe.example.org",
	} {
		c, err := Load(filepath.Join(out, env, "config.json"))
		if err != nil {
			t.Fatalf("%s: %v", env, err)
		}
		want := Config{
			Data:    "/var/lib/tribe-menus/" + env,
			Listen:  ListenSystemd,
			BaseURL: baseURL,
			SMTP:    exampleSMTP,
			Alerts:  exampleAlerts,
		}
		if c != want {
			t.Errorf("%s: config = %+v, want %+v", env, c, want)
		}
	}
	for _, name := range []string{"Caddyfile", "msmtprc", "aliases"} {
		content, err := os.ReadFile(filepath.Join(out, name))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(content), "codingmatters") {
			t.Errorf("%s names the real instance", name)
		}
	}

	// Nothing specific to the instance in the repository (ADR 0011, point 5).
	if strings.Contains(exampleInstance(t), "codingmatters") {
		t.Error("instance.example.conf names the real instance")
	}
}

// The instance file is data: a value is never evaluated by the shell, and an error names
// the line and the key, never the value.
func TestProvisionRefusesInstance(t *testing.T) {
	example := exampleInstance(t)
	tests := []struct {
		name    string
		replace [2]string
		envs    []string
		want    string
	}{
		{"unknown key", [2]string{"SMTP_PORT=465", "SMTP_PORT=465\nSMTP_TLS=on"}, nil, "unknown key SMTP_TLS"},
		{"missing key", [2]string{"SMTP_PORT=465", ""}, nil, "missing key SMTP_PORT"},
		{"key twice", [2]string{"SMTP_PORT=465", "SMTP_PORT=465\nSMTP_PORT=587"}, nil, "key SMTP_PORT written twice"},
		{"not key=value", [2]string{"SMTP_PORT=465", "SMTP_PORT 465"}, nil, "want KEY=value"},
		{"lower case key", [2]string{"SMTP_PORT=465", "smtp_port=465"}, nil, "want KEY=value"},
		{"command substitution", [2]string{"ADMIN_USER=alice", "ADMIN_USER=$(touch evaluated)"}, nil, "invalid value for ADMIN_USER"},
		{"backquotes", [2]string{"SMTP_HOST=smtp.example.org", "SMTP_HOST=`touch evaluated`"}, nil, "invalid value for SMTP_HOST"},
		{"quotes", [2]string{"ADMIN_EMAIL=admin@example.org", `ADMIN_EMAIL="admin@example.org"`}, nil, "invalid value for ADMIN_EMAIL"},
		{"JSON in a value", [2]string{"APP_SMTP_FROM=no-reply@example.org", `APP_SMTP_FROM=a@example.org","x":"`}, nil, "invalid value for APP_SMTP_FROM"},
		{"space", [2]string{"SMTP_HOST=smtp.example.org", "SMTP_HOST=smtp.example.org "}, nil, "invalid value for SMTP_HOST"},
		{"carriage return", [2]string{"SMTP_HOST=smtp.example.org", "SMTP_HOST=smtp.example.org\r"}, nil, "invalid value for SMTP_HOST"},
		{"Caddyfile in a host", [2]string{"RECETTE_HOST=recette.meltingtribe.example.org", "RECETTE_HOST=a.example.org {"}, nil, "invalid value for RECETTE_HOST"},
		{"port out of range", [2]string{"SMTP_PORT=465", "SMTP_PORT=65536"}, nil, "invalid value for SMTP_PORT"},
		{"root as administrator", [2]string{"ADMIN_USER=alice", "ADMIN_USER=root"}, nil, "ADMIN_USER must be a personal account"},
		{"service user as administrator", [2]string{"ADMIN_USER=alice", "ADMIN_USER=tribe-menus-recette"}, nil, "ADMIN_USER must be a personal account"},
		{"same host names", [2]string{"RECETTE_HOST=recette.meltingtribe.example.org", "RECETTE_HOST=meltingtribe.example.org"}, nil, "must differ"},
		{"unknown environment", [2]string{}, []string{"staging"}, "Usage"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := example
			if tt.replace[0] != "" {
				if !strings.Contains(example, tt.replace[0]) {
					t.Fatalf("%q not in the example", tt.replace[0])
				}
				instance = strings.Replace(example, tt.replace[0], tt.replace[1], 1)
			}
			envs := tt.envs
			if envs == nil {
				envs = []string{"production", "recette"}
			}
			out, stderr, err := provision(t, instance, envs...)
			if err == nil {
				t.Fatal("provision.sh accepted the instance file")
			}
			if !strings.Contains(stderr, tt.want) {
				t.Errorf("stderr = %q, want %q", stderr, tt.want)
			}
			if _, err := os.Stat(filepath.Join(filepath.Dir(out), "evaluated")); err == nil {
				t.Error("a value was evaluated by the shell")
			}
			if _, err := os.Stat(out); err == nil {
				t.Error("files made from a refused instance file")
			}
			for _, secret := range []string{"touch", "65536", "587"} {
				if strings.Contains(stderr, secret) {
					t.Errorf("stderr quotes the value: %q", stderr)
				}
			}
		})
	}
}

// On a real server, provision.sh refuses the values of the example left as they are: a host
// name of example.org gave Caddy no site for the real names (séance S2). --render keeps
// them, for the tests.
func TestProvisionRefusesExampleValues(t *testing.T) {
	script, err := filepath.Abs(filepath.Join("..", "..", "deploy", "provision.sh"))
	if err != nil {
		t.Fatal(err)
	}
	real := strings.NewReplacer("example.org", "codingmatters.test").Replace(exampleInstance(t))
	tests := []struct {
		name     string
		instance string
		want     string
	}{
		{"example as is", exampleInstance(t), "still has a value of the example"},
		{"one host left", strings.Replace(real, "RECETTE_HOST=recette.meltingtribe.codingmatters.test", "RECETTE_HOST=recette.meltingtribe.example.org", 1), "RECETTE_HOST still has a value of the example"},
		{"one address left", strings.Replace(real, "ADMIN_EMAIL=admin@codingmatters.test", "ADMIN_EMAIL=admin@example.org", 1), "ADMIN_EMAIL still has a value of the example"},
		{"real values", real, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "instance.conf")
			if err := os.WriteFile(file, []byte(tc.instance), 0o600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("bash", "-c", `source "$1"; read_instance "$2"; refuse_examples "$2"`, "_", script, file)
			out, err := cmd.CombinedOutput()
			if tc.want == "" {
				if err != nil {
					t.Errorf("refused: %v\n%s", err, out)
				}
				return
			}
			if err == nil || !strings.Contains(string(out), tc.want) {
				t.Errorf("err = %v, output %q, want %q", err, out, tc.want)
			}
			if strings.Contains(string(out), "codingmatters") {
				t.Errorf("error names a value: %s", out)
			}
		})
	}
}
