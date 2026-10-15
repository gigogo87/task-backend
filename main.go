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

func main() {
	// 1. 讀設定：PORT、RATE_LIMIT_RPS、RATE_LIMIT_BURST 都可以用環境變數覆蓋
	cfg, err := app.LoadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	// 2. 組裝：所有元件都由 dependency provider 建立並注入
	dp := app.NewDependencyProvider(cfg, os.Stdout)
	defer dp.Close()
	logger := dp.Logger

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           dp.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	// 3. 在另一個 goroutine 啟動 server，主程式繼續往下等待關閉訊號
	go func() {
		logger.Info("server listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// 4. Graceful shutdown：收到 Ctrl+C 或 docker stop 時，等正在處理的請求做完再關
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", "error", err)
	}
	logger.Info("server stopped")
}
