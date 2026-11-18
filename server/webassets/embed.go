// Package webassets embeds the built H5 and Admin frontends into the Go binary.
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

var sub struct {
	h5    fs.FS
	admin fs.FS
}

func init() {
	h5, err := fs.Sub(dist, "dist/h5")
	if err != nil {
		panic(err)
	}
	admin, err := fs.Sub(dist, "dist/admin")
	if err != nil {
		panic(err)
	}
	sub.h5 = h5
	sub.admin = admin
}

// FS returns the embedded filesystem for the given app ("h5" or "admin").
func FS(app string) fs.FS {
	if app == "h5" {
		return sub.h5
	}
	return sub.admin
}

// Exists reports whether a non-directory file exists at name in the app's fs.
func Exists(app, name string) bool {
	name = strings.TrimPrefix(path.Clean("/"+name), "/")
	if name == "" {
		return false
	}
	st, err := fs.Stat(FS(app), name)
	return err == nil && !st.IsDir()
}

// ReadFile reads an embedded file from the app's fs.
func ReadFile(app, name string) ([]byte, error) {
	name = strings.TrimPrefix(path.Clean("/"+name), "/")
	return fs.ReadFile(FS(app), name)
}

// Has reports whether the embedded assets were compiled in.
func Has() bool {
	_, err := fs.Stat(dist, "dist/h5/index.html")
	return err == nil
}

// FileServer returns an http.Handler serving embedded static files for an app,
// with SPA fallback to index.html for unknown non-asset paths.
func FileServer(app string) http.Handler {
	fsys := FS(app)
	prefix := "/" + app + "/"
	fileServer := http.FileServer(http.FS(fsys))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 去掉 /h5/ 或 /admin/ 前缀，得到 fs 内的相对路径
		name := strings.TrimPrefix(r.URL.Path, prefix)
		name = strings.TrimPrefix(path.Clean("/"+name), "/")
		if name != "" && name != "index.html" {
			if st, err := fs.Stat(fsys, name); err == nil && !st.IsDir() {
				// 构建产物文件名带 hash，可长缓存
				if strings.HasPrefix(name, "assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				r2 := r.Clone(r.Context())
				r2.URL.Path = "/" + name
				fileServer.ServeHTTP(w, r2)
				return
			}
		}
		// SPA fallback
		data, err := fs.ReadFile(fsys, "index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// index.html 不缓存，确保前端升级后立即加载新的资源清单
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	})
}
