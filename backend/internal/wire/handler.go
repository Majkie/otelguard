package wire

import (
	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/otelguard/otelguard/internal/api"
	"github.com/otelguard/otelguard/internal/api/handlers"
	"github.com/otelguard/otelguard/internal/config"
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
	ProvideLLMHandler,
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

// ProvideLLMHandler creates a new LLMHandler.
func ProvideLLMHandler(
	llmService *service.LLMServiceImpl,
	tokenizer *service.TokenizerService,
	pricing *service.PricingService,
	logger *zap.Logger,
) *handlers.LLMHandler {
	return handlers.NewLLMHandler(llmService, tokenizer, pricing, logger)
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
	llm *handlers.LLMHandler,
) *api.Handlers {
	return &api.Handlers{
		Health:    health,
		Auth:      auth,
		Org:       org,
		Trace:     trace,
		OTLP:      otlp,
		Prompt:    prompt,
		Guardrail: guardrail,
		LLM:       llm,
	}
}
