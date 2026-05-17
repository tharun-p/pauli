package clickhouse

import (
	"github.com/tharun/pauli/internal/config"
	"github.com/tharun/pauli/internal/storage"
)

// Store implements storage.Store for ClickHouse.
type Store struct {
	client *Client
	repo   storage.Repository
}

// NewStore creates a new ClickHouse-backed Store.
func NewStore(cfg *config.ClickHouseConf) (storage.Store, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}

	repo, err := NewRepository(client)
	if err != nil {
		_ = client.Close()
		return nil, err
	}

	return &Store{
		client: client,
		repo:   repo,
	}, nil
}

// RunMigrations runs database migrations for ClickHouse.
func (s *Store) RunMigrations() error {
	return s.client.RunMigrations()
}

// HealthCheck verifies the connection to ClickHouse is healthy.
func (s *Store) HealthCheck() error {
	return s.client.HealthCheck()
}

// Repository returns the ClickHouse-backed Repository implementation.
func (s *Store) Repository() storage.Repository {
	return s.repo
}

// Close closes underlying resources.
func (s *Store) Close() {
	if s.repo != nil {
		_ = s.repo.Close()
	}
	if s.client != nil {
		_ = s.client.Close()
	}
}
