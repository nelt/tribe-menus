// Package web embeds the built front end (web/dist) into the binary.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Dist returns the built front end, rooted at web/dist.
func Dist() (fs.FS, error) {
	return fs.Sub(dist, "dist")
}
