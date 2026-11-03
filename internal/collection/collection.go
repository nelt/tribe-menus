// Package collection serves the collections of the API (ADR 0029): bounded pages with a
// cursor, an ETag carrying the version of the collection, conditional requests, and the
// versions joined to the response of a write. Every collection route goes through it.
//
// The versions themselves are kept by the database of the tribe, increased by triggers on
// the tables of each collection (internal/storage).
package collection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Versions are the versions of collections, by name: in the response of a write, those of
// the collections it touched, after the write.
type Versions map[string]int64

// Page is a page of a collection, in the order of the identifiers. Next is the cursor of the
// next page, empty on the last one.
type Page[T any] struct {
	Items []T
	Next  string
}

// ErrBadCursor is returned by a source for a cursor it cannot read.
var ErrBadCursor = errors.New("bad cursor")

// Source reads a collection for Serve.
type Source[T any] struct {
	// Name is the name of the collection, in its ETag.
	Name string
	// Version reads the current version of the collection.
	Version func(ctx context.Context) (int64, error)
	// Page reads the version and the page after the cursor ("" for the first page) in one
	// read transaction, so that a page never carries the ETag of another version.
	Page func(ctx context.Context, after string) (int64, Page[T], error)
}

// ETag is the entity tag of a version of a collection: "<collection>-<version>".
func ETag(name string, version int64) string {
	return fmt.Sprintf(`"%s-%d"`, name, version)
}

// Error codes of the API written by Serve, in the "error" field.
const (
	errCollectionChanged = "collection_changed"
	errBadRequest        = "bad_request"
)

type pageResponse[T any] struct {
	Items []T    `json:"items"`
	Next  string `json:"next,omitempty"`
}

// Serve answers a GET of the collection, the cursor in the parameter after:
//   - the first page with If-None-Match: 304 without body if the version has not changed;
//   - a following page with If-Match: 412 collection_changed if the version has changed
//     during the loading;
//   - otherwise the page, with the ETag of its version.
//
// An error of the source is returned before anything is written, for the caller to answer.
func Serve[T any](w http.ResponseWriter, r *http.Request, src Source[T]) error {
	after := r.URL.Query().Get("after")
	if after == "" {
		if tags := r.Header.Get("If-None-Match"); tags != "" {
			version, err := src.Version(r.Context())
			if err != nil {
				return fmt.Errorf("collection %s: %w", src.Name, err)
			}
			etag := ETag(src.Name, version)
			if matchWeak(tags, etag) {
				w.Header().Set("ETag", etag)
				w.WriteHeader(http.StatusNotModified)
				return nil
			}
		}
	}
	version, page, err := src.Page(r.Context(), after)
	if errors.Is(err, ErrBadCursor) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": errBadRequest})
		return nil
	}
	if err != nil {
		return fmt.Errorf("collection %s: %w", src.Name, err)
	}
	etag := ETag(src.Name, version)
	w.Header().Set("ETag", etag)
	if tags := r.Header.Get("If-Match"); after != "" && tags != "" && !matchStrong(tags, etag) {
		writeJSON(w, http.StatusPreconditionFailed, map[string]string{"error": errCollectionChanged})
		return nil
	}
	items := page.Items
	if items == nil {
		items = []T{}
	}
	writeJSON(w, http.StatusOK, pageResponse[T]{Items: items, Next: page.Next})
	return nil
}

// matchWeak reports whether the list of tags of If-None-Match holds the tag, by the weak
// comparison of RFC 9110.
func matchWeak(tags, etag string) bool {
	for t := range strings.SplitSeq(tags, ",") {
		t = strings.TrimSpace(t)
		if t == "*" || strings.TrimPrefix(t, "W/") == etag {
			return true
		}
	}
	return false
}

// matchStrong reports whether the list of tags of If-Match holds the tag, by the strong
// comparison of RFC 9110: a weak tag never matches.
func matchStrong(tags, etag string) bool {
	for t := range strings.SplitSeq(tags, ",") {
		t = strings.TrimSpace(t)
		if t == "*" || t == etag {
			return true
		}
	}
	return false
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
