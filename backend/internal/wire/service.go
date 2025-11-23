package wire

import (
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/wire"
	"go.uber.org/zap"

	"github.com/otelguard/otelguard/internal/config"
	chrepo "github.com/otelguard/otelguard/internal/repository/clickhouse"
	pgrepo "github.com/otelguard/otelguard/internal/repository/postgres"
	"github.com/otelguard/otelguard/internal/service"
)

// ServiceSet provides all service instances.
var ServiceSet = wire.NewSet(
	ProvideAuthService,
	ProvideOrgService,
	ProvideTraceService,
	ProvidePromptService,
	ProvideGuardrailService,
	ProvideBatchWriter,
	ProvideSamplerConfig,
)

// ProvideAuthService creates a new AuthService.
func ProvideAuthService(
	userRepo *pgrepo.UserRepository,
	logger *zap.Logger,
	cfg *config.Config,
) *service.AuthService {
	return service.NewAuthService(userRepo, logger, cfg.Auth.BcryptCost)
}

// ProvideOrgService creates a new OrgService.
func ProvideOrgService(
	orgRepo *pgrepo.OrganizationRepository,
	projectRepo *pgrepo.ProjectRepository,
	userRepo *pgrepo.UserRepository,
	logger *zap.Logger,
	cfg *config.Config,
) *service.OrgService {
	return service.NewOrgService(orgRepo, projectRepo, userRepo, logger, cfg.Auth.BcryptCost)
}

// BatchWriterResult holds the batch writer and its lifecycle functions.
type BatchWriterResult struct {
	Writer  *chrepo.TraceBatchWriter
	Start   func()
	Cleanup func()
}

// ProvideBatchWriter creates a TraceBatchWriter if async writes are enabled.
func ProvideBatchWriter(
	conn clickhouse.Conn,
	cfg *config.Config,
	logger *zap.Logger,
) *BatchWriterResult {
	if !cfg.ClickHouse.AsyncWrite {
		return &BatchWriterResult{
			Writer:  nil,
			Start:   func() {},
			Cleanup: func() {},
		}
	}

	bwConfig := &chrepo.BatchWriterConfig{
		BatchSize:     cfg.ClickHouse.BatchSize,
		FlushInterval: cfg.ClickHouse.FlushInterval,
		MaxRetries:    cfg.ClickHouse.MaxRetries,
		RetryDelay:    cfg.ClickHouse.RetryDelay,
	}

	batchWriter := chrepo.NewTraceBatchWriter(conn, bwConfig, logger)

	return &BatchWriterResult{
		Writer: batchWriter,
		Start: func() {
			batchWriter.Start()
			logger.Info("batch writer started",
				zap.Int("batch_size", bwConfig.BatchSize),
				zap.Duration("flush_interval", bwConfig.FlushInterval),
			)
		},
		Cleanup: func() {
			// Note: Stop is called separately with context in main
		},
	}
}

// ProvideSamplerConfig creates a SamplerConfig if sampling is enabled.
func ProvideSamplerConfig(cfg *config.Config) *service.SamplerConfig {
	if !cfg.Sampler.Enabled {
		return nil
	}

	return &service.SamplerConfig{
		Type:          service.SamplerType(cfg.Sampler.Type),
		Rate:          cfg.Sampler.Rate,
		MaxPerSecond:  cfg.Sampler.MaxPerSecond,
		SampleErrors:  cfg.Sampler.SampleErrors,
		SampleSlow:    cfg.Sampler.SampleSlow,
		SlowThreshold: cfg.Sampler.SlowThreshold,
	}
}

// ProvideTraceService creates a new TraceService with optional batch writer and sampler.
func ProvideTraceService(
	traceRepo *chrepo.TraceRepository,
	batchWriterResult *BatchWriterResult,
	samplerConfig *service.SamplerConfig,
	cfg *config.Config,
	logger *zap.Logger,
) *service.TraceService {
	return service.NewTraceServiceFull(
		traceRepo,
		batchWriterResult.Writer,
		cfg.ClickHouse.AsyncWrite,
		samplerConfig,
		logger,
	)
}

// ProvidePromptService creates a new PromptService.
func ProvidePromptService(
	promptRepo *pgrepo.PromptRepository,
	logger *zap.Logger,
) *service.PromptService {
	return service.NewPromptService(promptRepo, logger)
}

// ProvideGuardrailService creates a new GuardrailService.
func ProvideGuardrailService(
	guardrailRepo *pgrepo.GuardrailRepository,
	guardrailEventRepo *chrepo.GuardrailEventRepository,
	logger *zap.Logger,
) *service.GuardrailService {
	return service.NewGuardrailService(guardrailRepo, guardrailEventRepo, logger)
}
