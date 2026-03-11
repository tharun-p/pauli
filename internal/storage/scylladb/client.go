package scylladb

import (
	"fmt"
	"strings"
	"time"

	"github.com/gocql/gocql"
	"github.com/tharun/pauli/internal/config"
	"github.com/tharun/pauli/internal/storage"
)

// Client wraps the ScyllaDB session with configuration.
type Client struct {
	Session    *gocql.Session
	Keyspace   string
	TTLSeconds int
}

func (c *Client) HealthCheck() error {
	panic("unimplemented")
}

// Store implements storage.Store for ScyllaDB.
type Store struct {
	client *Client
	repo   storage.Repository
}

// NewStore creates a new ScyllaDB-backed Store.
func NewStore(cfg *config.ScyllaDBConf) (storage.Store, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}

	repo := NewRepository(client)

	return &Store{
		client: client,
		repo:   repo,
	}, nil
}

// RunMigrations runs database migrations for ScyllaDB.
func (s *Store) RunMigrations() error {
	return s.client.RunMigrations()
}

// HealthCheck verifies the connection to ScyllaDB is healthy.
func (s *Store) HealthCheck() error {
	return s.client.HealthCheck()
}

// Repository returns the ScyllaDB-backed Repository implementation.
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

// NewClient creates a new ScyllaDB client with the given configuration.
func NewClient(cfg *config.ScyllaDBConf) (*Client, error) {
	// Validate and parse consistency level
	consistency, err := parseConsistency(cfg.Consistency)
	if err != nil {
		return nil, fmt.Errorf("invalid ScyllaDB consistency level %q: %w", cfg.Consistency, err)
	}

	// Helper to create cluster config
	createClusterConfig := func(keyspace string) *gocql.ClusterConfig {
		cluster := gocql.NewCluster(cfg.Hosts...)
		cluster.Timeout = cfg.Timeout()
		cluster.ConnectTimeout = cfg.Timeout()
		cluster.RetryPolicy = &gocql.ExponentialBackoffRetryPolicy{
			NumRetries: cfg.MaxRetries,
			Min:        100 * time.Millisecond,
			Max:        10 * time.Second,
		}

		// Set consistency level
		cluster.Consistency = consistency

		// Enable shard-aware connection pooling (ScyllaDB specific)
		// Each session needs its own HostSelectionPolicy instance
		cluster.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(
			gocql.RoundRobinHostPolicy(),
		)

		if keyspace != "" {
			cluster.Keyspace = keyspace
		}

		return cluster
	}

	// First connect without keyspace to create it if needed
	clusterWithoutKeyspace := createClusterConfig("")
	session, err := clusterWithoutKeyspace.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ScyllaDB: %w", err)
	}

	// Create keyspace if not exists
	if err := createKeyspace(session, cfg.Keyspace, cfg.ReplicationFactor); err != nil {
		session.Close()
		return nil, err
	}
	session.Close()

	// Create new cluster config with keyspace (new HostSelectionPolicy instance)
	clusterWithKeyspace := createClusterConfig(cfg.Keyspace)
	session, err = clusterWithKeyspace.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to keyspace %s: %w", cfg.Keyspace, err)
	}

	client := &Client{
		Session:    session,
		Keyspace:   cfg.Keyspace,
		TTLSeconds: cfg.TTLSeconds(),
	}

	return client, nil
}

// Close closes the ScyllaDB session.
func (c *Client) Close() {
	if c.Session != nil {
		c.Session.Close()
	}
}

// createKeyspace creates the keyspace with SimpleStrategy replication.
func createKeyspace(session *gocql.Session, keyspace string, replicationFactor int) error {
	query := fmt.Sprintf(`
		CREATE KEYSPACE IF NOT EXISTS %s
		WITH replication = {
			'class': 'SimpleStrategy',
			'replication_factor': %d
		}
	`, keyspace, replicationFactor)

	if err := session.Query(query).Exec(); err != nil {
		return fmt.Errorf("failed to create keyspace: %w", err)
	}
	return nil
}

// parseConsistency converts a string consistency level to gocql.Consistency.
func parseConsistency(level string) (gocql.Consistency, error) {
	switch strings.ToLower(level) {
	case "any":
		return gocql.Any, nil
	case "one":
		return gocql.One, nil
	case "two":
		return gocql.Two, nil
	case "three":
		return gocql.Three, nil
	case "quorum":
		return gocql.Quorum, nil
	case "all":
		return gocql.All, nil
	case "local_quorum":
		return gocql.LocalQuorum, nil
	case "each_quorum":
		return gocql.EachQuorum, nil
	case "local_one":
		return gocql.LocalOne, nil
	default:
		return gocql.LocalQuorum, fmt.Errorf("unknown consistency level: %s", level)
	}
}
