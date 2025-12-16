package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/jackc/pgx/v5"
	"github.com/otelguard/otelguard/internal/config"
)

func main() {
	log.Println("WARNING: This will delete ALL data in Postgres and ClickHouse databases.")
	log.Println("Waiting 5 seconds before proceeding... Press Ctrl+C to cancel.")
	time.Sleep(5 * time.Second)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Reset Postgres
	if err := resetPostgres(cfg); err != nil {
		log.Printf("Failed to reset Postgres: %v", err)
	} else {
		log.Println("Postgres reset successfully")
	}

	// Reset ClickHouse
	if err := resetClickHouse(cfg); err != nil {
		log.Printf("Failed to reset ClickHouse: %v", err)
	} else {
		log.Println("ClickHouse reset successfully")
	}

	log.Println("Databases reset. Restart the server to auto-migrate.")
}

func resetPostgres(cfg *config.Config) error {
	log.Println("Resetting Postgres...")
	dsn := cfg.Postgres.DSN()
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to postgres: %w", err)
	}
	defer conn.Close(ctx)

	// Drop schema public and recreate
	// This is brute force but effective for dev
	if _, err := conn.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"); err != nil {
		return fmt.Errorf("failed to drop/create schema: %w", err)
	}

	return nil
}

func resetClickHouse(cfg *config.Config) error {
	log.Println("Resetting ClickHouse...")

	opts := &clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", cfg.ClickHouse.Host, cfg.ClickHouse.Port)},
		Auth: clickhouse.Auth{
			Database: cfg.ClickHouse.Database,
			Username: cfg.ClickHouse.User,
			Password: cfg.ClickHouse.Password,
		},
	}

	// We can't easily "Drop Database" if we are connected to it in some versions/cloud
	// But we can drop all tables.
	// However, DROP DATABASE otelguard IF EXISTS; CREATE DATABASE otelguard; is better if allowed.
	// Note: We might need to connect to 'default' database to drop 'otelguard'.

	// Let's try connecting to default first
	opts.Auth.Database = "default"
	defaultConn, err := clickhouse.Open(opts)
	if err != nil {
		return fmt.Errorf("failed to connect to default db: %w", err)
	}

	ctx := context.Background()
	// Check connection
	if err := defaultConn.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping clickhouse default: %w", err)
	}

	targetDB := cfg.ClickHouse.Database
	if targetDB == "" {
		targetDB = "otelguard"
	}

	if err := defaultConn.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", targetDB)); err != nil {
		return fmt.Errorf("failed to drop database %s: %w", targetDB, err)
	}

	if err := defaultConn.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", targetDB)); err != nil {
		return fmt.Errorf("failed to create database %s: %w", targetDB, err)
	}

	return nil
}
