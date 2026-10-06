package tribe

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestCodeRequestCountsRefusal(t *testing.T) {
	cases := []struct {
		counts CodeRequestCounts
		want   AlertEvent
	}{
		{counts: CodeRequestCounts{}, want: ""},
		{counts: CodeRequestCounts{Email: 2, IP: 9}, want: ""},
		{counts: CodeRequestCounts{Email: 3}, want: EmailLimitRefusal},
		{counts: CodeRequestCounts{IP: 10}, want: IPLimitRefusal},
		// A request refused by both limits counts once.
		{counts: CodeRequestCounts{Email: 3, IP: 10}, want: EmailLimitRefusal},
	}
	for _, tc := range cases {
		if got := tc.counts.Refusal(); got != tc.want {
			t.Errorf("%+v.Refusal() = %q, want %q", tc.counts, got, tc.want)
		}
	}
}

func TestTribeRequestAllowed(t *testing.T) {
	for n, want := range map[int]bool{0: true, 29: true, 30: false, 31: false} {
		if got := TribeRequestAllowed(n); got != want {
			t.Errorf("TribeRequestAllowed(%d) = %v, want %v", n, got, want)
		}
	}
}

func TestRequestWindows(t *testing.T) {
	now := time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC)
	email, ip := RequestWindows(now)
	tribe := TribeRequestWindowStart(now)
	if !email.Equal(now.Add(-15*time.Minute)) || !ip.Equal(now.Add(-time.Hour)) || !tribe.Equal(now.Add(-time.Hour)) {
		t.Errorf("windows = %v, %v, %v", email, ip, tribe)
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

func TestIPKey(t *testing.T) {
	cases := []struct {
		ip, want string
	}{
		{ip: "192.0.2.1", want: "192.0.2.1"},
		{ip: "::ffff:192.0.2.1", want: "192.0.2.1"},
		{ip: "2001:db8:1:2::a", want: "2001:db8:1:2::/64"},
		{ip: "2001:db8:1:2:ffff:ffff:ffff:ffff", want: "2001:db8:1:2::/64"},
		{ip: "2001:DB8:1:2::B", want: "2001:db8:1:2::/64"},
		{ip: "2001:db8:1:3::a", want: "2001:db8:1:3::/64"},
		{ip: "fe80::1%eth0", want: "fe80::/64"},
		{ip: "::1", want: "::/64"},
		{ip: "unknown", want: "unknown"},
		{ip: "", want: ""},
	}
	for _, tc := range cases {
		if got := ipKey(tc.ip); got != tc.want {
			t.Errorf("ipKey(%q) = %q, want %q", tc.ip, got, tc.want)
		}
	}
}

// Two IPv6 of the same /64 share the limit by IP, two neighboring /64 do not (plan
// production, D4).
func TestNewCodeRequestIPv6Prefix(t *testing.T) {
	a := NewCodeRequest("martin", "alice@exemple.fr", "2001:db8:1:2::a")
	b := NewCodeRequest("martin", "alice@exemple.fr", "2001:db8:1:2::b")
	c := NewCodeRequest("martin", "alice@exemple.fr", "2001:db8:1:3::a")
	if !bytes.Equal(a.IPHash, b.IPHash) {
		t.Error("two IPv6 of the same /64 have different hashes")
	}
	if bytes.Equal(a.IPHash, c.IPHash) {
		t.Error("two neighboring /64 have the same hash")
	}
}
