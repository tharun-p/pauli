package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration.
type Config struct {
	BeaconNodeURL        string        `yaml:"beacon_node_url"`
	Validators           []uint64      `yaml:"validators"`
	PollingIntervalSlots int           `yaml:"polling_interval_slots"`
	WorkerPoolSize       int           `yaml:"worker_pool_size"`
	RateLimit            RateLimitConf `yaml:"rate_limit"`
	HTTP                 HTTPConf      `yaml:"http"`
	ScyllaDB             ScyllaDBConf  `yaml:"scylladb"`
}

// RateLimitConf configures the rate limiter.
type RateLimitConf struct {
	RequestsPerSecond float64 `yaml:"requests_per_second"`
	Burst             int     `yaml:"burst"`
}

// HTTPConf configures the HTTP client.
type HTTPConf struct {
	TimeoutSeconds int `yaml:"timeout_seconds"`
	MaxIdleConns   int `yaml:"max_idle_conns"`
}

// ScyllaDBConf configures ScyllaDB connection.
type ScyllaDBConf struct {
	Hosts             []string `yaml:"hosts"`
	Keyspace          string   `yaml:"keyspace"`
	ReplicationFactor int      `yaml:"replication_factor"`
	Consistency       string   `yaml:"consistency"`
	TimeoutSeconds    int      `yaml:"timeout_seconds"`
	MaxRetries        int      `yaml:"max_retries"`
	TTLDays           int      `yaml:"ttl_days"`
}

// Timeout returns the HTTP timeout as a time.Duration.
func (h *HTTPConf) Timeout() time.Duration {
	return time.Duration(h.TimeoutSeconds) * time.Second
}

// Timeout returns the ScyllaDB timeout as a time.Duration.
func (s *ScyllaDBConf) Timeout() time.Duration {
	return time.Duration(s.TimeoutSeconds) * time.Second
}

// TTLSeconds returns the TTL in seconds for ScyllaDB.
func (s *ScyllaDBConf) TTLSeconds() int {
	return s.TTLDays * 24 * 60 * 60
}

// SlotDuration returns the Ethereum slot duration (12 seconds).
func SlotDuration() time.Duration {
	return 12 * time.Second
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
	if len(c.ScyllaDB.Hosts) == 0 {
		return fmt.Errorf("at least one scylladb host is required")
	}
	if c.ScyllaDB.Keyspace == "" {
		return fmt.Errorf("scylladb keyspace is required")
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
	if c.ScyllaDB.ReplicationFactor <= 0 {
		c.ScyllaDB.ReplicationFactor = 3
	}
	if c.ScyllaDB.Consistency == "" {
		c.ScyllaDB.Consistency = "local_quorum"
	}
	if c.ScyllaDB.TimeoutSeconds <= 0 {
		c.ScyllaDB.TimeoutSeconds = 10
	}
	if c.ScyllaDB.MaxRetries <= 0 {
		c.ScyllaDB.MaxRetries = 3
	}
	if c.ScyllaDB.TTLDays <= 0 {
		c.ScyllaDB.TTLDays = 90
	}
}
