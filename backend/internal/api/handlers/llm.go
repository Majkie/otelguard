package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/service"
	"go.uber.org/zap"
)

// LLMHandler handles LLM-related endpoints
type LLMHandler struct {
	llmService service.LLMService
	tokenizer  *service.TokenizerService
	pricing    *service.PricingService
	logger     *zap.Logger
}

// NewLLMHandler creates a new LLM handler
func NewLLMHandler(llmService service.LLMService, tokenizer *service.TokenizerService, pricing *service.PricingService, logger *zap.Logger) *LLMHandler {
	return &LLMHandler{
		llmService: llmService,
		tokenizer:  tokenizer,
		pricing:    pricing,
		logger:     logger,
	}
}

// ListModels returns all available LLM models
// @Summary List available LLM models
// @Description Get all supported LLM models across providers
// @Tags llm
// @Accept json
// @Produce json
// @Success 200 {array} domain.LLMModel
// @Router /api/v1/llm/models [get]
func (h *LLMHandler) ListModels(c *gin.Context) {
	models, err := h.llmService.ListAvailableModels()
	if err != nil {
		h.logger.Error("failed to list models", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve models",
		})
		return
	}

	c.JSON(http.StatusOK, models)
}

// ExecutePrompt executes a prompt against an LLM
// @Summary Execute LLM prompt
// @Description Execute a prompt against the specified LLM model
// @Tags llm
// @Accept json
// @Produce json
// @Param request body domain.LLMRequest true "LLM execution request"
// @Success 200 {object} domain.LLMResponse
// @Router /api/v1/llm/execute [post]
func (h *LLMHandler) ExecutePrompt(c *gin.Context) {
	var req domain.LLMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
		return
	}

	response, err := h.llmService.ExecutePrompt(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("failed to execute prompt", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "execution_error",
			"message": "Failed to execute prompt",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// StreamPrompt streams a prompt execution
// @Summary Stream LLM prompt execution
// @Description Execute a prompt with streaming response
// @Tags llm
// @Accept json
// @Produce text/event-stream
// @Param request body domain.LLMRequest true "LLM execution request"
// @Success 200 {string} string "Server-sent events stream"
// @Router /api/v1/llm/stream [post]
func (h *LLMHandler) StreamPrompt(c *gin.Context) {
	var req domain.LLMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
		return
	}

	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	respCh, errCh := h.llmService.StreamPrompt(c.Request.Context(), req)

	for {
		select {
		case response, ok := <-respCh:
			if !ok {
				return
			}
			c.SSEvent("message", response)
			c.Writer.Flush()
		case err, ok := <-errCh:
			if !ok {
				return
			}
			h.logger.Error("streaming error", zap.Error(err))
			c.SSEvent("error", gin.H{"error": err.Error()})
			c.Writer.Flush()
			return
		case <-c.Request.Context().Done():
			return
		}
	}
}

// CountTokens counts tokens in text
// @Summary Count tokens
// @Description Count tokens in the given text for a specific model
// @Tags llm
// @Accept json
// @Produce json
// @Param text query string true "Text to count tokens for"
// @Param model query string true "Model to use for token counting"
// @Success 200 {object} gin.H{"tokens": 123}
// @Router /api/v1/llm/count-tokens [get]
func (h *LLMHandler) CountTokens(c *gin.Context) {
	text := c.Query("text")
	model := c.Query("model")

	if text == "" || model == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Both 'text' and 'model' parameters are required",
		})
		return
	}

	tokens, err := h.llmService.CountTokens(text, model)
	if err != nil {
		h.logger.Error("failed to count tokens", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "token_count_error",
			"message": "Failed to count tokens",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tokens": tokens,
		"text":   text,
		"model":  model,
	})
}

// EstimateCost estimates the cost of an LLM request
// @Summary Estimate cost
// @Description Estimate the cost of executing a prompt
// @Tags llm
// @Accept json
// @Produce json
// @Param request body domain.LLMRequest true "LLM request for cost estimation"
// @Success 200 {object} gin.H{"estimatedCost": 0.123, "currency": "USD"}
// @Router /api/v1/llm/estimate-cost [post]
func (h *LLMHandler) EstimateCost(c *gin.Context) {
	var req domain.LLMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
		return
	}

	// Count input tokens
	inputTokens, err := h.llmService.CountTokens(req.Prompt, req.Model)
	if err != nil {
		h.logger.Error("failed to count input tokens", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "token_count_error",
			"message": "Failed to count tokens",
		})
		return
	}

	// Estimate output tokens (rough approximation)
	estimatedOutputTokens := h.tokenizer.EstimateOutputTokens(inputTokens, req.Model)

	// Calculate cost
	cost, err := h.llmService.EstimateCost(req, estimatedOutputTokens)
	if err != nil {
		h.logger.Error("failed to estimate cost", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "cost_estimation_error",
			"message": "Failed to estimate cost",
		})
		return
	}

	pricing, _ := h.pricing.GetPricing(req.Provider, req.Model)

	c.JSON(http.StatusOK, gin.H{
		"estimatedCost":         cost,
		"currency":              pricing.Currency,
		"inputTokens":           inputTokens,
		"estimatedOutputTokens": estimatedOutputTokens,
		"formattedCost":         h.pricing.FormatCost(cost, pricing.Currency),
	})
}

// GetCostBreakdown provides detailed cost breakdown
// @Summary Get cost breakdown
// @Description Get detailed cost breakdown for token usage
// @Tags llm
// @Accept json
// @Produce json
// @Param provider query string true "LLM provider"
// @Param model query string true "Model name"
// @Param inputTokens query int true "Number of input tokens"
// @Param outputTokens query int true "Number of output tokens"
// @Success 200 {object} service.CostBreakdown
// @Router /api/v1/llm/cost-breakdown [get]
func (h *LLMHandler) GetCostBreakdown(c *gin.Context) {
	provider := c.Query("provider")
	model := c.Query("model")
	inputTokensStr := c.Query("inputTokens")
	outputTokensStr := c.Query("outputTokens")

	if provider == "" || model == "" || inputTokensStr == "" || outputTokensStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "provider, model, inputTokens, and outputTokens parameters are required",
		})
		return
	}

	// Parse token counts
	var inputTokens, outputTokens int
	if _, err := fmt.Sscanf(inputTokensStr, "%d", &inputTokens); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "inputTokens must be a valid integer",
		})
		return
	}
	if _, err := fmt.Sscanf(outputTokensStr, "%d", &outputTokens); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "outputTokens must be a valid integer",
		})
		return
	}

	breakdown, err := h.pricing.GetCostBreakdown(provider, model, inputTokens, outputTokens)
	if err != nil {
		h.logger.Error("failed to get cost breakdown", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "cost_breakdown_error",
			"message": "Failed to get cost breakdown",
		})
		return
	}

	c.JSON(http.StatusOK, breakdown)
}
