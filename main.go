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
	"syscall"
	"time"

	"main/internal/accountpolicies"
	"main/internal/auth"
	"main/internal/authorization"
	"main/internal/config"
	"main/internal/database"
	"main/internal/httpapi"
	"main/internal/identity"
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

	dbRuntime, err := database.Open(context.Background(), database.Config{
		Driver: cfg.Database.Driver,
		DSN:    cfg.Database.DSN,
	})
	if err != nil {
		return fmt.Errorf("open database driver %q: %w", cfg.Database.Driver, err)
	}
	defer closeDB(dbRuntime)

	authorizationService := authorization.NewService()
	identityService := identity.NewService(dbRuntime.Identity)
	accountPoliciesService := accountpolicies.NewService(dbRuntime.AccountPolicies)
	setupService := setup.NewService(dbRuntime.Setup)
	authService := auth.NewService(auth.Options{
		Secret:             []byte(cfg.JWTSecret),
		TTL:                cfg.JWTTTL,
		RefreshIdleTTL:     cfg.RefreshIdleTTL,
		RefreshAbsoluteTTL: cfg.RefreshAbsoluteTTL,
		RefreshCookieName:  cfg.RefreshCookieName,
	}, dbRuntime.Sessions, identityService, authorizationService, accountPoliciesService)

	server := &http.Server{
		Addr: listenAddr,
		Handler: httpapi.NewHandlerWithOptions(httpapi.HandlerOptions{
			Logger:          logger,
			LogStream:       logStream,
			Auth:            authService,
			Authorization:   authorizationService,
			Identity:        identityService,
			Setup:           setupService,
			AccountPolicies: accountPoliciesService,
			DataDir:         dataDir,
			FrontendFS:      frontendAssets,
			LogAPIRequests:  cfg.APIRequestLogEnabled,
		}),
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

func closeDB(runtime *database.Runtime) {
	if runtime == nil {
		return
	}
	if err := runtime.Close(); err != nil {
		slog.Error("failed to close database", "error", err)
		return
	}
	slog.Info("database closed", "driver", runtime.Driver())
}
