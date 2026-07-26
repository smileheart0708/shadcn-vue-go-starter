package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Regression: requestLogger wraps the ResponseWriter in statusRecorder. Without
// an Unwrap method, http.ResponseController could not reach the underlying
// Flusher and the SSE log stream failed with "streaming_unsupported" whenever
// API_REQUEST_LOG_ENABLED was true.
func TestRequestLoggerKeepsResponseControllerFlushable(t *testing.T) {
	t.Parallel()

	var flushErr error
	var sawRecorder bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, sawRecorder = w.(*statusRecorder)
		flushErr = http.NewResponseController(w).Flush()
	})

	logged := Chain(inner, requestLogger(slog.New(slog.NewTextHandler(io.Discard, nil))))
	logged.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/management/system-logs/stream", nil))

	if !sawRecorder {
		t.Fatal("expected requestLogger to wrap the ResponseWriter in statusRecorder")
	}
	if flushErr != nil {
		t.Fatalf("expected Flush through the request logger to succeed, got %v", flushErr)
	}
}

func TestParseSystemLogTail(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tests := []struct {
		name string
		raw  string
		want int
	}{
		{name: "missing replays full buffer", want: 0},
		{name: "all replays full buffer", raw: "all", want: 0},
		{name: "uppercase all replays full buffer", raw: "ALL", want: 0},
		{name: "latest 100", raw: "100", want: 100},
		{name: "latest 200", raw: "200", want: 200},
		{name: "latest 500", raw: "500", want: 500},
		{name: "latest 1000", raw: "1000", want: 1000},
		{name: "invalid falls back to full buffer", raw: "invalid", want: 0},
		{name: "unsupported number falls back to full buffer", raw: "50", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/api/management/system-logs/stream", nil)
			if tt.raw != "" {
				query := req.URL.Query()
				query.Set("tail", tt.raw)
				req.URL.RawQuery = query.Encode()
			}

			if got := parseSystemLogTail(req, logger); got != tt.want {
				t.Fatalf("expected tail %d, got %d", tt.want, got)
			}
		})
	}
}
