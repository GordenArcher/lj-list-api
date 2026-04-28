package database

import (
	"context"
	"fmt"
	"log"

	"github.com/GordenArcher/lj-list-api/internal/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect opens a connection pool to PostgreSQL. We use pgxpool rather than
// a single connection so concurrent requests don't serialize on database
// access. The caller is responsible for closing the pool on shutdown.
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Verify the connection is actually alive before returning. A bad URL
	// shouldn't fail later when the first request arrives.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	log.Println("database connection pool established")
	return pool, nil
}

// RunMigrations applies any pending SQL migrations from the migrations
// directory. It uses golang-migrate under the hood. We run migrations on
// every startup, idempotent by design, so already-applied migrations
// are skipped. The server refuses to start if migrations fail because
// serving requests against a mismatched schema is a production incident
// waiting to happen.
func RunMigrations(cfg config.Config) error {
	m, err := migrate.New(
		"file://internal/database/migrations",
		cfg.DatabaseURL,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}

	if err == migrate.ErrNoChange {
		log.Println("no pending migrations")
	} else {
		log.Println("migrations applied successfully")
	}

	return nil
}
