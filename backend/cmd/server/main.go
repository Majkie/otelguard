package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/otelguard/otelguard/internal/api"
	"github.com/otelguard/otelguard/internal/api/handlers"
	"github.com/otelguard/otelguard/internal/api/middleware"
	"github.com/otelguard/otelguard/internal/config"
	chrepo "github.com/otelguard/otelguard/internal/repository/clickhouse"
	pgrepo "github.com/otelguard/otelguard/internal/repository/postgres"
	"github.com/otelguard/otelguard/internal/service"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger := initLogger(cfg.IsDevelopment())
	defer logger.Sync()

	logger.Info("starting OTelGuard server",
		zap.String("environment", cfg.Server.Environment),
		zap.Int("port", cfg.Server.Port),
	)

	// Initialize PostgreSQL
	pgDB, err := initPostgres(cfg.Postgres)
	if err != nil {
		logger.Fatal("failed to connect to PostgreSQL", zap.Error(err))
	}
	defer pgDB.Close()
	logger.Info("connected to PostgreSQL")

	// Initialize ClickHouse
	chConn, err := initClickHouse(cfg.ClickHouse)
	if err != nil {
		logger.Fatal("failed to connect to ClickHouse", zap.Error(err))
	}
	defer chConn.Close()
	logger.Info("connected to ClickHouse")

	// Initialize repositories
	userRepo := pgrepo.NewUserRepository(pgDB)
	promptRepo := pgrepo.NewPromptRepository(pgDB)
	guardrailRepo := pgrepo.NewGuardrailRepository(pgDB)
	traceRepo := chrepo.NewTraceRepository(chConn)
	guardrailEventRepo := chrepo.NewGuardrailEventRepository(chConn)

	// Initialize services
	authService := service.NewAuthService(userRepo, logger, cfg.Auth.BcryptCost)
	traceService := service.NewTraceService(traceRepo, logger)
	promptService := service.NewPromptService(promptRepo, logger)
	guardrailService := service.NewGuardrailService(guardrailRepo, guardrailEventRepo, logger)

	// API key validator (stub for now)
	apiKeyValidator := func(keyHash string) (*middleware.APIKeyClaims, error) {
		// TODO: Implement proper API key validation from database
		// For now, accept any key for development
		return &middleware.APIKeyClaims{
			Scopes: []string{"*"},
		}, nil
	}

	// Initialize handlers
	h := &api.Handlers{
		Health:    handlers.NewHealthHandler(pgDB, logger),
		Auth:      handlers.NewAuthHandler(authService, &cfg.Auth, logger),
		Trace:     handlers.NewTraceHandler(traceService, logger),
		Prompt:    handlers.NewPromptHandler(promptService, logger),
		Guardrail: handlers.NewGuardrailHandler(guardrailService, logger),
	}

	// Setup router
	router := api.SetupRouter(h, cfg, logger, apiKeyValidator)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("server listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("server forced shutdown", zap.Error(err))
	}

	logger.Info("server stopped")
}

func initLogger(isDev bool) *zap.Logger {
	var config zap.Config
	if isDev {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
	}

	logger, err := config.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	return logger
}

func initPostgres(cfg config.PostgresConfig) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping: %w", err)
	}

	return db, nil
}

func initClickHouse(cfg config.ClickHouseConfig) (clickhouse.Conn, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)},
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.User,
			Password: cfg.Password,
		},
		DialTimeout: cfg.DialTimeout,
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		MaxOpenConns: cfg.MaxOpenConn,
		MaxIdleConns: cfg.MaxIdleConn,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	if err := conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping: %w", err)
	}

	return conn, nil
}
