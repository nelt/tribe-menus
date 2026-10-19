package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"
)

var (
	// sources are the addresses an HTML page loads: src= of any element, href= of a link
	// element (style sheet, icon).
	sources = []*regexp.Regexp{
		regexp.MustCompile(`\ssrc="([^"]*)"`),
		regexp.MustCompile(`<link\b[^>]*\shref="([^"]*)"`),
	}
	// cssURL is an address a style sheet loads.
	cssURL = regexp.MustCompile(`url\(\s*['"]?([^'")]*)`)
	// bannedHTML is what the CSP of the site refuses, or what the site goes without.
	bannedHTML = []string{"<script", "<style", " style=", "fonts.googleapis"}
)

// local succeeds for an address of the site itself: an absolute path, or a fragment.
func local(address string) bool {
	return (strings.HasPrefix(address, "/") && !strings.HasPrefix(address, "//")) || strings.HasPrefix(address, "#")
}

// siteProblems lists what in a site would load a third-party resource, run a script, or
// be refused by its CSP (style-src, img-src and font-src 'self', no inline style), and the
// names and markers that the archive or tribe-menus-deploy would refuse.
func siteProblems(site fs.FS) ([]string, error) {
	var problems []string
	err := fs.WalkDir(site, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || name == "." {
			return err
		}
		if !namePattern.MatchString("site/" + name) {
			problems = append(problems, name+": name refused by tribe-menus-deploy")
		}
		if d.IsDir() {
			return nil
		}
		data, err := fs.ReadFile(site, name)
		if err != nil {
			return err
		}
		text := string(data)
		switch {
		case strings.HasSuffix(name, ".html"):
			for _, m := range []string{"{{VERSION}}", "{{SOURCE_URL}}"} {
				if !strings.Contains(text, m) {
					problems = append(problems, fmt.Sprintf("%s: no %s", name, m))
				}
			}
			text = strings.NewReplacer("{{VERSION}}", "", "{{SOURCE_URL}}", "").Replace(text)
			if strings.Contains(text, "{{") {
				problems = append(problems, name+": unknown marker")
			}
			for _, banned := range bannedHTML {
				if strings.Contains(text, banned) {
					problems = append(problems, fmt.Sprintf("%s: %q", name, banned))
				}
			}
			for _, re := range sources {
				for _, m := range re.FindAllStringSubmatch(text, -1) {
					if !local(m[1]) {
						problems = append(problems, fmt.Sprintf("%s: loads %q", name, m[1]))
					}
				}
			}
		case strings.HasSuffix(name, ".css"):
			if strings.Contains(text, "@import") {
				problems = append(problems, name+": @import")
			}
			for _, m := range cssURL.FindAllStringSubmatch(text, -1) {
				if !local(m[1]) {
					problems = append(problems, fmt.Sprintf("%s: loads %q", name, m[1]))
				}
			}
		}
		return nil
	})
	return problems, err
}

// Each kind of problem is seen (review of PR #57, point 2).
func TestSiteProblems(t *testing.T) {
	page := `<link rel="stylesheet" href="/site.css"><img src="/images/symbole.svg" alt=""><a href="https://github.com/nelt/tribe-menus">x</a><a href="{{SOURCE_URL}}">{{VERSION}}</a>`
	tests := []struct {
		name, file, content, want string
	}{
		{"site of reference", "index.html", page, ""},
		{"third-party image", "index.html", page + `<img src="https://example.org/a.png">`, `loads "https://example.org/a.png"`},
		{"protocol-relative image", "index.html", page + `<img src="//example.org/a.png">`, `loads "//example.org/a.png"`},
		{"third-party style sheet", "index.html", page + `<link rel="stylesheet" href="https://example.org/a.css">`, `loads "https://example.org/a.css"`},
		{"style element", "index.html", page + `<style>p{}</style>`, `"<style"`},
		{"inline style", "index.html", page + `<p style="color:red">`, `" style="`},
		{"script", "index.html", page + `<script src="/a.js"></script>`, `"<script"`},
		{"marker missing", "index.html", `<a href="{{SOURCE_URL}}">`, "no {{VERSION}}"},
		{"unknown marker", "index.html", page + `{{DATE}}`, "unknown marker"},
		{"import in the style sheet", "site.css", `@import "/b.css";`, "@import"},
		{"third-party font", "site.css", `@font-face { src: url("https://example.org/f.woff2"); }`, `loads "https://example.org/f.woff2"`},
		{"unquoted third-party image", "site.css", `p { background: url(https://example.org/a.png); }`, `loads "https://example.org/a.png"`},
		{"name refused", "fonts/Figtree[wght].woff2", "", "name refused"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			site := fstest.MapFS{
				"index.html": {Data: []byte(page)},
				"site.css":   {Data: []byte(`@font-face { src: url("/fonts/figtree.woff2"); }`)},
			}
			site[tc.file] = &fstest.MapFile{Data: []byte(tc.content)}
			problems, err := siteProblems(site)
			if err != nil {
				t.Fatal(err)
			}
			got := strings.Join(problems, "\n")
			if tc.want == "" && got != "" {
				t.Errorf("problems:\n%s", got)
			}
			if tc.want != "" && !strings.Contains(got, tc.want) {
				t.Errorf("problems %q, want %q", got, tc.want)
			}
		})
	}
}

// The site of the repository goes into the archive as it is: three pages, nothing
// refused, and its copies of the fonts and of the identity stay those of web/src/fonts
// and docs/design/identite.
func TestSiteOfTheRepository(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	site := os.DirFS(filepath.Join(root, "site"))
	problems, err := siteProblems(site)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range problems {
		t.Errorf("site/%s", p)
	}
	pages, err := fs.Glob(site, "*.html")
	if err != nil {
		t.Fatal(err)
	}
	sub, err := fs.Glob(site, "*/index.html")
	if err != nil {
		t.Fatal(err)
	}
	if n := len(pages) + len(sub); n != 3 {
		t.Errorf("%d pages, want 3: presentation, legal notice, privacy", n)
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
