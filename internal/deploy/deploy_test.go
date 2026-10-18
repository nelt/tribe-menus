package deploy

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// run sources a script of deploy/bin and calls one of its functions with args, in dir.
func run(t *testing.T, dir, script, function string, args ...string) (string, error) {
	t.Helper()
	return runWithInput(t, dir, "", script, function, args...)
}

func runWithInput(t *testing.T, dir, input, script, function string, args ...string) (string, error) {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "deploy", "bin", script))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", append([]string{"-c", `source "$1"; shift; "$@"`, "_", path, function}, args...)...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(input)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err = cmd.Run()
	return out.String(), err
}

func deployFunc(t *testing.T, dir, function string, args ...string) (string, error) {
	t.Helper()
	return run(t, dir, "tribe-menus-deploy", function, args...)
}

const base = "tribe-menus-v1.2.3-linux-amd64"

// writeArchive writes a .tar.gz of these entries, and returns its path.
func writeArchive(t *testing.T, dir string, headers []tar.Header) string {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, h := range headers {
		var body []byte
		if h.Typeflag == tar.TypeReg {
			body = []byte("content of " + h.Name)
			h.Size = int64(len(body))
		}
		if h.Mode == 0 {
			h.Mode = 0o644
		}
		if err := tw.WriteHeader(&h); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, base+".tar.gz")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func dirEntry(name string) tar.Header {
	return tar.Header{Name: name, Typeflag: tar.TypeDir, Mode: 0o755}
}

func fileEntry(name string) tar.Header {
	return tar.Header{Name: name, Typeflag: tar.TypeReg}
}

// goodEntries is the shape of an archive of internal/tools/dist.
func goodEntries() []tar.Header {
	return []tar.Header{
		dirEntry(base + "/"),
		{Name: base + "/tribe-menus", Typeflag: tar.TypeReg, Mode: 0o755},
		fileEntry(base + "/LICENSE"),
		dirEntry(base + "/deploy/"),
		{Name: base + "/deploy/provision.sh", Typeflag: tar.TypeReg, Mode: 0o755},
		dirEntry(base + "/site/"),
		fileEntry(base + "/site/index.html"),
	}
}

// An archive goes past check_archive only with files and directories under its top
// directory (plan recette, lot C: unpacking of the archive).
func TestCheckArchive(t *testing.T) {
	tests := []struct {
		name  string
		extra tar.Header
		ok    bool
	}{
		{name: "archive of dist", ok: true},
		{name: "parent directory", extra: fileEntry(base + "/../escaped")},
		{name: "parent directory at the end", extra: dirEntry(base + "/site/..")},
		{name: "absolute path", extra: fileEntry("/etc/cron.d/x")},
		{name: "another top directory", extra: fileEntry("tribe-menus-v9.9.9-linux-amd64/tribe-menus")},
		{name: "top directory as a prefix", extra: fileEntry(base + "x/tribe-menus")},
		{name: "symbolic link", extra: tar.Header{Name: base + "/site/link", Typeflag: tar.TypeSymlink, Linkname: "/etc/shadow"}},
		{name: "hard link", extra: tar.Header{Name: base + "/site/hard", Typeflag: tar.TypeLink, Linkname: base + "/LICENSE"}},
		{name: "device", extra: tar.Header{Name: base + "/site/null", Typeflag: tar.TypeChar}},
		{name: "space in a name", extra: fileEntry(base + "/site/a b")},
		{name: "new line in a name", extra: fileEntry(base + "/site/a\n-rw-r--r-- x")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			entries := goodEntries()
			if tc.extra.Name != "" {
				entries = append(entries, tc.extra)
			}
			archive := writeArchive(t, dir, entries)
			out, err := deployFunc(t, dir, "check_archive", archive, base)
			if tc.ok && err != nil {
				t.Fatalf("refused: %v\n%s", err, out)
			}
			if !tc.ok && err == nil {
				t.Fatalf("accepted")
			}
			if !tc.ok && !strings.Contains(out, "archive") {
				t.Errorf("error does not name the archive: %s", out)
			}
		})
	}
}

