package tribe

import (
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestNewLoginCode(t *testing.T) {
	eightDigits := regexp.MustCompile(`^[0-9]{8}$`)
	seen := map[string]bool{}
	// Drawn among 10^8 codes, a single one of 1000 has 99 chances out of 100 to start
	// with something else than 00: all of them do only if the draw ignores the length.
	beyondSix := 0
	for range 1000 {
		code, err := newLoginCode()
		if err != nil {
			t.Fatal(err)
		}
		if !eightDigits.MatchString(code) {
			t.Fatalf("code %q is not 8 digits", code)
		}
		if !strings.HasPrefix(code, "00") {
			beyondSix++
		}
		seen[code] = true
	}
	if len(seen) < 990 {
		t.Errorf("%d distinct codes out of 1000", len(seen))
	}
	if beyondSix < 950 {
		t.Errorf("%d codes out of 1000 do not start with 00, want about 990: the draw ignores the length", beyondSix)
	}
}

func TestLoginCodeCheck(t *testing.T) {
	now := time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC)
	issued := newRealCode("12345678", now)
	withAttempts := func(n int) LoginCode {
		c := issued
		c.AttemptsLeft = n
		return c
	}
	cases := []struct {
		name         string
		code         LoginCode
		entered      string
		at           time.Time
		wantErr      error
		wantAttempts int // for an incorrect code
		wantKeep     bool
	}{
		{name: "right code", code: issued, entered: "12345678", at: now},
		{name: "right code just before expiry", code: issued, entered: "12345678", at: now.Add(LoginCodeValidity - time.Millisecond)},
		{name: "wrong code, first attempt", code: issued, entered: "87654321", at: now, wantErr: &IncorrectCodeError{}, wantAttempts: 2, wantKeep: true},
		{name: "wrong code, second attempt", code: withAttempts(2), entered: "87654321", at: now, wantErr: &IncorrectCodeError{}, wantAttempts: 1, wantKeep: true},
		{name: "wrong code, third attempt invalidates", code: withAttempts(1), entered: "87654321", at: now, wantErr: &IncorrectCodeError{}, wantAttempts: 0},
		{name: "right code after 2 wrong ones", code: withAttempts(1), entered: "12345678", at: now},
		{name: "right code without attempts left", code: withAttempts(0), entered: "12345678", at: now, wantErr: ErrNewCodeNeeded},
		{name: "right code at expiry", code: issued, entered: "12345678", at: now.Add(LoginCodeValidity), wantErr: ErrNewCodeNeeded},
		{name: "wrong code after expiry", code: issued, entered: "00000000", at: now.Add(time.Hour), wantErr: ErrNewCodeNeeded},
		{name: "decoy code", code: NewDecoyCode(now), entered: "12345678", at: now, wantErr: &IncorrectCodeError{}, wantAttempts: 2, wantKeep: true},
		{name: "decoy code, empty input", code: NewDecoyCode(now), entered: "", at: now, wantErr: &IncorrectCodeError{}, wantAttempts: 2, wantKeep: true},
		{name: "expired decoy code", code: NewDecoyCode(now), entered: "12345678", at: now.Add(LoginCodeValidity), wantErr: ErrNewCodeNeeded},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			after, keep, err := tc.code.Check(tc.entered, tc.at)
			var incorrect *IncorrectCodeError
			switch {
			case tc.wantErr == nil && err != nil:
				t.Fatalf("error = %v, want accepted", err)
			case errors.As(tc.wantErr, &incorrect):
				if !errors.As(err, &incorrect) || incorrect.AttemptsLeft != tc.wantAttempts {
					t.Fatalf("error = %v, want incorrect with %d attempts left", err, tc.wantAttempts)
				}
				if after.AttemptsLeft != tc.wantAttempts {
					t.Errorf("attempts left after = %d, want %d", after.AttemptsLeft, tc.wantAttempts)
				}
			case tc.wantErr != nil && !errors.Is(err, tc.wantErr):
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
			if keep != tc.wantKeep {
				t.Errorf("keep = %v, want %v", keep, tc.wantKeep)
			}
		})
	}
}

func TestNewSessionToken(t *testing.T) {
	a, err := newSessionToken()
	if err != nil {
		t.Fatal(err)
	}
	b, err := newSessionToken()
	if err != nil {
		t.Fatal(err)
	}
	// 32 bytes in unpadded base64url.
	if len(a) != 43 || a == b {
		t.Errorf("tokens %q and %q: want two distinct tokens of 43 characters", a, b)
	}
}
