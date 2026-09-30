package main

import (
	"bytes"
	"reflect"
	"testing"
	"testing/fstest"
)

func TestCheck(t *testing.T) {
	cases := []struct {
		name    string
		file    string
		content string
		want    []finding
	}{
		{name: "unsafeHTML import", file: "a.ts", content: `import { unsafeHTML } from "lit/directives/unsafe-html.js";`, want: []finding{{"a.ts", 1, "unsafeHTML"}}},
		{name: "unsafeSVG", file: "a.ts", content: "html`${unsafeSVG(icon)}`", want: []finding{{"a.ts", 1, "unsafeSVG"}}},
		{name: "innerHTML on second line", file: "a.js", content: "const a = 1;\nel.innerHTML = text;", want: []finding{{"a.js", 2, "innerHTML"}}},
		{name: "outerHTML", file: "a.mjs", content: "el.outerHTML = text;", want: []finding{{"a.mjs", 1, "outerHTML"}}},
		{name: "insertAdjacentHTML", file: "a.ts", content: `el.insertAdjacentHTML("beforeend", text);`, want: []finding{{"a.ts", 1, "insertAdjacentHTML"}}},
		{name: "document.write", file: "a.html", content: `<script>document.write("x")</script>`, want: []finding{{"a.html", 1, "document.write"}}},
		{name: "document.writeln", file: "a.ts", content: `document.writeln("x");`, want: []finding{{"a.ts", 1, "document.writeln"}}},
		{name: "several on one line", file: "a.ts", content: "a.innerHTML = b.outerHTML;", want: []finding{{"a.ts", 1, "innerHTML"}, {"a.ts", 1, "outerHTML"}}},
		{name: "nested directory", file: "screens/list.ts", content: "el.innerHTML = x;", want: []finding{{"screens/list.ts", 1, "innerHTML"}}},
		{name: "longer identifier", file: "a.ts", content: "const myinnerHTML = el.innerHTMLish;", want: nil},
		{name: "different case", file: "a.ts", content: "const sanitizedInnerHTML = 1;", want: nil},
		{name: "similar method", file: "a.ts", content: "stream.writer(); document.writeAll();", want: nil},
		{name: "escaped rendering", file: "a.ts", content: "html`<p>${text}</p>`", want: nil},
		{name: "css is ignored", file: "a.css", content: "/* innerHTML */", want: nil},
		{name: "markdown is ignored", file: "README.md", content: "never use innerHTML", want: nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fsys := fstest.MapFS{tc.file: {Data: []byte(tc.content)}}
			got, err := check(fsys)
			if err != nil {
				t.Fatalf("check: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("check = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRun(t *testing.T) {
	cases := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
	}{
		{name: "clean directory", args: []string{"testdata/clean"}, wantCode: 0},
		{name: "forbidden construct", args: []string{"testdata/unsafe"}, wantCode: 1, wantStdout: "testdata/unsafe/app.ts:2: innerHTML is forbidden (ADR 0004)\n"},
		{name: "missing directory", args: []string{"testdata/missing"}, wantCode: 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := run(tc.args, &stdout, &stderr); code != tc.wantCode {
				t.Errorf("code = %d, want %d (stderr: %s)", code, tc.wantCode, stderr.String())
			}
			if stdout.String() != tc.wantStdout {
				t.Errorf("stdout = %q, want %q", stdout.String(), tc.wantStdout)
			}
		})
	}
}
