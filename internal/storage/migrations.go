package storage

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

// migrations contains all table creation statements.
// TTL is set dynamically based on configuration.
var migrations = []string{
	`CREATE TABLE IF NOT EXISTS validator_snapshots (
		validator_index BIGINT,
		slot            BIGINT,
		status          TEXT,
		balance         BIGINT,
		effective_balance BIGINT,
		timestamp       TIMESTAMP,
		PRIMARY KEY ((validator_index), slot)
	) WITH CLUSTERING ORDER BY (slot DESC)`,

	`CREATE TABLE IF NOT EXISTS attestation_duties (
		validator_index   BIGINT,
		epoch             BIGINT,
		slot              BIGINT,
		committee_index   INT,
		committee_position INT,
		timestamp         TIMESTAMP,
		PRIMARY KEY ((validator_index), epoch, slot)
	) WITH CLUSTERING ORDER BY (epoch DESC, slot DESC)`,

	`CREATE TABLE IF NOT EXISTS attestation_rewards (
		validator_index BIGINT,
		epoch           BIGINT,
		head_reward     BIGINT,
		source_reward   BIGINT,
		target_reward   BIGINT,
		total_reward    BIGINT,
		timestamp       TIMESTAMP,
		PRIMARY KEY ((validator_index), epoch)
	) WITH CLUSTERING ORDER BY (epoch DESC)`,

	`CREATE TABLE IF NOT EXISTS validator_penalties (
		validator_index BIGINT,
		epoch           BIGINT,
		slot            BIGINT,
		penalty_type    TEXT,
		penalty_gwei    BIGINT,
		timestamp       TIMESTAMP,
		PRIMARY KEY ((validator_index), epoch, slot)
	) WITH CLUSTERING ORDER BY (epoch DESC, slot DESC)`,
}

// tableNames for TTL updates
var tableNames = []string{
	"validator_snapshots",
	"attestation_duties",
	"attestation_rewards",
	"validator_penalties",
}

// RunMigrations creates all required tables and sets TTL.
func (c *Client) RunMigrations() error {
	log.Info().Msg("Running ScyllaDB migrations...")

	for _, migration := range migrations {
		if err := c.Session.Query(migration).Exec(); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	// Set TTL on all tables
	for _, table := range tableNames {
		if err := c.setTableTTL(table); err != nil {
			return err
		}
	}

	log.Info().Msg("ScyllaDB migrations completed successfully")
	return nil
}

// setTableTTL sets the default TTL for a table.
func (c *Client) setTableTTL(table string) error {
	query := fmt.Sprintf(
		"ALTER TABLE %s WITH default_time_to_live = %d",
		table, c.TTLSeconds,
	)

	if err := c.Session.Query(query).Exec(); err != nil {
		return fmt.Errorf("failed to set TTL on %s: %w", table, err)
	}

	log.Debug().
		Str("table", table).
		Int("ttl_seconds", c.TTLSeconds).
		Msg("Set table TTL")

	return nil
}
