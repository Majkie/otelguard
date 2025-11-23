package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	grpcserver "github.com/otelguard/otelguard/internal/api/grpc"
	"github.com/otelguard/otelguard/internal/config"
	"github.com/otelguard/otelguard/pkg/validator"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize application with Wire-generated dependency injection
	app, err := InitializeApplication(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize application: %v\n", err)
		os.Exit(1)
	}
	defer app.Cleanup()
	defer app.Logger.Sync()

	// Initialize request validator
	validator.Init()
	app.Logger.Info("request validator initialized")

	app.Logger.Info("starting OTelGuard server",
		zap.String("environment", cfg.Server.Environment),
		zap.Int("port", cfg.Server.Port),
	)

	app.Logger.Info("connected to PostgreSQL")
	app.Logger.Info("connected to ClickHouse")

	// Start background services (batch writer, etc.)
	app.Start()

	// Log batch writer status if enabled
	if bw := app.GetBatchWriter(); bw != nil {
		app.Logger.Info("batch writer started")
	}

	// Log sampler status if enabled
	if cfg.Sampler.Enabled {
		app.Logger.Info("trace sampling enabled",
			zap.String("type", cfg.Sampler.Type),
			zap.Float64("rate", cfg.Sampler.Rate),
			zap.Int("max_per_sec", cfg.Sampler.MaxPerSecond),
		)
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      app.Router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start HTTP server in a goroutine
	go func() {
		app.Logger.Info("HTTP server listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			app.Logger.Fatal("HTTP server error", zap.Error(err))
		}
	}()

	// Start gRPC server if enabled
	var grpcServer *grpcserver.Server
	if app.GRPCComponents != nil {
		grpcServer = grpcserver.NewServer(
			app.GRPCComponents.Config,
			app.GRPCComponents.OTLPService,
			app.Logger,
		)
		if err := grpcServer.Start(); err != nil {
			app.Logger.Fatal("failed to start gRPC server", zap.Error(err))
		}
		app.Logger.Info("gRPC OTLP receiver started", zap.Int("port", cfg.GRPC.Port))
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	app.Logger.Info("shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		app.Logger.Fatal("HTTP server forced shutdown", zap.Error(err))
	}

	// Stop gRPC server
	if grpcServer != nil {
		app.Logger.Info("stopping gRPC server...")
		grpcCtx, grpcCancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := grpcServer.Stop(grpcCtx); err != nil {
			app.Logger.Error("failed to stop gRPC server cleanly", zap.Error(err))
		}
		grpcCancel()
	}

	// Stop batch writer and flush remaining data
	if batchWriter := app.GetBatchWriter(); batchWriter != nil {
		app.Logger.Info("stopping batch writer...")
		flushCtx, flushCancel := context.WithTimeout(context.Background(), 15*time.Second)
		if err := batchWriter.Stop(flushCtx); err != nil {
			app.Logger.Error("failed to stop batch writer cleanly", zap.Error(err))
		}
		flushCancel()

		// Log final metrics
		metrics := batchWriter.GetMetrics()
		app.Logger.Info("batch writer final metrics",
			zap.Int64("traces_written", metrics.TracesWritten),
			zap.Int64("spans_written", metrics.SpansWritten),
			zap.Int64("scores_written", metrics.ScoresWritten),
			zap.Int64("flush_count", metrics.FlushCount),
			zap.Int64("error_count", metrics.ErrorCount),
		)
	}

	app.Logger.Info("server stopped")
}
