package wire

import (
	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/otelguard/otelguard/internal/api"
	"github.com/otelguard/otelguard/internal/api/handlers"
	"github.com/otelguard/otelguard/internal/config"
	"github.com/otelguard/otelguard/internal/repository/postgres"
	"github.com/otelguard/otelguard/internal/service"
)

// HandlerSet provides all HTTP handler instances.
var HandlerSet = wire.NewSet(
	ProvideHealthHandler,
	ProvideAuthHandler,
	ProvideOrgHandler,
	ProvideTraceHandler,
	ProvideOTLPHandler,
	ProvidePromptHandler,
	ProvideGuardrailHandler,
	ProvideGuardrailAnalyticsHandler,
	ProvideAnnotationHandler,
	ProvideFeedbackHandler,
	ProvideLLMHandler,
	ProvideAgentHandler,
	ProvideEvaluatorHandler,
	ProvideDatasetHandler,
	ProvideExperimentHandler,
	ProvideScoreAnalyticsHandler,
	ProvideMetricsHandler,
	ProvideDashboardHandler,
	ProvideHandlers,
)

// ProvideHealthHandler creates a new HealthHandler.
func ProvideHealthHandler(db *pgxpool.Pool, logger *zap.Logger) *handlers.HealthHandler {
	return handlers.NewHealthHandler(db, logger)
}

// ProvideAuthHandler creates a new AuthHandler.
func ProvideAuthHandler(
	authService *service.AuthService,
	orgService *service.OrgService,
	cfg *config.Config,
	logger *zap.Logger,
) *handlers.AuthHandler {
	return handlers.NewAuthHandler(authService, orgService, &cfg.Auth, logger)
}

// ProvideOrgHandler creates a new OrgHandler.
func ProvideOrgHandler(
	orgService *service.OrgService,
	logger *zap.Logger,
) *handlers.OrgHandler {
	return handlers.NewOrgHandler(orgService, logger)
}

// ProvideTraceHandler creates a new TraceHandler.
func ProvideTraceHandler(
	traceService *service.TraceService,
	logger *zap.Logger,
) *handlers.TraceHandler {
	return handlers.NewTraceHandler(traceService, logger)
}

// ProvideOTLPHandler creates a new OTLPHandler.
func ProvideOTLPHandler(
	traceService *service.TraceService,
	logger *zap.Logger,
) *handlers.OTLPHandler {
	return handlers.NewOTLPHandler(traceService, logger)
}

// ProvidePromptHandler creates a new PromptHandler.
func ProvidePromptHandler(
	promptService *service.PromptService,
	traceService *service.TraceService,
	logger *zap.Logger,
) *handlers.PromptHandler {
	return handlers.NewPromptHandler(promptService, traceService, logger)
}

// ProvideGuardrailHandler creates a new GuardrailHandler.
func ProvideGuardrailHandler(
	guardrailService *service.GuardrailService,
	logger *zap.Logger,
) *handlers.GuardrailHandler {
	return handlers.NewGuardrailHandler(guardrailService, logger)
}

// ProvideAnnotationHandler creates a new AnnotationHandler.
func ProvideAnnotationHandler(
	annotationService *service.AnnotationService,
	logger *zap.Logger,
) *handlers.AnnotationHandler {
	return handlers.NewAnnotationHandler(annotationService, logger)
}

// ProvideFeedbackHandler creates a new FeedbackHandler.
func ProvideFeedbackHandler(
	feedbackService *service.FeedbackService,
	logger *zap.Logger,
) *handlers.FeedbackHandler {
	return handlers.NewFeedbackHandler(feedbackService, logger)
}

// ProvideLLMHandler creates a new LLMHandler.
func ProvideLLMHandler(
	llmService *service.LLMServiceImpl,
	tokenizer *service.TokenizerService,
	pricing *service.PricingService,
	logger *zap.Logger,
) *handlers.LLMHandler {
	return handlers.NewLLMHandler(llmService, tokenizer, pricing, logger)
}

