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
	ProvideValidatorService,
	ProvideRemediationService,
	ProvideGuardrailService,
	ProvideGuardrailAnalyticsService,
	ProvideAnnotationService,
	ProvideFeedbackService,
	ProvideFeedbackScoreMappingService,
	ProvideBatchWriter,
	ProvideSamplerConfig,
	ProvideTokenizerService,
	ProvidePricingService,
	ProvideLLMService,
	ProvideAgentService,
	ProvideEvaluatorService,
	ProvideDatasetService,
	ProvideExperimentService,
	ProvideScoreAnalyticsService,
	ProvideMetricsService,
	ProvideDashboardService,
)

// ProvideAuthService creates a new AuthService.
func ProvideAuthService(
	userRepo *pgrepo.UserRepository,
	apiKeyRepo *pgrepo.APIKeyRepository,
	logger *zap.Logger,
	cfg *config.Config,
) *service.AuthService {
	return service.NewAuthService(userRepo, apiKeyRepo, logger, cfg.Auth.BcryptCost, cfg.Auth.APIKeySalt)
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

// ProvideValidatorService creates a new ValidatorService.
func ProvideValidatorService(logger *zap.Logger) *service.ValidatorService {
	return service.NewValidatorService(logger)
}

// ProvideRemediationService creates a new RemediationService.
func ProvideRemediationService(logger *zap.Logger) *service.RemediationService {
	return service.NewRemediationService(logger)
}

// ProvideGuardrailService creates a new GuardrailService.
func ProvideGuardrailService(
	guardrailRepo *pgrepo.GuardrailRepository,
	guardrailEventRepo *chrepo.GuardrailEventRepository,
	validatorService *service.ValidatorService,
	remediationService *service.RemediationService,
	logger *zap.Logger,
) *service.GuardrailService {
	return service.NewGuardrailService(guardrailRepo, guardrailEventRepo, validatorService, remediationService, logger)
}

// ProvideTokenizerService creates a new TokenizerService.
func ProvideTokenizerService() *service.TokenizerService {
	return service.NewTokenizerService()
}

// ProvidePricingService creates a new PricingService.
func ProvidePricingService() *service.PricingService {
	return service.NewPricingService()
}

// ProvideAnnotationService creates a new AnnotationService.
func ProvideAnnotationService(
	annotationRepo *pgrepo.AnnotationRepository,
	projectRepo *pgrepo.ProjectRepository,
	userRepo *pgrepo.UserRepository,
	logger *zap.Logger,
) *service.AnnotationService {
	return service.NewAnnotationService(annotationRepo, projectRepo, userRepo, logger)
}

// ProvideFeedbackService creates a new FeedbackService.
func ProvideFeedbackService(
	feedbackRepo *pgrepo.FeedbackRepository,
	feedbackMappingSvc *service.FeedbackScoreMappingService,
	projectRepo *pgrepo.ProjectRepository,
	userRepo *pgrepo.UserRepository,
	logger *zap.Logger,
) *service.FeedbackService {
	return service.NewFeedbackService(feedbackRepo, feedbackMappingSvc, projectRepo, userRepo, logger)
}

// ProvideFeedbackScoreMappingService creates a new FeedbackScoreMappingService.
func ProvideFeedbackScoreMappingService(
	feedbackMappingRepo *pgrepo.FeedbackScoreMappingRepository,
	traceService *service.TraceService,
	projectRepo *pgrepo.ProjectRepository,
	logger *zap.Logger,
) *service.FeedbackScoreMappingService {
	return service.NewFeedbackScoreMappingService(feedbackMappingRepo, traceService, projectRepo, logger)
}

// ProvideLLMService creates a new LLMService.
func ProvideLLMService(
	logger *zap.Logger,
	tokenizer *service.TokenizerService,
	pricing *service.PricingService,
) *service.LLMServiceImpl {
	return service.NewLLMService(logger, tokenizer, pricing)
}

// ProvideAgentService creates a new AgentService.
func ProvideAgentService(
	agentRepo *chrepo.AgentRepository,
	traceRepo *chrepo.TraceRepository,
	logger *zap.Logger,
) *service.AgentService {
	return service.NewAgentService(agentRepo, traceRepo, logger)
}

// ProvideEvaluatorService creates a new EvaluatorService.
func ProvideEvaluatorService(
	evaluatorRepo *pgrepo.EvaluatorRepository,
	jobRepo *pgrepo.EvaluationJobRepository,
	resultRepo *chrepo.EvaluationResultRepository,
	traceRepo *chrepo.TraceRepository,
	llmService *service.LLMServiceImpl,
	pricing *service.PricingService,
	logger *zap.Logger,
) *service.EvaluatorService {
	return service.NewEvaluatorService(evaluatorRepo, jobRepo, resultRepo, traceRepo, llmService, pricing, logger)
}

// ProvideDatasetService creates a new DatasetService.
func ProvideDatasetService(
	datasetRepo *pgrepo.DatasetRepository,
	logger *zap.Logger,
) *service.DatasetService {
	return service.NewDatasetService(datasetRepo, logger)
}

// ProvideExperimentService creates a new ExperimentService.
func ProvideExperimentService(
	experimentRepo *pgrepo.ExperimentRepository,
	datasetRepo *pgrepo.DatasetRepository,
	promptRepo *pgrepo.PromptRepository,
	llmService *service.LLMServiceImpl,
	evaluatorSvc *service.EvaluatorService,
	logger *zap.Logger,
) *service.ExperimentService {
	return service.NewExperimentService(experimentRepo, datasetRepo, promptRepo, llmService, evaluatorSvc, logger)
}

// ProvideGuardrailAnalyticsService creates a new GuardrailAnalyticsService.
func ProvideGuardrailAnalyticsService(
	guardrailRepo *pgrepo.GuardrailRepository,
	guardrailEventRepo *chrepo.GuardrailEventRepository,
	logger *zap.Logger,
) *service.GuardrailAnalyticsService {
	return service.NewGuardrailAnalyticsService(guardrailRepo, guardrailEventRepo, logger)
}

// ProvideScoreAnalyticsService creates a new ScoreAnalyticsService.
func ProvideScoreAnalyticsService(
	evaluationRepo *chrepo.EvaluationResultRepository,
	logger *zap.Logger,
) *service.ScoreAnalyticsService {
	return service.NewScoreAnalyticsService(evaluationRepo, logger)
}

// ProvideMetricsService creates a new MetricsService.
func ProvideMetricsService(
	clickhouse clickhouse.Conn,
	logger *zap.Logger,
) *service.MetricsService {
	return service.NewMetricsService(clickhouse, logger)
}

// ProvideDashboardService creates a new DashboardService.
func ProvideDashboardService(
	dashboardRepo *pgrepo.DashboardRepository,
	logger *zap.Logger,
) *service.DashboardService {
	return service.NewDashboardService(dashboardRepo, logger)
}
