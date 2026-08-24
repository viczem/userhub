// Package migrations embeds and applies IAM's PostgreSQL schema migrations.
package migrations

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	migrationpgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	// Register the pgx driver for database/sql.
	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed *.sql
var files embed.FS

var migrationNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:_[a-z0-9]+)*$`)

// Up applies all pending IAM migrations to the configured PostgreSQL database.
func Up(databaseURL string) error {
	migrator, err := newMigrator(databaseURL)
	if err != nil {
		return err
	}
	defer func() { _, _ = migrator.Close() }()

	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}

// Down rolls back the most recently applied IAM migration.
func Down(databaseURL string) error {
	migrator, err := newMigrator(databaseURL)
	if err != nil {
		return err
	}
	defer func() { _, _ = migrator.Close() }()

	if err := migrator.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("roll back migration: %w", err)
	}

	return nil
}

// Create creates the next sequential up/down migration pair in directory.
func Create(directory, name string) error {
	name = strings.TrimSpace(name)
	if !migrationNamePattern.MatchString(name) {
		return errors.New("migration name must contain lowercase letters, digits, and single underscores")
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("read migrations directory: %w", err)
	}

	maxVersion := 0

	for _, entry := range entries {
		version, _, ok := strings.Cut(entry.Name(), "_")
		if !ok {
			continue
		}

		value, err := strconv.Atoi(version)
		if err == nil && value > maxVersion {
			maxVersion = value
		}
	}

	prefix := fmt.Sprintf("%06d_%s", maxVersion+1, name)
	created := make([]string, 0, 2)

	for _, suffix := range []string{".up.sql", ".down.sql"} {
		path := filepath.Join(directory, prefix+suffix)

		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			for _, createdPath := range created {
				_ = os.Remove(createdPath)
			}

			return fmt.Errorf("create migration %s: %w", path, err)
		}

		if err := file.Close(); err != nil {
			_ = os.Remove(path)
			for _, createdPath := range created {
				_ = os.Remove(createdPath)
			}

			return fmt.Errorf("close migration %s: %w", path, err)
		}

		created = append(created, path)
	}

	return nil
}

func newMigrator(databaseURL string) (*migrate.Migrate, error) {
	source, err := iofs.New(files, ".")
	if err != nil {
		return nil, fmt.Errorf("load migrations: %w", err)
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		_ = source.Close()
		return nil, fmt.Errorf("open migration database: %w", err)
	}

	driver, err := migrationpgx.WithInstance(db, &migrationpgx.Config{})
	if err != nil {
		_ = source.Close()
		_ = db.Close()

		return nil, fmt.Errorf("initialize migration database: %w", err)
	}

	migrator, err := migrate.NewWithInstance("iofs", source, "pgx5", driver)
	if err != nil {
		_ = source.Close()
		_ = driver.Close()

		return nil, fmt.Errorf("initialize migrator: %w", err)
	}

	return migrator, nil
}
