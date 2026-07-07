package api

import (
	"io"
	"io/fs"
	"net/http"
	"strings"
)

// serveSPA serves static files; missing paths fall back to index.html for client routing.
func serveSPA(w http.ResponseWriter, r *http.Request, static fs.FS) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path != "" {
		if f, err := static.Open(path); err == nil {
			_ = f.Close()
			http.FileServer(http.FS(static)).ServeHTTP(w, r)
			return
		}
	}
	index, err := static.Open("index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer index.Close()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.Copy(w, index)
}
