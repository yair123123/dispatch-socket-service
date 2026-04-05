package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppPort               string
	LogLevel              string
	JWTSecret             string
	JWTPublicKey          string
	RedisURL              string
	CoreServiceBaseURL    string
	CoreServiceTimeout    time.Duration
	WSPingInterval        time.Duration
	WSReadTimeout         time.Duration
	WSWriteTimeout        time.Duration
	DriverStateTTL        time.Duration
	DriverLocationTTL     time.Duration
	RedisAcceptLockTTL    time.Duration
	CoreSyncRetryInterval time.Duration
	CoreSyncMaxRetries    int
}

func Load() (Config, error) {
	cfg := Config{
		AppPort:               envOrDefault("APP_PORT", "8080"),
		LogLevel:              envOrDefault("LOG_LEVEL", "info"),
		JWTSecret:             os.Getenv("JWT_SECRET"),
		JWTPublicKey:          os.Getenv("JWT_PUBLIC_KEY"),
		RedisURL:              envOrDefault("REDIS_URL", "redis://localhost:6379/0"),
		CoreServiceBaseURL:    envOrDefault("CORE_SERVICE_BASE_URL", "http://localhost:8081"),
		CoreSyncMaxRetries:    intEnvOrDefault("CORE_SYNC_MAX_RETRIES", 10),
		CoreServiceTimeout:    durationSeconds("CORE_SERVICE_TIMEOUT_SECONDS", 3),
		WSPingInterval:        durationSeconds("WS_PING_INTERVAL_SECONDS", 10),
		WSReadTimeout:         durationSeconds("WS_READ_TIMEOUT_SECONDS", 30),
		WSWriteTimeout:        durationSeconds("WS_WRITE_TIMEOUT_SECONDS", 5),
		DriverStateTTL:        durationSeconds("DRIVER_STATE_TTL_SECONDS", 60),
		DriverLocationTTL:     durationSeconds("DRIVER_LOCATION_TTL_SECONDS", 60),
		RedisAcceptLockTTL:    durationSeconds("REDIS_ACCEPT_LOCK_TTL_SECONDS", 15),
		CoreSyncRetryInterval: durationSeconds("CORE_SYNC_RETRY_INTERVAL_SECONDS", 2),
	}

	if cfg.JWTSecret == "" && cfg.JWTPublicKey == "" {
		return Config{}, fmt.Errorf("either JWT_SECRET or JWT_PUBLIC_KEY must be set")
	}
	return cfg, nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func intEnvOrDefault(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		parsed, err := strconv.Atoi(v)
		if err == nil {
			return parsed
		}
	}
	return def
}

func durationSeconds(key string, def int) time.Duration {
	return time.Duration(intEnvOrDefault(key, def)) * time.Second
}
