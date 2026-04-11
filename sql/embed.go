package sql

import "embed"

// MigrationsPG contains all PostgreSQL migration files embedded in the binary.
//
//go:embed migrations_pg/*.sql
var MigrationsPG embed.FS
