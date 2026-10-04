package tribe

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestCodeRequestCountsAllowed(t *testing.T) {
	cases := []struct {
		counts CodeRequestCounts
		want   bool
	}{
		{counts: CodeRequestCounts{}, want: true},
		{counts: CodeRequestCounts{Email: 2, IP: 9, Tribe: 29}, want: true},
		{counts: CodeRequestCounts{Email: 3}, want: false},
		{counts: CodeRequestCounts{IP: 10}, want: false},
		{counts: CodeRequestCounts{Tribe: 30}, want: false},
	}
	for _, tc := range cases {
		if got := tc.counts.Allowed(); got != tc.want {
			t.Errorf("%+v.Allowed() = %v, want %v", tc.counts, got, tc.want)
		}
	}
}

func TestRequestWindows(t *testing.T) {
	now := time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC)
	email, ip, tribe := RequestWindows(now)
	if !email.Equal(now.Add(-15*time.Minute)) || !ip.Equal(now.Add(-time.Hour)) || !tribe.Equal(now.Add(-time.Hour)) {
		t.Errorf("RequestWindows = %v, %v, %v", email, ip, tribe)
	}
}

func TestNewCodeRequest(t *testing.T) {
	martin := NewCodeRequest("martin", "alice@exemple.fr", "192.0.2.1")
	durand := NewCodeRequest("durand", "alice@exemple.fr", "192.0.2.1")
	if bytes.Equal(martin.EmailHash, durand.EmailHash) {
		t.Error("the same address has the same hash in two tribes")
	}
	if !bytes.Equal(martin.IPHash, durand.IPHash) {
		t.Error("the same IP has different hashes in two tribes")
	}
	for _, h := range [][]byte{martin.EmailHash, martin.IPHash} {
		if len(h) != 32 || strings.Contains(string(h), "alice") || strings.Contains(string(h), "192.0.2.1") {
			t.Errorf("hash %x is not a SHA-256", h)
		}
	}
}

func TestRequestSlug(t *testing.T) {
	long := strings.Repeat("x", 10_000)
	cases := []struct {
		name      string
		slug      string
		wantClear bool
	}{
		{name: "slug of a tribe", slug: "martin", wantClear: true},
		{name: "well-formed slug without tribe", slug: "les-dupont-2", wantClear: true},
		{name: "upper case", slug: "Martin"},
		{name: "too short", slug: "ab"},
		{name: "too long", slug: long},
		{name: "free text", slug: "alice@exemple.fr, 12 rue des Lilas"},
		{name: "line feed", slug: "martin\nalice"},
		{name: "empty", slug: ""},
		{name: "already in the stored form", slug: "#" + strings.Repeat("0", 64)},
	}
	seen := map[string]string{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NewCodeRequest(tc.slug, "alice@exemple.fr", "192.0.2.1").Slug
			if tc.wantClear {
				if got != tc.slug {
					t.Errorf("stored slug = %q, want %q", got, tc.slug)
				}
				return
			}
			if len(got) != 65 || !strings.HasPrefix(got, "#") || got == tc.slug {
				t.Errorf("stored slug = %q, want # and a SHA-256 in hexadecimal", got)
			}
			if _, err := ParseSlug(got); err == nil {
				t.Errorf("stored slug %q could be the slug of a tribe", got)
			}
			if other, ok := seen[got]; ok {
				t.Errorf("%q and %q are stored alike", tc.slug, other)
			}
			seen[got] = tc.slug
		})
	}
}
