package postgres

import (
	"context"
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

// RunMigrations loads and executes all pending PostgreSQL migrations.
func (c *Client) RunMigrations() error {
	log.Info().Msg("Running PostgreSQL migrations...")

	// Load migrations from embedded files
	migrations, err := loadMigrationsPG()
	if err != nil {
		return fmt.Errorf("failed to load postgres migrations: %w", err)
	}

	log.Info().Int("total", len(migrations)).Msg("Loaded postgres migration files")

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

	if appliedCount > 0 {
		log.Info().Int("applied", appliedCount).Msg("PostgreSQL migrations applied successfully")
	} else {
		log.Info().Msg("No new PostgreSQL migrations to apply")
	}

	return nil
}

// loadMigrationsPG reads all SQL files from the embedded PostgreSQL filesystem.
func loadMigrationsPG() ([]*Migration, error) {
	var migrations []*Migration

	err := fs.WalkDir(sql.MigrationsPG, "migrations_pg", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".sql") {
			return nil
		}

		content, err := fs.ReadFile(sql.MigrationsPG, path)
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
	const query = `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version     TEXT PRIMARY KEY,
			name        TEXT NOT NULL,
			applied_at  TIMESTAMPTZ NOT NULL,
			checksum    TEXT NOT NULL
		)
	`
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := c.Pool.Exec(ctx, query); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}
	return nil
}

// getAppliedMigrations returns a map of already applied migration versions.
func (c *Client) getAppliedMigrations() (map[string]*MigrationRecord, error) {
	const query = `SELECT version, name, applied_at, checksum FROM schema_migrations`

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := c.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]*MigrationRecord)

	for rows.Next() {
		var record MigrationRecord
		if err := rows.Scan(&record.Version, &record.Name, &record.AppliedAt, &record.Checksum); err != nil {
			return nil, fmt.Errorf("failed to scan migration record: %w", err)
		}
		r := record
		applied[r.Version] = &r
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate migration records: %w", err)
	}

	return applied, nil
}

// applyMigration executes a single migration.
func (c *Client) applyMigration(m *Migration) error {
	log.Info().
		Str("version", m.Version).
		Str("name", m.Name).
		Msg("Applying postgres migration")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	tx, err := c.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin migration transaction: %w", err)
	}
	defer tx.Rollback(ctx) // safe to call even after commit

	// Execute the migration SQL
	statements := splitStatements(m.SQL)
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}

		if _, err := tx.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("migration %s failed: %w", m.Version, err)
		}
	}

	// Record the migration
	const recordQuery = `
		INSERT INTO schema_migrations (version, name, applied_at, checksum)
		VALUES ($1, $2, $3, $4)
	`
	if _, err := tx.Exec(ctx, recordQuery, m.Version, m.Name, time.Now().UTC(), m.Checksum); err != nil {
		return fmt.Errorf("failed to record migration %s: %w", m.Version, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit migration %s: %w", m.Version, err)
	}

	log.Info().
		Str("version", m.Version).
		Str("name", m.Name).
		Msg("Postgres migration applied successfully")

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