// What check_archive accepts, unpack extracts as the content of the release.
func TestUnpack(t *testing.T) {
	dir := t.TempDir()
	archive := writeArchive(t, dir, goodEntries())
	release := filepath.Join(dir, "release")
	if err := os.Mkdir(release, 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := deployFunc(t, dir, "unpack", archive, base, release); err != nil {
		t.Fatalf("unpack: %v\n%s", err, out)
	}
	for _, name := range []string{"tribe-menus", "LICENSE", "deploy/provision.sh", "site/index.html"} {
		if _, err := os.Stat(filepath.Join(release, name)); err != nil {
			t.Error(err)
		}
	}
	info, err := os.Stat(filepath.Join(release, "tribe-menus"))
	if err != nil || info.Mode().Perm()&0o111 == 0 {
		t.Errorf("binary not executable: %v", err)
	}
}

func TestCheckSum(t *testing.T) {
	dir := t.TempDir()
	archive := writeArchive(t, dir, goodEntries())
	content, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	good := hex.EncodeToString(sum[:])
	tests := []struct {
		name string
		file string
		ok   bool
	}{
		{name: "sha256sum format", file: good + "  " + base + ".tar.gz\n", ok: true},
		{name: "another sum", file: strings.Repeat("0", 64) + "  " + base + ".tar.gz\n"},
		{name: "another archive", file: good + "  tribe-menus-v0.0.1-linux-amd64.tar.gz\n"},
		{name: "path", file: good + "  ../" + base + ".tar.gz\n"},
		{name: "empty", file: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(archive+".sha256", []byte(tc.file), 0o644); err != nil {
				t.Fatal(err)
			}
			out, err := deployFunc(t, dir, "check_sum", archive, base)
			if tc.ok != (err == nil) {
				t.Errorf("check_sum: %v, want ok = %v\n%s", err, tc.ok, out)
			}
		})
	}
}

const (
	tribeA = "tribes/0123456789abcdef0123456789abcdef.db"
	tribeB = "tribes/fedcba9876543210fedcba9876543210.db"
)

func needSQLite(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not installed")
	}
}

// sqlite runs a statement on a database of dir and returns its output.
func sqlite(t *testing.T, dir, db, statement string) string {
	t.Helper()
	out, err := exec.Command("sqlite3", "-bail", filepath.Join(dir, db), statement).CombinedOutput()
	if err != nil {
		t.Fatalf("sqlite3 %s: %v\n%s", db, err, out)
	}
	return strings.TrimSpace(string(out))
}

// dataDir makes a data directory with the registry, the rate limit database and a tribe, at
// these schema versions.
func dataDir(t *testing.T) string {
	t.Helper()
	data := filepath.Join(t.TempDir(), "data")
	if err := os.MkdirAll(filepath.Join(data, "tribes"), 0o700); err != nil {
		t.Fatal(err)
	}
	sqlite(t, data, "registry.db", "PRAGMA journal_mode=WAL; CREATE TABLE tribes(slug); INSERT INTO tribes VALUES ('a'); PRAGMA user_version=2;")
	sqlite(t, data, "ratelimit.db", "PRAGMA journal_mode=WAL; CREATE TABLE counters(n); PRAGMA user_version=1;")
	sqlite(t, data, tribeA, "PRAGMA journal_mode=WAL; CREATE TABLE members(email); INSERT INTO members VALUES ('alice'); PRAGMA user_version=3;")
	return data
}

// A snapshot copies the registry and the tribes, not the rate limit database, and keeps
// every schema version (ADR 0016, point 1.3; ADR 0021, point 6).
func TestSnapshot(t *testing.T) {
	needSQLite(t)
	data := dataDir(t)
	snap := filepath.Join(filepath.Dir(data), "avant-v1.2.3")
	if out, err := deployFunc(t, data, "snapshot", data, snap); err != nil {
		t.Fatalf("snapshot: %v\n%s", err, out)
	}
	schema, err := os.ReadFile(filepath.Join(snap, "schema"))
	if err != nil {
		t.Fatal(err)
	}
	want := "registry.db 2\nratelimit.db 1\n" + tribeA + " 3\n"
	if string(schema) != want {
		t.Errorf("schema = %q, want %q", schema, want)
	}
	if got := sqlite(t, snap, tribeA, "SELECT email FROM members"); got != "alice" {
		t.Errorf("tribe in the snapshot: %q", got)
	}
	if _, err := os.Stat(filepath.Join(snap, "ratelimit.db")); !os.IsNotExist(err) {
		t.Errorf("rate limit database copied: %v", err)
	}
	if out, err := deployFunc(t, data, "schema_changed", data, snap); err == nil {
		t.Errorf("schema changed right after the snapshot:\n%s", out)
	}
}

