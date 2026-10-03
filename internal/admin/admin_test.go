package admin

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nelt/tribe-menus/internal/storage"
	"github.com/nelt/tribe-menus/internal/tribe"
)

var now = time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)

func openStore(t *testing.T, dir string) *storage.Store {
	t.Helper()
	migrations, err := storage.EmbeddedMigrations()
	if err != nil {
		t.Fatal(err)
	}
	s, err := storage.Open(context.Background(), dir, migrations)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func runInit(t *testing.T, store *storage.Store, input string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd := &Command{Store: store, In: strings.NewReader(input), Out: &out, BaseURL: "https://meltingtribe.example/", Now: func() time.Time { return now }}
	err := cmd.Init(context.Background())
	return out.String(), err
}

func TestInit(t *testing.T) {
	cases := []struct {
		name            string
		input           string
		wantErr         error
		wantRefusals    []string
		wantSlug        string
		wantEmail       tribe.Email
		wantDisplayName string
	}{
		{
			name:      "all answers valid",
			input:     "Les Martin\nmartin\nalice@exemple.fr\n\n",
			wantSlug:  "martin",
			wantEmail: "alice@exemple.fr",
		},
		{
			name:            "display name and normalized address",
			input:           "  Les Martin \nmartin\n Alice@Exemple.FR\nAlice\n",
			wantSlug:        "martin",
			wantEmail:       "alice@exemple.fr",
			wantDisplayName: "Alice",
		},
		{
			name:         "empty name asked again",
			input:        "\nLes Martin\nmartin\nalice@exemple.fr\n\n",
			wantRefusals: []string{refusedEmptyName},
			wantSlug:     "martin",
			wantEmail:    "alice@exemple.fr",
		},
		{
			name:         "invalid slugs asked again, not converted",
			input:        "Les Martin\nMartin\nles_martin\nmartin\nalice@exemple.fr\n\n",
			wantRefusals: []string{refusedSlugForm, refusedSlugForm},
			wantSlug:     "martin",
			wantEmail:    "alice@exemple.fr",
		},
		{
			name:         "taken slug asked again",
			input:        "Les Martin\ndurand\nmartin\nalice@exemple.fr\n\n",
			wantRefusals: []string{refusedSlugTaken},
			wantSlug:     "martin",
			wantEmail:    "alice@exemple.fr",
		},
		{
			name:         "invalid address asked again",
			input:        "Les Martin\nmartin\nalice\nalice@exemple.fr\n\n",
			wantRefusals: []string{refusedEmail},
			wantSlug:     "martin",
			wantEmail:    "alice@exemple.fr",
		},
		{name: "input closed", input: "Les Martin\nmartin\n", wantErr: ErrInputClosed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			store := openStore(t, dir)
			if _, err := runInit(t, store, "Les Durand\ndurand\nbruno@exemple.fr\n\n"); err != nil {
				t.Fatalf("init durand: %v", err)
			}

			out, err := runInit(t, store, tc.input)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Init error = %v, want %v (output: %s)", err, tc.wantErr, out)
			}
			for _, refusal := range tc.wantRefusals {
				if !strings.Contains(out, refusal) {
					t.Errorf("output = %q, want refusal %q", out, refusal)
				}
			}
			if got := strings.Count(out, refusedSlugForm) + strings.Count(out, refusedSlugTaken) +
				strings.Count(out, refusedEmail) + strings.Count(out, refusedEmptyName); got != len(tc.wantRefusals) {
				t.Errorf("%d refusals in output, want %d: %q", got, len(tc.wantRefusals), out)
			}

			if tc.wantErr != nil {
				if files, _ := filepath.Glob(filepath.Join(dir, "tribes", "*.db")); len(files) != 1 {
					t.Errorf("tribe databases = %v, want only durand's", files)
				}
				return
			}
			wantURL := "https://meltingtribe.example/tribes/" + tc.wantSlug + "/"
			if !strings.Contains(out, wantURL) {
				t.Errorf("output = %q, want the URL %s", out, wantURL)
			}
			db, err := store.Tribe(context.Background(), tc.wantSlug)
			if err != nil {
				t.Fatal(err)
			}
			member, err := tribe.NewStore(db).Member(context.Background(), tc.wantEmail)
			if err != nil {
				t.Fatal(err)
			}
			if member.Status != tribe.Active || member.DisplayName != tc.wantDisplayName {
				t.Errorf("member = %+v", member)
			}
		})
	}
}

func TestTribeURL(t *testing.T) {
	for _, base := range []string{"http://localhost:8080", "http://localhost:8080/"} {
		c := &Command{BaseURL: base}
		if got := c.TribeURL("martin"); got != "http://localhost:8080/tribes/martin/" {
			t.Errorf("TribeURL with %q = %q", base, got)
		}
	}
}
