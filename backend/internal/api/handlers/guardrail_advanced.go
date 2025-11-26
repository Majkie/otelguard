package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/service"
	"go.uber.org/zap"
)

// GuardrailAdvancedHandler handles advanced guardrail operations
type GuardrailAdvancedHandler struct {
	asyncService *service.AsyncEvaluationService
	batchService *service.BatchEvaluationService
	cacheService *service.CachedGuardrailService
	cbManager    *service.CircuitBreakerManager
	logger       *zap.Logger
}

// NewGuardrailAdvancedHandler creates a new advanced guardrail handler
func NewGuardrailAdvancedHandler(
	asyncService *service.AsyncEvaluationService,
	batchService *service.BatchEvaluationService,
	cacheService *service.CachedGuardrailService,
	cbManager    *service.CircuitBreakerManager,
	logger *zap.Logger,
) *GuardrailAdvancedHandler {
	return &GuardrailAdvancedHandler{
		asyncService: asyncService,
		batchService: batchService,
		cacheService: cacheService,
		cbManager:    cbManager,
		logger:       logger,
	}
}

// SubmitAsyncEvaluationRequest represents an async evaluation request
type SubmitAsyncEvaluationRequest struct {
	ProjectID   string                 `json:"project_id" binding:"required"`
	TraceID     *string                `json:"trace_id"`
	SpanID      *string                `json:"span_id"`
	PolicyID    *string                `json:"policy_id"`
	Input       string                 `json:"input" binding:"required"`
	Output      string                 `json:"output"`
	Context     map[string]interface{} `json:"context"`
	Model       string                 `json:"model"`
	Environment string                 `json:"environment"`
	Tags        []string               `json:"tags"`
	UserID      string                 `json:"user_id"`
	WebhookURL  string                 `json:"webhook_url" binding:"required,url"`
	WebhookAuth string                 `json:"webhook_auth"`
	MaxRetries  int                    `json:"max_retries"`
}

