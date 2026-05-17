package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/tharun/pauli/internal/config"
)

// Client wraps a ClickHouse connection with configuration.
type Client struct {
	Conn     driver.Conn
	Database string
	TTLDays  int
}

// NewClient opens a ClickHouse connection from configuration.
func NewClient(cfg *config.ClickHouseConf) (*Client, error) {
	opts := &clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)},
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.User,
			Password: cfg.Password,
		},
		MaxOpenConns: cfg.MaxConns,
		MaxIdleConns: cfg.MaxConns,
	}
	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open clickhouse: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := conn.Ping(ctx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to ping clickhouse: %w", err)
	}

	return &Client{
		Conn:     conn,
		Database: cfg.Database,
		TTLDays:  cfg.TTLDays,
	}, nil
}

// Close closes the ClickHouse connection.
func (c *Client) Close() error {
	if c.Conn != nil {
		return c.Conn.Close()
	}
	return nil
}

// HealthCheck verifies the connection to ClickHouse is healthy.
func (c *Client) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.Conn.Ping(ctx); err != nil {
		return fmt.Errorf("clickhouse health check failed: %w", err)
	}
	return nil
}

// ExecContext runs a statement without returning rows.
func (c *Client) ExecContext(ctx context.Context, query string, args ...any) error {
	return c.Conn.Exec(ctx, query, args...)
}

// QueryContext runs a query and returns rows.
func (c *Client) QueryContext(ctx context.Context, query string, args ...any) (driver.Rows, error) {
	return c.Conn.Query(ctx, query, args...)
}

// QueryRowContext runs a query expected to return at most one row.
func (c *Client) QueryRowContext(ctx context.Context, query string, args ...any) driver.Row {
	return c.Conn.QueryRow(ctx, query, args...)
}