// ProvideAgentHandler creates a new AgentHandler.
func ProvideAgentHandler(
	agentService *service.AgentService,
	logger *zap.Logger,
) *handlers.AgentHandler {
	return handlers.NewAgentHandler(agentService, logger)
}

// ProvideEvaluatorHandler creates a new EvaluatorHandler.
func ProvideEvaluatorHandler(
	evaluatorService *service.EvaluatorService,
	logger *zap.Logger,
) *handlers.EvaluatorHandler {
	return handlers.NewEvaluatorHandler(evaluatorService, logger)
}

// ProvideDatasetHandler creates a new DatasetHandler.
func ProvideDatasetHandler(
	datasetService *service.DatasetService,
	logger *zap.Logger,
) *handlers.DatasetHandler {
	return handlers.NewDatasetHandler(datasetService, logger)
}

// ProvideExperimentHandler creates a new ExperimentHandler.
func ProvideExperimentHandler(
	experimentService *service.ExperimentService,
	experimentRepo *postgres.ExperimentRepository,
	logger *zap.Logger,
) *handlers.ExperimentHandler {
	return handlers.NewExperimentHandler(experimentService, experimentRepo, logger)
}

// ProvideGuardrailAnalyticsHandler creates a new GuardrailAnalyticsHandler.
func ProvideGuardrailAnalyticsHandler(
	analyticsService *service.GuardrailAnalyticsService,
	logger *zap.Logger,
) *handlers.GuardrailAnalyticsHandler {
	return handlers.NewGuardrailAnalyticsHandler(analyticsService, logger)
}

// ProvideScoreAnalyticsHandler creates a new ScoreAnalyticsHandler.
func ProvideScoreAnalyticsHandler(
	analyticsService *service.ScoreAnalyticsService,
	logger *zap.Logger,
) *handlers.ScoreAnalyticsHandler {
	return handlers.NewScoreAnalyticsHandler(analyticsService, logger)
}

// ProvideMetricsHandler creates a new MetricsHandler.
func ProvideMetricsHandler(
	metricsService *service.MetricsService,
	logger *zap.Logger,
) *handlers.MetricsHandler {
	return handlers.NewMetricsHandler(metricsService, logger)
}

// ProvideDashboardHandler creates a new DashboardHandler.
func ProvideDashboardHandler(
	dashboardService *service.DashboardService,
	logger *zap.Logger,
) *handlers.DashboardHandler {
	return handlers.NewDashboardHandler(dashboardService, logger)
}

// ProvideHandlers creates the Handlers struct containing all handlers.
func ProvideHandlers(
	health *handlers.HealthHandler,
	auth *handlers.AuthHandler,
	org *handlers.OrgHandler,
	trace *handlers.TraceHandler,
	otlp *handlers.OTLPHandler,
	prompt *handlers.PromptHandler,
	guardrail *handlers.GuardrailHandler,
	guardrailAnalytics *handlers.GuardrailAnalyticsHandler,
	annotation *handlers.AnnotationHandler,
	feedback *handlers.FeedbackHandler,
	llm *handlers.LLMHandler,
	agent *handlers.AgentHandler,
	evaluator *handlers.EvaluatorHandler,
	dataset *handlers.DatasetHandler,
	experiment *handlers.ExperimentHandler,
	scoreAnalytics *handlers.ScoreAnalyticsHandler,
	metrics *handlers.MetricsHandler,
	dashboard *handlers.DashboardHandler,
) *api.Handlers {
	return &api.Handlers{
		Health:             health,
		Auth:               auth,
		Org:                org,
		Trace:              trace,
		OTLP:               otlp,
		Prompt:             prompt,
		Guardrail:          guardrail,
		GuardrailAnalytics: guardrailAnalytics,
		Annotation:         annotation,
		Feedback:           feedback,
		LLM:                llm,
		Agent:              agent,
		Evaluator:          evaluator,
		Dataset:            dataset,
		Experiment:         experiment,
		ScoreAnalytics:     scoreAnalytics,
		Metrics:            metrics,
		Dashboard:          dashboard,
	}
}
