package acceptance

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestCompare(t *testing.T) {
	failed := errors.New("step is undefined")
	cases := []struct {
		name    string
		results map[string]error
		pending []string
		want    []string
	}{
		{name: "nothing", want: nil},
		{name: "passing scenario not pending", results: map[string]error{"a.feature: A": nil}, want: nil},
		{name: "failing scenario pending", results: map[string]error{"a.feature: A": failed}, pending: []string{"a.feature: A"}, want: nil},
		{
			name:    "failing scenario not pending",
			results: map[string]error{"a.feature: A": failed},
			want:    []string{"a.feature: A: fails: step is undefined"},
		},
		{
			name:    "pending scenario passes",
			results: map[string]error{"a.feature: A": nil},
			pending: []string{"a.feature: A"},
			want:    []string{"a.feature: A: passes, remove it from pending.txt"},
		},
		{
			name:    "pending scenario does not exist",
			pending: []string{"a.feature: Renamed"},
			want:    []string{"a.feature: Renamed: listed in pending.txt but no such scenario was run"},
		},
		{
			name: "several problems, sorted",
			results: map[string]error{
				"b.feature: B": failed,
				"a.feature: A": nil,
				"c.feature: C": failed,
			},
			pending: []string{"a.feature: A", "c.feature: C"},
			want: []string{
				"a.feature: A: passes, remove it from pending.txt",
				"b.feature: B: fails: step is undefined",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := compare(tc.results, tc.pending); !slices.Equal(got, tc.want) {
				t.Errorf("compare = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParsePending(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    []string
		wantErr string
	}{
		{name: "empty", input: "", want: nil},
		{
			name:  "comments and blank lines",
			input: "# header\n\na.feature: A\n  b.feature: B, with a comma  \n",
			want:  []string{"a.feature: A", "b.feature: B, with a comma"},
		},
		{name: "malformed line", input: "a.feature A\n", wantErr: "pending.txt:1:"},
		{name: "duplicate", input: "a.feature: A\na.feature: A\n", wantErr: "pending.txt:2: \"a.feature: A\" listed twice"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parsePending(strings.NewReader(tc.input))
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want it to contain %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("parsePending = %q, want %q", got, tc.want)
			}
		})
	}
}
