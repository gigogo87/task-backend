package app

import (
	"fmt"
	"math"
	"os"
	"strconv"
)

// Names of the environment variables that override the defaults.
const (
	envPort           = "PORT"
	envRateLimitRPS   = "RATE_LIMIT_RPS"
	envRateLimitBurst = "RATE_LIMIT_BURST"
)

// Defaults used when the environment variables are not set.
const (
	defaultPort           = "8080"
	defaultRateLimitRPS   = 10 // requests per second allowed per IP
	defaultRateLimitBurst = 20 // maximum burst of requests per IP
)

// Config holds the program settings; every field can be overridden by an environment variable.
type Config struct {
	Port           string
	RateLimitRPS   float64
	RateLimitBurst int
}

// LoadConfig reads settings from the environment. Invalid values return an error, so the program fails at startup instead of misbehaving later.
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
