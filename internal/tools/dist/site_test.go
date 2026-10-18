package main

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The site of the repository goes into the archive: its names are accepted by
// tribe-menus-deploy, each page carries both markers and no other, and its copies of the
// fonts and of the identity stay those of web/src/fonts and docs/design/identite.
func TestSiteOfTheRepository(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	site := os.DirFS(filepath.Join(root, "site"))
	pages := 0
	err := fs.WalkDir(site, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == "." {
			return nil
		}
		if !namePattern.MatchString("site/" + name) {
			t.Errorf("site/%s: name refused by tribe-menus-deploy", name)
		}
		if d.IsDir() || !strings.HasSuffix(name, ".html") {
			return nil
		}
		pages++
		data, err := fs.ReadFile(site, name)
		if err != nil {
			return err
		}
		for _, m := range []string{"{{VERSION}}", "{{SOURCE_URL}}"} {
			if !bytes.Contains(data, []byte(m)) {
				t.Errorf("site/%s: no %s", name, m)
			}
		}
		rest := strings.NewReplacer("{{VERSION}}", "", "{{SOURCE_URL}}", "").Replace(string(data))
		if strings.Contains(rest, "{{") {
			t.Errorf("site/%s: unknown marker", name)
		}
		for _, banned := range []string{"<script", " style=", "fonts.googleapis", "http://"} {
			if strings.Contains(rest, banned) {
				t.Errorf("site/%s: %q, which the CSP of the site refuses or the site avoids", name, banned)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if pages != 3 {
		t.Errorf("%d pages, want 3: presentation, legal notice, privacy", pages)
	}

	copies := map[string]string{
		"site/fonts/bricolage-grotesque.woff2":  "web/src/fonts/BricolageGrotesque[opsz,wdth,wght].woff2",
		"site/fonts/figtree.woff2":              "web/src/fonts/Figtree[wght].woff2",
		"site/fonts/OFL-BricolageGrotesque.txt": "web/src/fonts/OFL-BricolageGrotesque.txt",
		"site/fonts/OFL-Figtree.txt":            "web/src/fonts/OFL-Figtree.txt",
		"site/images/symbole.svg":               "docs/design/identite/symbole.svg",
		"site/images/symbole-reduit.svg":        "docs/design/identite/symbole-reduit.svg",
		"site/images/logotype.svg":              "docs/design/identite/logotype.svg",
		"site/images/partage.png":               "docs/design/identite/partage.png",
	}
	for copy, source := range copies {
		a, err := os.ReadFile(filepath.Join(root, copy))
		if err != nil {
			t.Error(err)
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, source))
		if err != nil {
			t.Error(err)
			continue
		}
		if !bytes.Equal(a, b) {
			t.Errorf("%s differs from %s: copy it again", copy, source)
		}
	}
}
