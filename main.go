package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"taskapi/internal/app"
)

const (
	readHeaderTimeout = 5 * time.Second  // limit for reading the request headers
	readTimeout       = 10 * time.Second // limit for reading the whole request, including a body of up to 1 MB
	writeTimeout      = 15 * time.Second // limit from reading the request to finishing the response
	idleTimeout       = 60 * time.Second // how long an idle keep-alive connection is kept open

	// shutdownTimeout is how long graceful shutdown waits for in-flight requests.
	// It must be shorter than the platform's grace period before SIGKILL (10s for docker stop by default).
	shutdownTimeout = 5 * time.Second
)

func main() {
	// 1. Load config: PORT, RATE_LIMIT_RPS and RATE_LIMIT_BURST can be overridden by environment variables.
	cfg, err := app.LoadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	// 2. Wire up: the dependency provider builds and injects every component.
	dp := app.NewDependencyProvider(cfg, os.Stdout)
	defer dp.Close()
	logger := dp.Logger

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           dp.Router,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	// 3. Start the server in a goroutine; main goes on to wait for a shutdown signal.
	go func() {
		logger.Info("server listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// 4. Graceful shutdown: on Ctrl+C or docker stop, let in-flight requests finish before exiting.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", "error", err)
	}
	logger.Info("server stopped")
}
