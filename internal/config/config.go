package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration.
type Config struct {
	BeaconNodeURL        string   `yaml:"beacon_node_url"`
	BeaconAPIKey         string   `yaml:"beacon_api_key,omitempty"` // Optional API key for providers like Tatum
	Validators           []uint64 `yaml:"validators"`
	PollingIntervalSlots int      `yaml:"polling_interval_slots"`
	// SlotDurationSeconds allows overriding the default 12s slot duration.
	// For local devnets (e.g. kurtosis) you can set this to 2.
	SlotDurationSeconds int           `yaml:"slot_duration_seconds,omitempty"`
	WorkerPoolSize      int           `yaml:"worker_pool_size"`
	RateLimit           RateLimitConf `yaml:"rate_limit"`
	HTTP                HTTPConf      `yaml:"http"`
	// DatabaseDriver is optional; only "postgres" is supported (default when empty).
	DatabaseDriver string       `yaml:"database_driver,omitempty"`
	Postgres       PostgresConf `yaml:"postgres"`
}

// RateLimitConf configures the rate limiter.
type RateLimitConf struct {
	RequestsPerSecond float64 `yaml:"requests_per_second"`
	Burst             int     `yaml:"burst"`
}

// HTTPConf configures the HTTP client (beacon REST API).
type HTTPConf struct {
	TimeoutSeconds int `yaml:"timeout_seconds"`
	MaxIdleConns   int `yaml:"max_idle_conns"`
	// MaxRetries is the maximum number of retries after a failed attempt (timeouts, 429, 503, etc.).
	// Applied by the beacon client only; not related to database drivers.
	MaxRetries int `yaml:"max_retries"`
}

// PostgresConf configures PostgreSQL connection.
type PostgresConf struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	SSLMode  string `yaml:"ssl_mode"`
	MaxConns int32  `yaml:"max_conns"`
	TTLDays  int    `yaml:"ttl_days"`
}

// Timeout returns the HTTP timeout as a time.Duration.
func (h *HTTPConf) Timeout() time.Duration {
	return time.Duration(h.TimeoutSeconds) * time.Second
}

// SlotDuration returns the effective slot duration.
// Defaults to 12 seconds (mainnet), but can be overridden via config.
func (c *Config) SlotDuration() time.Duration {
	seconds := c.SlotDurationSeconds
	if seconds <= 0 {
		seconds = 12
	}
	return time.Duration(seconds) * time.Second
}

// SlotsPerEpoch returns the number of slots per epoch (32).
func SlotsPerEpoch() uint64 {
	return 32
}

// Load reads and parses the configuration from a YAML file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	cfg.setDefaults()

	return &cfg, nil
}

// validate checks the configuration for required fields.
func (c *Config) validate() error {
	if c.BeaconNodeURL == "" {
		return fmt.Errorf("beacon_node_url is required")
	}
	if len(c.Validators) == 0 {
		return fmt.Errorf("at least one validator index is required")
	}
	switch c.DatabaseDriver {
	case "", "postgres":
		if c.Postgres.Host == "" {
			return fmt.Errorf("postgres host is required")
		}
		if c.Postgres.Port == 0 {
			return fmt.Errorf("postgres port is required")
		}
		if c.Postgres.User == "" {
			return fmt.Errorf("postgres user is required")
		}
		if c.Postgres.Database == "" {
			return fmt.Errorf("postgres database is required")
		}
	case "scylladb":
		return fmt.Errorf("database_driver \"scylladb\" is no longer supported; use postgres only")
	default:
		return fmt.Errorf("unsupported database_driver: %s (only postgres is supported)", c.DatabaseDriver)
	}
	return nil
}

// setDefaults sets default values for optional fields.
func (c *Config) setDefaults() {
	if c.PollingIntervalSlots <= 0 {
		c.PollingIntervalSlots = 1
	}
	if c.WorkerPoolSize <= 0 {
		c.WorkerPoolSize = 10
	}
	if c.RateLimit.RequestsPerSecond <= 0 {
		c.RateLimit.RequestsPerSecond = 50
	}
	if c.RateLimit.Burst <= 0 {
		c.RateLimit.Burst = 100
	}
	if c.HTTP.TimeoutSeconds <= 0 {
		c.HTTP.TimeoutSeconds = 30
	}
	if c.HTTP.MaxIdleConns <= 0 {
		c.HTTP.MaxIdleConns = 100
	}
	if c.HTTP.MaxRetries <= 0 {
		c.HTTP.MaxRetries = 3
	}
	if c.DatabaseDriver == "" {
		c.DatabaseDriver = "postgres"
	}
	if c.Postgres.Port == 0 {
		c.Postgres.Port = 5432
	}
	if c.Postgres.SSLMode == "" {
		c.Postgres.SSLMode = "disable"
	}
	if c.Postgres.MaxConns <= 0 {
		c.Postgres.MaxConns = 10
	}
	if c.Postgres.TTLDays <= 0 {
		c.Postgres.TTLDays = 90
	}
}
