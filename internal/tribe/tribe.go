// Package tribe holds the tribe, its members and its audit log (EF-01 to EF-11).
package tribe

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
)

var (
	// ErrInvalidSlug is returned for a slug that does not follow the format of EF-08.
	ErrInvalidSlug = errors.New("invalid slug")
	// ErrInvalidEmail is returned for an address that is not of the form local@domain, with a
	// dot inside the domain.
	ErrInvalidEmail = errors.New("invalid email address")
)

const (
	minSlugLength = 3
	maxSlugLength = 40
)

// slugPattern: lower-case letters without accents, digits and hyphens, a letter first,
// no trailing hyphen and no consecutive hyphens.
var slugPattern = regexp.MustCompile(`^[a-z](?:[a-z0-9]|-[a-z0-9])*$`)

// Slug is the identifier of a tribe in its URL, /tribes/<slug>/.
type Slug string

// ParseSlug returns the slug, or ErrInvalidSlug. An input that does not follow the
// format is refused, not converted (EF-08).
func ParseSlug(s string) (Slug, error) {
	if len(s) < minSlugLength || len(s) > maxSlugLength || !slugPattern.MatchString(s) {
		return "", ErrInvalidSlug
	}
	return Slug(s), nil
}

// Email is a normalized address: without spaces, in lower case.
type Email string

// ParseEmail normalizes the address and checks that it has the form local@domain, with a
// dot in the domain, neither first nor last (D16): alice@exemple is refused.
func ParseEmail(s string) (Email, error) {
	normalized := strings.ToLower(strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s))
	local, domain, found := strings.Cut(normalized, "@")
	if !found || local == "" || strings.Contains(domain, "@") ||
		!strings.Contains(domain, ".") || strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return "", ErrInvalidEmail
	}
	return Email(normalized), nil
}

// Status is the status of a member.
type Status string

// Statuses of a member.
const (
	Active  Status = "active"
	Revoked Status = "revoked"
)

// AuditOperation is the kind of an audit log entry (EF-07), named after the glossary.
type AuditOperation string

// Audit operations.
const (
	TribeInitialized      AuditOperation = "tribe_initialized"
	MemberAdded           AuditOperation = "member_added"
	MemberRevoked         AuditOperation = "member_revoked"
	MemberReactivated     AuditOperation = "member_reactivated"
	MemberAnonymized      AuditOperation = "member_anonymized"
	SessionOpened         AuditOperation = "session_opened"
	SessionRevoked        AuditOperation = "session_revoked"
	OtherDevicesSignedOut AuditOperation = "other_devices_signed_out"
	SignedOut             AuditOperation = "signed_out"
	SessionClosed         AuditOperation = "session_closed"
)
