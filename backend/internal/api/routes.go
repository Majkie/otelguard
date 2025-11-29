package api

import (
	"os"
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
	Health             *handlers.HealthHandler
	Auth               *handlers.AuthHandler
	Org                *handlers.OrgHandler
	Trace              *handlers.TraceHandler
	OTLP               *handlers.OTLPHandler
	Prompt             *handlers.PromptHandler
	Guardrail          *handlers.GuardrailHandler
	GuardrailAnalytics *handlers.GuardrailAnalyticsHandler
	LLM                *handlers.LLMHandler
	Annotation         *handlers.AnnotationHandler
	Feedback           *handlers.FeedbackHandler
	Agent              *handlers.AgentHandler
	Evaluator          *handlers.EvaluatorHandler
	Dataset            *handlers.DatasetHandler
	Experiment         *handlers.ExperimentHandler
	ScoreAnalytics     *handlers.ScoreAnalyticsHandler
	Metrics            *handlers.MetricsHandler
	Dashboard          *handlers.DashboardHandler
	Alert              *handlers.AlertHandler
	WebSocket          *handlers.WebSocketHandler
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

	// CORS configuration for cookie-based authentication
	corsOrigins := []string{"http://localhost:3000", "http://localhost:5173"}
	if cfg.IsProduction() {
		// In production, restrict to specific domains
		corsOrigins = []string{os.Getenv("CORS_ALLOWED_ORIGINS")}
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-API-Key", "X-Request-ID", "X-CSRF-Token"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check endpoints (no auth required)
	r.GET("/health", h.Health.Health)
	r.GET("/ready", h.Health.Ready)

	// WebSocket endpoint (requires authentication via query param or header)
	r.GET("/ws", h.WebSocket.ServeWS)

	// API v1
	v1 := r.Group("/v1")
	{
		// Public auth routes
		auth := v1.Group("/auth")
		{
			auth.POST("/register", h.Auth.Register)
			auth.POST("/login", h.Auth.Login)
			auth.POST("/refresh", h.Auth.RefreshToken)
			auth.POST("/logout", h.Auth.Logout)
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
			sdk.POST("/guardrails/evaluate/batch", middleware.RequireScope("guardrail:evaluate"), h.Guardrail.BatchEvaluate)
			sdk.POST("/guardrails/evaluate/async", middleware.RequireScope("guardrail:evaluate"), h.Guardrail.AsyncEvaluate)
		}

		// Dashboard routes - JWT authentication with auto-refresh
		dashboard := v1.Group("")
		dashboard.Use(middleware.AutoRefreshAuth(cfg.Auth.JWTSecret, 15*time.Minute)) // Refresh when < 15 minutes left
		dashboard.Use(middleware.CSRFProtection())                                    // CSRF protection for state-changing operations
		dashboard.Use(middleware.SetProjectContext())                                 // Extract project_id from query params
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
				sessionRoutes.GET("/:id", h.Trace.GetSession)
				sessionRoutes.DELETE("/:sessionId", h.Org.RevokeSession)
				sessionRoutes.DELETE("", h.Org.RevokeAllSessions)
			}

			// Traces (dashboard view)
			traces := dashboard.Group("/traces")
			{
				traces.GET("", h.Trace.ListTraces)
				traces.GET("/:traceId", h.Trace.GetTrace)
				traces.GET("/:traceId/spans", h.Trace.GetSpans)
				traces.DELETE("/:traceId", h.Trace.DeleteTrace)
			}

			// Scores (dashboard view)
			scores := dashboard.Group("/scores")
			{
				scores.GET("", h.Trace.ListScores)
				scores.GET("/:scoreId", h.Trace.GetScoreByID)
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
				prompts.POST("/:id/duplicate", h.Prompt.Duplicate)
				prompts.GET("/:id/compare", h.Prompt.CompareVersions)
				prompts.POST("/extract-variables", h.Prompt.ExtractVariables)

				// Versions
				prompts.GET("/:id/versions", h.Prompt.ListVersions)
				prompts.POST("/:id/versions", h.Prompt.CreateVersion)
				prompts.GET("/:id/versions/:version", h.Prompt.GetVersion)
				prompts.PUT("/:id/versions/:version/labels", h.Prompt.UpdateVersionLabels)
				prompts.POST("/:id/versions/:version/promote", h.Prompt.PromoteVersion)
				prompts.GET("/:id/versions/by-label/:label", h.Prompt.GetVersionByLabel)
				prompts.GET("/:id/analytics", h.Prompt.GetAnalytics)
				prompts.GET("/:id/traces", h.Prompt.GetLinkedTraces)
				prompts.GET("/:id/performance", h.Prompt.GetPerformanceMetrics)
				prompts.GET("/:id/regressions", h.Prompt.DetectRegressions)
			}

			// Guardrails
			guardrails := dashboard.Group("/guardrails")
			{
				// Policies
				guardrails.GET("/policies", h.Guardrail.List)
				guardrails.POST("/policies", h.Guardrail.Create)
				guardrails.GET("/policies/:id", h.Guardrail.Get)
				guardrails.PUT("/policies/:id", h.Guardrail.Update)
				guardrails.DELETE("/policies/:id", h.Guardrail.Delete)

				// Rules
				guardrails.GET("/policies/:id/rules", h.Guardrail.ListRules)
				guardrails.POST("/policies/:id/rules", h.Guardrail.AddRule)
				guardrails.PUT("/policies/:id/rules/:ruleId", h.Guardrail.UpdateRule)
				guardrails.DELETE("/policies/:id/rules/:ruleId", h.Guardrail.DeleteRule)

				// Versions
				guardrails.POST("/policies/:id/versions", h.Guardrail.CreateVersion)
				guardrails.GET("/policies/:id/versions", h.Guardrail.ListVersions)
				guardrails.GET("/policies/:id/versions/:version", h.Guardrail.GetVersion)
				guardrails.POST("/policies/:id/versions/:version/restore", h.Guardrail.RestoreVersion)

				// Policy Testing
				guardrails.POST("/policies/:id/test", h.Guardrail.TestPolicy)
				guardrails.POST("/test", h.Guardrail.TestPolicy)

				// Cache Management
				guardrails.GET("/cache/stats", h.Guardrail.GetCacheStats)
				guardrails.POST("/cache/clear", h.Guardrail.ClearCache)
				guardrails.POST("/cache/invalidate", h.Guardrail.InvalidateCache)

				// Async Evaluation Jobs
				guardrails.GET("/jobs", h.Guardrail.ListAsyncJobs)
				guardrails.GET("/jobs/:jobId", h.Guardrail.GetAsyncJob)

				// Analytics
				guardrails.GET("/analytics/triggers", h.GuardrailAnalytics.GetTriggerStats)
				guardrails.GET("/analytics/trends", h.GuardrailAnalytics.GetViolationTrend)
				guardrails.GET("/analytics/remediation-success", h.GuardrailAnalytics.GetRemediationSuccessRates)
				guardrails.GET("/analytics/policies/:policyId", h.GuardrailAnalytics.GetPolicyAnalytics)
				guardrails.GET("/analytics/cost-impact", h.GuardrailAnalytics.GetCostImpactAnalysis)
				guardrails.GET("/analytics/latency-impact", h.GuardrailAnalytics.GetLatencyImpact)
			}

			// LLM
			llm := dashboard.Group("/llm")
			{
				llm.GET("/models", h.LLM.ListModels)
				llm.POST("/execute", h.LLM.ExecutePrompt)
				llm.POST("/stream", h.LLM.StreamPrompt)
				llm.GET("/count-tokens", h.LLM.CountTokens)
				llm.POST("/estimate-cost", h.LLM.EstimateCost)
				llm.GET("/cost-breakdown", h.LLM.GetCostBreakdown)
			}

			// Analytics
			analytics := dashboard.Group("/analytics")
			{
				analytics.GET("/overview", h.Trace.GetOverview)
				analytics.GET("/costs", h.Trace.GetCostAnalytics)
				analytics.GET("/usage", h.Trace.GetUsageAnalytics)
				analytics.GET("/ingestion", h.Trace.GetIngestionStats)

				// Score analytics
				analytics.GET("/scores/aggregations", h.Trace.GetScoreAggregations)
				analytics.GET("/scores/trends", h.Trace.GetScoreTrends)
				analytics.GET("/scores/comparisons", h.Trace.GetScoreComparisons)
				analytics.GET("/scores/distribution", h.ScoreAnalytics.GetScoreDistribution)
				analytics.GET("/scores/correlation", h.ScoreAnalytics.GetCorrelation)
				analytics.GET("/scores/breakdown", h.ScoreAnalytics.GetScoreBreakdown)
				analytics.GET("/scores/cohen-kappa", h.ScoreAnalytics.GetCohenKappa)
				analytics.GET("/scores/f1", h.ScoreAnalytics.GetF1Score)
				analytics.GET("/scores/trend", h.ScoreAnalytics.GetScoreTrend)

				// Agent analytics
				analytics.GET("/agents", h.Agent.GetAgentStatistics)
				analytics.GET("/tool-calls", h.Agent.GetToolCallStatistics)
			}

			// Metrics (new aggregated metrics endpoints)
			metrics := dashboard.Group("/metrics")
			{
				// Core metrics
				metrics.GET("/core", h.Metrics.GetCoreMetrics)

				// Time series
				metrics.GET("/timeseries", h.Metrics.GetTimeSeries)

				// Breakdowns
				metrics.GET("/models", h.Metrics.GetModelBreakdown)
				metrics.GET("/users", h.Metrics.GetUserBreakdown)
				metrics.GET("/cost", h.Metrics.GetCostBreakdown)
				metrics.GET("/quality", h.Metrics.GetQualityMetrics)
			}

			// Dashboards (custom dashboard management)
			dashboards := dashboard.Group("/dashboards")
			{
				// Dashboard CRUD
				dashboards.POST("", h.Dashboard.CreateDashboard)
				dashboards.GET("", h.Dashboard.ListDashboards)
				dashboards.GET("/:dashboardId", h.Dashboard.GetDashboard)
				dashboards.PUT("/:dashboardId", h.Dashboard.UpdateDashboard)
				dashboards.DELETE("/:dashboardId", h.Dashboard.DeleteDashboard)

				// Widget management
				dashboards.POST("/:dashboardId/widgets", h.Dashboard.AddWidget)
				dashboards.PUT("/:dashboardId/widgets/:widgetId", h.Dashboard.UpdateWidget)
				dashboards.DELETE("/:dashboardId/widgets/:widgetId", h.Dashboard.DeleteWidget)

				// Layout management (drag-and-drop)
				dashboards.PUT("/:dashboardId/layout", h.Dashboard.UpdateLayout)

				// Dashboard sharing
				dashboards.POST("/:dashboardId/share", h.Dashboard.CreateShare)
				dashboards.GET("/:dashboardId/shares", h.Dashboard.ListShares)
				dashboards.DELETE("/:dashboardId/shares/:shareId", h.Dashboard.DeleteShare)

				// Dashboard cloning
				dashboards.POST("/:dashboardId/clone", h.Dashboard.CloneDashboard)
			}

			// Shared dashboards (public access)
			dashboard.GET("/dashboards/shared/:token", h.Dashboard.GetSharedDashboard)

			// Agents (multi-agent visualization)
			agents := dashboard.Group("/agents")
			{
				agents.GET("", h.Agent.ListAgents)
				agents.POST("", h.Agent.CreateAgent)
				agents.GET("/:id", h.Agent.GetAgent)
				agents.GET("/:id/tool-calls", h.Agent.GetToolCallsByAgent)
				agents.GET("/:id/states", h.Agent.GetAgentStates)
			}

			// Agent graphs for traces
			dashboard.GET("/traces/:traceId/agents", h.Agent.GetAgentsByTrace)
			dashboard.GET("/traces/:traceId/agents/hierarchy", h.Agent.GetAgentHierarchy)
			dashboard.POST("/traces/:traceId/agents/detect", h.Agent.DetectAgents)
			dashboard.GET("/traces/:traceId/tool-calls", h.Agent.GetToolCallsByTrace)
			dashboard.GET("/traces/:traceId/agent-messages", h.Agent.GetAgentMessages)
			dashboard.GET("/traces/:traceId/graph", h.Agent.GetAgentGraph)
			dashboard.GET("/traces/:traceId/graph/:nodeId/subgraph", h.Agent.GetSubgraph)

			// Annotation queues
			annotationQueues := dashboard.Group("/annotation-queues")
			{
				annotationQueues.POST("", h.Annotation.CreateQueue)
				annotationQueues.GET("/:queueId", h.Annotation.GetQueue)
				annotationQueues.PUT("/:queueId", h.Annotation.UpdateQueue)
				annotationQueues.DELETE("/:queueId", h.Annotation.DeleteQueue)
				annotationQueues.GET("/:queueId/items", h.Annotation.ListQueueItems)
				annotationQueues.POST("/:queueId/items", h.Annotation.CreateQueueItem)
				annotationQueues.POST("/:queueId/assign", h.Annotation.AssignNextItem)
				annotationQueues.GET("/:queueId/stats", h.Annotation.GetQueueStats)
				annotationQueues.POST("/:queueId/items/:queueItemId/agreement", h.Annotation.CalculateAgreement)
				annotationQueues.GET("/:queueId/agreements", h.Annotation.GetQueueAgreements)
				annotationQueues.GET("/:queueId/agreement-stats", h.Annotation.GetQueueAgreementStats)
				annotationQueues.GET("/:queueId/export", h.Annotation.ExportAnnotations)
			}

			// Project-specific annotation routes
			projectRoutes := dashboard.Group("/projects")
			{
				projectRoutes.GET("/:projectId/annotation-queues", h.Annotation.ListQueuesByProject)
			}

			// Annotation assignments
			assignments := dashboard.Group("/annotation-assignments")
			{
				assignments.POST("/:assignmentId/start", h.Annotation.StartAssignment)
				assignments.POST("/:assignmentId/skip", h.Annotation.SkipAssignment)
			}

			// Annotations
			annotations := dashboard.Group("/annotations")
			{
				annotations.POST("", h.Annotation.CreateAnnotation)
				annotations.GET("/:annotationId", h.Annotation.GetAnnotation)
			}

			// Queue items
			queueItems := dashboard.Group("/annotation-queue-items")
			{
				queueItems.GET("/:queueItemId/annotations", h.Annotation.ListAnnotationsByQueueItem)
			}

			// Feedback
			feedback := dashboard.Group("/feedback")
			{
				feedback.POST("", h.Feedback.CreateFeedback)
				feedback.GET("", h.Feedback.ListFeedback)
				feedback.GET("/:id", h.Feedback.GetFeedback)
				feedback.PUT("/:id", h.Feedback.UpdateFeedback)
				feedback.DELETE("/:id", h.Feedback.DeleteFeedback)
				feedback.GET("/analytics", h.Feedback.GetFeedbackAnalytics)
				feedback.GET("/trends", h.Feedback.GetFeedbackTrends)
			}

			// Evaluators (LLM-as-a-Judge)
			evaluators := dashboard.Group("/evaluators")
			{
				evaluators.GET("", h.Evaluator.ListEvaluators)
				evaluators.POST("", h.Evaluator.CreateEvaluator)
				evaluators.GET("/templates", h.Evaluator.GetTemplates)
				evaluators.GET("/templates/:templateId", h.Evaluator.GetTemplate)
				evaluators.GET("/:id", h.Evaluator.GetEvaluator)
				evaluators.PUT("/:id", h.Evaluator.UpdateEvaluator)
				evaluators.DELETE("/:id", h.Evaluator.DeleteEvaluator)
			}

			// Evaluations
			evaluations := dashboard.Group("/evaluations")
			{
				evaluations.POST("/run", h.Evaluator.RunEvaluation)
				evaluations.POST("/batch", h.Evaluator.BatchEvaluation)
				evaluations.GET("/jobs", h.Evaluator.ListJobs)
				evaluations.GET("/jobs/:jobId", h.Evaluator.GetJob)
				evaluations.GET("/results", h.Evaluator.GetResults)
				evaluations.GET("/stats", h.Evaluator.GetStats)
				evaluations.GET("/costs", h.Evaluator.GetCostSummary)
			}

			// Datasets
			datasets := dashboard.Group("/datasets")
			{
				datasets.GET("", h.Dataset.List)
				datasets.POST("", h.Dataset.Create)
				datasets.GET("/:id", h.Dataset.Get)
				datasets.PUT("/:id", h.Dataset.Update)
				datasets.DELETE("/:id", h.Dataset.Delete)

				// Dataset items
				datasets.GET("/:id/items", h.Dataset.ListItems)
				datasets.POST("/items", h.Dataset.CreateItem)
				datasets.GET("/items/:itemId", h.Dataset.GetItem)
				datasets.PUT("/items/:itemId", h.Dataset.UpdateItem)
				datasets.DELETE("/items/:itemId", h.Dataset.DeleteItem)

				// Import
				datasets.POST("/import", h.Dataset.Import)
			}

			// Experiments
			experiments := dashboard.Group("/experiments")
			{
				experiments.GET("", h.Experiment.List)
				experiments.POST("", h.Experiment.Create)
				experiments.GET("/datasets/:datasetId", h.Experiment.ListByDataset)
				experiments.GET("/:id", h.Experiment.Get)
				experiments.POST("/:id/execute", h.Experiment.Execute)

				// Runs
				experiments.GET("/:id/runs", h.Experiment.ListRuns)
				experiments.GET("/runs/:runId", h.Experiment.GetRun)
				experiments.GET("/runs/:runId/results", h.Experiment.GetResults)

				// Comparison
				experiments.POST("/compare", h.Experiment.CompareRuns)
				experiments.POST("/statistical-comparison", h.Experiment.StatisticalComparison)

				// Scheduling
				experiments.POST("/schedules", h.Experiment.CreateSchedule)
				experiments.GET("/schedules", h.Experiment.ListSchedules)
				experiments.GET("/schedules/:scheduleId", h.Experiment.GetSchedule)
				experiments.PUT("/schedules/:scheduleId", h.Experiment.UpdateSchedule)
				experiments.DELETE("/schedules/:scheduleId", h.Experiment.DeleteSchedule)
			}

			// Alerts
			alerts := dashboard.Group("/projects/:projectId/alerts")
			{
				// Alert rules
				alerts.POST("/rules", h.Alert.CreateAlertRule)
				alerts.GET("/rules", h.Alert.ListAlertRules)
				alerts.GET("/rules/:ruleId", h.Alert.GetAlertRule)
				alerts.PUT("/rules/:ruleId", h.Alert.UpdateAlertRule)
				alerts.DELETE("/rules/:ruleId", h.Alert.DeleteAlertRule)

				// Alert history
				alerts.GET("/history", h.Alert.ListAlertHistory)
				alerts.GET("/history/:alertId", h.Alert.GetAlertHistory)
				alerts.POST("/history/:alertId/acknowledge", h.Alert.AcknowledgeAlert)

				// Manual evaluation trigger
				alerts.POST("/evaluate", h.Alert.EvaluateAlerts)
			}

			// User-specific routes
			user := dashboard.Group("/user")
			{
				user.GET("/annotation-assignments", h.Annotation.ListUserAssignments)
				user.GET("/annotation-stats", h.Annotation.GetUserStats)
			}
		}
	}

	return r
}
