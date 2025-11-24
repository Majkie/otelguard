package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/api/middleware"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/service"
	"go.uber.org/zap"
)

// FeedbackHandler handles feedback endpoints
type FeedbackHandler struct {
	feedbackService *service.FeedbackService
	logger          *zap.Logger
}

// NewFeedbackHandler creates a new feedback handler
func NewFeedbackHandler(feedbackService *service.FeedbackService, logger *zap.Logger) *FeedbackHandler {
	return &FeedbackHandler{
		feedbackService: feedbackService,
		logger:          logger,
	}
}

// CreateFeedback handles POST /v1/feedback
func (h *FeedbackHandler) CreateFeedback(c *gin.Context) {
	var req CreateFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	projectID := c.GetString(string(middleware.ContextProjectID))
	if projectID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "authentication_error",
			"message": "Project context not found",
		})
		return
	}

	req.ProjectID = uuid.MustParse(projectID)

	createReq := &domain.UserFeedbackCreate{
		ProjectID: req.ProjectID,
		UserID:    req.UserID,
		SessionID: req.SessionID,
		TraceID:   req.TraceID,
		SpanID:    req.SpanID,
		ItemType:  req.ItemType,
		ItemID:    req.ItemID,
		ThumbsUp:  req.ThumbsUp,
		Rating:    req.Rating,
		Comment:   req.Comment,
		Metadata:  req.Metadata,
	}

	feedback, err := h.feedbackService.CreateFeedback(c.Request.Context(), createReq)
	if err != nil {
		h.logger.Error("failed to create feedback", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to create feedback",
		})
		return
	}

	c.JSON(http.StatusCreated, h.feedbackToResponse(feedback))
}

// ListFeedback handles GET /v1/feedback
func (h *FeedbackHandler) ListFeedback(c *gin.Context) {
	var filter FeedbackFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// Set defaults
	if filter.Limit == 0 {
		filter.Limit = 50
	}
	if filter.Limit > 1000 {
		filter.Limit = 1000
	}

	projectID := c.GetString(string(middleware.ContextProjectID))
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "projectId is required",
		})
		return
	}

	domainFilter := domain.FeedbackFilter{
		ProjectID: projectID,
		UserID:    filter.UserID,
		ItemType:  filter.ItemType,
		ItemID:    filter.ItemID,
		TraceID:   filter.TraceID,
		SessionID: filter.SessionID,
		ThumbsUp:  filter.ThumbsUp,
		Rating:    filter.Rating,
		OrderBy:   filter.OrderBy,
		OrderDesc: filter.OrderDesc,
		Limit:     filter.Limit,
		Offset:    filter.Offset,
	}

	// Parse dates
	if filter.StartDate != "" {
		if startDate, err := time.Parse("2006-01-02", filter.StartDate); err == nil {
			domainFilter.StartDate = startDate
		}
	}

	if filter.EndDate != "" {
		if endDate, err := time.Parse("2006-01-02", filter.EndDate); err == nil {
			domainFilter.EndDate = endDate
		}
	}

	feedback, total, err := h.feedbackService.ListFeedback(c.Request.Context(), domainFilter)
	if err != nil {
		h.logger.Error("failed to list feedback", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to list feedback",
		})
		return
	}

	response := FeedbackListResponse{
		Feedback: make([]*FeedbackResponse, len(feedback)),
		Total:    total,
		Limit:    filter.Limit,
		Offset:   filter.Offset,
	}

	for i, f := range feedback {
		response.Feedback[i] = h.feedbackToResponse(f)
	}

	c.JSON(http.StatusOK, response)
}

// GetFeedback handles GET /v1/feedback/:id
func (h *FeedbackHandler) GetFeedback(c *gin.Context) {
	id := c.Param("id")

	feedback, err := h.feedbackService.GetFeedback(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("failed to get feedback", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Feedback not found",
		})
		return
	}

	c.JSON(http.StatusOK, h.feedbackToResponse(feedback))
}

// UpdateFeedback handles PUT /v1/feedback/:id
func (h *FeedbackHandler) UpdateFeedback(c *gin.Context) {
	id := c.Param("id")

	var req UpdateFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	updateReq := &domain.UserFeedbackUpdate{
		ThumbsUp: req.ThumbsUp,
		Rating:   req.Rating,
		Comment:  req.Comment,
		Metadata: req.Metadata,
	}

	feedback, err := h.feedbackService.UpdateFeedback(c.Request.Context(), id, updateReq)
	if err != nil {
		h.logger.Error("failed to update feedback", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to update feedback",
		})
		return
	}

	c.JSON(http.StatusOK, h.feedbackToResponse(feedback))
}

