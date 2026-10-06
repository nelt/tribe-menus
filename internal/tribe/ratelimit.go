package tribe

import (
	"crypto/sha256"
	"net/netip"
	"time"
)

// Rate limits of the code requests (ENF-01). The limits by address and by IP apply the
// same way to an unknown, revoked or other tribe's address, and to an unknown tribe
// (ENF-02). The limit by tribe counts only the requests for its active members (plan
// revue-securite, D2): it is kept in the tribe database, so that the rate limit database
// never learns which address is a member's.
const (
	EmailRequestLimit  = 3
	EmailRequestWindow = 15 * time.Minute
	IPRequestLimit     = 10
	IPRequestWindow    = time.Hour
	TribeRequestLimit  = 30
	TribeRequestWindow = time.Hour

	// RateLimitRetention is the longest window of the rate limit database: older requests
	// are purged.
	RateLimitRetention = time.Hour
)

// CodeRequest is a request for a login code, as the rate limit sees it: no address nor
// IP in clear (ADR 0021).
type CodeRequest struct {
	// EmailHash is the hash of the address with the slug: the same address has unrelated
	// hashes in two tribes (ENF-02).
	EmailHash []byte
	// IPHash is the hash of the IP alone, normalized by ipKey: the limit by IP holds for the
	// whole instance.
	IPHash []byte
}

// NewCodeRequest returns the request of email for the tribe slug from ip.
func NewCodeRequest(slug string, email Email, ip string) CodeRequest {
	return CodeRequest{EmailHash: emailHash(slug, email), IPHash: hash(ipKey(ip))}
}

// ipv6Prefix is the prefix an IPv6 counts for: the least a provider gives a subscriber
// (plan production, D4). Without it, the limit by IP would be bypassed by changing the
// address within one's own prefix.
const ipv6Prefix = 64

// ipKey is the form of the IP counted by the limit by IP: an IPv4 as is, an IPv4 mapped
// in an IPv6 as the IPv4, an IPv6 as its /64 prefix, without zone. Anything else, such as
// the unknown IP of a request behind the proxy, is kept as is.
func ipKey(ip string) string {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return ip
	}
	addr = addr.Unmap().WithZone("")
	if addr.Is4() {
		return addr.String()
	}
	prefix, err := addr.Prefix(ipv6Prefix)
	if err != nil {
		return addr.String()
	}
	return prefix.String()
}

// emailHash hashes an address with the slug of the tribe. Neither contains a line feed;
// a slug of any length or content is hashed, never stored.
func emailHash(slug string, email Email) []byte {
	return hash(slug + "\n" + string(email))
}

// hash is the stored form of a login code, a session token, an address or an IP: SHA-256,
// never the value in clear.
func hash(s string) []byte {
	h := sha256.Sum256([]byte(s))
	return h[:]
}

// CodeRequestCounts are the past requests sharing the address or the IP of a new request,
// each counted over its window.
type CodeRequestCounts struct {
	Email, IP int
}

// Allowed reports whether a new request is allowed after these requests.
func (c CodeRequestCounts) Allowed() bool {
	return c.Email < EmailRequestLimit && c.IP < IPRequestLimit
}

// RequestWindows returns the start of each window for a request at now: requests made
// strictly after it count.
func RequestWindows(now time.Time) (email, ip time.Time) {
	return now.Add(-EmailRequestWindow), now.Add(-IPRequestWindow)
}

// TribeRequestAllowed reports whether a new request for an active member is allowed after
// n requests for the members of the tribe in the window.
func TribeRequestAllowed(n int) bool {
	return n < TribeRequestLimit
}

// TribeRequestWindowStart returns the start of the window of the limit by tribe at now.
func TribeRequestWindowStart(now time.Time) time.Time {
	return now.Add(-TribeRequestWindow)
}
