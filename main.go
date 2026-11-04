package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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

	logging.LogStartupBanner(logger, listenAddr, dataDir)

	var installationService *installation.Service
	var jwtSecret string
	buildAPI := func(runtime *database.Runtime, secret string) http.Handler {
		store := runtime.Store
		authorizationService := authorization.NewService()
		identityService := identity.NewService(store)
		accountPoliciesService := accountpolicies.NewService(store)
		setupService := setup.NewService(store)
		authService := auth.NewService(auth.Options{
			Secret:             []byte(secret),
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

	var dispatcher *httpapi.Dispatcher
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
		Publish: func(runtime *database.Runtime) {
			dispatcher.Publish(buildAPI(runtime, jwtSecret))
		},
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
		dispatcher.Publish(buildAPI(runtime, jwtSecret))
	}

	server := &http.Server{
		Addr:              listenAddr,
		Handler:           dispatcher,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		// Long-lived responses (SSE) clear this deadline per-request via
		// http.ResponseController, so slow clients on normal routes stay bounded.
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	serverErrCh := make(chan error, 1)
	go func() {
		if listenErr := server.ListenAndServe(); listenErr != nil && !errors.Is(listenErr, http.ErrServerClosed) {
			serverErrCh <- listenErr
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
		return fmt.Errorf("shutdown http server: %w", shutdownErr)
	}

	if serverErr := <-serverErrCh; serverErr != nil {
		return fmt.Errorf("http server stopped unexpectedly: %w", serverErr)
	}

	slog.Info("application stopped")
	return nil
}

func closeInstallation(service *installation.Service) {
	if service == nil {
		return
	}
	hadDatabase := service.Runtime() != nil
	if err := service.Close(); err != nil {
		slog.Error("failed to close installation database", "error", err)
		return
	}
	if hadDatabase {
		slog.Info("installation database closed")
	}
}
