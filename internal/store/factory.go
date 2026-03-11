package store

import (
	"fmt"

	"github.com/tharun/pauli/internal/config"
	"github.com/tharun/pauli/internal/storage"
	"github.com/tharun/pauli/internal/storage/postgres"
	"github.com/tharun/pauli/internal/storage/scylladb"
)

// NewStore creates a new storage.Store based on the configured database driver.
func NewStore(cfg *config.Config) (storage.Store, error) {
	switch cfg.DatabaseDriver {
	case "postgres":
		return postgres.NewStore(&cfg.Postgres)
	case "scylladb", "":
		return scylladb.NewStore(&cfg.ScyllaDB)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.DatabaseDriver)
	}
}
