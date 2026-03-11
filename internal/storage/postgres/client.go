package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tharun/pauli/internal/config"
	"github.com/tharun/pauli/internal/storage"
)

// Client wraps the PostgreSQL connection pool with configuration.
type Client struct {
	Pool    *pgxpool.Pool
	TTLDays int
}

// Store implements storage.Store for PostgreSQL.
type Store struct {
	client *Client
	repo   storage.Repository
}

// NewStore creates a new PostgreSQL-backed Store.
func NewStore(cfg *config.PostgresConf) (storage.Store, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}

	repo, err := NewRepository(client)
	if err != nil {
		client.Close()
		return nil, err
	}

	return &Store{
		client: client,
		repo:   repo,
	}, nil
}

// RunMigrations runs database migrations for PostgreSQL.
func (s *Store) RunMigrations() error {
	return s.client.RunMigrations()
}

// HealthCheck verifies the connection to PostgreSQL is healthy.
func (s *Store) HealthCheck() error {
	return s.client.HealthCheck()
}

// Repository returns the PostgreSQL-backed Repository implementation.
func (s *Store) Repository() storage.Repository {
	return s.repo
}

// Close closes underlying resources.
func (s *Store) Close() {
	if s.repo != nil {
		_ = s.repo.Close()
	}
	if s.client != nil {
		s.client.Close()
	}
}

// NewClient creates a new PostgreSQL client with the given configuration.
func NewClient(cfg *config.PostgresConf) (*Client, error) {
	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.SSLMode,
	)

	pgxCfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres config: %w", err)
	}

	if cfg.MaxConns > 0 {
		pgxCfg.MaxConns = cfg.MaxConns
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, pgxCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	client := &Client{
		Pool:    pool,
		TTLDays: cfg.TTLDays,
	}

	return client, nil
}

// Close closes the PostgreSQL pool.
func (c *Client) Close() {
	if c.Pool != nil {
		c.Pool.Close()
	}
}

// HealthCheck verifies the connection to PostgreSQL is healthy.
func (c *Client) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var one int
	if err := c.Pool.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil {
		return fmt.Errorf("postgres health check failed: %w", err)
	}
	return nil
}
