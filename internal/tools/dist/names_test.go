package main

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// deployScript is the script whose check_archive refuses the names of an archive.
var deployScript = filepath.Join("..", "..", "..", "deploy", "bin", "tribe-menus-deploy")

// checkArchive runs check_archive of tribe-menus-deploy on an archive holding the top
// directory and one file, top/name, and tells whether the script accepts it.
func checkArchive(t *testing.T, name string) bool {
	t.Helper()
	const top = "tribe-menus-v1.2.3-linux-amd64"
	archive := filepath.Join(t.TempDir(), top+".tar.gz")
	f, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	headers := []tar.Header{
		{Name: top + "/", Typeflag: tar.TypeDir, Mode: 0o755},
		{Name: top + "/" + name, Typeflag: tar.TypeReg, Mode: 0o644},
	}
	for _, h := range headers {
		if err := tw.WriteHeader(&h); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range []interface{ Close() error }{tw, gz, f} {
		if err := c.Close(); err != nil {
			t.Fatal(err)
		}
	}
	script, err := filepath.Abs(deployScript)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", "-c", `source "$1"; check_archive "$2" "$3"`, "_", script, archive, top)
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return true
	case !errors.As(err, &exit):
		t.Fatalf("bash: %v", err)
	case !strings.Contains(string(out), "archive: unexpected entry name"):
		t.Fatalf("check_archive refused for another reason: %s", out)
	}
	return false
}

// TestNameAcceptedAgreesWithTribeMenusDeploy makes the names this tool lets into an
// archive and those that tribe-menus-deploy accepts the same, the rule being written in
// both: a change on one side only fails here, not at the deployment.
func TestNameAcceptedAgreesWithTribeMenusDeploy(t *testing.T) {
	names := []string{
		// Names of the archive.
		"LICENSE",
		"site/index.html",
		"site/fonts/figtree-latin.woff2",
		"deploy/bin/tribe-menus-deploy",
		"deploy/systemd/tribe-menus@.service",
		"site/a+b_c-d.e",
		"site/.well-known/security.txt",
		"site/..hidden",
		// Every character of the rule, and some around it.
		"site/ABCXYZabcxyz0189._@+-",
		"ABCXYZabcxyz0189._@+-",
		"ABCXYZabcxyz0189._@+-/a",
		"a,b/c",
		"site/Figtree[wght].woff2",
		"site/a,b",
		"site/a b",
		"site/a:b",
		"site/a~b",
		"site/a*b",
		"site/a$b",
		"site/a'b",
		`site/a\b`,
		"site/é",
		// Segments.
		"site//a",
		"site/./a",
		"site/../a",
		"site/a/..",
		"..",
		".",
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			got, want := nameAccepted(name), checkArchive(t, name)
			if got != want {
				t.Errorf("nameAccepted(%q) = %v, tribe-menus-deploy accepts: %v", name, got, want)
			}
		})
	}
}
