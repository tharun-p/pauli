package clickhouse

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

// RunMigrations loads and executes all pending ClickHouse migrations.
func (c *Client) RunMigrations() error {
	log.Debug().Msg("Running ClickHouse migrations...")

	migrations, err := loadMigrationsCH()
	if err != nil {
		return fmt.Errorf("failed to load clickhouse migrations: %w", err)
	}

	log.Debug().Int("total", len(migrations)).Msg("Loaded clickhouse migration files")

	if err := c.ensureMigrationsTable(); err != nil {
		return err
	}

	applied, err := c.getAppliedMigrations()
	if err != nil {
		return err
	}

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
		log.Debug().Int("applied", appliedCount).Msg("ClickHouse migrations applied successfully")
	} else {
		log.Debug().Msg("No new ClickHouse migrations to apply")
	}

	return nil
}

func loadMigrationsCH() ([]*Migration, error) {
	var migrations []*Migration

	err := fs.WalkDir(sql.MigrationsCH, "migrations_ch", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".sql") {
			return nil
		}

		content, err := fs.ReadFile(sql.MigrationsCH, path)
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

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

func parseMigrationFilename(filename string) (version, name string) {
	base := strings.TrimSuffix(filename, ".sql")
	parts := strings.SplitN(base, "_", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return base, base
}

func (c *Client) ensureMigrationsTable() error {
	const query = `
		CREATE TABLE IF NOT EXISTS schema_migrations
		(
			version    String,
			name       String,
			applied_at DateTime64(3),
			checksum   String
		)
		ENGINE = ReplacingMergeTree(applied_at)
		ORDER BY version
		SETTINGS storage_policy = 'default'
	`
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := c.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}
	return nil
}

func (c *Client) getAppliedMigrations() (map[string]*MigrationRecord, error) {
	const query = `SELECT version, name, applied_at, checksum FROM schema_migrations FINAL`

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rows, err := c.QueryContext(ctx, query)
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

func (c *Client) applyMigration(m *Migration) error {
	log.Debug().Str("version", m.Version).Str("name", m.Name).Msg("Applying clickhouse migration")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	for _, stmt := range splitStatements(m.SQL) {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}
		if err := c.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migration %s failed: %w", m.Version, err)
		}
	}

	const recordQuery = `
		INSERT INTO schema_migrations (version, name, applied_at, checksum)
		VALUES (?, ?, ?, ?)
	`
	if err := c.ExecContext(ctx, recordQuery, m.Version, m.Name, time.Now().UTC(), m.Checksum); err != nil {
		return fmt.Errorf("failed to record migration %s: %w", m.Version, err)
	}

	log.Debug().Str("version", m.Version).Str("name", m.Name).Msg("ClickHouse migration applied successfully")
	return nil
}

func splitStatements(sqlText string) []string {
	var statements []string
	var current strings.Builder
	inComment := false

	lines := strings.Split(sqlText, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
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

	if remaining := strings.TrimSpace(current.String()); remaining != "" {
		statements = append(statements, remaining)
	}
	return statements
}
