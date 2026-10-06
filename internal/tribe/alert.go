package tribe

import "time"

// Alerts on the code request limits reached repeatedly (ENF-01, ADR 0023). The events are
// counted by ten-minute slot, without hash nor identifier, each in the database that sees
// it: the rate limit database for those that do not depend on membership, the tribe
// database for those that suppose a member (plan production, D7).
const (
	// AlertSlot is the length of a slot of the alert counters.
	AlertSlot = 10 * time.Minute
	// AlertWindow is the window over which the counters are summed: the slots that start
	// strictly after its start count. Older slots are purged.
	AlertWindow = time.Hour
)

// AlertEvent is the nature of an event counted for the alerts.
type AlertEvent string

// The events counted for the alerts. Their names are those of the databases.
const (
	// EmailLimitRefusal: a request refused by the limit by address (rate limit database).
	EmailLimitRefusal AlertEvent = "email_limit"
	// IPLimitRefusal: a request refused by the limit by IP (rate limit database).
	IPLimitRefusal AlertEvent = "ip_limit"
	// TribeLimitRefusal: a request refused by the limit by tribe (tribe database).
	TribeLimitRefusal AlertEvent = "tribe_limit"
	// DecoyCodeExhausted: a decoy code invalidated by its last attempt (rate limit database).
	DecoyCodeExhausted AlertEvent = "decoy_code_exhausted"
	// LoginCodeExhausted: a real code invalidated by its last attempt (tribe database).
	LoginCodeExhausted AlertEvent = "login_code_exhausted"
)

// AlertEvents are the events, in the order of the alert email.
var AlertEvents = []AlertEvent{EmailLimitRefusal, IPLimitRefusal, TribeLimitRefusal, DecoyCodeExhausted, LoginCodeExhausted}

// alertSlot returns the stored start of the slot of an event at now.
func alertSlot(now time.Time) string {
	return formatTime(now.Truncate(AlertSlot))
}

// alertWindowStart returns the start of the window of the alerts at now.
func alertWindowStart(now time.Time) time.Time {
	return now.Add(-AlertWindow)
}
