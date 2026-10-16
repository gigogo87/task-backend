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

// http.Server 預設沒有任何逾時：client 故意慢慢傳資料，就能一直佔住連線（Slowloris 攻擊）。
// 這個 API 的 body 上限是 1 MB、handler 都是記憶體操作，所以下面的值都很寬鬆，正常請求不會被切斷。
const (
	readHeaderTimeout = 5 * time.Second  // 讀完 header 的上限
	readTimeout       = 10 * time.Second // 讀完整個請求的上限，包含最多 1 MB 的 body
	writeTimeout      = 15 * time.Second // 從讀完請求到寫完回應的上限；必須比最慢的 handler 長
	idleTimeout       = 60 * time.Second // keep-alive 連線閒置多久後關閉

	// shutdownTimeout 是優雅關閉時最多等待處理中請求的時間。
	// 要比平台送 SIGKILL 前的寬限期短（docker stop 預設 10 秒），關閉流程才跑得完。
	shutdownTimeout = 5 * time.Second
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
		Handler:           dp.Router,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
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
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", "error", err)
	}
	logger.Info("server stopped")
}
