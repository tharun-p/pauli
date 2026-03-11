package sql

import "embed"

// Migrations contains all ScyllaDB CQL migration files embedded in the binary.
//
//go:embed migrations/*.sql
var Migrations embed.FS

// MigrationsPG contains all PostgreSQL migration files embedded in the binary.
//
//go:embed migrations_pg/*.sql
var MigrationsPG embed.FS
