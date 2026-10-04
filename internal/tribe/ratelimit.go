package tribe

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// Rate limits of the code requests (ENF-01). They apply the same way to an unknown,
// revoked or other tribe's address, and to an unknown tribe (ENF-02).
const (
	EmailRequestLimit  = 3
	EmailRequestWindow = 15 * time.Minute
	IPRequestLimit     = 10
	IPRequestWindow    = time.Hour
	TribeRequestLimit  = 30
	TribeRequestWindow = time.Hour

	// RateLimitRetention is the longest window: older requests are purged.
	RateLimitRetention = time.Hour
)

// CodeRequest is a request for a login code, as the rate limit sees it: no address nor
// IP in clear (ADR 0021).
type CodeRequest struct {
	// Slug is the slug of the URL, whether a tribe has it or not. A slug that does not
	// follow the format of EF-08, which no tribe can have, is replaced by its hash: nothing
	// of arbitrary length nor content chosen by the client is stored.
	Slug string
	// EmailHash is the hash of the address with the slug: the same address has unrelated
	// hashes in two tribes (ENF-02).
	EmailHash []byte
	// IPHash is the hash of the IP alone: the limit by IP holds for the whole instance.
	IPHash []byte
}

// NewCodeRequest returns the request of email for the tribe slug from ip.
func NewCodeRequest(slug string, email Email, ip string) CodeRequest {
	return CodeRequest{Slug: requestSlug(slug), EmailHash: emailHash(slug, email), IPHash: hash(ip)}
}

// malformedSlugPrefix starts the stored form of a malformed slug: no slug has this character.
const malformedSlugPrefix = "#"

// requestSlug is the slug as the rate limit database keeps it: in clear when it follows
// the format of EF-08, as its SHA-256 hash in hexadecimal otherwise.
func requestSlug(slug string) string {
	if _, err := ParseSlug(slug); err == nil {
		return slug
	}
	return malformedSlugPrefix + hex.EncodeToString(hash(slug))
}

// emailHash hashes an address with the slug of the tribe. Neither contains a line feed.
func emailHash(slug string, email Email) []byte {
	return hash(slug + "\n" + string(email))
}

// hash is the stored form of a login code, a session token, an address or an IP: SHA-256,
// never the value in clear.
func hash(s string) []byte {
	h := sha256.Sum256([]byte(s))
	return h[:]
}

// CodeRequestCounts are the past requests sharing the address, the IP or the tribe of a
// new request, each counted over its window.
type CodeRequestCounts struct {
	Email, IP, Tribe int
}

// Allowed reports whether a new request is allowed after these requests.
func (c CodeRequestCounts) Allowed() bool {
	return c.Email < EmailRequestLimit && c.IP < IPRequestLimit && c.Tribe < TribeRequestLimit
}

// RequestWindows returns the start of each window for a request at now: requests made
// strictly after it count.
func RequestWindows(now time.Time) (email, ip, tribe time.Time) {
	return now.Add(-EmailRequestWindow), now.Add(-IPRequestWindow), now.Add(-TribeRequestWindow)
}
