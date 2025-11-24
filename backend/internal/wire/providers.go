package wire

import (
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/otelguard/otelguard/internal/api"
	grpcserver "github.com/otelguard/otelguard/internal/api/grpc"
	"github.com/otelguard/otelguard/internal/api/middleware"
	"github.com/otelguard/otelguard/internal/config"
	chrepo "github.com/otelguard/otelguard/internal/repository/clickhouse"
	"github.com/otelguard/otelguard/internal/service"
)

// ProviderSet is the main provider set that includes all application dependencies.
var ProviderSet = wire.NewSet(
	DatabaseSet,
	RepositorySet,
	ServiceSet,
	HandlerSet,
	ProvideLogger,
	ProvideRouter,
	ProvideGRPCComponents,
	ProvideApplication,
)

// Application holds all the dependencies needed to run the server.
type Application struct {
	Config         *config.Config
	Logger         *zap.Logger
	PostgresDB     *pgxpool.Pool
	ClickHouseConn clickhouse.Conn
	Router            *gin.Engine
	Handlers          *api.Handlers
	TraceService      *service.TraceService
	EvaluatorService  *service.EvaluatorService
	ExperimentService *service.ExperimentService
	BatchWriter       *BatchWriterResult
	GRPCComponents    *GRPCComponents

	// Database wrappers with cleanup
	postgresWrapper   *PostgresDB
	clickhouseWrapper *ClickHouseDB
}

// Start starts all background services.
func (a *Application) Start() {
	// Start batch writer if enabled
	if a.BatchWriter != nil && a.BatchWriter.Writer != nil {
		a.BatchWriter.Start()
	}

	// Start evaluator service
	if a.EvaluatorService != nil {
		a.EvaluatorService.Start()
	}

	// Start experiment service
	if a.ExperimentService != nil {
		a.ExperimentService.Start()
	}
}

// Cleanup releases all resources.
func (a *Application) Cleanup() {
	if a.clickhouseWrapper != nil && a.clickhouseWrapper.Cleanup != nil {
		a.clickhouseWrapper.Cleanup()
	}
	if a.postgresWrapper != nil && a.postgresWrapper.Cleanup != nil {
		a.postgresWrapper.Cleanup()
	}
}

// GetBatchWriter returns the batch writer if async writes are enabled.
func (a *Application) GetBatchWriter() *chrepo.TraceBatchWriter {
	if a.BatchWriter == nil {
		return nil
	}
	return a.BatchWriter.Writer
}

// GRPCComponents holds gRPC-related dependencies.
type GRPCComponents struct {
	OTLPService *grpcserver.OTLPTraceService
	Config      *grpcserver.ServerConfig
}

// ProvideLogger creates a configured zap logger.
func ProvideLogger(cfg *config.Config) *zap.Logger {
	var zapConfig zap.Config
	if cfg.IsDevelopment() {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		zapConfig = zap.NewProductionConfig()
	}

	logger, err := zapConfig.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	return logger
}

// ProvideRouter creates the Gin router with all routes configured.
func ProvideRouter(
	h *api.Handlers,
	cfg *config.Config,
	logger *zap.Logger,
) *gin.Engine {
	// API key validator (stub for now)
	apiKeyValidator := func(keyHash string) (*middleware.APIKeyClaims, error) {
		// TODO: Implement proper API key validation from database
		// For now, accept any key for development
		return &middleware.APIKeyClaims{
			Scopes: []string{"*"},
		}, nil
	}

	return api.SetupRouter(h, cfg, logger, apiKeyValidator)
}

// ProvideGRPCComponents creates the gRPC-related components.
func ProvideGRPCComponents(
	traceService *service.TraceService,
	cfg *config.Config,
	logger *zap.Logger,
) *GRPCComponents {
	if !cfg.GRPC.Enabled {
		return nil
	}

	otlpService := grpcserver.NewOTLPTraceService(traceService, logger)
	grpcConfig := &grpcserver.ServerConfig{
		Port:             cfg.GRPC.Port,
		MaxRecvMsgSize:   cfg.GRPC.MaxRecvMsgSize,
		MaxSendMsgSize:   cfg.GRPC.MaxSendMsgSize,
		EnableReflection: cfg.GRPC.EnableReflection,
	}

	return &GRPCComponents{
		OTLPService: otlpService,
		Config:      grpcConfig,
	}
}

// ProvideApplication creates the main Application struct with all dependencies.
func ProvideApplication(
	cfg *config.Config,
	logger *zap.Logger,
	pgWrapper *PostgresDB,
	chWrapper *ClickHouseDB,
	router *gin.Engine,
	handlers *api.Handlers,
	traceService *service.TraceService,
	evaluatorService *service.EvaluatorService,
	experimentService *service.ExperimentService,
	batchWriter *BatchWriterResult,
	grpcComponents *GRPCComponents,
) *Application {
	return &Application{
		Config:            cfg,
		Logger:            logger,
		PostgresDB:        pgWrapper.DB,
		ClickHouseConn:    chWrapper.Conn,
		Router:            router,
		Handlers:          handlers,
		TraceService:      traceService,
		EvaluatorService:  evaluatorService,
		ExperimentService: experimentService,
		BatchWriter:       batchWriter,
		GRPCComponents:    grpcComponents,
		postgresWrapper:   pgWrapper,
		clickhouseWrapper: chWrapper,
	}
}
