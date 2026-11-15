package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func newTestSPAHandler() http.Handler {
	return newSPAHandler(fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<!doctype html><div id=\"app\"></div>")},
		// Mirrors what Vite emits: a content hash in the name plus a precompressed sibling.
		"assets/app-abc123.js":    &fstest.MapFile{Data: []byte("console.log(1)")},
		"assets/app-abc123.js.gz": &fstest.MapFile{Data: []byte("precompressed")},
		// Root-level assets keep their name across deploys, so they must not be pinned.
		"logo.svg": &fstest.MapFile{Data: []byte("<svg/>")},
	})
}

// The cache policy is split on the logical path: everything under assets/ is content-hashed by
// Vite and can be pinned for a year, while index.html and root assets keep their names and must
// revalidate — pinning index.html leaves clients on a shell that references asset hashes the
// server no longer serves.
func TestSPAHandlerCacheControl(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		path      string
		wantCode  int
		wantCache string
	}{
		{
			name:      "content-hashed asset is immutable",
			path:      "/assets/app-abc123.js",
			wantCode:  http.StatusOK,
			wantCache: "public, max-age=31536000, immutable",
		},
		{
			name:      "entry document must revalidate",
			path:      "/",
			wantCode:  http.StatusOK,
			wantCache: "no-cache",
		},
		{
			name:      "root asset keeps its name and must revalidate",
			path:      "/logo.svg",
			wantCode:  http.StatusOK,
			wantCache: "no-cache",
		},
		{
			// A 404 that carried the immutable header would outlive the deploy that fixes it.
			name:      "missing asset is not cached",
			path:      "/assets/gone.js",
			wantCode:  http.StatusNotFound,
			wantCache: "",
		},
	}

	handler := newTestSPAHandler()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, test.path, nil))

			if recorder.Code != test.wantCode {
				t.Fatalf("GET %s status = %d, want %d", test.path, recorder.Code, test.wantCode)
			}
			if got := recorder.Header().Get("Cache-Control"); got != test.wantCache {
				t.Fatalf("GET %s Cache-Control = %q, want %q", test.path, got, test.wantCache)
			}
		})
	}
}

// Vary has to be advertised on both encoding branches. It used to be set only when the
// precompressed copy was served, which is harmless while nothing is cached but becomes wrong as
// soon as the immutable header lets a shared cache store the response: the cache would then hand
// a gzip body to a client that never sent Accept-Encoding.
func TestSPAHandlerVaryOnBothEncodingBranches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		acceptEncoding string
		wantEncoding   string
	}{
		{name: "gzip accepted", acceptEncoding: "gzip", wantEncoding: "gzip"},
		{name: "gzip not accepted", acceptEncoding: "", wantEncoding: ""},
	}

	handler := newTestSPAHandler()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/assets/app-abc123.js", nil)
			if test.acceptEncoding != "" {
				request.Header.Set("Accept-Encoding", test.acceptEncoding)
			}

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
			}
			if got := recorder.Header().Get("Vary"); got != "Accept-Encoding" {
				t.Fatalf("Vary = %q, want %q", got, "Accept-Encoding")
			}
			if got := recorder.Header().Get("Content-Encoding"); got != test.wantEncoding {
				t.Fatalf("Content-Encoding = %q, want %q", got, test.wantEncoding)
			}
		})
	}
}
