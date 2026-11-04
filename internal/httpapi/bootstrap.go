package httpapi

import (
	"log/slog"
	"net/http"

	"main/internal/accountpolicies"
	"main/internal/installation"
)

type BootstrapOptions struct {
	Logger         *slog.Logger
	Installation   *installation.Service
	LogAPIRequests bool
}

// NewBootstrapAPIHandler serves only setup and the public frontend state while no
// database (or no completed installation) exists. Any other API route answers
// setup_required so clients see the same contract as the full application.
func NewBootstrapAPIHandler(options BootstrapOptions) http.Handler {
	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}
	api := &API{
		installation: options.Installation,
		logger:       logger,
	}

	mux := http.NewServeMux()
	registerSharedRoutes(mux, api)
	mux.HandleFunc("GET /api/auth/public-config", func(w http.ResponseWriter, _ *http.Request) {
		writeSuccessJSON(w, http.StatusOK, accountpolicies.PublicAuthConfig{RegistrationEnabled: false})
	})
	setupRequired := func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, CodeSetupRequired)
	}
	mux.HandleFunc("/api", setupRequired)
	mux.HandleFunc("/api/", setupRequired)

	handler := withErrorLogger(logger)(mux)
	if options.LogAPIRequests {
		handler = Chain(handler, requestLogger(logger))
	}
	return handler
}
