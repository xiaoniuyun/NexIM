// Package webassets embeds the built Admin frontend into the Go binary.
package webassets

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var dist embed.FS

var admin fs.FS

func init() {
	var err error
	admin, err = fs.Sub(dist, "dist/admin")
	if err != nil {
		panic(err)
	}
}

// FS returns the embedded Admin filesystem.
func FS(app string) fs.FS { return admin }

// Exists reports whether a non-directory file exists at name in the admin fs.
func Exists(app, name string) bool {
	name = strings.TrimPrefix(path.Clean("/"+name), "/")
	if name == "" {
		return false
	}
	st, err := fs.Stat(FS(app), name)
	return err == nil && !st.IsDir()
}

// ReadFile reads an embedded file from the admin fs.
func ReadFile(app, name string) ([]byte, error) {
	name = strings.TrimPrefix(path.Clean("/"+name), "/")
	return fs.ReadFile(FS(app), name)
}

// Has reports whether the embedded Admin assets were compiled in.
func Has() bool {
	_, err := fs.Stat(dist, "dist/admin/index.html")
	return err == nil
}

// FileServer returns an http.Handler serving embedded static files for an app,
// with SPA fallback to index.html for unknown non-asset paths.
func FileServer(app string) http.Handler {
	fsys := FS(app)
	prefix := "/" + app + "/"
	fileServer := http.FileServer(http.FS(fsys))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, prefix)
		name = strings.TrimPrefix(path.Clean("/"+name), "/")
		if name != "" && name != "index.html" {
			if st, err := fs.Stat(fsys, name); err == nil && !st.IsDir() {
				if strings.HasPrefix(name, "assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				r2 := r.Clone(r.Context())
				r2.URL.Path = "/" + name
				fileServer.ServeHTTP(w, r2)
				return
			}
		}
		data, err := fs.ReadFile(fsys, "index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	})
}
