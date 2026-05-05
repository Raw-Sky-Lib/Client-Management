package database

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var portalMigrations embed.FS

//go:embed client_migrations/*.sql
var clientMigrations embed.FS

const createMigrationsTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    filename   TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`

func MigratePortalDB(pool *pgxpool.Pool) error {
	return runMigrations(pool, portalMigrations, "migrations")
}

func MigrateClientDB(dbURL string) error {
	pool, err := Connect(dbURL)
	if err != nil {
		return fmt.Errorf("connect client db: %w", err)
	}
	defer pool.Close()
	return runMigrations(pool, clientMigrations, "client_migrations")
}

func runMigrations(pool *pgxpool.Pool, files embed.FS, dir string) error {
	ctx := context.Background()

	if _, err := pool.Exec(ctx, createMigrationsTable); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	entries, err := fs.ReadDir(files, dir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		var exists bool
		if err := pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename = $1)", name,
		).Scan(&exists); err != nil {
			return fmt.Errorf("check %s: %w", name, err)
		}
		if exists {
			continue
		}

		sql, err := files.ReadFile(dir + "/" + name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}

		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}

		if _, err := pool.Exec(ctx,
			"INSERT INTO schema_migrations (filename) VALUES ($1)", name,
		); err != nil {
			return fmt.Errorf("record %s: %w", name, err)
		}

		fmt.Printf("applied: %s\n", name)
	}

	return nil
}
