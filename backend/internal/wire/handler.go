package wire

import (
	"github.com/google/wire"
	"github.com/jmoiron/sqlx"
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
	ProvideHandlers,
)

// ProvideHealthHandler creates a new HealthHandler.
func ProvideHealthHandler(db *sqlx.DB, logger *zap.Logger) *handlers.HealthHandler {
	return handlers.NewHealthHandler(db, logger)
}

// ProvideAuthHandler creates a new AuthHandler.
func ProvideAuthHandler(
	authService *service.AuthService,
	cfg *config.Config,
	logger *zap.Logger,
) *handlers.AuthHandler {
	return handlers.NewAuthHandler(authService, &cfg.Auth, logger)
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
	logger *zap.Logger,
) *handlers.PromptHandler {
	return handlers.NewPromptHandler(promptService, logger)
}

// ProvideGuardrailHandler creates a new GuardrailHandler.
func ProvideGuardrailHandler(
	guardrailService *service.GuardrailService,
	logger *zap.Logger,
) *handlers.GuardrailHandler {
	return handlers.NewGuardrailHandler(guardrailService, logger)
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
) *api.Handlers {
	return &api.Handlers{
		Health:    health,
		Auth:      auth,
		Org:       org,
		Trace:     trace,
		OTLP:      otlp,
		Prompt:    prompt,
		Guardrail: guardrail,
	}
}
