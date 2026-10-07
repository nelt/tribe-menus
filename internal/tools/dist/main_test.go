package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func repository() fstest.MapFS {
	return fstest.MapFS{
		"bin/tribe-menus":            {Data: []byte("ELF"), Mode: 0o755},
		"bin/tribe-menus-dev":        {Data: []byte("not archived"), Mode: 0o755},
		"LICENSE":                    {Data: []byte("GNU AFFERO GENERAL PUBLIC LICENSE"), Mode: 0o664},
		"site/index.html":            {Data: []byte("<title>Melting Tribe</title>"), Mode: 0o600},
		"site/robots.txt":            {Data: []byte("Disallow: /tribes/\n")},
		"deploy/README.md":           {Data: []byte("# deploy/")},
		"deploy/config.example.json": {Data: []byte("{}"), ModTime: time.Now()},
		"deploy/provision.sh":        {Data: []byte("#!/bin/sh"), Mode: 0o775},
		"web/dist/index.html":        {Data: []byte("embedded in the binary")},
	}
}

// archived is what an entry of the archive says of itself.
type archived struct {
	name    string
	mode    int64
	content string
}

func unpack(t *testing.T, archive []byte) []archived {
	t.Helper()
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		t.Fatalf("gzip: %v", err)
	}
	tr := tar.NewReader(gz)
	var got []archived
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return got
		}
		if err != nil {
			t.Fatalf("tar: %v", err)
		}
		if h.Uid != 0 || h.Gid != 0 || h.Uname != "" || h.Gname != "" || !h.ModTime.Equal(time.Unix(0, 0)) {
			t.Errorf("%s: owner %d:%d (%q:%q), date %v, want root and the epoch", h.Name, h.Uid, h.Gid, h.Uname, h.Gname, h.ModTime)
		}
		content, err := io.ReadAll(tr)
		if err != nil {
			t.Fatalf("read %s: %v", h.Name, err)
		}
		got = append(got, archived{name: h.Name, mode: h.Mode, content: string(content)})
	}
}

func TestWrite(t *testing.T) {
	dir := t.TempDir()
	archive, err := write(repository(), dir, "v1.2.3")
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if want := filepath.Join(dir, "tribe-menus-v1.2.3-linux-amd64.tar.gz"); archive != want {
		t.Errorf("archive = %q, want %q", archive, want)
	}
	data, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}

	top := "tribe-menus-v1.2.3-linux-amd64/"
	want := []archived{
		{name: top, mode: 0o755},
		{name: top + "LICENSE", mode: 0o644, content: "GNU AFFERO GENERAL PUBLIC LICENSE"},
		{name: top + "deploy/", mode: 0o755},
		{name: top + "deploy/README.md", mode: 0o644, content: "# deploy/"},
		{name: top + "deploy/config.example.json", mode: 0o644, content: "{}"},
		{name: top + "deploy/provision.sh", mode: 0o755, content: "#!/bin/sh"},
		{name: top + "site/", mode: 0o755},
		{name: top + "site/index.html", mode: 0o644, content: "<title>Melting Tribe</title>"},
		{name: top + "site/robots.txt", mode: 0o644, content: "Disallow: /tribes/\n"},
		{name: top + "tribe-menus", mode: 0o755, content: "ELF"},
	}
	got := unpack(t, data)
	if len(got) != len(want) {
		t.Fatalf("entries = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d = %+v, want %+v", i, got[i], want[i])
		}
	}

	sum, err := os.ReadFile(archive + ".sha256")
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(data)
	if want := hex.EncodeToString(hash[:]) + "  tribe-menus-v1.2.3-linux-amd64.tar.gz\n"; string(sum) != want {
		t.Errorf("SHA-256 file = %q, want %q", sum, want)
	}
}

func TestWriteIsReproducible(t *testing.T) {
	first, err := write(repository(), t.TempDir(), "pr-12")
	if err != nil {
		t.Fatal(err)
	}
	// Other dates and permissions outside the execute bit change nothing.
	again := repository()
	for name, file := range again {
		file.ModTime = time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
		if file.Mode&0o111 == 0 {
			file.Mode = 0o640
		}
		again[name] = file
	}
	second, err := write(again, t.TempDir(), "pr-12")
	if err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"", ".sha256"} {
		a, _ := os.ReadFile(first + suffix)
		b, _ := os.ReadFile(second + suffix)
		if len(a) == 0 || !bytes.Equal(a, b) {
			t.Errorf("%s differs from one run to the next", filepath.Base(first+suffix))
		}
	}
}

func TestWriteRefuses(t *testing.T) {
	cases := []struct {
		name   string
		change func(fstest.MapFS)
		want   string
	}{
		{name: "binary not built", change: func(r fstest.MapFS) { delete(r, "bin/tribe-menus") }, want: "binary not built"},
		{name: "licence missing", change: func(r fstest.MapFS) { delete(r, "LICENSE") }, want: "read LICENSE"},
		{name: "site missing", change: func(r fstest.MapFS) {
			delete(r, "site/index.html")
			delete(r, "site/robots.txt")
		}, want: "read site"},
		{name: "symbolic link", change: func(r fstest.MapFS) {
			r["deploy/link"] = &fstest.MapFile{Data: []byte("/etc/passwd"), Mode: fs.ModeSymlink}
		}, want: "deploy/link is not a file"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := repository()
			tc.change(r)
			_, err := write(r, t.TempDir(), "dev")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("write error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestRun(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		wantCode int
	}{
		{name: "no version", args: nil, wantCode: 2},
		{name: "path in the version", args: []string{"-version", "../v1"}, wantCode: 2},
		{name: "extra argument", args: []string{"-version", "v1.2.3", "extra"}, wantCode: 2},
		{name: "binary not built", args: []string{"-version", "v1.2.3"}, wantCode: 1},
	}
	t.Chdir(t.TempDir())
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := run(tc.args, &stdout, &stderr); code != tc.wantCode {
				t.Errorf("code = %d, want %d (stderr %q)", code, tc.wantCode, stderr.String())
			}
		})
	}
}
