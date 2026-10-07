// Command dist writes the archive of a version (ADR 0012, point 3): the binary built by
// make build, the public site, the deployment files and the licence, under a directory named
// after the archive, with its SHA-256 in a file next to it.
//
// Usage: go run ./internal/tools/dist -version <version> [-out dist]
//
// It runs from the root of the repository, after make build. Entries are sorted and their
// dates, owners and permissions fixed: two runs on the same commit give the same archive.
package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

const (
	// binary is where make build writes the binary for linux/amd64.
	binary = "bin/tribe-menus"
	// platform is the one of the server (ADR 0007).
	platform = "linux-amd64"
)

var (
	// dirs are copied whole, files one by one, under the same name.
	dirs  = []string{"site", "deploy"}
	files = []string{"LICENSE"}

	// versionPattern accepts a release (v1.2.3), a pull request build (pr-12) or dev.
	versionPattern = regexp.MustCompile(`^[0-9A-Za-z][0-9A-Za-z.-]*$`)
	// modTime is the date of every entry. A fixed date keeps the archive the same from one
	// run to the next; the commit is given by tribe-menus version.
	modTime = time.Unix(0, 0).UTC()
)

// entry is a file or a directory of the archive.
type entry struct {
	// name in the archive, below the top directory.
	name string
	// source in the repository, empty for a directory.
	source string
	mode   int64
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("dist", flag.ContinueOnError)
	flags.SetOutput(stderr)
	version := flags.String("version", "", "version of the archive (v1.2.3, pr-12, dev)")
	out := flags.String("out", "dist", "directory of the archive")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 || !versionPattern.MatchString(*version) {
		fmt.Fprintln(stderr, "usage: dist -version <version> [-out dist]")
		return 2
	}
	name, err := write(os.DirFS("."), *out, *version)
	if err != nil {
		fmt.Fprintf(stderr, "dist: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, name)
	return 0
}

// write writes the archive and its SHA-256 into dir, and returns the path of the archive.
func write(fsys fs.FS, dir, version string) (string, error) {
	base := "tribe-menus-" + version + "-" + platform
	entries, err := collect(fsys)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}
	archive := filepath.Join(dir, base+".tar.gz")
	f, err := os.Create(archive)
	if err != nil {
		return "", fmt.Errorf("create archive: %w", err)
	}
	hash := sha256.New()
	if err := pack(io.MultiWriter(f, hash), fsys, base, entries); err != nil {
		_ = f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("write archive: %w", err)
	}
	// The format of sha256sum, so that sha256sum -c checks the archive.
	sum := hex.EncodeToString(hash.Sum(nil)) + "  " + base + ".tar.gz\n"
	if err := os.WriteFile(archive+".sha256", []byte(sum), 0o644); err != nil {
		return "", fmt.Errorf("write SHA-256: %w", err)
	}
	return archive, nil
}

// collect lists the entries of the archive, sorted by name. A file keeps the execute
// permission it has in the repository; anything but a file or a directory is refused.
func collect(fsys fs.FS) ([]entry, error) {
	info, err := fs.Stat(fsys, binary)
	if err != nil {
		return nil, fmt.Errorf("binary not built, run make build: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a file", binary)
	}
	entries := []entry{{name: path.Base(binary), source: binary, mode: 0o755}}
	for _, file := range files {
		info, err := fs.Stat(fsys, file)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", file, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("%s is not a file", file)
		}
		entries = append(entries, entry{name: file, source: file, mode: fileMode(info)})
	}
	for _, dir := range dirs {
		err := fs.WalkDir(fsys, dir, func(name string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			switch {
			case d.IsDir():
				entries = append(entries, entry{name: name, mode: 0o755})
			case d.Type().IsRegular():
				info, err := d.Info()
				if err != nil {
					return err
				}
				entries = append(entries, entry{name: name, source: name, mode: fileMode(info)})
			default:
				return fmt.Errorf("%s is not a file", name)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", dir, err)
		}
	}
	slices.SortFunc(entries, func(a, b entry) int { return strings.Compare(a.name, b.name) })
	return entries, nil
}

// fileMode is 0755 for a file the repository marks executable, 0644 otherwise.
func fileMode(info fs.FileInfo) int64 {
	if info.Mode().Perm()&0o111 != 0 {
		return 0o755
	}
	return 0o644
}

// pack writes the entries as a gzip-compressed tar, under the directory top.
func pack(w io.Writer, fsys fs.FS, top string, entries []entry) error {
	gz, err := gzip.NewWriterLevel(w, gzip.BestCompression)
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	tw := tar.NewWriter(gz)
	if err := writeEntry(tw, fsys, top, entry{mode: 0o755}); err != nil {
		return err
	}
	for _, e := range entries {
		if err := writeEntry(tw, fsys, top, e); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("tar: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	return nil
}

// writeEntry writes one entry, owned by root, dated modTime.
func writeEntry(tw *tar.Writer, fsys fs.FS, top string, e entry) error {
	header := &tar.Header{
		Name:    path.Join(top, e.name),
		Mode:    e.mode,
		ModTime: modTime,
		Format:  tar.FormatUSTAR,
	}
	if e.source == "" {
		header.Typeflag = tar.TypeDir
		header.Name += "/"
		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("tar %s: %w", header.Name, err)
		}
		return nil
	}
	data, err := fs.ReadFile(fsys, e.source)
	if err != nil {
		return fmt.Errorf("read %s: %w", e.source, err)
	}
	header.Typeflag = tar.TypeReg
	header.Size = int64(len(data))
	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("tar %s: %w", header.Name, err)
	}
	if _, err := tw.Write(data); err != nil {
		return fmt.Errorf("tar %s: %w", header.Name, err)
	}
	return nil
}
