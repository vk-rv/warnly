package server

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"regexp"
	"strings"
)

// Include SvelteKit's _app directory, which embed excludes without all:.
//
//go:embed all:frontend
var frontendAssets embed.FS

func newFrontendHandler() (http.Handler, error) {
	assets, err := fs.Sub(frontendAssets, "frontend")
	if err != nil {
		return nil, fmt.Errorf("frontend assets: %w", err)
	}
	return serveFrontend(assets), nil
}

func serveFrontend(assets fs.FS) http.Handler {
	files := http.StripPrefix("/", http.FileServer(http.FS(assets)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if strings.HasPrefix(name, "_app/") {
			if info, err := fs.Stat(assets, "dist/"+name); err != nil || info.IsDir() {
				http.NotFound(w, r)
				return
			}
			if strings.HasPrefix(name, "_app/immutable/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			assetRequest := r.Clone(r.Context())
			assetRequest.URL.Path = "/dist/" + name
			files.ServeHTTP(w, assetRequest)
			return
		}
		if !isFrontendRoute(r.URL.Path) {
			http.NotFound(w, r)
			return
		}
		data, err := fs.ReadFile(assets, "dist/index.html")
		if err != nil {
			http.Error(w, "Frontend is not built. Run make frontend-build and rebuild Warnly.", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		if r.Method != http.MethodHead {
			_, _ = w.Write(data)
		}
	})
}

var frontendRoutes = regexp.MustCompile(
	`^/(?:login|oncall|analytics|notready|error|` +
		`system(?:/(?:schema|errors))?|settings(?:/projects/[0-9]+)?|` +
		`alerts(?:/new|/[0-9]+/edit)?|` +
		`projects(?:/new|/[0-9]+(?:/getting-started|/issues/[0-9]+(?:/(?:events|fields|discussions))?)?)?` +
		`)?/?$`,
)

func isFrontendRoute(urlPath string) bool {
	return frontendRoutes.MatchString(urlPath)
}
