package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestReadCredential(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "plain"), "s3cret")
	writeFile(t, filepath.Join(dir, "line-feed"), "s3cret\n")
	writeFile(t, filepath.Join(dir, "crlf"), "s3cret\r\n")
	writeFile(t, filepath.Join(dir, "spaces"), " s3cret \n")
	writeFile(t, filepath.Join(dir, "empty"), "")
	writeFile(t, filepath.Join(dir, "only-line-feed"), "\n")
	cases := []struct {
		name, dir, file string
		want, wantErr   string
	}{
		{name: "plain", dir: dir, file: "plain", want: "s3cret"},
		{name: "final line feed", dir: dir, file: "line-feed", want: "s3cret"},
		{name: "final CRLF", dir: dir, file: "crlf", want: "s3cret"},
		{name: "spaces kept", dir: dir, file: "spaces", want: " s3cret "},
		{name: "directory not set", dir: "", file: "plain", wantErr: "credential plain: CREDENTIALS_DIRECTORY not set"},
		{name: "missing", dir: dir, file: "missing", wantErr: "credential missing: no such file or directory"},
		{name: "empty", dir: dir, file: "empty", wantErr: "credential empty: empty"},
		{name: "only a line feed", dir: dir, file: "only-line-feed", wantErr: "credential only-line-feed: empty"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := readCredential(func(k string) string {
				if k == "CREDENTIALS_DIRECTORY" {
					return tc.dir
				}
				return ""
			}, tc.file)
			if tc.wantErr != "" {
				if err == nil || !strings.HasPrefix(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Errorf("credential = %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}
