package tribe

import (
	"testing"
	"time"
)

func TestFormatTimeSortsChronologically(t *testing.T) {
	paris := time.FixedZone("CEST", 2*60*60)
	base := time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC)
	cases := []struct {
		name          string
		before, after time.Time
	}{
		{name: "same second, without then with fraction", before: base, after: base.Add(500 * time.Millisecond)},
		{name: "same second, short then long fraction", before: base.Add(100 * time.Millisecond), after: base.Add(120 * time.Millisecond)},
		{name: "fraction then next second", before: base.Add(999 * time.Millisecond), after: base.Add(time.Second)},
		{name: "other time zone", before: base.In(paris), after: base.Add(time.Millisecond)},
		{name: "across days", before: time.Date(2026, 10, 5, 23, 59, 59, 0, time.UTC), after: time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before, after := formatTime(tc.before), formatTime(tc.after)
			if len(before) != len(after) {
				t.Errorf("widths differ: %q, %q", before, after)
			}
			if before >= after {
				t.Errorf("%q does not sort before %q", before, after)
			}
			for _, tm := range []time.Time{tc.before, tc.after} {
				got, err := parseTime(formatTime(tm))
				if err != nil || !got.Equal(tm.Truncate(time.Millisecond)) {
					t.Errorf("parseTime(formatTime(%v)) = %v, %v", tm, got, err)
				}
			}
		})
	}
}
