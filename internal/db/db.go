package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

type DB struct {
	Pool *pgxpool.Pool
}

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

func Connect(cfg Config) (*DB, error) {
	connString := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	)

	pool, err := pgxpool.Connect(context.Background(), connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

func (db *DB) RunMigrations(migrationsDir string) error {
	log.Printf("Running migrations from %s", migrationsDir)

	_, err := db.Pool.Exec(context.Background(), `
        CREATE TABLE IF NOT EXISTS migrations (
            id SERIAL PRIMARY KEY,
            name VARCHAR(255) NOT NULL UNIQUE,
            applied_at TIMESTAMP NOT NULL DEFAULT NOW()
        )
    `)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	appliedMigrations := make(map[string]bool)
	rows, err := db.Pool.Query(context.Background(), "SELECT name FROM migrations")
	if err != nil {
		return fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("failed to scan migration row: %w", err)
		}
		appliedMigrations[name] = true
	}

	if rows.Err() != nil {
		return fmt.Errorf("error iterating migrations rows: %w", rows.Err())
	}

	migrationFiles, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return fmt.Errorf("failed to list migration files: %w", err)
	}

	sort.Strings(migrationFiles)

	for _, file := range migrationFiles {
		filename := filepath.Base(file)

		if appliedMigrations[filename] {
			log.Printf("Migration %s already applied, skipping", filename)
			continue
		}

		log.Printf("Applying migration: %s", filename)

		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		tx, err := db.Pool.Begin(context.Background())
		if err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}

		if _, err := tx.Exec(context.Background(), string(content)); err != nil {
			tx.Rollback(context.Background())
			return fmt.Errorf("failed to execute migration %s: %w", filename, err)
		}

		if _, err := tx.Exec(context.Background(),
			"INSERT INTO migrations (name) VALUES ($1)", filename); err != nil {
			tx.Rollback(context.Background())
			return fmt.Errorf("failed to record migration %s: %w", filename, err)
		}

		if err := tx.Commit(context.Background()); err != nil {
			return fmt.Errorf("failed to commit migration transaction: %w", err)
		}

		log.Printf("Successfully applied migration: %s", filename)
	}

	return nil
}
