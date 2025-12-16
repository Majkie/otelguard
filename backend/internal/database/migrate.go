package database

import (
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/clickhouse"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"go.uber.org/zap"
)

//go:embed migrations
var migrationsFS embed.FS

// MigrateConfig holds migration configuration
type MigrateConfig struct {
	DatabaseURL string
	Logger      *zap.Logger
}

// RunPostgresMigrations runs all pending PostgreSQL migrations
func RunPostgresMigrations(cfg *MigrateConfig) error {
	d, err := iofs.New(migrationsFS, "migrations/postgres")
	if err != nil {
		return fmt.Errorf("failed to create postgres migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to create postgres migrate instance: %w", err)
	}
	defer m.Close()

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run postgres migrations: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get postgres migration version: %w", err)
	}

	if cfg.Logger != nil {
		cfg.Logger.Info("postgres migrations completed",
			zap.Uint("version", version),
			zap.Bool("dirty", dirty),
		)
	}

	return nil
}

// RunClickHouseMigrations runs all pending ClickHouse migrations
func RunClickHouseMigrations(cfg *MigrateConfig) error {
	d, err := iofs.New(migrationsFS, "migrations/clickhouse")
	if err != nil {
		return fmt.Errorf("failed to create clickhouse migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to create clickhouse migrate instance: %w", err)
	}
	defer m.Close()

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run clickhouse migrations: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get clickhouse migration version: %w", err)
	}

	if cfg.Logger != nil {
		cfg.Logger.Info("clickhouse migrations completed",
			zap.Uint("version", version),
			zap.Bool("dirty", dirty),
		)
	}

	return nil
}
