package httpapi

import (
	"bytes"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
)

type spaHandler struct {
	fsys fs.FS
}

func newSPAHandler(fsys fs.FS) http.Handler {
	if fsys == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "frontend assets are unavailable", http.StatusServiceUnavailable)
		})
	}

	return &spaHandler{fsys: fsys}
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.NotFound(w, r)
		return
	}

	requestedPath := cleanFrontendPath(r.URL.Path)
	switch {
	case requestedPath == "" || requestedPath == ".":
		h.serveFrontendFile(w, r, "index.html")
	case h.fileExists(requestedPath):
		h.serveFrontendFile(w, r, requestedPath)
	case looksLikeStaticAsset(requestedPath):
		http.NotFound(w, r)
	default:
		h.serveFrontendFile(w, r, "index.html")
	}
}

func (h *spaHandler) serveFrontendFile(w http.ResponseWriter, r *http.Request, name string) {
	if clientAcceptsGzip(r.Header.Get("Accept-Encoding")) && h.fileExists(name+".gz") {
		setVary(w.Header(), "Accept-Encoding")
		w.Header().Set("Content-Encoding", "gzip")
		serveFSContent(w, r, h.fsys, name+".gz", name)
		return
	}

	serveFSContent(w, r, h.fsys, name, name)
}

func (h *spaHandler) fileExists(name string) bool {
	info, err := fs.Stat(h.fsys, name)
	return err == nil && !info.IsDir()
}

func serveFSContent(w http.ResponseWriter, r *http.Request, fsys fs.FS, storedName string, originalName string) {
	data, err := fs.ReadFile(fsys, storedName)
	if err != nil {
		status := http.StatusNotFound
		if originalName == "index.html" {
			status = http.StatusServiceUnavailable
		}
		http.Error(w, http.StatusText(status), status)
		return
	}

	if contentType := mime.TypeByExtension(path.Ext(originalName)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}

	http.ServeContent(w, r, originalName, time.Time{}, bytes.NewReader(data))
}

func cleanFrontendPath(requestPath string) string {
	cleaned := path.Clean("/" + requestPath)
	return strings.TrimPrefix(cleaned, "/")
}

func looksLikeStaticAsset(name string) bool {
	return strings.Contains(path.Base(name), ".")
}

func clientAcceptsGzip(header string) bool {
	for part := range strings.SplitSeq(header, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		encoding, params, _ := strings.Cut(part, ";")
		encoding = strings.ToLower(strings.TrimSpace(encoding))
		if encoding != "gzip" && encoding != "*" {
			continue
		}

		if qualityValueAllowed(params) {
			return true
		}
	}
	return false
}

func qualityValueAllowed(params string) bool {
	params = strings.TrimSpace(params)
	if params == "" {
		return true
	}

	for param := range strings.SplitSeq(params, ";") {
		param = strings.TrimSpace(param)
		if param == "" {
			continue
		}

		key, value, ok := strings.Cut(param, "=")
		if !ok || !strings.EqualFold(strings.TrimSpace(key), "q") {
			continue
		}

		quality, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil {
			return false
		}
		return quality > 0
	}

	return true
}

func setVary(header http.Header, value string) {
	if current := header.Values("Vary"); len(current) > 0 {
		for _, entry := range current {
			for item := range strings.SplitSeq(entry, ",") {
				if strings.EqualFold(strings.TrimSpace(item), value) {
					return
				}
			}
		}
	}
	header.Add("Vary", value)
}
