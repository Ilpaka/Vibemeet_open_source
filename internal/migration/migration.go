// Package migration wraps goose to apply, roll back, and inspect database
// migrations. Migration SQL files are embedded into the binary so the server
// is self-contained: no separate goose CLI, no SQL files mounted at runtime.
package migration

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var fs embed.FS

const (
	dialect = "postgres"
	dir     = "."
)

func init() {
	if err := goose.SetDialect(dialect); err != nil {
		panic(fmt.Sprintf("migration: set dialect: %v", err))
	}
	goose.SetBaseFS(fs)
}

// Apply runs every pending migration in order.
func Apply(ctx context.Context, db *sql.DB) error {
	return goose.UpContext(ctx, db, dir)
}

// ApplyOne advances the schema by exactly one migration.
func ApplyOne(ctx context.Context, db *sql.DB) error {
	return goose.UpByOneContext(ctx, db, dir)
}

// Down rolls back the most recent migration.
func Down(ctx context.Context, db *sql.DB) error {
	return goose.DownContext(ctx, db, dir)
}

// Redo rolls back the latest migration and re-applies it. Useful while
// iterating on a migration in development.
func Redo(ctx context.Context, db *sql.DB) error {
	return goose.RedoContext(ctx, db, dir)
}

// Status prints the applied/pending state of every known migration to the
// goose logger.
func Status(ctx context.Context, db *sql.DB) error {
	return goose.StatusContext(ctx, db, dir)
}

// Version returns the current schema version (0 if none applied).
func Version(ctx context.Context, db *sql.DB) (int64, error) {
	return goose.GetDBVersionContext(ctx, db)
}
