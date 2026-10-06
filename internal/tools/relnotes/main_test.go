package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const changelog = `# Journal des modifications

Format : une ligne par PR.

## Non publié

- Une ligne à venir.

## 0.2.0 — 2026-11-02

- ` + "`recette/B`" + ` : déploiement.
- Une autre ligne.

### Migrations

- Table ajoutée.

## 0.1.10 — 2026-10-30

- Une version à deux chiffres.

## 0.1.1 — 2026-10-25

- Un correctif.

## 0.1.0 — 2026-10-20

- Première version.
`

func TestSection(t *testing.T) {
	cases := []struct {
		name      string
		changelog string
		tag       string
		want      string
		wantErr   error
	}{
		{name: "section with a sub-heading", tag: "v0.2.0", want: "- `recette/B` : déploiement.\n- Une autre ligne.\n\n### Migrations\n\n- Table ajoutée.\n"},
		{name: "last section", tag: "v0.1.0", want: "- Première version.\n"},
		{name: "prefix of another version", tag: "v0.1.1", want: "- Un correctif.\n"},
		{name: "two-digit patch", tag: "v0.1.10", want: "- Une version à deux chiffres.\n"},
		{name: "heading without date", changelog: "## 1.0.0\n\n- V1.\n", tag: "v1.0.0", want: "- V1.\n"},
		{name: "CRLF line endings", changelog: "## 1.0.0 — 2027-01-04\r\n\r\n- V1.\r\n", tag: "v1.0.0", want: "- V1.\n"},
		{name: "missing section", tag: "v0.3.0", wantErr: errMissing},
		{name: "empty section", changelog: "## 0.1.0 — 2026-10-20\n\n## 0.0.1\n\n- x\n", tag: "v0.1.0", wantErr: errEmpty},
		{name: "section written twice", changelog: "## 0.1.0 — 2026-10-20\n\n- a\n\n## 0.1.0 — 2026-10-21\n\n- b\n", tag: "v0.1.0", wantErr: errTwice},
		{name: "version in the text, not in a heading", changelog: "- avant la 0.4.0\n", tag: "v0.4.0", wantErr: errMissing},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			text := tc.changelog
			if text == "" {
				text = changelog
			}
			got, err := section(text, tc.tag)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("notes = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSectionRefusesWhatIsNotATag(t *testing.T) {
	for _, tag := range []string{"Non publié", "0.2.0", "v0.2", "v0.2.0-rc1", "pr-12", "dev", ""} {
		if _, err := section(changelog, tag); err == nil || !strings.Contains(err.Error(), "not a release tag") {
			t.Errorf("section(%q) error = %v, want not a release tag", tag, err)
		}
	}
}

func TestRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), "CHANGELOG.md")
	if err := os.WriteFile(path, []byte(changelog), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
	}{
		{name: "notes", args: []string{"-changelog", path, "v0.1.0"}, wantStdout: "- Première version.\n"},
		{name: "missing section", args: []string{"-changelog", path, "v9.9.9"}, wantCode: 1},
		{name: "no tag", args: []string{"-changelog", path}, wantCode: 2},
		{name: "missing changelog", args: []string{"-changelog", path + ".absent", "v0.1.0"}, wantCode: 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := run(tc.args, &stdout, &stderr); code != tc.wantCode {
				t.Fatalf("code = %d, want %d (stderr %q)", code, tc.wantCode, stderr.String())
			}
			if stdout.String() != tc.wantStdout {
				t.Errorf("stdout = %q, want %q", stdout.String(), tc.wantStdout)
			}
		})
	}
}
