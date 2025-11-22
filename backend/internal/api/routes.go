package api

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/otelguard/otelguard/internal/api/handlers"
	"github.com/otelguard/otelguard/internal/api/middleware"
	"github.com/otelguard/otelguard/internal/config"
	"go.uber.org/zap"
)

// Handlers holds all HTTP handlers
type Handlers struct {
	Health    *handlers.HealthHandler
	Auth      *handlers.AuthHandler
	Org       *handlers.OrgHandler
	Trace     *handlers.TraceHandler
	OTLP      *handlers.OTLPHandler
	Prompt    *handlers.PromptHandler
	Guardrail *handlers.GuardrailHandler
}

// SetupRouter configures the Gin router with all routes and middleware
func SetupRouter(h *Handlers, cfg *config.Config, logger *zap.Logger, apiKeyValidator middleware.APIKeyValidator) *gin.Engine {
	// Set Gin mode based on environment
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Global middleware
	r.Use(gin.Recovery())
	r.Use(requestid.New())
	r.Use(middleware.Logger(logger))
	r.Use(middleware.ErrorHandler())

	// CORS configuration
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-API-Key", "X-Request-ID"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check endpoints (no auth required)
	r.GET("/health", h.Health.Health)
	r.GET("/ready", h.Health.Ready)

	// API v1
	v1 := r.Group("/v1")
	{
		// Public auth routes
		auth := v1.Group("/auth")
		{
			auth.POST("/register", h.Auth.Register)
			auth.POST("/login", h.Auth.Login)
			auth.POST("/refresh", h.Auth.RefreshToken)
			auth.POST("/password-reset/request", h.Org.RequestPasswordReset)
			auth.POST("/password-reset/confirm", h.Org.ResetPassword)
		}

		// SDK/API routes - API key authentication
		sdk := v1.Group("")
		sdk.Use(middleware.APIKeyAuth(cfg.Auth.APIKeySalt, apiKeyValidator))
		{
			// Trace ingestion
			sdk.POST("/traces", middleware.RequireScope("trace:write"), h.Trace.IngestTrace)
			sdk.POST("/traces/batch", middleware.RequireScope("trace:write"), h.Trace.IngestBatch)
			sdk.POST("/spans", middleware.RequireScope("trace:write"), h.Trace.IngestSpan)

			// OTLP trace ingestion (OpenTelemetry Protocol)
			sdk.POST("/otlp/v1/traces", middleware.RequireScope("trace:write"), h.OTLP.IngestTraces)

			// Scores
			sdk.POST("/scores", middleware.RequireScope("trace:write"), h.Trace.SubmitScore)

			// Prompt retrieval (for SDK)
			sdk.GET("/prompts/:id/compile", middleware.RequireScope("prompt:read"), h.Prompt.Compile)

			// Guardrail evaluation
			sdk.POST("/guardrails/evaluate", middleware.RequireScope("guardrail:evaluate"), h.Guardrail.Evaluate)
		}

		// Dashboard routes - JWT authentication
		dashboard := v1.Group("")
		dashboard.Use(middleware.JWTAuth(cfg.Auth.JWTSecret))
		{
			// User profile
			dashboard.GET("/me", h.Auth.Me)
			dashboard.PUT("/me", h.Auth.UpdateProfile)
			dashboard.PUT("/me/password", h.Auth.ChangePassword)

			// Organizations
			orgs := dashboard.Group("/organizations")
			{
				orgs.GET("", h.Org.ListOrganizations)
				orgs.POST("", h.Org.CreateOrganization)
				orgs.GET("/:orgId", h.Org.GetOrganization)
				orgs.PUT("/:orgId", h.Org.UpdateOrganization)
				orgs.DELETE("/:orgId", h.Org.DeleteOrganization)
				orgs.GET("/:orgId/members", h.Org.ListMembers)
				orgs.POST("/:orgId/members", h.Org.AddMember)
				orgs.DELETE("/:orgId/members/:userId", h.Org.RemoveMember)
			}

			// Projects
			projects := dashboard.Group("/projects")
			{
				projects.GET("", h.Org.ListProjects)
				projects.POST("", h.Org.CreateProject)
				projects.GET("/:projectId", h.Org.GetProject)
				projects.PUT("/:projectId", h.Org.UpdateProject)
				projects.DELETE("/:projectId", h.Org.DeleteProject)

				// API Keys
				projects.GET("/:projectId/api-keys", h.Auth.ListAPIKeys)
				projects.POST("/:projectId/api-keys", h.Auth.CreateAPIKey)
				projects.DELETE("/:projectId/api-keys/:keyId", h.Auth.RevokeAPIKey)
			}

			// Sessions
			sessionRoutes := dashboard.Group("/sessions")
			{
				sessionRoutes.GET("", h.Org.ListSessions)
				sessionRoutes.DELETE("/:sessionId", h.Org.RevokeSession)
				sessionRoutes.DELETE("", h.Org.RevokeAllSessions)
			}

			// Traces (dashboard view)
			traces := dashboard.Group("/traces")
			{
				traces.GET("", h.Trace.ListTraces)
				traces.GET("/:id", h.Trace.GetTrace)
				traces.GET("/:id/spans", h.Trace.GetSpans)
				traces.DELETE("/:id", h.Trace.DeleteTrace)
			}

			// Sessions
			sessions := dashboard.Group("/sessions")
			{
				sessions.GET("", h.Trace.ListSessions)
				sessions.GET("/:id", h.Trace.GetSession)
			}

			// Users (tracked users from traces)
			users := dashboard.Group("/users")
			{
				users.GET("", h.Trace.ListUsers)
				users.GET("/:id", h.Trace.GetUser)
			}

			// Search
			dashboard.GET("/search/traces", h.Trace.SearchTraces)

			// Prompts
			prompts := dashboard.Group("/prompts")
			{
				prompts.GET("", h.Prompt.List)
				prompts.POST("", h.Prompt.Create)
				prompts.GET("/:id", h.Prompt.Get)
				prompts.PUT("/:id", h.Prompt.Update)
				prompts.DELETE("/:id", h.Prompt.Delete)

				// Versions
				prompts.GET("/:id/versions", h.Prompt.ListVersions)
				prompts.POST("/:id/versions", h.Prompt.CreateVersion)
				prompts.GET("/:id/versions/:version", h.Prompt.GetVersion)
			}

			// Guardrails
			guardrails := dashboard.Group("/guardrails")
			{
				guardrails.GET("", h.Guardrail.List)
				guardrails.POST("", h.Guardrail.Create)
				guardrails.GET("/:id", h.Guardrail.Get)
				guardrails.PUT("/:id", h.Guardrail.Update)
				guardrails.DELETE("/:id", h.Guardrail.Delete)

				// Rules
				guardrails.GET("/:id/rules", h.Guardrail.ListRules)
				guardrails.POST("/:id/rules", h.Guardrail.AddRule)
				guardrails.PUT("/:id/rules/:ruleId", h.Guardrail.UpdateRule)
				guardrails.DELETE("/:id/rules/:ruleId", h.Guardrail.DeleteRule)
			}

			// Analytics
			analytics := dashboard.Group("/analytics")
			{
				analytics.GET("/overview", h.Trace.GetOverview)
				analytics.GET("/costs", h.Trace.GetCostAnalytics)
				analytics.GET("/usage", h.Trace.GetUsageAnalytics)
				analytics.GET("/ingestion", h.Trace.GetIngestionStats)
			}
		}
	}

	return r
}
