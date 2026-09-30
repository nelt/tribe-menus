package main

import (
	"reflect"
	"testing"
)

func TestCheck(t *testing.T) {
	const (
		name  = "Ada Lovelace"
		email = "ada@example.org"
	)
	signed := func(message string) commit {
		return commit{hash: "abc", authorName: name, authorEmail: email, parents: 1, message: message}
	}
	cases := []struct {
		name     string
		commit   commit
		unsigned bool
	}{
		{name: "sign-off", commit: signed("Fix\n\nSigned-off-by: Ada Lovelace <ada@example.org>\n")},
		{name: "sign-off among other trailers", commit: signed("Fix\n\nCo-Authored-By: Bob <bob@example.org>\nSigned-off-by: Ada Lovelace <ada@example.org>\n")},
		{name: "several sign-offs", commit: signed("Fix\n\nSigned-off-by: Bob <bob@example.org>\nSigned-off-by: Ada Lovelace <ada@example.org>\n")},
		{name: "email case differs", commit: signed("Fix\n\nSigned-off-by: Ada Lovelace <Ada@Example.org>\n")},
		{name: "no sign-off", commit: signed("Fix\n"), unsigned: true},
		{name: "sign-off by someone else", commit: signed("Fix\n\nSigned-off-by: Bob <bob@example.org>\n"), unsigned: true},
		{name: "different email", commit: signed("Fix\n\nSigned-off-by: Ada Lovelace <ada@example.com>\n"), unsigned: true},
		{name: "different name", commit: signed("Fix\n\nSigned-off-by: Ada <ada@example.org>\n"), unsigned: true},
		{name: "malformed sign-off", commit: signed("Fix\n\nSigned-off-by: Ada Lovelace ada@example.org\n"), unsigned: true},
		{name: "sign-off quoted in the subject", commit: signed("Require Signed-off-by: lines\n"), unsigned: true},
		{name: "merge commit", commit: commit{hash: "abc", authorName: name, authorEmail: email, parents: 2, message: "Merge\n"}},
		{name: "dependabot", commit: commit{hash: "abc", authorName: "dependabot[bot]", authorEmail: dependabotEmail, parents: 1, message: "Bump lit\n"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var want []commit
			if tc.unsigned {
				want = []commit{tc.commit}
			}
			if got := check([]commit{tc.commit}); !reflect.DeepEqual(got, want) {
				t.Errorf("check = %v, want %v", got, want)
			}
		})
	}
}

func TestParseLog(t *testing.T) {
	out := "aaa\x00Ada\x00ada@example.org\x00p1\x00Fix\n\nSigned-off-by: Ada <ada@example.org>\n\x1e\n" +
		"bbb\x00Bob\x00bob@example.org\x00p1 p2\x00Merge\n\x1e\n"
	want := []commit{
		{hash: "aaa", authorName: "Ada", authorEmail: "ada@example.org", parents: 1, message: "Fix\n\nSigned-off-by: Ada <ada@example.org>\n"},
		{hash: "bbb", authorName: "Bob", authorEmail: "bob@example.org", parents: 2, message: "Merge\n"},
	}
	got, err := parseLog(out)
	if err != nil {
		t.Fatalf("parseLog: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseLog = %v, want %v", got, want)
	}

	if _, err := parseLog("aaa\x00Ada\x1e"); err == nil {
		t.Error("parseLog of a truncated record: want an error")
	}
	if got, err := parseLog(""); err != nil || got != nil {
		t.Errorf("parseLog of an empty range = %v, %v, want nil, nil", got, err)
	}
}
