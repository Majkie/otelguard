package main

import (
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	log.Println("Checking migration status...")

	// Database connection
	pgHost := getEnv("POSTGRES_HOST", "localhost")
	pgPort := getEnv("POSTGRES_PORT", "5432")
	pgUser := getEnv("POSTGRES_USER", "otelguard")
	pgPass := getEnv("POSTGRES_PASSWORD", "otelguard")
	pgDB := getEnv("POSTGRES_DB", "otelguard")

	pgDSN := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		pgUser, pgPass, pgHost, pgPort, pgDB)

	// Create file source
	sourceDriver, err := (&file.File{}).Open("file://./internal/database/migrations/postgres")
	if err != nil {
		log.Fatalf("Failed to create migration source: %v", err)
	}

	m, err := migrate.NewWithSourceInstance("file", sourceDriver, pgDSN)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}
	defer m.Close()

	version, dirty, err := m.Version()
	if err != nil {
		log.Fatalf("Failed to get version: %v", err)
	}

	fmt.Printf("Current version: %d, Dirty: %v\n", version, dirty)

	if dirty {
		log.Println("Database is dirty, forcing to version 3...")
		if err := m.Force(3); err != nil {
			log.Fatalf("Failed to force version: %v", err)
		}
		log.Println("Forced to version 3, now running migrations...")
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Failed to run migrations: %v", err)
		}
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
