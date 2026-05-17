package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration.
type Config struct {
	BeaconNodeURL string `yaml:"beacon_node_url"`
	BeaconAPIKey  string `yaml:"beacon_api_key,omitempty"` // Optional API key for providers like Tatum
	// ExecutionNodeURL is optional JSON-RPC URL (e.g. http://localhost:8545). When set, the monitor
	// fetches execution-layer priority fees for proposed blocks via eth_getBlockByNumber + eth_getBlockReceipts.
	ExecutionNodeURL string `yaml:"execution_node_url,omitempty"`
	// ExecutionAPIKey is optional; how it is sent depends on execution_auth_header (default: Bearer).
	ExecutionAPIKey string `yaml:"execution_api_key,omitempty"`
	// ExecutionAuthHeader selects how execution_api_key is attached: "bearer" (default, Authorization: Bearer <key>),
	// "x_api_key" (x-api-key header), "authorization" (raw Authorization value, no Bearer prefix), "token" (Token: <key>),
	// or "none" / "off" to send no auth headers (bare JSON-RPC), even if execution_api_key is set.
	ExecutionAuthHeader string `yaml:"execution_auth_header,omitempty"`
	Validators          []uint64 `yaml:"validators"`
	PollingIntervalSlots int      `yaml:"polling_interval_slots"`
	// SlotDurationSeconds allows overriding the default 12s slot duration.
	// For local devnets (e.g. kurtosis) you can set this to 2.
	SlotDurationSeconds int           `yaml:"slot_duration_seconds,omitempty"`
	WorkerPoolSize      int           `yaml:"worker_pool_size"`
	RateLimit           RateLimitConf `yaml:"rate_limit"`
	HTTP                HTTPConf      `yaml:"http"`
	// DatabaseDriver selects storage: "postgres" (default) or "clickhouse".
	DatabaseDriver string         `yaml:"database_driver,omitempty"`
	Postgres       PostgresConf   `yaml:"postgres"`
	ClickHouse     ClickHouseConf `yaml:"clickhouse"`
	Backfill       BackfillConf   `yaml:"backfill"`
}

// BackfillConf configures the historical backfill runner (slot + epoch tracks).
type BackfillConf struct {
	Enabled       bool    `yaml:"enabled"`
	StartSlot     uint64  `yaml:"start_slot"`
	StartEpoch    uint64  `yaml:"start_epoch"`
	EndSlot       *uint64 `yaml:"end_slot,omitempty"`
	EndEpoch      *uint64 `yaml:"end_epoch,omitempty"`
	LagBehindHead uint64  `yaml:"lag_behind_head"`
	SlotsPerPass  int     `yaml:"slots_per_pass"`
	EpochsPerPass int     `yaml:"epochs_per_pass"`
	PollDelayMs      int `yaml:"poll_delay_ms"`
	IdlePollDelayMs  int `yaml:"idle_poll_delay_ms"`
}

// PollDelay returns pacing between backfill passes while catching up.
func (b *BackfillConf) PollDelay() time.Duration {
	if b.PollDelayMs <= 0 {
		return 100 * time.Millisecond
	}
	return time.Duration(b.PollDelayMs) * time.Millisecond
}

// IdlePollDelay returns pacing between backfill passes when fully caught up.
func (b *BackfillConf) IdlePollDelay() time.Duration {
	if b.IdlePollDelayMs <= 0 {
		return 12 * time.Second
	}
	return time.Duration(b.IdlePollDelayMs) * time.Millisecond
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

// ApplyDefaults sets default values for optional Postgres fields.
func (p *PostgresConf) ApplyDefaults() {
	if p.Port == 0 {
		p.Port = 5432
	}
	if p.SSLMode == "" {
		p.SSLMode = "disable"
	}
	if p.MaxConns <= 0 {
		p.MaxConns = 10
	}
	if p.TTLDays <= 0 {
		p.TTLDays = 90
	}
}

// ClickHouseConf configures the ClickHouse connection (monitor and API).
// Hot retention (ttl_days) is enforced via table TTL in sql/migrations_ch (default 90 days, then DELETE).
type ClickHouseConf struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	MaxConns int    `yaml:"max_conns"`
	TTLDays  int    `yaml:"ttl_days"`
}

// ApplyDefaults sets default values for optional ClickHouse fields.
func (c *ClickHouseConf) ApplyDefaults() {
	if c.Port == 0 {
		c.Port = 9000
	}
	if c.User == "" {
		c.User = "default"
	}
	if c.Database == "" {
		c.Database = "pauli"
	}
	if c.MaxConns <= 0 {
		c.MaxConns = 10
	}
	if c.TTLDays <= 0 {
		c.TTLDays = 90
	}
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

func validatePostgres(p *PostgresConf) error {
	if p.Host == "" {
		return fmt.Errorf("postgres host is required")
	}
	if p.Port == 0 {
		return fmt.Errorf("postgres port is required")
	}
	if p.User == "" {
		return fmt.Errorf("postgres user is required")
	}
	if p.Database == "" {
		return fmt.Errorf("postgres database is required")
	}
	return nil
}

func validateClickHouse(c *ClickHouseConf) error {
	if c.Host == "" {
		return fmt.Errorf("clickhouse host is required")
	}
	if c.Port == 0 {
		return fmt.Errorf("clickhouse port is required")
	}
	if c.Database == "" {
		return fmt.Errorf("clickhouse database is required")
	}
	return nil
}

// validate checks the configuration for required fields.
func (c *Config) validate() error {
	if c.BeaconNodeURL == "" {
		return fmt.Errorf("beacon_node_url is required")
	}
	// validators is optional: network-wide epoch indexing does not use it for RPC.
	switch c.DatabaseDriver {
	case "", "postgres":
		if err := validatePostgres(&c.Postgres); err != nil {
			return err
		}
	case "clickhouse":
		if err := validateClickHouse(&c.ClickHouse); err != nil {
			return err
		}
	case "scylladb":
		return fmt.Errorf("database_driver \"scylladb\" is no longer supported; use postgres or clickhouse")
	default:
		return fmt.Errorf("unsupported database_driver: %s (use postgres or clickhouse)", c.DatabaseDriver)
	}
	return nil
}

// setDefaults sets default values for optional fields.
func (c *Config) setDefaults() {
	if c.PollingIntervalSlots <= 0 {
		c.PollingIntervalSlots = 32
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
	c.Postgres.ApplyDefaults()
	c.ClickHouse.ApplyDefaults()
	c.Backfill.setDefaults()
}

func (b *BackfillConf) setDefaults() {
	if b.LagBehindHead == 0 {
		b.LagBehindHead = 4
	}
	if b.SlotsPerPass <= 0 {
		b.SlotsPerPass = 8
	}
	if b.EpochsPerPass <= 0 {
		b.EpochsPerPass = 2
	}
	if b.PollDelayMs <= 0 {
		b.PollDelayMs = 100
	}
	if b.IdlePollDelayMs <= 0 {
		b.IdlePollDelayMs = 12000
	}
}
