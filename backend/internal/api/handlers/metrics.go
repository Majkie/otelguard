package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/service"
	"go.uber.org/zap"
)

// MetricsHandler handles metrics and analytics endpoints
type MetricsHandler struct {
	metricsService *service.MetricsService
	logger         *zap.Logger
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler(metricsService *service.MetricsService, logger *zap.Logger) *MetricsHandler {
	return &MetricsHandler{
		metricsService: metricsService,
		logger:         logger,
	}
}

// GetCoreMetrics returns core aggregated metrics
// @Summary Get core metrics
// @Tags metrics
// @Produce json
// @Param projectId query string true "Project ID"
// @Param startTime query string false "Start time (RFC3339)"
// @Param endTime query string false "End time (RFC3339)"
// @Param model query string false "Filter by model"
// @Param userId query string false "Filter by user ID"
// @Param sessionId query string false "Filter by session ID"
// @Success 200 {object} service.CoreMetrics
// @Router /v1/metrics/core [get]
func (h *MetricsHandler) GetCoreMetrics(c *gin.Context) {
	filter, err := h.parseFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	metrics, err := h.metricsService.GetCoreMetrics(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("failed to get core metrics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve metrics",
		})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// GetTimeSeries returns time-series data for a specific metric
// @Summary Get time-series metrics
// @Tags metrics
// @Produce json
// @Param projectId query string true "Project ID"
// @Param metric query string true "Metric name (traces, latency, cost, tokens, errors, error_rate)"
// @Param interval query string false "Time interval (hour, day, week, month)"
// @Param startTime query string false "Start time (RFC3339)"
// @Param endTime query string false "End time (RFC3339)"
// @Success 200 {object} service.TimeSeriesData
// @Router /v1/metrics/timeseries [get]
func (h *MetricsHandler) GetTimeSeries(c *gin.Context) {
	filter, err := h.parseFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	metricName := c.Query("metric")
	if metricName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "metric parameter is required",
		})
		return
	}

	interval := c.DefaultQuery("interval", "hour")

	data, err := h.metricsService.GetTimeSeriesMetrics(c.Request.Context(), filter, metricName, interval)
	if err != nil {
		h.logger.Error("failed to get time series", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, data)
}

// GetModelBreakdown returns metrics broken down by model
// @Summary Get model breakdown
// @Tags metrics
// @Produce json
// @Param projectId query string true "Project ID"
// @Param startTime query string false "Start time (RFC3339)"
// @Param endTime query string false "End time (RFC3339)"
// @Success 200 {array} service.ModelMetrics
// @Router /v1/metrics/models [get]
func (h *MetricsHandler) GetModelBreakdown(c *gin.Context) {
	filter, err := h.parseFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	breakdown, err := h.metricsService.GetModelBreakdown(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("failed to get model breakdown", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve model breakdown",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": breakdown,
	})
}

// GetUserBreakdown returns metrics broken down by user
// @Summary Get user breakdown
// @Tags metrics
// @Produce json
// @Param projectId query string true "Project ID"
// @Param limit query int false "Max users to return"
// @Param startTime query string false "Start time (RFC3339)"
// @Param endTime query string false "End time (RFC3339)"
// @Success 200 {array} service.UserMetrics
// @Router /v1/metrics/users [get]
func (h *MetricsHandler) GetUserBreakdown(c *gin.Context) {
	filter, err := h.parseFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := parseIntParam(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	breakdown, err := h.metricsService.GetUserBreakdown(c.Request.Context(), filter, limit)
	if err != nil {
		h.logger.Error("failed to get user breakdown", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve user breakdown",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": breakdown,
	})
}

// GetCostBreakdown returns comprehensive cost analytics
// @Summary Get cost breakdown
// @Tags metrics
// @Produce json
// @Param projectId query string true "Project ID"
// @Param startTime query string false "Start time (RFC3339)"
// @Param endTime query string false "End time (RFC3339)"
// @Success 200 {object} service.CostBreakdown
// @Router /v1/metrics/cost [get]
func (h *MetricsHandler) GetCostBreakdown(c *gin.Context) {
	filter, err := h.parseFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	breakdown, err := h.metricsService.GetCostBreakdown(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("failed to get cost breakdown", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve cost breakdown",
		})
		return
	}

	c.JSON(http.StatusOK, breakdown)
}

// GetQualityMetrics returns quality and evaluation metrics
// @Summary Get quality metrics
// @Tags metrics
// @Produce json
// @Param projectId query string true "Project ID"
// @Param startTime query string false "Start time (RFC3339)"
// @Param endTime query string false "End time (RFC3339)"
// @Success 200 {object} service.QualityMetrics
// @Router /v1/metrics/quality [get]
func (h *MetricsHandler) GetQualityMetrics(c *gin.Context) {
	filter, err := h.parseFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	metrics, err := h.metricsService.GetQualityMetrics(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("failed to get quality metrics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve quality metrics",
		})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// parseFilter parses query parameters into a MetricsFilter
func (h *MetricsHandler) parseFilter(c *gin.Context) (*service.MetricsFilter, error) {
	projectIDStr := c.Query("projectId")
	if projectIDStr == "" {
		return nil, gin.Error{Err: gin.Error{Err: nil}, Type: 0, Meta: "projectId is required"}
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		return nil, gin.Error{Err: err, Type: 0, Meta: "invalid projectId format"}
	}

	filter := &service.MetricsFilter{
		ProjectID: projectID,
		Model:     c.Query("model"),
		UserID:    c.Query("userId"),
		SessionID: c.Query("sessionId"),
	}

	// Parse time range (default to last 24 hours)
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	if startTimeStr := c.Query("startTime"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = t
		}
	}

	if endTimeStr := c.Query("endTime"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = t
		}
	}

	filter.StartTime = startTime
	filter.EndTime = endTime

	return filter, nil
}

// parseIntParam safely parses an integer parameter
func parseIntParam(s string) (int, error) {
	var i int
	if _, err := fmt.Sscanf(s, "%d", &i); err != nil {
		return 0, err
	}
	return i, nil
}
