package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// ServerConfig captures all tunable parameters for the HTTP API process.
// Values are primarily loaded from environment variables with sane defaults
// so the binary can run locally without excessive setup.
type ServerConfig struct {
	HTTPAddr        string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration

	RedisAddr     string
	RedisPassword string
	RedisGeoKey   string

	KafkaBrokers []string
	KafkaTopic   string

	PGDSN string

	DefaultSpeedMps float64
	MatcherTopN     int

	LogLevel      string
	RunMigrations bool
}

func defaultServerConfig() ServerConfig {
	return ServerConfig{
		HTTPAddr:        ":8080",
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    10 * time.Second,
		IdleTimeout:     120 * time.Second,
		ShutdownTimeout: 15 * time.Second,
		RedisGeoKey:     "drivers_geo",
		KafkaTopic:      "driver-locations",
		DefaultSpeedMps: 10,
		MatcherTopN:     8,
		LogLevel:        "info",
	}
}

func LoadServerConfig() (ServerConfig, error) {
	cfg := defaultServerConfig()
	var errs []error

	setStringFromEnv(&cfg.HTTPAddr, "HTTP_ADDR")
	setDurationFromEnv(&cfg.ReadTimeout, "HTTP_READ_TIMEOUT", &errs)
	setDurationFromEnv(&cfg.WriteTimeout, "HTTP_WRITE_TIMEOUT", &errs)
	setDurationFromEnv(&cfg.IdleTimeout, "HTTP_IDLE_TIMEOUT", &errs)
	setDurationFromEnv(&cfg.ShutdownTimeout, "HTTP_SHUTDOWN_TIMEOUT", &errs)

	cfg.RedisAddr = strings.TrimSpace(os.Getenv("REDIS_ADDR"))
	cfg.RedisPassword = os.Getenv("REDIS_PASSWORD")
	setStringFromEnv(&cfg.RedisGeoKey, "REDIS_GEO_KEY")

	if brokers := os.Getenv("KAFKA_BROKERS"); brokers != "" {
		cfg.KafkaBrokers = splitAndTrim(brokers)
	}
	setStringFromEnv(&cfg.KafkaTopic, "KAFKA_TOPIC")

	cfg.PGDSN = os.Getenv("PG_DSN")

	setFloatFromEnv(&cfg.DefaultSpeedMps, "MATCHER_DEFAULT_SPEED_MPS", &errs)
	setIntFromEnv(&cfg.MatcherTopN, "MATCHER_TOP_N", &errs)

	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = strings.ToLower(v)
	}

	cfg.RunMigrations = strings.EqualFold(os.Getenv("MIGRATE"), "true")

	if cfg.MatcherTopN <= 0 {
		errs = append(errs, fmt.Errorf("MATCHER_TOP_N must be > 0"))
	}

	return cfg, errors.Join(errs...)
}

func setDurationFromEnv(target *time.Duration, key string, errs *[]error) {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			*errs = append(*errs, fmt.Errorf("invalid %s: %w", key, err))
			return
		}
		*target = d
	}
}

func setFloatFromEnv(target *float64, key string, errs *[]error) {
	if v := os.Getenv(key); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			*errs = append(*errs, fmt.Errorf("invalid %s: %w", key, err))
			return
		}
		*target = f
	}
}

func setIntFromEnv(target *int, key string, errs *[]error) {
	if v := os.Getenv(key); v != "" {
		i, err := strconv.Atoi(v)
		if err != nil {
			*errs = append(*errs, fmt.Errorf("invalid %s: %w", key, err))
			return
		}
		*target = i
	}
}

func setStringFromEnv(target *string, key string) {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		*target = v
	}
}

func splitAndTrim(v string) []string {
	raw := strings.Split(v, ",")
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		out = append(out, r)
	}
	return out
}
