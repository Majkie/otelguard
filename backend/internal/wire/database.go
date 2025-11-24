package wire

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/otelguard/otelguard/internal/config"
	"github.com/otelguard/otelguard/internal/database"
)

// PostgresDB wraps the database connection with its cleanup function.
type PostgresDB struct {
	DB      *pgxpool.Pool
	Cleanup func()
}

// ClickHouseDB wraps the ClickHouse connection with its cleanup function.
type ClickHouseDB struct {
	Conn    clickhouse.Conn
	Cleanup func()
}

// DatabaseSet provides database connections.
var DatabaseSet = wire.NewSet(
	ProvidePostgresDB,
	ProvideClickHouseConn,
	wire.FieldsOf(new(*PostgresDB), "DB"),
	wire.FieldsOf(new(*ClickHouseDB), "Conn"),
)

// ProvidePostgresDB creates a PostgreSQL database connection.
func ProvidePostgresDB(cfg *config.Config, logger *zap.Logger) (*PostgresDB, error) {
	config, err := pgxpool.ParseConfig(cfg.Postgres.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres config: %w", err)
	}

	config.MaxConns = int32(cfg.Postgres.MaxOpenConns)
	config.MinConns = int32(cfg.Postgres.MaxIdleConns)
	config.MaxConnLifetime = cfg.Postgres.ConnMaxLifetime

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres pool: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	// Run database migrations
	migrateCfg := &database.MigrateConfig{
		DatabaseURL: cfg.Postgres.MigrationDSN(),
		Logger:      logger,
	}
	if err := database.RunMigrations(migrateCfg); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &PostgresDB{
		DB: pool,
		Cleanup: func() {
			pool.Close()
		},
	}, nil
}

// ProvideClickHouseConn creates a ClickHouse database connection.
func ProvideClickHouseConn(cfg *config.Config) (*ClickHouseDB, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", cfg.ClickHouse.Host, cfg.ClickHouse.Port)},
		Auth: clickhouse.Auth{
			Database: cfg.ClickHouse.Database,
			Username: cfg.ClickHouse.User,
			Password: cfg.ClickHouse.Password,
		},
		DialTimeout: cfg.ClickHouse.DialTimeout,
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		MaxOpenConns: cfg.ClickHouse.MaxOpenConn,
		MaxIdleConns: cfg.ClickHouse.MaxIdleConn,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open clickhouse connection: %w", err)
	}

	if err := conn.Ping(context.Background()); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping clickhouse: %w", err)
	}

	return &ClickHouseDB{
		Conn: conn,
		Cleanup: func() {
			conn.Close()
		},
	}, nil
}
