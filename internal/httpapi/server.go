package httpapi

import (
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

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

func isAPIRequest(requestPath string) bool {
	return requestPath == "/api" || strings.HasPrefix(requestPath, "/api/")
}
