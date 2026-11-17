// Package web contains the embedded browser assets.
package web

import (
	"embed"
	"io/fs"
)

//go:embed static
var staticFiles embed.FS

// StaticFS returns the embedded static assets rooted at the static directory.
func StaticFS() fs.FS {
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		// The embedded directory is known at compile time.
		panic("web: resolve static assets: " + err.Error())
	}
	return sub
}
