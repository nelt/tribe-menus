package tribe

import (
	"errors"
	"strings"
	"testing"
)

func TestParseSlug(t *testing.T) {
	cases := []struct {
		input string
		valid bool
	}{
		{input: "martin", valid: true},
		{input: "les-durand", valid: true},
		{input: "tribu42", valid: true},
		{input: "a-1-b", valid: true},
		{input: "abc", valid: true},
		{input: "a" + strings.Repeat("b", 39), valid: true},
		{input: "a" + strings.Repeat("b", 40)},
		{input: "ab"},
		{input: ""},
		{input: "Martin"},
		{input: "les_durand"},
		{input: "é-nous"},
		{input: "42"},
		{input: "1abc"},
		{input: "-martin"},
		{input: "martin-"},
		{input: "les--durand"},
		{input: "les durand"},
		{input: "martin/"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			slug, err := ParseSlug(tc.input)
			if tc.valid && (err != nil || string(slug) != tc.input) {
				t.Errorf("ParseSlug(%q) = %q, %v; want it valid and unchanged", tc.input, slug, err)
			}
			if !tc.valid && !errors.Is(err, ErrInvalidSlug) {
				t.Errorf("ParseSlug(%q) error = %v, want ErrInvalidSlug", tc.input, err)
			}
		})
	}
}

func TestParseEmail(t *testing.T) {
	cases := []struct {
		input string
		want  Email
	}{
		{input: "alice@exemple.fr", want: "alice@exemple.fr"},
		{input: "  Alice@Exemple.FR ", want: "alice@exemple.fr"},
		{input: "ali ce@exem\tple.fr", want: "alice@exemple.fr"},
		{input: "chloé@exemple.fr", want: "chloé@exemple.fr"},
		{input: ""},
		{input: "alice"},
		{input: "@exemple.fr"},
		{input: "alice@"},
		{input: "alice@exemple@fr"},
		{input: "alice@exemple"},
		{input: "alice@.exemple"},
		{input: "alice@exemple."},
		{input: "alice@."},
		{input: "alice@mail.exemple.fr", want: "alice@mail.exemple.fr"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got, err := ParseEmail(tc.input)
			if tc.want != "" && (err != nil || got != tc.want) {
				t.Errorf("ParseEmail(%q) = %q, %v; want %q", tc.input, got, err, tc.want)
			}
			if tc.want == "" && !errors.Is(err, ErrInvalidEmail) {
				t.Errorf("ParseEmail(%q) error = %v, want ErrInvalidEmail", tc.input, err)
			}
		})
	}
}