// DeleteFeedback handles DELETE /v1/feedback/:id
func (h *FeedbackHandler) DeleteFeedback(c *gin.Context) {
	id := c.Param("id")

	if err := h.feedbackService.DeleteFeedback(c.Request.Context(), id); err != nil {
		h.logger.Error("failed to delete feedback", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to delete feedback",
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// GetFeedbackAnalytics handles GET /v1/feedback/analytics
func (h *FeedbackHandler) GetFeedbackAnalytics(c *gin.Context) {
	projectID := c.GetString(string(middleware.ContextProjectID))
	itemType := c.Query("itemType")
	startDateStr := c.Query("startDate")
	endDateStr := c.Query("endDate")

	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "projectId is required",
		})
		return
	}

	if itemType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "itemType is required",
		})
		return
	}

	// Parse dates
	startDate := time.Now().AddDate(0, -30, 0) // Default to last 30 days
	endDate := time.Now()

	if startDateStr != "" {
		if sd, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = sd
		}
	}

	if endDateStr != "" {
		if ed, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = ed
		}
	}

	analytics, err := h.feedbackService.GetFeedbackAnalytics(c.Request.Context(), projectID, itemType, startDate, endDate)
	if err != nil {
		h.logger.Error("failed to get feedback analytics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to get feedback analytics",
		})
		return
	}

	response := FeedbackAnalyticsResponse{
		ProjectID:       analytics.ProjectID.String(),
		ItemType:        analytics.ItemType,
		TotalFeedback:   analytics.TotalFeedback,
		ThumbsUpCount:   analytics.ThumbsUpCount,
		ThumbsDownCount: analytics.ThumbsDownCount,
		RatingCounts:    analytics.RatingCounts,
		CommentCount:    analytics.CommentCount,
		DateRange:       analytics.DateRange,
		Trends:          make([]FeedbackTrendResponse, len(analytics.Trends)),
	}

	if analytics.AverageRating.Valid {
		response.AverageRating = &analytics.AverageRating.Float64
	}

	for i, trend := range analytics.Trends {
		response.Trends[i] = FeedbackTrendResponse{
			Date:          trend.Date,
			TotalFeedback: trend.TotalFeedback,
			ThumbsUpRate:  trend.ThumbsUpRate,
			AverageRating: trend.AverageRating,
			CommentCount:  trend.CommentCount,
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetFeedbackTrends handles GET /v1/feedback/trends
func (h *FeedbackHandler) GetFeedbackTrends(c *gin.Context) {
	projectID := c.GetString(string(middleware.ContextProjectID))
	itemType := c.Query("itemType")
	startDateStr := c.Query("startDate")
	endDateStr := c.Query("endDate")
	interval := c.DefaultQuery("interval", "day")

	if itemType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "itemType is required",
		})
		return
	}

	// Parse dates
	startDate := time.Now().AddDate(0, -30, 0) // Default to last 30 days
	endDate := time.Now()

	if startDateStr != "" {
		if sd, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = sd
		}
	}

	if endDateStr != "" {
		if ed, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = ed
		}
	}

	trends, err := h.feedbackService.GetFeedbackTrends(c.Request.Context(), projectID, itemType, startDate, endDate, interval)
	if err != nil {
		h.logger.Error("failed to get feedback trends", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to get feedback trends",
		})
		return
	}

	response := make([]FeedbackTrendResponse, len(trends))
	for i, trend := range trends {
		response[i] = FeedbackTrendResponse{
			Date:          trend.Date,
			TotalFeedback: trend.TotalFeedback,
			ThumbsUpRate:  trend.ThumbsUpRate,
			AverageRating: trend.AverageRating,
			CommentCount:  trend.CommentCount,
		}
	}

	c.JSON(http.StatusOK, response)
}

// Helper methods

func (h *FeedbackHandler) feedbackToResponse(feedback *domain.UserFeedback) *FeedbackResponse {
	response := &FeedbackResponse{
		ID:        feedback.ID.String(),
		ProjectID: feedback.ProjectID.String(),
		ItemType:  feedback.ItemType,
		ItemID:    feedback.ItemID,
		Comment:   feedback.Comment,
		UserAgent: feedback.UserAgent,
		IPAddress: feedback.IPAddress,
		CreatedAt: feedback.CreatedAt.Format(time.RFC3339),
		UpdatedAt: feedback.UpdatedAt.Format(time.RFC3339),
	}

	if feedback.UserID != nil {
		userID := feedback.UserID.String()
		response.UserID = &userID
	}

	if feedback.SessionID != nil {
		response.SessionID = feedback.SessionID
	}

	if feedback.TraceID != nil {
		response.TraceID = feedback.TraceID
	}

	if feedback.SpanID != nil {
		response.SpanID = feedback.SpanID
	}

	if feedback.ThumbsUp.Valid {
		response.ThumbsUp = &feedback.ThumbsUp.Bool
	}

	if feedback.Rating.Valid {
		rating := int(feedback.Rating.Int32)
		response.Rating = &rating
	}

	if len(feedback.Metadata) > 0 {
		var metadata map[string]interface{}
		if err := json.Unmarshal(feedback.Metadata, &metadata); err == nil {
			response.Metadata = metadata
		}
	}

	return response
}