// After a migration, the schema has changed and a restore puts the snapshot back, setting
// aside what the data directory had (plan recette, D7).
func TestSchemaChangedAndRestore(t *testing.T) {
	needSQLite(t)
	tests := []struct {
		name    string
		change  func(t *testing.T, data string)
		changed bool
	}{
		{name: "nothing", change: func(*testing.T, string) {}},
		{name: "rows only", change: func(t *testing.T, data string) {
			sqlite(t, data, tribeA, "INSERT INTO members VALUES ('bruno')")
		}},
		{name: "rate limit database only", change: func(t *testing.T, data string) {
			sqlite(t, data, "ratelimit.db", "PRAGMA user_version=2")
		}},
		{name: "tribe migrated", changed: true, change: func(t *testing.T, data string) {
			sqlite(t, data, tribeA, "PRAGMA user_version=4")
		}},
		{name: "registry migrated", changed: true, change: func(t *testing.T, data string) {
			sqlite(t, data, "registry.db", "PRAGMA user_version=3")
		}},
		{name: "tribe created at the same version", change: func(t *testing.T, data string) {
			sqlite(t, data, tribeB, "PRAGMA user_version=3")
		}},
		{name: "tribe created at a newer version", changed: true, change: func(t *testing.T, data string) {
			sqlite(t, data, tribeB, "PRAGMA user_version=4")
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data := dataDir(t)
			snap := filepath.Join(filepath.Dir(data), "avant-v1.2.3")
			if out, err := deployFunc(t, data, "snapshot", data, snap); err != nil {
				t.Fatalf("snapshot: %v\n%s", err, out)
			}
			tc.change(t, data)
			out, err := deployFunc(t, data, "schema_changed", data, snap)
			if changed := err == nil; changed != tc.changed {
				t.Fatalf("schema_changed = %v, want %v\n%s", changed, tc.changed, out)
			}
			if !tc.changed {
				return
			}
			aside := filepath.Join(filepath.Dir(data), "remplacees")
			if out, err := deployFunc(t, data, "restore", data, snap, aside); err != nil {
				t.Fatalf("restore: %v\n%s", err, out)
			}
			if out, err := deployFunc(t, data, "schema_changed", data, snap); err == nil {
				t.Errorf("schema still changed after the restore:\n%s", out)
			}
			if got := sqlite(t, data, tribeA, "PRAGMA user_version"); got != "3" {
				t.Errorf("tribe at version %s after the restore", got)
			}
			if _, err := os.Stat(filepath.Join(aside, tribeA)); err != nil {
				t.Errorf("replaced tribe not set aside: %v", err)
			}
			info, err := os.Stat(filepath.Join(data, tribeA))
			if err != nil || info.Mode().Perm() != 0o600 {
				t.Errorf("restored tribe: %v, mode %v", err, info.Mode())
			}
		})
	}
}

