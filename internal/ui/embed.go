package ui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist/*
var distFS embed.FS

// Handler returns an http.Handler that serves the embedded React SPA (web/dist).
// Unknown paths fall back to index.html so the SPA router can handle client-side routes.
func Handler() http.Handler {
	sub, _ := fs.Sub(distFS, "dist")
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "" || path == "/" {
			path = "/index.html"
		}
		// Path for fs: strip leading slash
		fsPath := path
		if len(fsPath) > 0 && fsPath[0] == '/' {
			fsPath = fsPath[1:]
		}
		f, err := sub.Open(fsPath)
		if err != nil {
			// SPA fallback: serve index.html with 200 so client-side routing works (no redirect)
			http.ServeFileFS(w, r, sub, "index.html")
			return
		}
		_ = f.Close()
		fileServer.ServeHTTP(w, r)
	})
}