func (h *FeedbackHandler) addClientInfo(c *gin.Context, metadata map[string]interface{}) map[string]interface{} {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	// Add user agent
	if userAgent := c.GetHeader("User-Agent"); userAgent != "" {
		metadata["userAgent"] = userAgent
	}

	// Add IP address
	if ip := c.ClientIP(); ip != "" {
		metadata["ipAddress"] = ip
	}

	// Add referrer
	if referrer := c.GetHeader("Referer"); referrer != "" {
		metadata["referrer"] = referrer
	}

	return metadata
}

// Request/Response types (keeping the existing ones that were defined)

// Request/Response types

// FeedbackResponse represents user feedback in API responses
type FeedbackResponse struct {
	ID        string                 `json:"id"`
	ProjectID string                 `json:"projectId"`
	UserID    *string                `json:"userId,omitempty"`
	SessionID *string                `json:"sessionId,omitempty"`
	TraceID   *string                `json:"traceId,omitempty"`
	SpanID    *string                `json:"spanId,omitempty"`
	ItemType  string                 `json:"itemType"`
	ItemID    string                 `json:"itemId"`
	ThumbsUp  *bool                  `json:"thumbsUp,omitempty"`
	Rating    *int                   `json:"rating,omitempty"`
	Comment   string                 `json:"comment,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	UserAgent string                 `json:"userAgent,omitempty"`
	IPAddress string                 `json:"ipAddress,omitempty"`
	CreatedAt string                 `json:"createdAt"`
	UpdatedAt string                 `json:"updatedAt"`
}

// FeedbackListResponse represents a list of feedback with pagination
type FeedbackListResponse struct {
	Feedback []*FeedbackResponse `json:"feedback"`
	Total    int64               `json:"total"`
	Limit    int                 `json:"limit"`
	Offset   int                 `json:"offset"`
}

// FeedbackAnalyticsResponse represents feedback analytics
type FeedbackAnalyticsResponse struct {
	ProjectID       string                  `json:"projectId"`
	ItemType        string                  `json:"itemType"`
	TotalFeedback   int64                   `json:"totalFeedback"`
	ThumbsUpCount   int64                   `json:"thumbsUpCount"`
	ThumbsDownCount int64                   `json:"thumbsDownCount"`
	AverageRating   *float64                `json:"averageRating,omitempty"`
	RatingCounts    map[int]int64           `json:"ratingCounts"`
	CommentCount    int64                   `json:"commentCount"`
	DateRange       string                  `json:"dateRange"`
	Trends          []FeedbackTrendResponse `json:"trends,omitempty"`
}

// FeedbackTrendResponse represents feedback trends
type FeedbackTrendResponse struct {
	Date          string  `json:"date"`
	TotalFeedback int64   `json:"totalFeedback"`
	ThumbsUpRate  float64 `json:"thumbsUpRate"`
	AverageRating float64 `json:"averageRating"`
	CommentCount  int64   `json:"commentCount"`
}

// Request Types

// CreateFeedbackRequest represents a request to create feedback
type CreateFeedbackRequest struct {
	ProjectID uuid.UUID              `json:"projectId" binding:"required"`
	UserID    *uuid.UUID             `json:"userId,omitempty"`
	SessionID *string                `json:"sessionId,omitempty"`
	TraceID   *string                `json:"traceId,omitempty"`
	SpanID    *string                `json:"spanId,omitempty"`
	ItemType  string                 `json:"itemType" binding:"required,oneof=trace session span prompt"`
	ItemID    string                 `json:"itemId" binding:"required"`
	ThumbsUp  *bool                  `json:"thumbsUp,omitempty"`
	Rating    *int                   `json:"rating,omitempty" binding:"omitempty,min=1,max=5"`
	Comment   string                 `json:"comment,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateFeedbackRequest represents a request to update feedback
type UpdateFeedbackRequest struct {
	ThumbsUp *bool                   `json:"thumbsUp,omitempty"`
	Rating   *int                    `json:"rating,omitempty" binding:"omitempty,min=1,max=5"`
	Comment  *string                 `json:"comment,omitempty"`
	Metadata *map[string]interface{} `json:"metadata,omitempty"`
}

// FeedbackFilter represents query parameters for filtering feedback
type FeedbackFilter struct {
	ProjectID string `form:"projectId"`
	UserID    string `form:"userId"`
	ItemType  string `form:"itemType"`
	ItemID    string `form:"itemId"`
	TraceID   string `form:"traceId"`
	SessionID string `form:"sessionId"`
	ThumbsUp  *bool  `form:"thumbsUp"`
	Rating    *int   `form:"rating"`
	StartDate string `form:"startDate"`
	EndDate   string `form:"endDate"`
	OrderBy   string `form:"orderBy"`
	OrderDesc bool   `form:"orderDesc"`
	Limit     int    `form:"limit"`
	Offset    int    `form:"offset"`
}