func TestDropRatelimit(t *testing.T) {
	needSQLite(t)
	data := dataDir(t)
	snap := filepath.Join(filepath.Dir(data), "avant-v1.2.3")
	aside := filepath.Join(filepath.Dir(data), "remplacees")
	if out, err := deployFunc(t, data, "snapshot", data, snap); err != nil {
		t.Fatalf("snapshot: %v\n%s", err, out)
	}
	if out, err := deployFunc(t, data, "drop_ratelimit", data, snap, aside); err != nil {
		t.Fatalf("drop_ratelimit: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(data, "ratelimit.db")); err != nil {
		t.Errorf("unchanged rate limit database set aside: %v", err)
	}
	sqlite(t, data, "ratelimit.db", "PRAGMA user_version=2")
	if out, err := deployFunc(t, data, "drop_ratelimit", data, snap, aside); err != nil {
		t.Fatalf("drop_ratelimit: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(data, "ratelimit.db")); !os.IsNotExist(err) {
		t.Errorf("migrated rate limit database kept: %v", err)
	}
	if _, err := os.Stat(filepath.Join(aside, "ratelimit.db")); err != nil {
		t.Errorf("rate limit database not set aside: %v", err)
	}
}

// A file of another form in tribes/ stops the script: its name goes into SQL and paths.
func TestDatabasesRefusesOddNames(t *testing.T) {
	data := t.TempDir()
	if err := os.MkdirAll(filepath.Join(data, "tribes"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(data, "tribes", "x';.db"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := deployFunc(t, data, "snapshot", data, filepath.Join(data, "snap")); err == nil {
		t.Errorf("snapshot accepted an odd name:\n%s", out)
	}
}

// The relay keeps only the records of the application that carry the alert attribute, as
// written; the nature, which goes into the subject of the email, is a name or "unknown"
// (ADR 0023, point 1).
func TestAlerts(t *testing.T) {
	journal := strings.Join([]string{
		`{"time":"2026-10-17T10:00:00Z","level":"INFO","msg":"server started"}`,
		`{"time":"2026-10-17T10:10:00Z","level":"ERROR","msg":"code request limits reached repeatedly","alert":"code_request_limits","signals":"refused_requests"}`,
		`not JSON, "alert":"smtp_failures"`,
		`["alert","smtp_failures"]`,
		`{"level":"ERROR","msg":"x","alert":7}`,
		`{"level":"ERROR","msg":"request failed","error":"\"alert\":\"smtp_failures\""}`,
		`{"level":"ERROR","msg":"repeated SMTP failures","alert":"smtp_failures","failures":3}`,
		`{"level":"ERROR","msg":"y","alert":"x\nSubject: injected"}`,
	}, "\n") + "\n"
	out, err := runWithInput(t, t.TempDir(), journal, "tribe-menus-alert", "alerts")
	if err != nil {
		t.Fatalf("alerts: %v\n%s", err, out)
	}
	want := `{"time":"2026-10-17T10:10:00Z","level":"ERROR","msg":"code request limits reached repeatedly","alert":"code_request_limits","signals":"refused_requests"}
{"level":"ERROR","msg":"repeated SMTP failures","alert":"smtp_failures","failures":3}
{"level":"ERROR","msg":"y","alert":"unknown"}
`
	if out != want {
		t.Errorf("alerts:\n%s\nwant:\n%s", out, want)
	}
}

// prune keeps five releases, the current and previous ones whatever their age, their
// snapshots, and the five last sets of databases put aside (ADR 0016, point 1.7).
func TestPrune(t *testing.T) {
	root := t.TempDir()
	releases := filepath.Join(root, "releases")
	backups := filepath.Join(root, "backups")
	start := time.Now().Add(-24 * time.Hour)
	// v1 is the oldest, previous; v8 the newest; v3 the current one.
	for i := 1; i <= 8; i++ {
		name := fmt.Sprintf("v0.%d.0", i)
		for _, dir := range []string{filepath.Join(releases, name), filepath.Join(backups, "avant-"+name)} {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			at := start.Add(time.Duration(i) * time.Hour)
			if err := os.Chtimes(dir, at, at); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.MkdirAll(filepath.Join(backups, fmt.Sprintf("remplacees-2026101%dT000000", i)), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(backups, "manuel-20261001T000000"), 0o700); err != nil {
		t.Fatal(err)
	}
	for link, target := range map[string]string{"current": "releases/v0.3.0", "previous": "releases/v0.1.0"} {
		if err := os.Symlink(target, filepath.Join(root, link)); err != nil {
			t.Fatal(err)
		}
	}
	script, err := filepath.Abs(filepath.Join("..", "..", "deploy", "bin", "tribe-menus-deploy"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", "-c", `source "$1"; RELEASES=$2/releases CURRENT=$2/current PREVIOUS=$2/previous BACKUPS=$2/backups; prune`, "_", script, root)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("prune: %v\n%s", err, out)
	}
	names := func(dir string) string {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		var n []string
		for _, e := range entries {
			n = append(n, e.Name())
		}
		return strings.Join(n, " ")
	}
	if got, want := names(releases), "v0.1.0 v0.3.0 v0.6.0 v0.7.0 v0.8.0"; got != want {
		t.Errorf("releases: %s, want %s", got, want)
	}
	want := "avant-v0.1.0 avant-v0.3.0 avant-v0.6.0 avant-v0.7.0 avant-v0.8.0 manuel-20261001T000000 " +
		"remplacees-20261014T000000 remplacees-20261015T000000 remplacees-20261016T000000 remplacees-20261017T000000 remplacees-20261018T000000"
	if got := names(backups); got != want {
		t.Errorf("backups:\n%s\nwant:\n%s", got, want)
	}
}

// fakeServer is the shell prelude of the recovery tests: tribe-menus-deploy sourced in a
// temporary root, systemctl, curl, runuser and the download replaced. systemctl keeps the
// masked units in a file and fails restart when FAIL_RESTART is set; /healthz answers when
// HEALTHY is set, after a SIGHUP to the script when HUP_FIRST is set.
const fakeServer = `
set -euo pipefail
source "$SCRIPT"
ROOT=$PWD
ENV=recette SERVICE_USER='' RELEASES=$ROOT/releases CURRENT=$ROOT/current PREVIOUS=$ROOT/previous
DATA=$ROOT/data BACKUPS=$ROOT/data/backups SOCKET=$ROOT/sock
systemctl() {
	local command=$1
	shift
	case $command in
	is-enabled) if grep -qx "$1" "$ROOT/masked" 2>/dev/null; then echo masked; else echo enabled; fi ;;
	mask) shift; printf '%s\n' "$@" >>"$ROOT/masked" ;;
	unmask) shift; for u in "$@"; do grep -vx "$u" "$ROOT/masked" >"$ROOT/masked.new" || true; mv "$ROOT/masked.new" "$ROOT/masked"; done ;;
	restart) [[ -z ${FAIL_RESTART:-} ]] ;;
	esac
}
curl() {
	if [[ -n ${HUP_FIRST:-} && ! -e $ROOT/hup ]]; then
		touch "$ROOT/hup"
		kill -HUP $$
	fi
	if [[ -n ${HEALTHY:-} ]]; then echo ok; fi
}
runuser() { shift 3; "$@"; }
logger() { :; }
journalctl() { :; }
chown() { :; }
sleep() { :; }
fetch_release() {
	cp "$ROOT/archive/tribe-menus-$1-linux-amd64.tar.gz" "$ROOT/archive/tribe-menus-$1-linux-amd64.tar.gz.sha256" "$2/"
}
trap on_exit EXIT
`

// recoveryRoot prepares a root with v0.1.0 in service, v0.0.9 as previous, the archive of
// v0.2.0 to deploy, and the environment on hold or not.
func recoveryRoot(t *testing.T, onHold bool) string {
	t.Helper()
	root := t.TempDir()
	for _, v := range []string{"v0.0.9", "v0.1.0"} {
		dir := filepath.Join(root, "releases", v)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "tribe-menus"), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for link, target := range map[string]string{"current": "releases/v0.1.0", "previous": "releases/v0.0.9"} {
		if err := os.Symlink(target, filepath.Join(root, link)); err != nil {
			t.Fatal(err)
		}
	}
	snap := filepath.Join(root, "data", "backups", "avant-v0.1.0")
	if err := os.MkdirAll(snap, 0o700); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{"schema": "registry.db 1\n", "taken": "2026-10-17T10:00:00+02:00\n", "registry.db": ""} {
		if err := os.WriteFile(filepath.Join(snap, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	const v = "tribe-menus-v0.2.0-linux-amd64"
	archiveDir := filepath.Join(root, "archive")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	binary := []byte("#!/bin/sh\necho 'tribe-menus v0.2.0 (abcdef0)'\n")
	for _, h := range []tar.Header{
		{Name: v + "/", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: v + "/tribe-menus", Typeflag: tar.TypeReg, Mode: 0o755, Size: int64(len(binary))},
		{Name: v + "/site/", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: v + "/deploy/", Typeflag: tar.TypeDir, Mode: 0o755},
	} {
		if err := tw.WriteHeader(&h); err != nil {
			t.Fatal(err)
		}
		if h.Typeflag == tar.TypeReg {
			if _, err := tw.Write(binary); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(archiveDir, v+".tar.gz")
	if err := os.WriteFile(archive, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(buf.Bytes())
	if err := os.WriteFile(archive+".sha256", []byte(hex.EncodeToString(sum[:])+"  "+v+".tar.gz\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	masked := ""
	if onHold {
		masked = "tribe-menus@recette.socket\ntribe-menus@recette.service\n"
	}
	if err := os.WriteFile(filepath.Join(root, "masked"), []byte(masked), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// A deployment, a rollback or a restore that fails or is interrupted puts the environment
// back: the previous version in service, the hold back if there was one; a failed restore
// puts it on hold (review of PR #56, points 1 and 2).
func TestRecovery(t *testing.T) {
	script, err := filepath.Abs(filepath.Join("..", "..", "deploy", "bin", "tribe-menus-deploy"))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		command    string
		env        []string
		onHold     bool
		wantOK     bool
		wantLink   string
		wantMasked bool
	}{
		{name: "restart fails", command: "deploy v0.2.0", env: []string{"FAIL_RESTART=1"}, wantLink: "releases/v0.1.0"},
		{name: "restart fails, on hold", command: "deploy v0.2.0", env: []string{"FAIL_RESTART=1"}, onHold: true, wantLink: "releases/v0.1.0", wantMasked: true},
		{name: "no answer, on hold", command: "deploy v0.2.0", onHold: true, wantLink: "releases/v0.1.0", wantMasked: true},
		{name: "SIGHUP during the wait", command: "deploy v0.2.0", env: []string{"HEALTHY=1", "HUP_FIRST=1"}, wantOK: true, wantLink: "releases/v0.2.0"},
		{name: "deployed, hold lifted", command: "deploy v0.2.0", env: []string{"HEALTHY=1"}, onHold: true, wantOK: true, wantLink: "releases/v0.2.0"},
		{name: "rollback without answer, on hold", command: "rollback 0", onHold: true, wantLink: "releases/v0.0.9", wantMasked: true},
		{name: "restore without answer", command: "restore_snapshot avant-v0.1.0", wantLink: "releases/v0.1.0", wantMasked: true},
		// Every write fails, as on a terminal gone with its SSH session: the script goes on
		// (review of PR #56, after the session S2).
		{name: "terminal gone", command: "exec >/dev/full 2>/dev/full; deploy v0.2.0", env: []string{"HEALTHY=1"}, wantOK: true, wantLink: "releases/v0.2.0"},
		{name: "terminal gone, no answer", command: "exec >/dev/full 2>/dev/full; deploy v0.2.0", onHold: true, wantLink: "releases/v0.1.0", wantMasked: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := recoveryRoot(t, tc.onHold)
			cmd := exec.Command("bash", "-c", fakeServer+tc.command)
			cmd.Dir = root
			cmd.Env = append(append(os.Environ(), "SCRIPT="+script), tc.env...)
			out, err := cmd.CombinedOutput()
			if tc.wantOK != (err == nil) {
				t.Fatalf("%s: %v, want ok = %v\n%s", tc.command, err, tc.wantOK, out)
			}
			link, err := os.Readlink(filepath.Join(root, "current"))
			if err != nil || link != tc.wantLink {
				t.Errorf("current -> %s (%v), want %s\n%s", link, err, tc.wantLink, out)
			}
			masked, err := os.ReadFile(filepath.Join(root, "masked"))
			if err != nil {
				t.Fatal(err)
			}
			want := ""
			if tc.wantMasked {
				want = "tribe-menus@recette.socket\ntribe-menus@recette.service\n"
			}
			if string(masked) != want {
				t.Errorf("masked units %q, want %q\n%s", masked, want, out)
			}
		})
	}
}

// An alert that msmtp could not send is read again and sent at the next run of the relay:
// the cursor moves only once every email is sent (review of PR #56, point 3).
func TestRelayKeepsUnsentAlerts(t *testing.T) {
	script, err := filepath.Abs(filepath.Join("..", "..", "deploy", "bin", "tribe-menus-alert"))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	journal := `{"level":"INFO","msg":"server started"}
{"level":"ERROR","msg":"repeated SMTP failures","alert":"smtp_failures","failures":3}
`
	if err := os.WriteFile(filepath.Join(root, "journal"), []byte(journal), 0o644); err != nil {
		t.Fatal(err)
	}
	// journalctl prints the lines after the cursor, then "-- cursor: i=<lines read>", and
	// refuses two starting points, as the real one does (séance S2); send fails while the
	// file fail exists.
	prelude := `
set -euo pipefail
source "$SCRIPT"
ROOT=$PWD
journalctl() {
	local from=0 starts=0 arg
	for arg in "$@"; do
		case $arg in
		--after-cursor=i=*) from=${arg#--after-cursor=i=}; starts=$((starts + 1)) ;;
		--after-cursor=*) return 1 ;;
		--since=* | --cursor=* | --cursor-file=*) starts=$((starts + 1)) ;;
		esac
	done
	if ((starts != 1)); then
		echo "Please specify only one of --since=, --cursor=, --cursor-file=, and --after-cursor=." >&2
		return 1
	fi
	local total
	total=$(wc -l <"$ROOT/journal" | tr -d ' ')
	if ((total > from)); then
		tail -n +"$((from + 1))" "$ROOT/journal"
		echo "-- cursor: i=$total"
	fi
}
send() {
	cat >/dev/null
	if [[ -e $ROOT/fail ]]; then return 1; fi
	printf '%s\n' "$1" >>"$ROOT/sent"
}
logger() { :; }
mkdir -p "$ROOT/state"
relay_env recette "$ROOT/state"
`
	relay := func() error {
		cmd := exec.Command("bash", "-c", prelude)
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "SCRIPT="+script)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("relay: %v\n%s", err, out)
		}
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "fail"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := relay(); err == nil {
		t.Fatal("relay succeeded although send failed")
	}
	if err := os.Remove(filepath.Join(root, "fail")); err != nil {
		t.Fatal(err)
	}
	if err := relay(); err != nil {
		t.Fatal(err)
	}
	sent, _ := os.ReadFile(filepath.Join(root, "sent"))
	if string(sent) != "alerte smtp_failures (recette)\n" {
		t.Errorf("sent after the failure: %q", sent)
	}
	// Nothing new: nothing sent again.
	if err := relay(); err != nil {
		t.Fatal(err)
	}
	sent, _ = os.ReadFile(filepath.Join(root, "sent"))
	if string(sent) != "alerte smtp_failures (recette)\n" {
		t.Errorf("sent after a run without news: %q", sent)
	}
	// A cursor that journalctl refuses: the last hour is read again, the relay goes on.
	if err := os.WriteFile(filepath.Join(root, "state", "recette.cursor"), []byte("x=zz\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := relay(); err != nil {
		t.Errorf("relay stopped on a refused cursor: %v", err)
	}
}

// The first run of the relay passes options that the real journalctl accepts: a single
// starting point (séance S2).
func TestRelayFirstRunWithJournalctl(t *testing.T) {
	if _, err := exec.LookPath("journalctl"); err != nil {
		t.Skip("journalctl not installed")
	}
	script, err := filepath.Abs(filepath.Join("..", "..", "deploy", "bin", "tribe-menus-alert"))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	cmd := exec.Command("bash", "-c", `
set -euo pipefail
source "$SCRIPT"
journalctl() { command journalctl --directory="$PWD/journal" "$@"; }
logger() { :; }
mkdir -p journal state
relay_env recette "$PWD/state"
`)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "SCRIPT="+script)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("relay with journalctl: %v\n%s", err, out)
	}
}

// The report of tribe-menus-check names no host: journalctl writes the host name in its
// short formats unless told not to (review of PR #56, point 4).
func TestReportWithoutHostName(t *testing.T) {
	script, err := filepath.Abs(filepath.Join("..", "..", "deploy", "bin", "tribe-menus-check"))
	if err != nil {
		t.Fatal(err)
	}
	prelude := `
set -euo pipefail
source "$SCRIPT"
environments() { printf 'recette\n'; }
journalctl() {
	local arg host=machine-de-l-instance short=0
	for arg in "$@"; do
		case $arg in
		--no-hostname) host="" ;;
		--output) ;;
		short-iso) short=1 ;;
		esac
	done
	if ((short)); then
		printf '2026-10-18T10:00:00+0200 %s tribe-menus-deploy[1]: recette: v0.1.0 deployed\n' "$host"
	else
		printf '{"level":"WARN","msg":"client IP unknown behind the proxy: X-Forwarded-For missing or unreadable"}\n'
	fi
}
report
`
	cmd := exec.Command("bash", "-c", prelude)
	cmd.Env = append(os.Environ(), "SCRIPT="+script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("report: %v\n%s", err, out)
	}
	if strings.Contains(string(out), "machine-de-l-instance") {
		t.Errorf("report names the host:\n%s", out)
	}
	if !strings.Contains(string(out), "v0.1.0 deployed") || !strings.Contains(string(out), "client IP unknown") {
		t.Errorf("report incomplete:\n%s", out)
	}
}

// The checks of SSH read the whole output of sshd -T: under pipefail, grep -q stopping at
// the first match made sshd fail on SIGPIPE, and the check with it (séance S2). sudo asks
// a password of the administration account unless a NOPASSWD rule or a Defaults
// !authenticate applies to it (D14; review of PR #56 after S2, point 2); a NOPASSWD rule
// for root alone, as cloud-init writes, does not count.
func TestCheckSSH(t *testing.T) {
	script, err := filepath.Abs(filepath.Join("..", "..", "deploy", "bin", "tribe-menus-check"))
	if err != nil {
		t.Fatal(err)
	}
	const sudoPoint = "sudo asks the administration account for a password"
	tests := []struct {
		name   string
		rules  string
		sudoOK bool
	}{
		{"password asked", `Matching Defaults entries for alice on host:\n    env_reset, use_pty\n\nUser alice may run the following commands on host:\n    (ALL : ALL) ALL`, true},
		{"NOPASSWD", `User alice may run the following commands on host:\n    (ALL : ALL) NOPASSWD: ALL`, false},
		{"Defaults !authenticate", `Matching Defaults entries for alice on host:\n    env_reset, !authenticate\n\nUser alice may run the following commands on host:\n    (ALL : ALL) ALL`, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prelude := `
set -euo pipefail
source "$SCRIPT"
sshd() {
	printf '%s\n' 'permitrootlogin no' 'authenticationmethods publickey' 'passwordauthentication no' \
		'kbdinteractiveauthentication no' 'allowusers alice' 'allowtcpforwarding no'
	seq 1 200000 | sed 's/^/padding /'
}
sudo() {
	[[ $1 == -l && $2 == -U && $3 == alice ]] || return 1
	printf "$RULES\n"
}
check_ssh
`
			cmd := exec.Command("bash", "-c", prelude)
			cmd.Env = append(os.Environ(), "SCRIPT="+script, "RULES="+tc.rules)
			out, _ := cmd.CombinedOutput()
			for _, point := range []string{"root may not connect", "keys only", "no password", "no keyboard-interactive",
				"a single account admitted, not root", "no forwarding"} {
				if !strings.Contains(string(out), "ok    "+point+"\n") {
					t.Errorf("%q not ok:\n%s", point, out)
				}
			}
			want := "FAIL  " + sudoPoint + "\n"
			if tc.sudoOK {
				want = "ok    " + sudoPoint + "\n"
			}
			if !strings.Contains(string(out), want) {
				t.Errorf("want %q:\n%s", want, out)
			}
		})
	}
}

// The ports listening outside the loopback are listed, none included: a filter that failed
// on an empty result stopped tribe-menus-check under pipefail (séance S2).
func TestListeningPorts(t *testing.T) {
	script, err := filepath.Abs(filepath.Join("..", "..", "deploy", "bin", "tribe-menus-check"))
	if err != nil {
		t.Fatal(err)
	}
	prelude := `
set -euo pipefail
source "$SCRIPT"
ss() {
	case $2 in
	-t) printf '%s\n' 'LISTEN 0 4096 0.0.0.0:22 0.0.0.0:*' 'LISTEN 0 4096 [::]:443 [::]:*' \
		'LISTEN 0 4096 *:80 *:*' 'LISTEN 0 4096 127.0.0.1:2019 0.0.0.0:*' 'LISTEN 0 4096 127.0.0.53%lo:53 0.0.0.0:*' ;;
	-u) printf '%s\n' 'UNCONN 0 0 127.0.0.1:323 0.0.0.0:*' 'UNCONN 0 0 [::1]:323 [::]:*' ;;
	esac
}
# Assigned, as check_firewall does: set -e applies to an assignment.
tcp=$(listening_ports -t)
udp=$(listening_ports -u)
printf 'tcp=%s\nudp=%s\n' "$tcp" "$udp"
`
	cmd := exec.Command("bash", "-c", prelude)
	cmd.Env = append(os.Environ(), "SCRIPT="+script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("listening_ports: %v\n%s", err, out)
	}
	if want := "tcp=22 80 443\nudp=\n"; string(out) != want {
		t.Errorf("listening_ports:\n%s\nwant:\n%s", out, want)
	}
}
