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

// AlertSignal is a sign of the limits reached repeatedly: each has its threshold over the
// window, and is alerted on at most once per AlertInterval (plan production, D8 and D12).
type AlertSignal string

// The signals. Their names are those of the rate limit database and of the logs.
const (
	// RefusedRequests: requests refused by any limit, the trace of a flood.
	RefusedRequests AlertSignal = "refused_requests"
	// ExhaustedCodes: codes, real or decoy, invalidated by their last attempt, the trace of
	// a brute force at the pace of the limits, which is never refused.
	ExhaustedCodes AlertSignal = "exhausted_codes"
	// RepeatedRequests: an address hash with RepeatedRequestsThreshold accepted requests or
	// more, the trace of a targeted lockout at the exact pace of the limit by address.
	RepeatedRequests AlertSignal = "repeated_requests"
)

// AlertSignals are the signals, in the order of the alert email.
var AlertSignals = []AlertSignal{RefusedRequests, ExhaustedCodes, RepeatedRequests}

// Thresholds of the signals over the window, for the whole instance; constants of the code,
// not of the configuration (socle, D15).
const (
	RefusedRequestsThreshold = 10
	ExhaustedCodesThreshold  = 5
	// RepeatedRequestsThreshold is the number of requests of one address hash: twelve an
	// hour is the most the limit by address allows.
	RepeatedRequestsThreshold = 9
	// AlertInterval is the least time between two alerts on the same signal.
	AlertInterval = 6 * time.Hour
)

// AlertCounts are the counters of the window, for the whole instance.
type AlertCounts struct {
	// Events are the counted events, summed over the rate limit database and every tribe.
	Events map[AlertEvent]int
	// RepeatedAddresses is the number of address hashes with RepeatedRequestsThreshold
	// requests or more.
	RepeatedAddresses int
}

// add adds the events of a database to the counts.
func (c *AlertCounts) add(events map[AlertEvent]int) {
	if c.Events == nil {
		c.Events = map[AlertEvent]int{}
	}
	for event, n := range events {
		c.Events[event] += n
	}
}

// Refused returns the requests refused by any limit.
func (c AlertCounts) Refused() int {
	return c.Events[EmailLimitRefusal] + c.Events[IPLimitRefusal] + c.Events[TribeLimitRefusal]
}

// Exhausted returns the codes, real or decoy, invalidated by their last attempt.
func (c AlertCounts) Exhausted() int {
	return c.Events[DecoyCodeExhausted] + c.Events[LoginCodeExhausted]
}

// crossed reports whether the counts reach the threshold of the signal.
func (c AlertCounts) crossed(s AlertSignal) bool {
	switch s {
	case RefusedRequests:
		return c.Refused() >= RefusedRequestsThreshold
	case ExhaustedCodes:
		return c.Exhausted() >= ExhaustedCodesThreshold
	case RepeatedRequests:
		return c.RepeatedAddresses > 0
	default:
		return false
	}
}

// DueAlerts returns the signals whose threshold the counts reach, and whose last alert, in
// lastSent, is AlertInterval old or more, in the order of AlertSignals.
func DueAlerts(counts AlertCounts, lastSent map[AlertSignal]time.Time, now time.Time) []AlertSignal {
	var due []AlertSignal
	for _, s := range AlertSignals {
		last, sent := lastSent[s]
		if counts.crossed(s) && (!sent || !now.Before(last.Add(AlertInterval))) {
			due = append(due, s)
		}
	}
	return due
}

// AlertReport is an alert on the code request limits reached repeatedly: neither address,
// nor IP, nor tribe (ADR 0023, point 6).
type AlertReport struct {
	At time.Time
	// Signals are the signals alerted on, in the order of AlertSignals.
	Signals []AlertSignal
	Counts  AlertCounts
}

// AlertNotifier writes an alert to the log and sends it to the administrator.
type AlertNotifier interface {
	NotifyCodeRequestLimits(r AlertReport)
}
