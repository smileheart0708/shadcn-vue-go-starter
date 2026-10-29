package httpapi

import (
	"log/slog"
	"net"
	"net/http"
	"time"
)

func requestLogger(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		requestLogger := logger.With("log_source", "http_access")
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			start := time.Now()

			next.ServeHTTP(recorder, r)

			route := r.Pattern
			if route == "" {
				route = r.URL.Path
			}

			requestLogger.InfoContext(
				r.Context(),
				"http request",
				"method", r.Method,
				"route", route,
				"status", recorder.status,
				"duration", time.Since(start),
				"remote_addr", clientAddr(r.RemoteAddr),
			)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// Unwrap exposes the underlying ResponseWriter to http.ResponseController so
// wrapping a handler does not hide Flusher/deadline support from it. Without
// this, enabling API request logging breaks the SSE log stream.
func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func clientAddr(remoteAddr string) string {
	if remoteAddr == "" {
		return ""
	}

	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		return host
	}
	return remoteAddr
}
