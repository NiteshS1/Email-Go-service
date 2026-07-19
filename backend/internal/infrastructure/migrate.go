package infrastructure

import (
	"fmt"
	"path/filepath"

	"github.com/emailservice/internal/config"
	migrate "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

// RunMigrations runs all pending SQL migrations from the migrations directory.
func RunMigrations(cfg *config.Config) error {
	absPath, err := filepath.Abs("migrations")
	if err != nil {
		return fmt.Errorf("migrations path: %w", err)
	}

	sourceURL := "file://" + absPath
	m, err := migrate.New(sourceURL, DSN(cfg))
	if err != nil {
		return fmt.Errorf("migrate new: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}
