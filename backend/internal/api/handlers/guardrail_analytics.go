package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/service"
	"go.uber.org/zap"
)

// GuardrailAnalyticsHandler handles guardrail analytics endpoints
type GuardrailAnalyticsHandler struct {
	analyticsService *service.GuardrailAnalyticsService
	logger           *zap.Logger
}

// NewGuardrailAnalyticsHandler creates a new guardrail analytics handler
func NewGuardrailAnalyticsHandler(
	analyticsService *service.GuardrailAnalyticsService,
	logger *zap.Logger,
) *GuardrailAnalyticsHandler {
	return &GuardrailAnalyticsHandler{
		analyticsService: analyticsService,
		logger:           logger,
	}
}

// GetTriggerStats returns trigger statistics
// GET /v1/guardrails/analytics/triggers
func (h *GuardrailAnalyticsHandler) GetTriggerStats(c *gin.Context) {
	projectID := c.Query("projectId")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "projectId is required",
		})
		return
	}

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid projectId format",
		})
		return
	}

	// Parse time range
	startTime, endTime, err := parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	stats, err := h.analyticsService.GetTriggerStats(c.Request.Context(), projectUUID, startTime, endTime)
	if err != nil {
		h.logger.Error("failed to get trigger stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve trigger statistics",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetViolationTrend returns violation trends over time
// GET /v1/guardrails/analytics/trends
func (h *GuardrailAnalyticsHandler) GetViolationTrend(c *gin.Context) {
	projectID := c.Query("projectId")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "projectId is required",
		})
		return
	}

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid projectId format",
		})
		return
	}

	// Parse time range
	startTime, endTime, err := parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	interval := c.DefaultQuery("interval", "1h")

	trends, err := h.analyticsService.GetViolationTrend(c.Request.Context(), projectUUID, startTime, endTime, interval)
	if err != nil {
		h.logger.Error("failed to get violation trends", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve violation trends",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": trends,
	})
}

// GetRemediationSuccessRates returns success rates for remediation actions
// GET /v1/guardrails/analytics/remediation-success
func (h *GuardrailAnalyticsHandler) GetRemediationSuccessRates(c *gin.Context) {
	projectID := c.Query("projectId")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "projectId is required",
		})
		return
	}

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid projectId format",
		})
		return
	}

	// Parse time range
	startTime, endTime, err := parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	rates, err := h.analyticsService.GetRemediationSuccessRates(c.Request.Context(), projectUUID, startTime, endTime)
	if err != nil {
		h.logger.Error("failed to get remediation success rates", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve remediation success rates",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": rates,
	})
}

// GetPolicyAnalytics returns detailed analytics for a specific policy
// GET /v1/guardrails/analytics/policies/:policyId
func (h *GuardrailAnalyticsHandler) GetPolicyAnalytics(c *gin.Context) {
	policyID := c.Param("policyId")
	if policyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "policyId is required",
		})
		return
	}

	policyUUID, err := uuid.Parse(policyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid policyId format",
		})
		return
	}

	// Parse time range
	startTime, endTime, err := parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	stats, err := h.analyticsService.GetPolicyAnalytics(c.Request.Context(), policyUUID, startTime, endTime)
	if err != nil {
		h.logger.Error("failed to get policy analytics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve policy analytics",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetCostImpactAnalysis returns cost impact analysis
// GET /v1/guardrails/analytics/cost-impact
func (h *GuardrailAnalyticsHandler) GetCostImpactAnalysis(c *gin.Context) {
	projectID := c.Query("projectId")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "projectId is required",
		})
		return
	}

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid projectId format",
		})
		return
	}

	// Parse time range
	startTime, endTime, err := parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	analysis, err := h.analyticsService.GetCostImpactAnalysis(c.Request.Context(), projectUUID, startTime, endTime)
	if err != nil {
		h.logger.Error("failed to get cost impact analysis", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve cost impact analysis",
		})
		return
	}

	c.JSON(http.StatusOK, analysis)
}

// GetLatencyImpact returns latency impact analysis
// GET /v1/guardrails/analytics/latency-impact
func (h *GuardrailAnalyticsHandler) GetLatencyImpact(c *gin.Context) {
	projectID := c.Query("projectId")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "projectId is required",
		})
		return
	}

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid projectId format",
		})
		return
	}

	// Parse time range
	startTime, endTime, err := parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	impact, err := h.analyticsService.GetLatencyImpact(c.Request.Context(), projectUUID, startTime, endTime)
	if err != nil {
		h.logger.Error("failed to get latency impact", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve latency impact",
		})
		return
	}

	c.JSON(http.StatusOK, impact)
}

// parseTimeRange parses start_time and end_time query parameters
func parseTimeRange(c *gin.Context) (startTime, endTime time.Time, err error) {
	// Default to last 7 days
	endTime = time.Now()
	startTime = endTime.Add(-7 * 24 * time.Hour)

	if startTimeStr := c.Query("startTime"); startTimeStr != "" {
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			return startTime, endTime, err
		}
	}

	if endTimeStr := c.Query("endTime"); endTimeStr != "" {
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			return startTime, endTime, err
		}
	}

	return startTime, endTime, nil
}
