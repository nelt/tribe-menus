package mealplan

import (
	"errors"
	"testing"
)

func TestParseDate(t *testing.T) {
	cases := []struct {
		text string
		err  error
	}{
		{"2026-10-06", nil},
		{"2000-01-01", nil},
		{"2099-12-31", nil},
		{"2028-02-29", nil},
		{"1999-12-31", ErrInvalidDate},
		{"2100-01-01", ErrInvalidDate},
		{"2026-02-29", ErrInvalidDate},
		{"2026-02-30", ErrInvalidDate},
		{"2026-04-31", ErrInvalidDate},
		{"2026-13-01", ErrInvalidDate},
		{"2026-10-6", ErrInvalidDate},
		{"2026-1-06", ErrInvalidDate},
		{"26-10-06", ErrInvalidDate},
		{"2026/10/06", ErrInvalidDate},
		{"2026-10-06T00:00:00Z", ErrInvalidDate},
		{" 2026-10-06", ErrInvalidDate},
		{"", ErrInvalidDate},
	}
	for _, c := range cases {
		t.Run(c.text, func(t *testing.T) {
			d, err := ParseDate(c.text)
			if !errors.Is(err, c.err) {
				t.Fatalf("error %v, want %v", err, c.err)
			}
			if err == nil && string(d) != c.text {
				t.Errorf("date %q", d)
			}
		})
	}
}

func TestParseMealtime(t *testing.T) {
	cases := []struct {
		text string
		want Mealtime
		err  error
	}{
		{"lunch", Lunch, nil},
		{"dinner", Dinner, nil},
		{"Lunch", "", ErrInvalidMealtime},
		{"midi", "", ErrInvalidMealtime},
		{"breakfast", "", ErrInvalidMealtime},
		{"", "", ErrInvalidMealtime},
	}
	for _, c := range cases {
		m, err := ParseMealtime(c.text)
		if m != c.want || !errors.Is(err, c.err) {
			t.Errorf("%q: %q, %v; want %q, %v", c.text, m, err, c.want, c.err)
		}
	}
}

func TestCheckServings(t *testing.T) {
	cases := []struct {
		servings int
		err      error
	}{
		{1, nil}, {4, nil}, {50, nil},
		{0, ErrInvalidServings}, {-1, ErrInvalidServings}, {51, ErrInvalidServings},
	}
	for _, c := range cases {
		if err := CheckServings(c.servings); !errors.Is(err, c.err) {
			t.Errorf("%d: %v, want %v", c.servings, err, c.err)
		}
	}
}
