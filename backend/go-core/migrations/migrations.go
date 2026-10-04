package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
)

//go:embed *.sql
var FS embed.FS

// Run applies all pending SQL migrations in ascending numerical order.
// Uses a schema_migrations tracking table to ensure each script runs exactly once.
func Run(ctx context.Context, db *sql.DB) error {
	createTrackerSQL := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		filename VARCHAR(255) PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`

	if _, err := db.ExecContext(ctx, createTrackerSQL); err != nil {
		return fmt.Errorf("migrations.Run: create schema_migrations: %w", err)
	}

	entries, err := FS.ReadDir(".")
	if err != nil {
		return fmt.Errorf("migrations.Run: read dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, filename := range files {
		var exists bool
		checkSQL := `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename = $1);`
		if err := db.QueryRowContext(ctx, checkSQL, filename).Scan(&exists); err != nil {
			return fmt.Errorf("migrations.Run: check %s: %w", filename, err)
		}
		if exists {
			continue
		}

		content, err := FS.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("migrations.Run: read file %s: %w", filename, err)
		}

		log.Info().Str("migration", filename).Msg("applying database migration")

		if _, err := db.ExecContext(ctx, string(content)); err != nil {
			return fmt.Errorf("migrations.Run: execute %s: %w", filename, err)
		}

		recordSQL := `INSERT INTO schema_migrations (filename) VALUES ($1);`
		if _, err := db.ExecContext(ctx, recordSQL, filename); err != nil {
			return fmt.Errorf("migrations.Run: record %s: %w", filename, err)
		}

		log.Info().Str("migration", filename).Msg("database migration applied successfully")
	}

	return nil
}
