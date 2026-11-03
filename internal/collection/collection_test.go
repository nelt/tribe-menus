package collection

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fruits is a collection of strings, cut in pages of two, in the order of the items.
type fruits struct {
	version int64
	items   []string
	// pages counts the pages read.
	pages int
}

func (f *fruits) source() Source[string] {
	return Source[string]{
		Name:    "fruits",
		Version: func(context.Context) (int64, error) { return f.version, nil },
		Page: func(_ context.Context, after string) (int64, Page[string], error) {
			f.pages++
			if after == "bad" {
				return 0, Page[string]{}, ErrBadCursor
			}
			var page Page[string]
			for _, item := range f.items {
				if item <= after {
					continue
				}
				if len(page.Items) == 2 {
					page.Next = page.Items[1]
					break
				}
				page.Items = append(page.Items, item)
			}
			return f.version, page, nil
		},
	}
}

func get(t *testing.T, src Source[string], query string, header map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/fruits"+query, nil)
	for k, v := range header {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	if err := Serve(w, r, src); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	return w
}

func TestServe(t *testing.T) {
	cases := []struct {
		name      string
		query     string
		header    map[string]string
		status    int
		etag      string
		body      string
		pagesRead int
	}{
		{name: "first page", status: 200, etag: `"fruits-7"`, body: `{"items":["apple","banana"],"next":"banana"}`, pagesRead: 1},
		{name: "next page", query: "?after=banana", status: 200, etag: `"fruits-7"`, body: `{"items":["cherry"]}`, pagesRead: 1},
		{name: "past the last item", query: "?after=zzz", status: 200, etag: `"fruits-7"`, body: `{"items":[]}`, pagesRead: 1},
		{
			name: "first page unchanged", header: map[string]string{"If-None-Match": `"fruits-7"`},
			status: 304, etag: `"fruits-7"`, body: "", pagesRead: 0,
		},
		{
			name: "first page unchanged, among other tags", header: map[string]string{"If-None-Match": `"fruits-6", W/"fruits-7"`},
			status: 304, etag: `"fruits-7"`, body: "", pagesRead: 0,
		},
		{
			name: "first page changed", header: map[string]string{"If-None-Match": `"fruits-6"`},
			status: 200, etag: `"fruits-7"`, body: `{"items":["apple","banana"],"next":"banana"}`, pagesRead: 1,
		},
		{
			name: "tag of another collection", header: map[string]string{"If-None-Match": `"vegetables-7"`},
			status: 200, etag: `"fruits-7"`, body: `{"items":["apple","banana"],"next":"banana"}`, pagesRead: 1,
		},
		{
			name: "next page, same version", query: "?after=banana", header: map[string]string{"If-Match": `"fruits-7"`},
			status: 200, etag: `"fruits-7"`, body: `{"items":["cherry"]}`, pagesRead: 1,
		},
		{
			name: "next page, version changed", query: "?after=banana", header: map[string]string{"If-Match": `"fruits-6"`},
			status: 412, etag: `"fruits-7"`, body: `{"error":"collection_changed"}`, pagesRead: 1,
		},
		{
			name: "weak tag does not match If-Match", query: "?after=banana", header: map[string]string{"If-Match": `W/"fruits-7"`},
			status: 412, etag: `"fruits-7"`, body: `{"error":"collection_changed"}`, pagesRead: 1,
		},
		{name: "bad cursor", query: "?after=bad", status: 400, body: `{"error":"bad_request"}`, pagesRead: 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := &fruits{version: 7, items: []string{"apple", "banana", "cherry"}}
			w := get(t, f.source(), c.query, c.header)
			if w.Code != c.status {
				t.Errorf("status %d, want %d", w.Code, c.status)
			}
			if got := w.Header().Get("ETag"); got != c.etag {
				t.Errorf("ETag %q, want %q", got, c.etag)
			}
			if got := strings.TrimSpace(w.Body.String()); got != c.body {
				t.Errorf("body %s, want %s", got, c.body)
			}
			if f.pages != c.pagesRead {
				t.Errorf("%d pages read, want %d", f.pages, c.pagesRead)
			}
			if c.body != "" && w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("Content-Type %q", w.Header().Get("Content-Type"))
			}
		})
	}
}

func TestServeError(t *testing.T) {
	failure := errors.New("disk on fire")
	src := Source[string]{
		Name:    "fruits",
		Version: func(context.Context) (int64, error) { return 0, failure },
		Page:    func(context.Context, string) (int64, Page[string], error) { return 0, Page[string]{}, failure },
	}
	for _, header := range []map[string]string{nil, {"If-None-Match": `"fruits-1"`}} {
		r := httptest.NewRequest(http.MethodGet, "/fruits", nil)
		for k, v := range header {
			r.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		if err := Serve(w, r, src); !errors.Is(err, failure) {
			t.Errorf("error %v, want %v", err, failure)
		}
		if w.Body.Len() != 0 || len(w.Header()) != 0 {
			t.Errorf("response written: %d %v %s", w.Code, w.Header(), w.Body)
		}
	}
}

func TestETag(t *testing.T) {
	if got := ETag("dishes", 42); got != `"dishes-42"` {
		t.Errorf("ETag %s", got)
	}
}

func TestVersionsJSON(t *testing.T) {
	data, err := json.Marshal(Versions{"dishes": 43, "ingredients": 18})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"dishes":43,"ingredients":18}` {
		t.Errorf("versions %s", data)
	}
}
