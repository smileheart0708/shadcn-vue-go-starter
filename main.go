package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"main/internal/accountpolicies"
	"main/internal/auth"
	"main/internal/authorization"
	"main/internal/avatar"
	"main/internal/config"
	"main/internal/database"
	"main/internal/httpapi"
	"main/internal/identity"
	"main/internal/installation"
	"main/internal/logging"
	"main/internal/setup"
)

const shutdownTimeout = 10 * time.Second

func main() {
	if err := run(); err != nil {
		slog.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	logger, logStream := logging.NewWithStream(logging.StreamOptions{
		MaxBytes:  logging.DefaultStreamMaxBytes,
		Retention: logging.DefaultRetention,
	})
	slog.SetDefault(logger)

	if err := config.Load(); err != nil {
		return fmt.Errorf("load environment: %w", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	if seed := strings.TrimSpace(cfg.JWTSecret); seed != "" && len(seed) < auth.MinJWTSecretLength {
		// Not fatal: a stored secret still wins. It only fails where the seed is
		// actually consumed, i.e. when the database holds no signing secret yet.
		slog.Warn("JWT_SECRET is shorter than the minimum and will be rejected unless a signing secret is already stored", "minimum", auth.MinJWTSecretLength)
	}

	frontendAssets, err := frontendFS()
	if err != nil {
		return fmt.Errorf("load embedded frontend assets: %w", err)
	}

	dataDir, err := filepath.Abs(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("resolve data directory %q: %w", cfg.DataDir, err)
	}

	listenAddr := fmt.Sprintf(":%d", cfg.Port)

	// Bind before touching the database: a port conflict then fails in
	// milliseconds instead of paying for a migration that can never serve.
	// The cost is a window where the socket accepts connections nobody answers,
	// so a slow database dial looks like a hang instead of "connection refused".
	// shortcut: noctx rejects net.Listen; keep the ListenConfig form until the
	// linter set drops it.
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(context.Background(), "tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("open listener: %w", err)
	}
	// No Close here: Serve closes the listener on return, and the only exit
	// before Serve is a failed installation.New, which ends in os.Exit.
	// Re-add a Close only if run() ever becomes callable in-process.

	var installationService *installation.Service
	// Set by PrepareRuntime before anything is published. buildAPI must never run
	// while it is empty, or every token would be signed with an empty key.
	var jwtSecret string
	buildAPI := func(runtime *database.Runtime) http.Handler {
		store := runtime.Store
		authorizationService := authorization.NewService()
		identityService := identity.NewService(store)
		accountPoliciesService := accountpolicies.NewService(store)
		setupService := setup.NewService(store)
		authService := auth.NewService(auth.Options{
			Secret:             []byte(jwtSecret),
			TTL:                cfg.JWTTTL,
			RefreshIdleTTL:     cfg.RefreshIdleTTL,
			RefreshAbsoluteTTL: cfg.RefreshAbsoluteTTL,
			RefreshCookieName:  cfg.RefreshCookieName,
		}, store, identityService, authorizationService, accountPoliciesService)

		return httpapi.NewAPIHandlerWithOptions(httpapi.HandlerOptions{
			Logger:          logger,
			LogStream:       logStream,
			Auth:            authService,
			Authorization:   authorizationService,
			Identity:        identityService,
			Setup:           setupService,
			Installation:    installationService,
			AccountPolicies: accountPoliciesService,
			Avatars:         avatar.New(dataDir),
			LogAPIRequests:  cfg.APIRequestLogEnabled,
		})
	}

	// installation.New only activates a runtime and never publishes, so the closure
	// below cannot observe this nil between here and the assignment.
	var dispatcher *httpapi.Dispatcher
	// Reconcile before publishing the upload route: no Save can be in flight yet, so
	// an unreferenced avatar file is genuinely orphaned. The Publish callback also runs
	// on setup completion, where the bootstrap API is still the only handler.
	publish := func(runtime *database.Runtime) {
		reconcileAvatars(runtime, dataDir, logger)
		dispatcher.Publish(buildAPI(runtime))
	}
	installationService, err = installation.New(context.Background(), installation.Options{
		DataDir:        dataDir,
		SelectionStore: database.NewSelectionStore(dataDir),
		// PrepareRuntime resolves the signing secret before the owner transaction
		// commits; buildAPI then reuses it instead of reading the database again.
		PrepareRuntime: func(ctx context.Context, runtime *database.Runtime) error {
			secret, resolveErr := auth.ResolveJWTSecret(ctx, runtime.Store, cfg.JWTSecret)
			if resolveErr != nil {
				return fmt.Errorf("resolve JWT secret: %w", resolveErr)
			}
			jwtSecret = secret
			return nil
		},
		Publish: publish,
	})
	if err != nil {
		return fmt.Errorf("initialize installation: %w", err)
	}
	defer closeInstallation(installationService)

	bootstrapAPI := httpapi.NewBootstrapAPIHandler(httpapi.BootstrapOptions{
		Logger:         logger,
		Installation:   installationService,
		LogAPIRequests: cfg.APIRequestLogEnabled,
	})
	dispatcher = httpapi.NewDispatcher(frontendAssets, bootstrapAPI)

	if runtime := installationService.Runtime(); runtime != nil && installationService.State().Phase == installation.PhaseCompleted {
		publish(runtime)
	}

	server := &http.Server{
		Handler:           dispatcher,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		// Long-lived responses (SSE) clear this deadline per-request via
		// http.ResponseController, so slow clients on normal routes stay bounded.
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
		// Handler panics and accept failures otherwise go to the standard logger,
		// i.e. never into the system log stream the UI shows.
		ErrorLog: slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}
	// The system log stream never returns on its own, and Shutdown waits for
	// in-flight handlers, so close its subscriptions before waiting.
	server.RegisterOnShutdown(func() { logStream.Shutdown("server_shutdown") })

	logging.LogStartupBanner(logger, listenAddr, dataDir)

	serverErrCh := make(chan error, 1)
	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			serverErrCh <- serveErr
			return
		}
		serverErrCh <- nil
	}()

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case serverErr := <-serverErrCh:
		if serverErr != nil {
			return fmt.Errorf("http server stopped unexpectedly: %w", serverErr)
		}
		return nil
	case <-signalCtx.Done():
		slog.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if shutdownErr := server.Shutdown(shutdownCtx); shutdownErr != nil {
		// A handler that ignores cancellation keeps its connection active past the
		// deadline; that is a slow stop, not a startup failure, so drop the rest
		// instead of exiting non-zero on every stop.
		slog.Warn("graceful shutdown timed out, closing remaining connections", "error", shutdownErr)
		if closeErr := server.Close(); closeErr != nil {
			slog.Warn("failed to close remaining http connections", "error", closeErr)
		}
	}

	if serverErr := <-serverErrCh; serverErr != nil {
		return fmt.Errorf("http server stopped unexpectedly: %w", serverErr)
	}

	slog.Info("application stopped")
	return nil
}

// reconcileAvatars deletes avatar files no user row references. Callers must run it
// before the upload route is published; see avatar.Store.Reconcile for why.
func reconcileAvatars(runtime *database.Runtime, dataDir string, logger *slog.Logger) {
	removed, err := avatar.New(dataDir).Reconcile(context.Background(), runtime.Store.ListAvatarPaths)
	if err != nil {
		logger.Warn("avatar reconciliation incomplete", "removed", removed, "error", err)
	} else if removed > 0 {
		logger.Info("removed orphaned avatar files", "count", removed)
	}
}

func closeInstallation(service *installation.Service) {
	hadDatabase := service.Runtime() != nil
	if err := service.Close(); err != nil {
		slog.Error("failed to close installation database", "error", err)
		return
	}
	if hadDatabase {
		slog.Info("installation database closed")
	}
}
