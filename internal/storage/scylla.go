package storage

import (
	"fmt"
	"strings"
	"time"

	"github.com/gocql/gocql"
	"github.com/tharun/pauli/internal/config"
)

// Client wraps the ScyllaDB session with configuration.
type Client struct {
	Session    *gocql.Session
	Keyspace   string
	TTLSeconds int
}

// NewClient creates a new ScyllaDB client with the given configuration.
func NewClient(cfg *config.ScyllaDBConf) (*Client, error) {
	// Create cluster configuration
	cluster := gocql.NewCluster(cfg.Hosts...)
	cluster.Timeout = cfg.Timeout()
	cluster.ConnectTimeout = cfg.Timeout()
	cluster.RetryPolicy = &gocql.ExponentialBackoffRetryPolicy{
		NumRetries: cfg.MaxRetries,
		Min:        100 * time.Millisecond,
		Max:        10 * time.Second,
	}

	// Set consistency level
	consistency, err := parseConsistency(cfg.Consistency)
	if err != nil {
		return nil, err
	}
	cluster.Consistency = consistency

	// Enable shard-aware connection pooling (ScyllaDB specific)
	cluster.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(
		gocql.RoundRobinHostPolicy(),
	)

	// First connect without keyspace to create it if needed
	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ScyllaDB: %w", err)
	}

	// Create keyspace if not exists
	if err := createKeyspace(session, cfg.Keyspace, cfg.ReplicationFactor); err != nil {
		session.Close()
		return nil, err
	}
	session.Close()

	// Reconnect with keyspace
	cluster.Keyspace = cfg.Keyspace
	session, err = cluster.CreateSession()
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

// HealthCheck verifies the connection to ScyllaDB is healthy.
func (c *Client) HealthCheck() error {
	var result int
	if err := c.Session.Query("SELECT 1").Scan(&result); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	return nil
}
