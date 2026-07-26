package httpapi

import (
	"bytes"
	"io/fs"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"main/internal/accountpolicies"
	"main/internal/auth"
	"main/internal/authorization"
	"main/internal/identity"
	"main/internal/logging"
	"main/internal/setup"
)

type HandlerOptions struct {
	Logger          *slog.Logger
	LogStream       *logging.Stream
	Auth            *auth.Service
	Authorization   *authorization.Service
	Identity        *identity.Service
	Setup           *setup.Service
	AccountPolicies *accountpolicies.Service
	DataDir         string
	FrontendFS      fs.FS
	LogAPIRequests  bool
}

func NewHandlerWithOptions(options HandlerOptions) http.Handler {
	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}

	apiService := &API{
		auth:          options.Auth,
		authorization: options.Authorization,
		identities:    options.Identity,
		setup:         options.Setup,
		policies:      options.AccountPolicies,
		dataDir:       options.DataDir,
		logger:        logger,
		logStream:     options.LogStream,
	}

	api := newAPIMux(apiService)
	if options.LogAPIRequests {
		api = Chain(api, requestLogger(logger))
	}
	spa := newSPAHandler(options.FrontendFS)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isAPIRequest(r.URL.Path) {
			api.ServeHTTP(w, r)
			return
		}
		spa.ServeHTTP(w, r)
	})
}

func newAPIMux(api *API) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/healthz", healthz)
	mux.Handle("GET /api/install/state", http.HandlerFunc(api.installStateHandler))
	mux.Handle("POST /api/install/setup", http.HandlerFunc(api.installSetupHandler))
	mux.Handle("GET /api/auth/public-config", http.HandlerFunc(api.publicAuthConfigHandler))
	mux.Handle("POST /api/auth/login", Chain(http.HandlerFunc(api.loginHandler), RequireSetupCompleted(api.setup)))
	mux.Handle("POST /api/auth/register", Chain(http.HandlerFunc(api.registerHandler), RequireSetupCompleted(api.setup)))
	mux.Handle("POST /api/auth/refresh", Chain(http.HandlerFunc(api.refreshHandler), RequireSetupCompleted(api.setup)))
	mux.Handle("POST /api/auth/logout", Chain(http.HandlerFunc(api.logoutHandler), RequireSetupCompleted(api.setup)))
	mux.Handle("GET /api/auth/me", Chain(http.HandlerFunc(api.meHandler), RequireSetupCompleted(api.setup), RequireAuth(api.auth)))
	mux.Handle("PATCH /api/account/profile", Chain(http.HandlerFunc(api.updateProfileHandler), RequireSetupCompleted(api.setup), RequireAuth(api.auth)))
	mux.Handle("POST /api/account/avatar", Chain(http.HandlerFunc(api.updateAvatarHandler), RequireSetupCompleted(api.setup), RequireAuth(api.auth)))
	mux.Handle("POST /api/account/password", Chain(http.HandlerFunc(api.changePasswordHandler), RequireSetupCompleted(api.setup), RequireAuth(api.auth)))
	mux.Handle("DELETE /api/account", Chain(http.HandlerFunc(api.deleteAccountHandler), RequireSetupCompleted(api.setup), RequireAuth(api.auth)))
	mux.Handle("GET /api/avatars/{filename}", http.HandlerFunc(api.avatarHandler))

	mux.Handle(
		"GET /api/system/settings",
		Chain(
			http.HandlerFunc(api.getAccountPoliciesHandler),
			RequireSetupCompleted(api.setup),
			RequireAuth(api.auth),
			RequireCapability(authorization.CapabilitySystemSettingsRead),
		),
	)
	mux.Handle(
		"PATCH /api/system/settings",
		Chain(
			http.HandlerFunc(api.updateAccountPoliciesHandler),
			RequireSetupCompleted(api.setup),
			RequireAuth(api.auth),
			RequireCapability(authorization.CapabilitySystemSettingsUpdate),
		),
	)

	mux.Handle(
		"GET /api/management/users",
		Chain(
			http.HandlerFunc(api.listManagementUsersHandler),
			RequireSetupCompleted(api.setup),
			RequireAuth(api.auth),
			RequireCapability(authorization.CapabilityManagementUsersRead),
		),
	)
	mux.Handle(
		"POST /api/management/users",
		Chain(
			http.HandlerFunc(api.createManagementUserHandler),
			RequireSetupCompleted(api.setup),
			RequireAuth(api.auth),
			RequireCapability(authorization.CapabilityManagementUsersCreate),
		),
	)
	mux.Handle(
		"PATCH /api/management/users/{id}",
		Chain(
			http.HandlerFunc(api.updateManagementUserHandler),
			RequireSetupCompleted(api.setup),
			RequireAuth(api.auth),
			RequireCapability(authorization.CapabilityManagementUsersUpdate),
		),
	)
	mux.Handle(
		"POST /api/management/users/{id}/disable",
		Chain(
			http.HandlerFunc(api.disableManagementUserHandler),
			RequireSetupCompleted(api.setup),
			RequireAuth(api.auth),
			RequireCapability(authorization.CapabilityManagementUsersDisable),
		),
	)
	mux.Handle(
		"POST /api/management/users/{id}/enable",
		Chain(
			http.HandlerFunc(api.enableManagementUserHandler),
			RequireSetupCompleted(api.setup),
			RequireAuth(api.auth),
			RequireCapability(authorization.CapabilityManagementUsersEnable),
		),
	)
	mux.Handle(
		"GET /api/management/system-logs/stream",
		Chain(
			http.HandlerFunc(api.streamSystemLogsHandler),
			RequireSetupCompleted(api.setup),
			RequireAuth(api.auth),
			RequireCapability(authorization.CapabilityManagementSystemLogsRead),
		),
	)

	return mux
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

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

func isAPIRequest(requestPath string) bool {
	return requestPath == "/api" || strings.HasPrefix(requestPath, "/api/")
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
