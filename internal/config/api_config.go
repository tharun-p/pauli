package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// APIConfig is configuration for the pauli-api HTTP server (Postgres only).
type APIConfig struct {
	Listen   string       `yaml:"listen"`
	Postgres PostgresConf `yaml:"postgres"`
}

// LoadAPIConfig reads API server configuration from a YAML file.
// Postgres defaults are applied before validation so listen and port can be omitted.
func LoadAPIConfig(path string) (*APIConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg APIConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	cfg.Postgres.ApplyDefaults()
	if cfg.Listen == "" {
		cfg.Listen = ":8080"
	}

	if err := validatePostgres(&cfg.Postgres); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}
