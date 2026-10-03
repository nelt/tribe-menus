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
