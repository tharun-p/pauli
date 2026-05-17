package store

import (
	"fmt"

	"github.com/tharun/pauli/internal/config"
	"github.com/tharun/pauli/internal/storage"
	"github.com/tharun/pauli/internal/storage/clickhouse"
	"github.com/tharun/pauli/internal/storage/postgres"
)

// NewStore creates a storage.Store for the configured database driver.
func NewStore(cfg *config.Config) (storage.Store, error) {
	switch cfg.DatabaseDriver {
	case "", "postgres":
		return postgres.NewStore(&cfg.Postgres)
	case "clickhouse":
		return clickhouse.NewStore(&cfg.ClickHouse)
	default:
		return nil, fmt.Errorf("unsupported database_driver: %s", cfg.DatabaseDriver)
	}
}
