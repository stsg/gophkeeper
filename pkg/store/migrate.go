package postgres

import (
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations
var migrations embed.FS

// migrate performs database migrations using the provided pgxpool.Pool and target version.
//
// It sets the base file system for migrations using the migrations embed.FS.
// It sets the database dialect to "postgres" using goose.SetDialect.
// It opens a database connection from the provided pgxpool.Pool using stdlib.OpenDBFromPool.
// It applies database migrations up to the target version using goose.UpTo.
// It closes the database connection using db.Close.
// It returns an error if any of the above operations fail.
//
// Parameters:
// - pool: The pgxpool.Pool used to connect to the database.
// - version: The target version up to which migrations should be applied.
//
// Returns:
// - error: An error if any of the operations fail.
func migrate(pool *pgxpool.Pool, version int64) error {
	goose.SetBaseFS(migrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("postgres migrate set dialect postgres: %w", err)
	}

	db := stdlib.OpenDBFromPool(pool)

	if err := goose.UpTo(db, "migrations", version); err != nil {
		return fmt.Errorf("postgres migrate up: %w", err)
	}

	if err := db.Close(); err != nil {
		return fmt.Errorf("postgres migrate close db: %w", err)
	}
	return nil
}