// SubmitAsyncEvaluationResponse represents the response
type SubmitAsyncEvaluationResponse struct {
	JobID     string `json:"job_id"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	QueueSize int    `json:"queue_size"`
}

// SubmitAsyncEvaluation submits an async evaluation
// @Summary Submit async evaluation
// @Tags guardrails
// @Accept json
// @Produce json
// @Param request body SubmitAsyncEvaluationRequest true "Async evaluation request"
// @Success 202 {object} SubmitAsyncEvaluationResponse
// @Failure 400 {object} ErrorResponse
// @Router /v1/guardrails/evaluate/async [post]
func (h *GuardrailAdvancedHandler) SubmitAsyncEvaluation(c *gin.Context) {
	var req SubmitAsyncEvaluationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_project_id",
			Message: "Project ID must be a valid UUID",
		})
		return
	}

	// Build evaluation input
	input := &service.EvaluationInput{
		ProjectID:   projectID,
		Input:       req.Input,
		Output:      req.Output,
		Context:     req.Context,
		Model:       req.Model,
		Environment: req.Environment,
		Tags:        req.Tags,
		UserID:      req.UserID,
	}

	if req.TraceID != nil {
		traceID, err := uuid.Parse(*req.TraceID)
		if err == nil {
			input.TraceID = &traceID
		}
	}

	if req.SpanID != nil {
		spanID, err := uuid.Parse(*req.SpanID)
		if err == nil {
			input.SpanID = &spanID
		}
	}

	if req.PolicyID != nil {
		policyID, err := uuid.Parse(*req.PolicyID)
		if err == nil {
			input.PolicyID = &policyID
		}
	}

	// Submit job
	job, err := h.asyncService.SubmitEvaluation(input, req.WebhookURL, req.WebhookAuth, req.MaxRetries)
	if err != nil {
		h.logger.Error("failed to submit async evaluation", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "submission_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, SubmitAsyncEvaluationResponse{
		JobID:     job.ID.String(),
		Status:    "pending",
		Message:   "Evaluation job submitted successfully",
		QueueSize: h.asyncService.GetQueueSize(),
	})
}

// BatchEvaluateRequest represents a batch evaluation request
type BatchEvaluateRequest struct {
	Items       []SubmitAsyncEvaluationRequest `json:"items" binding:"required,min=1,max=100"`
	MaxParallel int                            `json:"max_parallel"`
	UseCache    bool                           `json:"use_cache"`
	StopOnFailure bool                         `json:"stop_on_failure"`
}

// BatchEvaluate evaluates multiple inputs in batch
// @Summary Batch evaluate
// @Tags guardrails
// @Accept json
// @Produce json
// @Param request body BatchEvaluateRequest true "Batch evaluation request"
// @Success 200 {object} service.BatchEvaluationResponse
// @Failure 400 {object} ErrorResponse
// @Router /v1/guardrails/evaluate/batch [post]
func (h *GuardrailAdvancedHandler) BatchEvaluate(c *gin.Context) {
	var req BatchEvaluateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// Convert items to evaluation inputs
	items := make([]*service.EvaluationInput, len(req.Items))
	for i, item := range req.Items {
		projectID, err := uuid.Parse(item.ProjectID)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_project_id",
				Message: fmt.Sprintf("Item %d: invalid project ID", i),
			})
			return
		}

		input := &service.EvaluationInput{
			ProjectID:   projectID,
			Input:       item.Input,
			Output:      item.Output,
			Context:     item.Context,
			Model:       item.Model,
			Environment: item.Environment,
			Tags:        item.Tags,
			UserID:      item.UserID,
		}

		if item.TraceID != nil {
			traceID, err := uuid.Parse(*item.TraceID)
			if err == nil {
				input.TraceID = &traceID
			}
		}

		if item.SpanID != nil {
			spanID, err := uuid.Parse(*item.SpanID)
			if err == nil {
				input.SpanID = &spanID
			}
		}

		if item.PolicyID != nil {
			policyID, err := uuid.Parse(*item.PolicyID)
			if err == nil {
				input.PolicyID = &policyID
			}
		}

		items[i] = input
	}

	// Execute batch evaluation
	batchReq := &service.BatchEvaluationRequest{
		Items:         items,
		MaxParallel:   req.MaxParallel,
		UseCache:      req.UseCache,
		StopOnFailure: req.StopOnFailure,
	}

	response, err := h.batchService.Evaluate(c.Request.Context(), batchReq)
	if err != nil {
		h.logger.Error("batch evaluation failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "batch_evaluation_failed",
			Message: err.Error(),
		})
		return
	}

	// Get statistics
	stats := h.batchService.GetStatistics(response)

	c.JSON(http.StatusOK, gin.H{
		"batch":      response,
		"statistics": stats,
	})
}

// GetCacheStats returns cache statistics
// @Summary Get cache statistics
// @Tags guardrails
// @Produce json
// @Success 200 {object} service.CacheStats
// @Router /v1/guardrails/cache/stats [get]
func (h *GuardrailAdvancedHandler) GetCacheStats(c *gin.Context) {
	stats := h.cacheService.GetCacheStats()
	c.JSON(http.StatusOK, stats)
}

// InvalidateCacheRequest represents a cache invalidation request
type InvalidateCacheRequest struct {
	ProjectID string  `json:"project_id" binding:"required"`
	PolicyID  *string `json:"policy_id"`
}

// InvalidateCache invalidates cache entries
// @Summary Invalidate cache
// @Tags guardrails
// @Accept json
// @Produce json
// @Param request body InvalidateCacheRequest true "Cache invalidation request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Router /v1/guardrails/cache/invalidate [post]
func (h *GuardrailAdvancedHandler) InvalidateCache(c *gin.Context) {
	var req InvalidateCacheRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	count := h.cacheService.InvalidateCache(req.ProjectID, req.PolicyID)

	c.JSON(http.StatusOK, gin.H{
		"message":           "Cache invalidated successfully",
		"entries_cleared":   count,
	})
}

// GetCircuitBreakerStats returns circuit breaker statistics
// @Summary Get circuit breaker statistics
// @Tags guardrails
// @Produce json
// @Success 200 {object} map[string]service.CircuitBreakerStats
// @Router /v1/guardrails/circuit-breakers/stats [get]
func (h *GuardrailAdvancedHandler) GetCircuitBreakerStats(c *gin.Context) {
	stats := h.cbManager.GetAllStats()
	c.JSON(http.StatusOK, stats)
}

// ResetCircuitBreakersRequest represents a circuit breaker reset request
type ResetCircuitBreakersRequest struct {
	Name string `json:"name"` // Optional: reset specific breaker
}

// ResetCircuitBreakers resets circuit breakers
// @Summary Reset circuit breakers
// @Tags guardrails
// @Accept json
// @Produce json
// @Param request body ResetCircuitBreakersRequest true "Reset request"
// @Success 200 {object} map[string]interface{}
// @Router /v1/guardrails/circuit-breakers/reset [post]
func (h *GuardrailAdvancedHandler) ResetCircuitBreakers(c *gin.Context) {
	var req ResetCircuitBreakersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Reset all if no body provided
		h.cbManager.ResetAll()
		c.JSON(http.StatusOK, gin.H{
			"message": "All circuit breakers reset successfully",
		})
		return
	}

	if req.Name != "" {
		breaker, exists := h.cbManager.Get(req.Name)
		if !exists {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("Circuit breaker '%s' not found", req.Name),
			})
			return
		}
		breaker.Reset()
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("Circuit breaker '%s' reset successfully", req.Name),
		})
	} else {
		h.cbManager.ResetAll()
		c.JSON(http.StatusOK, gin.H{
			"message": "All circuit breakers reset successfully",
		})
	}
}
