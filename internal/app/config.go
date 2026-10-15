package app

import (
	"fmt"
	"math"
	"os"
	"strconv"
)

// 可以覆蓋設定的環境變數名稱。
const (
	envPort           = "PORT"
	envRateLimitRPS   = "RATE_LIMIT_RPS"
	envRateLimitBurst = "RATE_LIMIT_BURST"
)

// 沒有設定環境變數時使用的預設值。
const (
	defaultPort           = "8080"
	defaultRateLimitRPS   = 10 // 每個 IP 每秒可以打幾次
	defaultRateLimitBurst = 20 // 每個 IP 瞬間最多可以連打幾次
)

// Config 是程式的設定，全部可以用環境變數覆蓋（名稱與預設值見上方常數）。
type Config struct {
	Port           string
	RateLimitRPS   float64
	RateLimitBurst int
}

// LoadConfig 從環境變數讀設定；格式錯誤時回傳 error，讓程式在啟動時就失敗，而不是跑起來才出問題。
func LoadConfig() (Config, error) {
	cfg := Config{
		Port:           defaultPort,
		RateLimitRPS:   defaultRateLimitRPS,
		RateLimitBurst: defaultRateLimitBurst,
	}

	if v := os.Getenv(envPort); v != "" {
		cfg.Port = v
	}
	if v := os.Getenv(envRateLimitRPS); v != "" {
		rps, err := strconv.ParseFloat(v, 64)
		if err != nil || rps <= 0 || math.IsNaN(rps) || math.IsInf(rps, 0) {
			return Config{}, fmt.Errorf("%s must be a positive number, got %q", envRateLimitRPS, v)
		}
		cfg.RateLimitRPS = rps
	}
	if v := os.Getenv(envRateLimitBurst); v != "" {
		burst, err := strconv.Atoi(v)
		if err != nil || burst < 1 {
			return Config{}, fmt.Errorf("%s must be a positive integer, got %q", envRateLimitBurst, v)
		}
		cfg.RateLimitBurst = burst
	}
	return cfg, nil
}
