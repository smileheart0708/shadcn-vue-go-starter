package httpapi

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Regression: requestLogger wraps the ResponseWriter in statusRecorder. Without
// an Unwrap method, http.ResponseController could not reach the underlying
// Flusher and the SSE log stream failed with a 500 error whenever
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

// Regression: the log stream never returns on its own, so Shutdown burns its full
// deadline on every stop while a log page is open. main.go ends the stream through
// Server.RegisterOnShutdown; without that hook Shutdown reports a timeout and the
// process treats a normal stop as a failure.
func TestServerShutdownEndsOpenLogStream(t *testing.T) {
	t.Parallel()

	ctx := newTestContext(t)
	session, _ := performSetup(t, ctx)

	server := &http.Server{
		Handler:           ctx.handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	server.RegisterOnShutdown(func() { ctx.logStream.Shutdown("server_shutdown") })
	t.Cleanup(func() {
		if closeErr := server.Close(); closeErr != nil {
			t.Fatalf("close test server: %v", closeErr)
		}
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	serveErrCh := make(chan error, 1)
	go func() { serveErrCh <- server.Serve(listener) }()

	request, err := http.NewRequest(http.MethodGet, "http://"+listener.Addr().String()+"/api/management/system-logs/stream", nil)
	if err != nil {
		t.Fatalf("build stream request: %v", err)
	}
	request.Header.Set("Authorization", "Bearer "+session.AccessToken)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("open log stream: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("stream status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	// The handler subscribes before it flushes the response headers.
	if count := ctx.logStream.SubscriberCount(); count != 1 {
		t.Fatalf("subscribers = %d, want 1", count)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if shutdownErr := server.Shutdown(shutdownCtx); shutdownErr != nil {
		t.Fatalf("Shutdown() error = %v, want nil once the log stream is closed", shutdownErr)
	}
	if count := ctx.logStream.SubscriberCount(); count != 0 {
		t.Fatalf("subscribers after Shutdown() = %d, want 0", count)
	}
	if serveErr := <-serveErrCh; !errors.Is(serveErr, http.ErrServerClosed) {
		t.Fatalf("Serve() error = %v, want ErrServerClosed", serveErr)
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
