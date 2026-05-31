// Package web embeds the HTML templates and static assets so the server ships
// as a single self-contained binary.
package web

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed static
var staticFS embed.FS

// Templates parses the embedded HTML templates.
func Templates() (*template.Template, error) {
	return template.ParseFS(templatesFS, "templates/*.html")
}

// StaticHandler returns an http.Handler serving the embedded static assets
// (rooted so that /static/css/pixel.css maps to static/css/pixel.css).
func StaticHandler() http.Handler {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(sub))
}
