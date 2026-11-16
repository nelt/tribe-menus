package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"path/filepath"
	"slices"
	"time"
)

// serve serves the volume of the videos on a loopback address until ctx is done, and lists the
// control pages. A browser puts a video at the instant of a step (#t=) only when the server
// answers the request of a part of the file (206), which http.FileServer does.
func serve(ctx context.Context, videos, addr string, stdout io.Writer) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("-addr: %w", err)
	}
	if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
		return errors.New("-addr: a loopback address only, the videos show the tribe of the staging instance")
	}
	pages, err := controlPages(videos)
	if err != nil {
		return err
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	fmt.Fprintf(stdout, "Volume of the videos served on http://localhost:%s/ (forward the port in VS Code), Ctrl-C stops.\n", port)
	for _, p := range pages {
		fmt.Fprintf(stdout, "  http://localhost:%s/%s\n", port, filepath.ToSlash(p))
	}
	srv := &http.Server{Handler: pagesHandler(videos), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		_ = srv.Close() // ends Serve, whose error says so
	}()
	if err := srv.Serve(l); !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

// pagesHandler serves the files of the volume, the videos with their type.
func pagesHandler(videos string) http.Handler {
	_ = mime.AddExtensionType(".webm", "video/webm") // a valid type: never an error
	return http.FileServer(http.Dir(videos))
}

// controlPages returns the control pages of the volume, vX.Y.Z/<time>/controle.html, oldest first.
func controlPages(videos string) ([]string, error) {
	found, err := filepath.Glob(filepath.Join(videos, "*", "*", "controle.html"))
	if err != nil {
		return nil, fmt.Errorf("control pages: %w", err)
	}
	pages := make([]string, 0, len(found))
	for _, f := range found {
		rel, err := filepath.Rel(videos, f)
		if err != nil {
			return nil, fmt.Errorf("control pages: %w", err)
		}
		pages = append(pages, rel)
	}
	slices.Sort(pages)
	return pages, nil
}
