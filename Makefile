.PHONY: help build run test lint clean docker-up docker-down migrate dev

# Default target
help:
	@echo "OTelGuard Development Commands"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Development:"
	@echo "  dev           Start development environment (Docker + hot reload)"
	@echo "  docker-up     Start Docker services (PostgreSQL, ClickHouse)"
	@echo "  docker-down   Stop Docker services"
	@echo "  run-backend   Run backend server"
	@echo "  run-frontend  Run frontend dev server"
	@echo ""
	@echo "Build:"
	@echo "  build         Build all services"
	@echo "  build-backend Build Go backend"
	@echo "  build-frontend Build React frontend"
	@echo ""
	@echo "Database:"
	@echo "  migrate-up    Run database migrations"
	@echo "  migrate-down  Rollback database migrations"
	@echo "  seed          Seed database with sample data"
	@echo ""
	@echo "Testing:"
	@echo "  test          Run all tests"
	@echo "  test-backend  Run backend tests"
	@echo "  test-frontend Run frontend tests"
	@echo "  lint          Run linters"
	@echo ""
	@echo "Other:"
	@echo "  clean         Clean build artifacts"
	@echo "  install       Install all dependencies"

# Development
dev: docker-up
	@echo "Starting development environment..."
	@trap 'make docker-down' EXIT; \
	(cd backend && go run cmd/server/main.go) & \
	(cd frontend && npm run dev) & \
	wait

docker-up:
	@echo "Starting Docker services..."
	docker-compose up -d postgres clickhouse
	@echo "Waiting for services to be healthy..."
	@sleep 5

docker-down:
	@echo "Stopping Docker services..."
	docker-compose down

run-backend:
	@echo "Starting backend server..."
	cd backend && go run cmd/server/main.go

run-frontend:
	@echo "Starting frontend dev server..."
	cd frontend && npm run dev

# Build
build: build-backend build-frontend

build-backend:
	@echo "Building backend..."
	cd backend && go build -o bin/otelguard ./cmd/server

build-frontend:
	@echo "Building frontend..."
	cd frontend && npm run build

# Database
migrate-up:
	@echo "Running PostgreSQL migrations..."
	docker exec -i otelguard-postgres psql -U otelguard -d otelguard < backend/migrations/postgres/001_initial_schema.sql
	@echo "Running ClickHouse migrations..."
	docker exec -i otelguard-clickhouse clickhouse-client --database=otelguard < backend/migrations/clickhouse/001_traces.sql

migrate-down:
	@echo "Rolling back migrations..."
	@echo "Warning: This will drop all tables!"
	docker exec -i otelguard-postgres psql -U otelguard -d otelguard -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"

seed:
	@echo "Seeding database..."
	@go run backend/scripts/seed.go

# Testing
test: test-backend test-frontend

test-backend:
	@echo "Running backend tests..."
	cd backend && go test -v ./...

test-frontend:
	@echo "Running frontend tests..."
	cd frontend && npm test

lint: lint-backend lint-frontend

lint-backend:
	@echo "Linting backend..."
	cd backend && golangci-lint run

lint-frontend:
	@echo "Linting frontend..."
	cd frontend && npm run lint

# Install
install: install-backend install-frontend

install-backend:
	@echo "Installing backend dependencies..."
	cd backend && go mod download

install-frontend:
	@echo "Installing frontend dependencies..."
	cd frontend && npm install

# Clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf backend/bin
	rm -rf frontend/dist
	rm -rf frontend/node_modules/.cache
