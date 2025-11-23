package wire

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/wire"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/otelguard/otelguard/internal/config"
)

// PostgresDB wraps the database connection with its cleanup function.
type PostgresDB struct {
	DB      *sqlx.DB
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
func ProvidePostgresDB(cfg *config.Config) (*PostgresDB, error) {
	db, err := sqlx.Connect("postgres", cfg.Postgres.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	db.SetMaxOpenConns(cfg.Postgres.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Postgres.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Postgres.ConnMaxLifetime)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return &PostgresDB{
		DB: db,
		Cleanup: func() {
			db.Close()
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
