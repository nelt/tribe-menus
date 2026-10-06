package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// smtpPasswordCredential is the systemd credential of the SMTP password (ADR 0015, point
// 10).
const smtpPasswordCredential = "smtp-password"

// readCredential returns the systemd credential of this name: the file of the directory
// CREDENTIALS_DIRECTORY, read by getenv, without its final line feed. A secret is never
// read from an option, an environment variable or the config file, and its value is
// never in an error.
func readCredential(getenv func(string) string, name string) (string, error) {
	dir := getenv("CREDENTIALS_DIRECTORY")
	if dir == "" {
		return "", fmt.Errorf("credential %s: CREDENTIALS_DIRECTORY not set, the secret is passed by systemd only", name)
	}
	content, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		var pathErr *os.PathError
		if errors.As(err, &pathErr) {
			err = pathErr.Err
		}
		return "", fmt.Errorf("credential %s: %w", name, err)
	}
	value := strings.TrimSuffix(strings.TrimSuffix(string(content), "\n"), "\r")
	if value == "" {
		return "", fmt.Errorf("credential %s: empty", name)
	}
	return value, nil
}
