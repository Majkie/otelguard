.PHONY: help build run test lint lint-fix format clean docker-up docker-down migrate dev dev-backend dev-frontend hooks hooks-install install-tools

# Default target
help:
	@echo "OTelGuard Development Commands"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Development:"
	@echo "  dev            Start full dev environment (databases + hot reload)"
	@echo "  dev-backend    Start backend with hot reload (requires databases)"
	@echo "  dev-frontend   Start frontend with hot reload"
	@echo "  docker-up      Start Docker services (PostgreSQL, ClickHouse)"
	@echo "  docker-down    Stop Docker services"
	@echo ""
	@echo "Build:"
	@echo "  build          Build all services"
	@echo "  build-backend  Build Go backend"
	@echo "  build-frontend Build React frontend"
	@echo ""
	@echo "Database:"
	@echo "  migrate-up     Run database migrations"
	@echo "  migrate-down   Rollback database migrations"
	@echo "  migrate-status Show current migration version"
	@echo "  migrate-create Create new migration (NAME=...)"
	@echo "  seed           Seed database with sample data"
	@echo ""
	@echo "Testing & Quality:"
	@echo "  test           Run all tests"
	@echo "  test-backend   Run backend tests"
	@echo "  test-frontend  Run frontend tests"
	@echo "  lint           Run linters"
	@echo "  lint-fix       Fix auto-fixable lint issues"
	@echo "  format         Format code"
	@echo ""
	@echo "Git Hooks:"
	@echo "  hooks-install  Install git hooks (lefthook)"
	@echo "  hooks          Run pre-commit hooks manually"
	@echo ""
	@echo "Other:"
	@echo "  clean          Clean build artifacts"
	@echo "  install        Install all dependencies"
	@echo "  install-tools  Install development tools (air, etc.)"

# Development - Full stack with hot reload
dev: docker-up install-tools
	@echo ""
	@echo "=========================================="
	@echo "Starting OTelGuard Development Environment"
	@echo "=========================================="
	@echo ""
	@echo "Services:"
	@echo "  Backend API:    http://localhost:8080"
	@echo "  Frontend:       http://localhost:3000"
	@echo "  PostgreSQL:     localhost:5432"
	@echo "  ClickHouse:     localhost:8123 (HTTP), localhost:9000 (Native)"
	@echo ""
	@echo "Hot reload enabled for both backend and frontend!"
	@echo "Press Ctrl+C to stop all services."
	@echo ""
	@trap 'echo ""; echo "Stopping services..."; make docker-down' EXIT; \
	(cd backend && export $$(cat .env.development 2>/dev/null | xargs) && air) & \
	(cd frontend && npm run dev) & \
	wait

# Start only databases
docker-up:
	@echo "Starting Docker services (PostgreSQL, ClickHouse)..."
	docker-compose up -d postgres clickhouse
	@echo "Waiting for services to be healthy..."
	@sleep 5
	@echo ""
	@echo "Database services started:"
	@echo "  PostgreSQL: localhost:5432"
	@echo "  ClickHouse: localhost:8123 (HTTP), localhost:9000 (Native)"

docker-down:
	@echo "Stopping Docker services..."
	docker-compose down

# Backend with hot reload
dev-backend: install-tools
	@echo "Starting backend with hot reload..."
	@echo "Make sure databases are running (make docker-up)"
	@echo ""
	cd backend && export $$(cat .env.development 2>/dev/null | xargs) && air

# Frontend with hot reload
dev-frontend:
	@echo "Starting frontend with hot reload..."
	@echo "Frontend will be available at http://localhost:3000"
	@echo ""
	cd frontend && npm run dev

# Run without hot reload (simple)
run-backend:
	@echo "Starting backend server..."
	cd backend && go run ./cmd/server

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
POSTGRES_URL ?= postgres://otelguard:otelguard@localhost:5432/otelguard?sslmode=disable
MIGRATIONS_PATH ?= backend/internal/database/migrations/postgres

migrate-up:
	@echo "Running PostgreSQL migrations..."
	@which migrate > /dev/null || go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	migrate -path $(MIGRATIONS_PATH) -database "$(POSTGRES_URL)" up
	@echo "Running ClickHouse migrations..."
	docker exec -i otelguard-clickhouse clickhouse-client --database=otelguard < backend/migrations/clickhouse/001_traces.sql 2>/dev/null || true
	docker exec -i otelguard-clickhouse clickhouse-client --database=otelguard < backend/migrations/clickhouse/002_events.sql 2>/dev/null || true
	docker exec -i otelguard-clickhouse clickhouse-client --database=otelguard < backend/migrations/clickhouse/003_attributes.sql 2>/dev/null || true

migrate-down:
	@echo "Rolling back PostgreSQL migrations..."
	@echo "Warning: This will drop all tables!"
	migrate -path $(MIGRATIONS_PATH) -database "$(POSTGRES_URL)" down -all

migrate-status:
	@echo "Checking migration status..."
	migrate -path $(MIGRATIONS_PATH) -database "$(POSTGRES_URL)" version

migrate-create:
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-create NAME=migration_name"; exit 1; fi
	migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $(NAME)

seed:
	@echo "Seeding database..."
	cd backend && go run ./scripts/seed/main.go

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

format: format-backend format-frontend

format-backend:
	@echo "Formatting backend..."
	cd backend && gofmt -s -w . && goimports -w .

format-frontend:
	@echo "Formatting frontend..."
	cd frontend && npm run format

lint-fix: lint-fix-backend lint-fix-frontend

lint-fix-backend:
	@echo "Fixing backend lint issues..."
	cd backend && golangci-lint run --fix

lint-fix-frontend:
	@echo "Fixing frontend lint issues..."
	cd frontend && npm run lint:fix

# Install dependencies
install: install-backend install-frontend

install-backend:
	@echo "Installing backend dependencies..."
	cd backend && go mod download

install-frontend:
	@echo "Installing frontend dependencies..."
	cd frontend && npm install

# Install development tools
install-tools:
	@echo "Checking development tools..."
	@which air > /dev/null 2>&1 || (echo "Installing Air for Go hot reload..." && go install github.com/air-verse/air@latest)
	@echo "Development tools ready!"

# Git Hooks
hooks-install:
	@echo "Installing lefthook..."
	@which lefthook > /dev/null || go install github.com/evilmartians/lefthook@latest
	lefthook install
	@echo "Git hooks installed successfully!"

hooks:
	@echo "Running pre-commit hooks..."
	lefthook run pre-commit

# Clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf backend/bin
	rm -rf backend/tmp
	rm -rf frontend/dist
	rm -rf frontend/node_modules/.cache

# Production docker build
docker-build:
	@echo "Building production Docker images..."
	docker-compose --profile prod build

docker-prod:
	@echo "Starting production environment..."
	docker-compose --profile prod up -d
