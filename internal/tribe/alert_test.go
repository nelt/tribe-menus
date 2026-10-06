package tribe

import (
	"fmt"
	"testing"
	"time"
)

func TestDueAlerts(t *testing.T) {
	now := time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC)
	refused := map[AlertEvent]int{EmailLimitRefusal: 4, IPLimitRefusal: 5, TribeLimitRefusal: 1}
	exhausted := map[AlertEvent]int{DecoyCodeExhausted: 3, LoginCodeExhausted: 2}
	cases := []struct {
		name     string
		counts   AlertCounts
		lastSent map[AlertSignal]time.Time
		want     []AlertSignal
	}{
		{name: "nothing", counts: AlertCounts{}},
		{name: "refused below the threshold", counts: AlertCounts{Events: map[AlertEvent]int{EmailLimitRefusal: 4, IPLimitRefusal: 5}}},
		{name: "refused by every limit", counts: AlertCounts{Events: refused}, want: []AlertSignal{RefusedRequests}},
		{name: "exhausted below the threshold", counts: AlertCounts{Events: map[AlertEvent]int{DecoyCodeExhausted: 2, LoginCodeExhausted: 2}}},
		{name: "exhausted, real and decoy", counts: AlertCounts{Events: exhausted}, want: []AlertSignal{ExhaustedCodes}},
		{name: "repeated requests", counts: AlertCounts{RepeatedAddresses: 1}, want: []AlertSignal{RepeatedRequests}},
		{
			name:   "every signal",
			counts: AlertCounts{Events: map[AlertEvent]int{IPLimitRefusal: 10, LoginCodeExhausted: 5}, RepeatedAddresses: 2},
			want:   []AlertSignal{RefusedRequests, ExhaustedCodes, RepeatedRequests},
		},
		{
			name:     "alerted on less than six hours ago",
			counts:   AlertCounts{Events: refused},
			lastSent: map[AlertSignal]time.Time{RefusedRequests: now.Add(-AlertInterval + time.Second)},
		},
		{
			name:     "alerted on six hours ago",
			counts:   AlertCounts{Events: refused},
			lastSent: map[AlertSignal]time.Time{RefusedRequests: now.Add(-AlertInterval)},
			want:     []AlertSignal{RefusedRequests},
		},
		{
			name:     "another signal alerted on recently",
			counts:   AlertCounts{Events: exhausted},
			lastSent: map[AlertSignal]time.Time{RefusedRequests: now},
			want:     []AlertSignal{ExhaustedCodes},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DueAlerts(tc.counts, tc.lastSent, now); fmt.Sprint(got) != fmt.Sprint(tc.want) {
				t.Errorf("DueAlerts() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAlertSlot(t *testing.T) {
	for at, want := range map[string]string{
		"2026-10-05T09:00:00Z": "2026-10-05T09:00:00.000Z",
		"2026-10-05T09:09:59Z": "2026-10-05T09:00:00.000Z",
		"2026-10-05T09:10:00Z": "2026-10-05T09:10:00.000Z",
	} {
		now, err := time.Parse(time.RFC3339, at)
		if err != nil {
			t.Fatal(err)
		}
		if got := alertSlot(now); got != want {
			t.Errorf("alertSlot(%s) = %s, want %s", at, got, want)
		}
	}
}
