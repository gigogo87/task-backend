package app

import (
	"fmt"
	"math"
	"os"
	"strconv"
)

// Config 是程式的設定，全部可以用環境變數覆蓋。
type Config struct {
	Port           string  // PORT，預設 8080
	RateLimitRPS   float64 // RATE_LIMIT_RPS：每個 IP 每秒可以打幾次，預設 10
	RateLimitBurst int     // RATE_LIMIT_BURST：瞬間最多可以連打幾次，預設 20
}

// LoadConfig 從環境變數讀設定；格式錯誤時回傳 error，讓程式在啟動時就失敗，而不是跑起來才出問題。
func LoadConfig() (Config, error) {
	cfg := Config{
		Port:           "8080",
		RateLimitRPS:   10,
		RateLimitBurst: 20,
	}

	if v := os.Getenv("PORT"); v != "" {
		cfg.Port = v
	}
	if v := os.Getenv("RATE_LIMIT_RPS"); v != "" {
		rps, err := strconv.ParseFloat(v, 64)
		if err != nil || rps <= 0 || math.IsNaN(rps) || math.IsInf(rps, 0) {
			return Config{}, fmt.Errorf("RATE_LIMIT_RPS must be a positive number, got %q", v)
		}
		cfg.RateLimitRPS = rps
	}
	if v := os.Getenv("RATE_LIMIT_BURST"); v != "" {
		burst, err := strconv.Atoi(v)
		if err != nil || burst < 1 {
			return Config{}, fmt.Errorf("RATE_LIMIT_BURST must be a positive integer, got %q", v)
		}
		cfg.RateLimitBurst = burst
	}
	return cfg, nil
}
