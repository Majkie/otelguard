package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/service"
	"go.uber.org/zap"
)

type ScoreAnalyticsHandler struct {
	analyticsService *service.ScoreAnalyticsService
	logger           *zap.Logger
}

func NewScoreAnalyticsHandler(
	analyticsService *service.ScoreAnalyticsService,
	logger *zap.Logger,
) *ScoreAnalyticsHandler {
	return &ScoreAnalyticsHandler{
		analyticsService: analyticsService,
		logger:           logger,
	}
}

// GetScoreDistribution returns distribution statistics for a score
func (h *ScoreAnalyticsHandler) GetScoreDistribution(c *gin.Context) {
	scoreName := c.Query("score_name")
	if scoreName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "score_name is required"})
		return
	}

	projectID, err := uuid.Parse(c.GetString("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project_id"})
		return
	}

	startTime, endTime := parseTimeRangeSimple(c)

	distribution, err := h.analyticsService.GetScoreDistribution(
		c.Request.Context(),
		projectID,
		scoreName,
		startTime,
		endTime,
	)

	if err != nil {
		h.logger.Error("failed to get score distribution", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get distribution"})
		return
	}

	c.JSON(http.StatusOK, distribution)
}

// GetCorrelation returns correlation between two scores
func (h *ScoreAnalyticsHandler) GetCorrelation(c *gin.Context) {
	score1 := c.Query("score1")
	score2 := c.Query("score2")

	if score1 == "" || score2 == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "both score1 and score2 are required"})
		return
	}

	projectID, err := uuid.Parse(c.GetString("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project_id"})
		return
	}

	startTime, endTime := parseTimeRangeSimple(c)

	correlation, err := h.analyticsService.GetCorrelation(
		c.Request.Context(),
		projectID,
		score1,
		score2,
		startTime,
		endTime,
	)

	if err != nil {
		h.logger.Error("failed to get correlation", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get correlation"})
		return
	}

	c.JSON(http.StatusOK, correlation)
}

// GetScoreBreakdown returns score statistics by dimension
func (h *ScoreAnalyticsHandler) GetScoreBreakdown(c *gin.Context) {
	scoreName := c.Query("score_name")
	dimension := c.Query("dimension")

	if scoreName == "" || dimension == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "score_name and dimension are required"})
		return
	}

	projectID, err := uuid.Parse(c.GetString("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project_id"})
		return
	}

	startTime, endTime := parseTimeRangeSimple(c)

	breakdown, err := h.analyticsService.GetScoreBreakdown(
		c.Request.Context(),
		projectID,
		scoreName,
		dimension,
		startTime,
		endTime,
	)

	if err != nil {
		h.logger.Error("failed to get score breakdown", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get breakdown"})
		return
	}

	c.JSON(http.StatusOK, breakdown)
}

// GetCohenKappa calculates inter-annotator agreement
func (h *ScoreAnalyticsHandler) GetCohenKappa(c *gin.Context) {
	scoreName := c.Query("score_name")
	annotator1Str := c.Query("annotator1")
	annotator2Str := c.Query("annotator2")

	if scoreName == "" || annotator1Str == "" || annotator2Str == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "score_name, annotator1, and annotator2 are required"})
		return
	}

	annotator1, err := uuid.Parse(annotator1Str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid annotator1 UUID"})
		return
	}

	annotator2, err := uuid.Parse(annotator2Str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid annotator2 UUID"})
		return
	}

	projectID, err := uuid.Parse(c.GetString("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project_id"})
		return
	}

	startTime, endTime := parseTimeRangeSimple(c)

	kappa, err := h.analyticsService.CalculateCohenKappa(
		c.Request.Context(),
		projectID,
		scoreName,
		annotator1,
		annotator2,
		startTime,
		endTime,
	)

	if err != nil {
		h.logger.Error("failed to calculate Cohen's Kappa", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to calculate kappa"})
		return
	}

	c.JSON(http.StatusOK, kappa)
}

// GetF1Score calculates F1 score and related metrics
func (h *ScoreAnalyticsHandler) GetF1Score(c *gin.Context) {
	scoreName := c.Query("score_name")
	groundTruthSource := c.Query("ground_truth_source")

	if scoreName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "score_name is required"})
		return
	}

	var req struct {
		Threshold float64 `json:"threshold"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Threshold = 0.5 // Default threshold
	}

	projectID, err := uuid.Parse(c.GetString("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project_id"})
		return
	}

	startTime, endTime := parseTimeRangeSimple(c)

	f1Result, err := h.analyticsService.CalculateF1Score(
		c.Request.Context(),
		projectID,
		scoreName,
		req.Threshold,
		groundTruthSource,
		startTime,
		endTime,
	)

	if err != nil {
		h.logger.Error("failed to calculate F1 score", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to calculate F1"})
		return
	}

	c.JSON(http.StatusOK, f1Result)
}

// GetScoreTrend returns score trends over time
func (h *ScoreAnalyticsHandler) GetScoreTrend(c *gin.Context) {
	scoreName := c.Query("score_name")
	intervalStr := c.DefaultQuery("interval", "1d")

	if scoreName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "score_name is required"})
		return
	}

	projectID, err := uuid.Parse(c.GetString("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project_id"})
		return
	}

	startTime, endTime := parseTimeRangeSimple(c)

	// Parse interval
	interval := 24 * time.Hour // Default: 1 day
	if intervalStr != "" {
		switch intervalStr {
		case "1h":
			interval = time.Hour
		case "6h":
			interval = 6 * time.Hour
		case "12h":
			interval = 12 * time.Hour
		case "1d":
			interval = 24 * time.Hour
		case "1w":
			interval = 7 * 24 * time.Hour
		}
	}

	trend, err := h.analyticsService.GetScoreTrend(
		c.Request.Context(),
		projectID,
		scoreName,
		startTime,
		endTime,
		interval,
	)

	if err != nil {
		h.logger.Error("failed to get score trend", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get trend"})
		return
	}

	c.JSON(http.StatusOK, trend)
}

// Helper function to parse time range from query parameters
func parseTimeRangeSimple(c *gin.Context) (time.Time, time.Time) {
	// Default: last 7 days
	endTime := time.Now()
	startTime := endTime.Add(-7 * 24 * time.Hour)

	if startStr := c.Query("start_time"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			startTime = t
		}
	}

	if endStr := c.Query("end_time"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			endTime = t
		}
	}

	return startTime, endTime
}
