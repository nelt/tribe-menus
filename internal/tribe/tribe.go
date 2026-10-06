// Package tribe holds the tribe, its members and its audit log (EF-01 to EF-11).
package tribe

import (
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"unicode"
)

var (
	// ErrInvalidSlug is returned for a slug that does not follow the format of EF-08.
	ErrInvalidSlug = errors.New("invalid slug")
	// ErrInvalidEmail is returned for an address that does not follow the rule of ParseEmail.
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

// Email is a normalized address: without spaces, in lower case, following the rule of
// ParseEmail. It is written as is in the headers and the envelope of an email.
type Email string

const (
	maxEmailLength = 254
	maxLocalLength = 64
)

var (
	// localPattern: dot-separated atoms of letters without accents, digits and
	// !#$%&'*+/=?^_{|}~- (plan production, D5): no quotes, no comments, no dot first, last
	// or doubled.
	localPattern = regexp.MustCompile(`^[a-z0-9!#$%&'*+/=?^_{|}~-]+(?:\.[a-z0-9!#$%&'*+/=?^_{|}~-]+)*$`)
	// domainPattern: at least two labels of letters, digits and hyphens, without hyphen first
	// or last, 63 characters at most each (D16).
	domainPattern = regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)
)

// ParseEmail normalizes the address (without spaces, in lower case) and checks the rule of
// the specs (plan production, D5): ASCII only, a local part of atoms, a domain of labels
// with a dot (D16), so that alice@exemple is refused. An input that does not follow it is
// refused, not corrected. What the rule accepts is read back unchanged by net/mail, without
// display name: it can be written as is in a header and in the envelope (plan
// revue-securite, finding 7).
func ParseEmail(s string) (Email, error) {
	normalized := strings.ToLower(strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s))
	local, domain, found := strings.Cut(normalized, "@")
	if !found || len(normalized) > maxEmailLength || len(local) > maxLocalLength ||
		!localPattern.MatchString(local) || !domainPattern.MatchString(domain) {
		return "", ErrInvalidEmail
	}
	parsed, err := mail.ParseAddress(normalized)
	if err != nil || parsed.Name != "" || parsed.Address != normalized {
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
