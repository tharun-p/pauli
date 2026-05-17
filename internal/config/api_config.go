package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// APIConfig is configuration for the pauli-api HTTP server.
type APIConfig struct {
	Listen         string         `yaml:"listen"`
	DatabaseDriver string         `yaml:"database_driver,omitempty"`
	Postgres       PostgresConf   `yaml:"postgres"`
	ClickHouse     ClickHouseConf `yaml:"clickhouse"`
}

// LoadAPIConfig reads API server configuration from a YAML file.
func LoadAPIConfig(path string) (*APIConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg APIConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if cfg.Listen == "" {
		cfg.Listen = ":8080"
	}
	if cfg.DatabaseDriver == "" {
		cfg.DatabaseDriver = "postgres"
	}
	cfg.Postgres.ApplyDefaults()
	cfg.ClickHouse.ApplyDefaults()

	switch cfg.DatabaseDriver {
	case "postgres":
		if err := validatePostgres(&cfg.Postgres); err != nil {
			return nil, fmt.Errorf("invalid configuration: %w", err)
		}
	case "clickhouse":
		if err := validateClickHouse(&cfg.ClickHouse); err != nil {
			return nil, fmt.Errorf("invalid configuration: %w", err)
		}
	default:
		return nil, fmt.Errorf("invalid configuration: unsupported database_driver: %s", cfg.DatabaseDriver)
	}

	return &cfg, nil
}

// StoreConfig builds a monitor-style Config for store.NewStore.
func (c *APIConfig) StoreConfig() *Config {
	return &Config{
		DatabaseDriver: c.DatabaseDriver,
		Postgres:       c.Postgres,
		ClickHouse:     c.ClickHouse,
	}
}
