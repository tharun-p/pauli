package scylladb

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/tharun/pauli/sql"
)

// Migration represents a database migration.
type Migration struct {
	Version  string
	Name     string
	SQL      string
	Checksum string
}

// MigrationRecord represents an applied migration in the database.
type MigrationRecord struct {
	Version   string
	Name      string
	AppliedAt time.Time
	Checksum  string
}

// tableNames for TTL updates
var tableNames = []string{
	"validator_snapshots",
	"attestation_duties",
	"attestation_rewards",
	"validator_penalties",
}

// RunMigrations loads and executes all pending migrations.
func (c *Client) RunMigrations() error {
	log.Info().Msg("Running ScyllaDB migrations...")

	// Load migrations from embedded files
	migrations, err := loadMigrations()
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	log.Info().Int("total", len(migrations)).Msg("Loaded migration files")

	// Ensure schema_migrations table exists (bootstrap)
	if err := c.ensureMigrationsTable(); err != nil {
		return err
	}

	// Get already applied migrations
	applied, err := c.getAppliedMigrations()
	if err != nil {
		return err
	}

	// Apply pending migrations
	appliedCount := 0
	for _, m := range migrations {
		if _, exists := applied[m.Version]; exists {
			log.Debug().Str("version", m.Version).Str("name", m.Name).Msg("Migration already applied")
			continue
		}

		if err := c.applyMigration(m); err != nil {
			return err
		}
		appliedCount++
	}

	// Apply dynamic TTL settings
	if err := c.applyTTL(); err != nil {
		return err
	}

	if appliedCount > 0 {
		log.Info().Int("applied", appliedCount).Msg("Migrations applied successfully")
	} else {
		log.Info().Msg("No new migrations to apply")
	}

	return nil
}

// loadMigrations reads all SQL files from the embedded filesystem.
func loadMigrations() ([]*Migration, error) {
	var migrations []*Migration

	err := fs.WalkDir(sql.Migrations, "migrations", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".sql") {
			return nil
		}

		content, err := fs.ReadFile(sql.Migrations, path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		filename := filepath.Base(path)
		version, name := parseMigrationFilename(filename)

		hash := sha256.Sum256(content)
		checksum := hex.EncodeToString(hash[:])

		migrations = append(migrations, &Migration{
			Version:  version,
			Name:     name,
			SQL:      string(content),
			Checksum: checksum,
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// parseMigrationFilename extracts version and name from filename.
// Example: "001_create_validator_snapshots.sql" -> ("001", "create_validator_snapshots")
func parseMigrationFilename(filename string) (version, name string) {
	// Remove .sql extension
	base := strings.TrimSuffix(filename, ".sql")

	// Split on first underscore
	parts := strings.SplitN(base, "_", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return base, base
}

// ensureMigrationsTable creates the schema_migrations table if it doesn't exist.
func (c *Client) ensureMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version     TEXT,
			name        TEXT,
			applied_at  TIMESTAMP,
			checksum    TEXT,
			PRIMARY KEY (version)
		)
	`
	if err := c.Session.Query(query).Exec(); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}
	return nil
}

// getAppliedMigrations returns a map of already applied migration versions.
func (c *Client) getAppliedMigrations() (map[string]*MigrationRecord, error) {
	query := `SELECT version, name, applied_at, checksum FROM schema_migrations`

	iter := c.Session.Query(query).Iter()
	applied := make(map[string]*MigrationRecord)

	var record MigrationRecord
	for iter.Scan(&record.Version, &record.Name, &record.AppliedAt, &record.Checksum) {
		r := record
		applied[r.Version] = &r
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	return applied, nil
}

// applyMigration executes a single migration.
func (c *Client) applyMigration(m *Migration) error {
	log.Info().
		Str("version", m.Version).
		Str("name", m.Name).
		Msg("Applying migration")

	// Execute the migration SQL
	// Split by semicolons for multi-statement migrations
	statements := splitStatements(m.SQL)
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}

		if err := c.Session.Query(stmt).Exec(); err != nil {
			return fmt.Errorf("migration %s failed: %w", m.Version, err)
		}
	}

	// Record the migration
	recordQuery := `
		INSERT INTO schema_migrations (version, name, applied_at, checksum)
		VALUES (?, ?, ?, ?)
	`
	if err := c.Session.Query(recordQuery, m.Version, m.Name, time.Now().UTC(), m.Checksum).Exec(); err != nil {
		return fmt.Errorf("failed to record migration %s: %w", m.Version, err)
	}

	log.Info().
		Str("version", m.Version).
		Str("name", m.Name).
		Msg("Migration applied successfully")

	return nil
}

// splitStatements splits SQL content by semicolons, handling comments.
func splitStatements(sqlText string) []string {
	var statements []string
	var current strings.Builder
	inComment := false

	lines := strings.Split(sqlText, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip comment lines
		if strings.HasPrefix(trimmed, "--") {
			continue
		}

		// Check for block comments
		if strings.Contains(line, "/*") {
			inComment = true
		}
		if strings.Contains(line, "*/") {
			inComment = false
			continue
		}
		if inComment {
			continue
		}

		current.WriteString(line)
		current.WriteString("\n")

		if strings.HasSuffix(trimmed, ";") {
			stmt := strings.TrimSpace(current.String())
			stmt = strings.TrimSuffix(stmt, ";")
			if stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
		}
	}

	// Handle statement without trailing semicolon
	if remaining := strings.TrimSpace(current.String()); remaining != "" {
		statements = append(statements, remaining)
	}

	return statements
}

// applyTTL sets the TTL on all tables based on configuration.
func (c *Client) applyTTL() error {
	log.Debug().Int("ttl_seconds", c.TTLSeconds).Msg("Applying TTL to tables")

	for _, table := range tableNames {
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
	}

	return nil
}

// GetMigrationStatus returns the status of all migrations.
func (c *Client) GetMigrationStatus() ([]MigrationStatus, error) {
	migrations, err := loadMigrations()
	if err != nil {
		return nil, err
	}

	applied, err := c.getAppliedMigrations()
	if err != nil {
		return nil, err
	}

	var status []MigrationStatus
	for _, m := range migrations {
		s := MigrationStatus{
			Version: m.Version,
			Name:    m.Name,
		}

		if record, exists := applied[m.Version]; exists {
			s.Applied = true
			s.AppliedAt = record.AppliedAt
			s.Checksum = record.Checksum
			s.ChecksumMatch = record.Checksum == m.Checksum
		}

		status = append(status, s)
	}

	return status, nil
}

// MigrationStatus represents the status of a migration.
type MigrationStatus struct {
	Version       string
	Name          string
	Applied       bool
	AppliedAt     time.Time
	Checksum      string
	ChecksumMatch bool
}
