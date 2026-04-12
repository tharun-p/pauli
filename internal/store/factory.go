package store

import (
	"github.com/tharun/pauli/internal/config"
	"github.com/tharun/pauli/internal/storage"
	"github.com/tharun/pauli/internal/storage/postgres"
)

// NewStore creates a new PostgreSQL-backed storage.Store.
func NewStore(cfg *config.Config) (storage.Store, error) {
	return postgres.NewStore(&cfg.Postgres)
}
