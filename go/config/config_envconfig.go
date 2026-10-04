package config

import (
	"fmt"
	"strconv"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config holds all service configuration, loaded from environment variables.
// Struct tags define env var names and defaults.
type Config struct {
	DatabaseURL   string `envconfig:"DATABASE_URL" default:"postgres://postgres:postgres@localhost:5432/mydb?sslmode=disable"`
	RedisHost     string `envconfig:"REDIS_HOST" default:"localhost"`
	RedisPort     string `envconfig:"REDIS_PORT" default:"6379"`
	Env           string `envconfig:"APP_ENV" default:"dev"`
	PollIntervalS string `envconfig:"POLL_INTERVAL_SEC" default:"10"`
	DailyAPILimit int64  `envconfig:"DAILY_API_LIMIT" default:"100000"`
	EnableTLS     bool   `envconfig:"ENABLE_TLS" default:"false"`
	DebugMode     bool   `envconfig:"DEBUG" default:"false"`

	// Computed — not from env
	PollInterval time.Duration `ignored:"true"`
}

// Load reads config from environment variables with defaults.
// Invalid values are errors; callers must handle them before boot.
func Load() (Config, error) {
	var c Config
	if err := envconfig.Process("", &c); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}

	n, err := strconv.ParseInt(c.PollIntervalS, 10, 64)
	if err != nil || n <= 0 || n > int64((1<<63-1)/int64(time.Second)) {
		return Config{}, fmt.Errorf("POLL_INTERVAL_SEC must be a positive bounded integer")
	}
	if c.DailyAPILimit <= 0 {
		return Config{}, fmt.Errorf("DAILY_API_LIMIT must be positive")
	}
	c.PollInterval = time.Duration(n) * time.Second
	return c, nil
}
