package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/api/middleware"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/repository/clickhouse"
	"github.com/otelguard/otelguard/internal/service"
	"github.com/otelguard/otelguard/pkg/validator"
	"go.uber.org/zap"
)

const (
	// MaxInputOutputSize is the maximum size for input/output fields in bytes (500KB)
	MaxInputOutputSize = 500000
	// TruncationSuffix is appended to truncated content
	TruncationSuffix = "\n...[truncated]"
)

// truncateString truncates a string to maxLen bytes, appending a suffix if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	// Find a safe place to cut (avoid breaking UTF-8 sequences)
	truncateAt := maxLen - len(TruncationSuffix)
	if truncateAt < 0 {
		truncateAt = 0
	}
	// Ensure we don't cut in the middle of a UTF-8 character
	for truncateAt > 0 && s[truncateAt]&0xC0 == 0x80 {
		truncateAt--
	}
	return s[:truncateAt] + TruncationSuffix
}

// TraceHandler handles trace-related endpoints
type TraceHandler struct {
	traceService *service.TraceService
	logger       *zap.Logger
}

// NewTraceHandler creates a new trace handler
func NewTraceHandler(traceService *service.TraceService, logger *zap.Logger) *TraceHandler {
	return &TraceHandler{
		traceService: traceService,
		logger:       logger,
	}
}

// IngestTraceRequest represents a trace ingestion request
type IngestTraceRequest struct {
	ID               string   `json:"id,omitempty" binding:"omitempty,uuid"`
	SessionID        string   `json:"sessionId,omitempty" binding:"max=255"`
	UserID           string   `json:"userId,omitempty" binding:"max=255"`
	Name             string   `json:"name" binding:"required,min=1,max=255"`
	Input            string   `json:"input" binding:"max=1000000"`
	Output           string   `json:"output" binding:"max=1000000"`
	Metadata         any      `json:"metadata,omitempty"`
	StartTime        string   `json:"startTime"`
	EndTime          string   `json:"endTime"`
	LatencyMs        uint32   `json:"latencyMs" binding:"gte=0"`
	TotalTokens      uint32   `json:"totalTokens" binding:"gte=0"`
	PromptTokens     uint32   `json:"promptTokens" binding:"gte=0"`
	CompletionTokens uint32   `json:"completionTokens" binding:"gte=0"`
	Cost             float64  `json:"cost" binding:"gte=0"`
	Model            string   `json:"model" binding:"max=100"`
	Tags             []string `json:"tags" binding:"max=50,dive,max=50"`
	Status           string   `json:"status" binding:"omitempty,status"`
	ErrorMessage     string   `json:"errorMessage,omitempty" binding:"max=5000"`
	PromptID         string   `json:"promptId,omitempty" binding:"omitempty,uuid"`
	PromptVersion    *int     `json:"promptVersion,omitempty"`
}

// IngestTraceResponse represents the trace ingestion response
type IngestTraceResponse struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
}

// IngestTrace handles single trace ingestion
func (h *TraceHandler) IngestTrace(c *gin.Context) {
	var req IngestTraceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		validationErrors := validator.ParseValidationErrors(err)
		if len(validationErrors) > 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "validation_error",
				"message": validationErrors.Error(),
				"details": validationErrors,
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	projectID := c.GetString(string(middleware.ContextProjectID))
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_project",
			"message": "Project ID is required",
		})
		return
	}

	// Generate ID if not provided
	traceID := req.ID
	if traceID == "" {
		traceID = uuid.New().String()
	}

	// Parse timestamps
	startTime := time.Now()
	endTime := time.Now()
	if req.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, req.StartTime); err == nil {
			startTime = t
		}
	}
	if req.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, req.EndTime); err == nil {
			endTime = t
		}
	}

	projectUUID, _ := uuid.Parse(projectID)
	traceUUID, _ := uuid.Parse(traceID)

	// Apply truncation to large input/output fields
	input := truncateString(req.Input, MaxInputOutputSize)
	output := truncateString(req.Output, MaxInputOutputSize)

	trace := &domain.Trace{
		ID:               traceUUID,
		ProjectID:        projectUUID,
		Name:             req.Name,
		Input:            input,
		Output:           output,
		StartTime:        startTime,
		EndTime:          endTime,
		LatencyMs:        req.LatencyMs,
		TotalTokens:      req.TotalTokens,
		PromptTokens:     req.PromptTokens,
		CompletionTokens: req.CompletionTokens,
		Cost:             req.Cost,
		Model:            req.Model,
		Tags:             req.Tags,
		Status:           req.Status,
	}

	if req.SessionID != "" {
		trace.SessionID = &req.SessionID
	}
	if req.UserID != "" {
		trace.UserID = &req.UserID
	}
	if req.ErrorMessage != "" {
		trace.ErrorMessage = &req.ErrorMessage
	}
	if req.PromptID != "" {
		promptUUID, _ := uuid.Parse(req.PromptID)
		trace.PromptID = &promptUUID
	}
	if req.PromptVersion != nil {
		trace.PromptVersion = req.PromptVersion
	}
	if trace.Status == "" {
		trace.Status = domain.StatusSuccess
	}

	if err := h.traceService.IngestTrace(c.Request.Context(), trace); err != nil {
		h.logger.Error("failed to ingest trace",
			zap.Error(err),
			zap.String("project_id", projectID),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "ingestion_failed",
			"message": "Failed to ingest trace",
		})
		return
	}

	c.JSON(http.StatusCreated, IngestTraceResponse{
		ID:        traceID,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// IngestBatch handles batch trace ingestion
func (h *TraceHandler) IngestBatch(c *gin.Context) {
	var req struct {
		Traces []IngestTraceRequest `json:"traces" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	projectID := c.GetString(string(middleware.ContextProjectID))
	projectUUID, _ := uuid.Parse(projectID)

	traces := make([]*domain.Trace, 0, len(req.Traces))
	for _, r := range req.Traces {
		traceID := r.ID
		if traceID == "" {
			traceID = uuid.New().String()
		}
		traceUUID, _ := uuid.Parse(traceID)

		// Apply truncation to large input/output fields
		input := truncateString(r.Input, MaxInputOutputSize)
		output := truncateString(r.Output, MaxInputOutputSize)

		trace := &domain.Trace{
			ID:               traceUUID,
			ProjectID:        projectUUID,
			Name:             r.Name,
			Input:            input,
			Output:           output,
			StartTime:        time.Now(),
			EndTime:          time.Now(),
			LatencyMs:        r.LatencyMs,
			TotalTokens:      r.TotalTokens,
			PromptTokens:     r.PromptTokens,
			CompletionTokens: r.CompletionTokens,
			Cost:             r.Cost,
			Model:            r.Model,
			Tags:             r.Tags,
			Status:           domain.StatusSuccess,
		}
		if r.SessionID != "" {
			trace.SessionID = &r.SessionID
		}
		if r.UserID != "" {
			trace.UserID = &r.UserID
		}
		if r.PromptID != "" {
			promptUUID, _ := uuid.Parse(r.PromptID)
			trace.PromptID = &promptUUID
		}
		if r.PromptVersion != nil {
			trace.PromptVersion = r.PromptVersion
		}
		traces = append(traces, trace)
	}

	if err := h.traceService.IngestBatch(c.Request.Context(), traces); err != nil {
		h.logger.Error("failed to ingest batch", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "ingestion_failed",
			"message": "Failed to ingest traces",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"count":     len(traces),
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// IngestSpan handles span ingestion
func (h *TraceHandler) IngestSpan(c *gin.Context) {
	var req struct {
		TraceID      string  `json:"traceId" binding:"required"`
		ParentSpanID string  `json:"parentSpanId,omitempty"`
		ID           string  `json:"id,omitempty"`
		Name         string  `json:"name" binding:"required"`
		Type         string  `json:"type" binding:"required"`
		Input        string  `json:"input"`
		Output       string  `json:"output"`
		StartTime    string  `json:"startTime"`
		EndTime      string  `json:"endTime"`
		LatencyMs    uint32  `json:"latencyMs"`
		Tokens       uint32  `json:"tokens"`
		Cost         float64 `json:"cost"`
		Model        string  `json:"model,omitempty"`
		Status       string  `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	projectID := c.GetString(string(middleware.ContextProjectID))
	projectUUID, _ := uuid.Parse(projectID)

	spanID := req.ID
	if spanID == "" {
		spanID = uuid.New().String()
	}

	spanUUID, _ := uuid.Parse(spanID)
	traceUUID, _ := uuid.Parse(req.TraceID)

	// Apply truncation to large input/output fields
	input := truncateString(req.Input, MaxInputOutputSize)
	output := truncateString(req.Output, MaxInputOutputSize)

	span := &domain.Span{
		ID:        spanUUID,
		TraceID:   traceUUID,
		ProjectID: projectUUID,
		Name:      req.Name,
		Type:      req.Type,
		Input:     input,
		Output:    output,
		StartTime: time.Now(),
		EndTime:   time.Now(),
		LatencyMs: req.LatencyMs,
		Tokens:    req.Tokens,
		Cost:      req.Cost,
		Status:    domain.StatusSuccess,
	}

	if req.ParentSpanID != "" {
		parentUUID, _ := uuid.Parse(req.ParentSpanID)
		span.ParentSpanID = &parentUUID
	}
	if req.Model != "" {
		span.Model = &req.Model
	}

	if err := h.traceService.IngestSpan(c.Request.Context(), span); err != nil {
		h.logger.Error("failed to ingest span", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "ingestion_failed",
			"message": "Failed to ingest span",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":        spanID,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// SubmitScore handles score submission
func (h *TraceHandler) SubmitScore(c *gin.Context) {
	var req struct {
		TraceID     string  `json:"traceId" binding:"required"`
		SpanID      string  `json:"spanId,omitempty"`
		Name        string  `json:"name" binding:"required"`
		Value       float64 `json:"value"`
		StringValue string  `json:"stringValue,omitempty"`
		DataType    string  `json:"dataType" binding:"required"`
		Source      string  `json:"source"`
		Comment     string  `json:"comment,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	projectID := c.GetString(string(middleware.ContextProjectID))
	projectUUID, _ := uuid.Parse(projectID)
	traceUUID, _ := uuid.Parse(req.TraceID)

	score := &domain.Score{
		ID:        uuid.New(),
		ProjectID: projectUUID,
		TraceID:   traceUUID,
		Name:      req.Name,
		Value:     req.Value,
		DataType:  req.DataType,
		Source:    req.Source,
		CreatedAt: time.Now(),
	}

	if req.SpanID != "" {
		spanUUID, _ := uuid.Parse(req.SpanID)
		score.SpanID = &spanUUID
	}
	if req.StringValue != "" {
		score.StringValue = &req.StringValue
	}
	if req.Comment != "" {
		score.Comment = &req.Comment
	}
	if score.Source == "" {
		score.Source = "api"
	}

	if err := h.traceService.SubmitScore(c.Request.Context(), score); err != nil {
		h.logger.Error("failed to submit score", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "submission_failed",
			"message": "Failed to submit score",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":        score.ID.String(),
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// ListScores returns paginated scores with filtering
func (h *TraceHandler) ListScores(c *gin.Context) {
	projectIDStr := c.Query("projectId")
	if projectIDStr == "" {
		h.logger.Warn("projectId not found in query parameters")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "projectId is required",
		})
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.logger.Error("invalid projectId format", zap.String("projectId", projectIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid projectId format",
		})
		return
	}

	filter := &clickhouse.ScoreFilter{
		ProjectID: projectID,
	}

	// Optional filters
	if traceIDStr := c.Query("traceId"); traceIDStr != "" {
		if traceID, err := uuid.Parse(traceIDStr); err == nil {
			filter.TraceID = &traceID
		}
	}

	if spanIDStr := c.Query("spanId"); spanIDStr != "" {
		if spanID, err := uuid.Parse(spanIDStr); err == nil {
			filter.SpanID = &spanID
		}
	}

	if name := c.Query("name"); name != "" {
		filter.Name = &name
	}

	if source := c.Query("source"); source != "" {
		filter.Source = &source
	}

	if dataType := c.Query("dataType"); dataType != "" {
		filter.DataType = &dataType
	}

	if startTimeStr := c.Query("startTime"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			filter.StartTime = &startTime
		}
	}

	if endTimeStr := c.Query("endTime"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			filter.EndTime = &endTime
		}
	}

	// Pagination
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 1000 {
		limit = 1000
	}
	if limit < 1 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	filter.Limit = limit
	filter.Offset = offset

	scores, total, err := h.traceService.GetScores(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("failed to get scores", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve scores",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"scores": scores,
		"pagination": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetScoreByID retrieves a single score by ID
func (h *TraceHandler) GetScoreByID(c *gin.Context) {
	projectIDStr := c.Query("projectId")
	scoreIDStr := c.Param("scoreId")

	if projectIDStr == "" {
		h.logger.Warn("projectId not found in query parameters")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "projectId is required",
		})
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.logger.Error("invalid projectId format", zap.String("projectId", projectIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid projectId format",
		})
		return
	}

	scoreID, err := uuid.Parse(scoreIDStr)
	if err != nil {
		h.logger.Error("invalid scoreId format", zap.String("scoreId", scoreIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid scoreId format",
		})
		return
	}

	score, err := h.traceService.GetScoreByID(c.Request.Context(), projectID, scoreID)
	if err != nil {
		h.logger.Error("failed to get score", zap.String("scoreId", scoreID.String()), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve score",
		})
		return
	}

	if score == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Score not found",
		})
		return
	}

	c.JSON(http.StatusOK, score)
}

// GetScoreAggregations retrieves aggregated statistics for scores
func (h *TraceHandler) GetScoreAggregations(c *gin.Context) {
	projectIDStr := c.Query("projectId")
	if projectIDStr == "" {
		h.logger.Warn("projectId not found in query parameters")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "projectId is required",
		})
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.logger.Error("invalid projectId format", zap.String("projectId", projectIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid projectId format",
		})
		return
	}

	filter := &clickhouse.ScoreFilter{
		ProjectID: projectID,
	}

	// Optional filters (same as ListScores)
	if traceIDStr := c.Query("traceId"); traceIDStr != "" {
		if traceID, err := uuid.Parse(traceIDStr); err == nil {
			filter.TraceID = &traceID
		}
	}

	if spanIDStr := c.Query("spanId"); spanIDStr != "" {
		if spanID, err := uuid.Parse(spanIDStr); err == nil {
			filter.SpanID = &spanID
		}
	}

	if name := c.Query("name"); name != "" {
		filter.Name = &name
	}

	if source := c.Query("source"); source != "" {
		filter.Source = &source
	}

	if startTimeStr := c.Query("startTime"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			filter.StartTime = &startTime
		}
	}

	if endTimeStr := c.Query("endTime"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			filter.EndTime = &endTime
		}
	}

	aggregations, err := h.traceService.GetScoreAggregations(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("failed to get score aggregations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve score aggregations",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"aggregations": aggregations,
	})
}

// GetScoreTrends retrieves score trends over time
func (h *TraceHandler) GetScoreTrends(c *gin.Context) {
	projectIDStr := c.Query("projectId")
	if projectIDStr == "" {
		h.logger.Warn("projectId not found in query parameters")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "projectId is required",
		})
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.logger.Error("invalid projectId format", zap.String("projectId", projectIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid projectId format",
		})
		return
	}

	filter := &clickhouse.ScoreFilter{
		ProjectID: projectID,
	}

	// Optional filters
	if traceIDStr := c.Query("traceId"); traceIDStr != "" {
		if traceID, err := uuid.Parse(traceIDStr); err == nil {
			filter.TraceID = &traceID
		}
	}

	if spanIDStr := c.Query("spanId"); spanIDStr != "" {
		if spanID, err := uuid.Parse(spanIDStr); err == nil {
			filter.SpanID = &spanID
		}
	}

	if name := c.Query("name"); name != "" {
		filter.Name = &name
	}

	if source := c.Query("source"); source != "" {
		filter.Source = &source
	}

	if startTimeStr := c.Query("startTime"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			filter.StartTime = &startTime
		}
	}

	if endTimeStr := c.Query("endTime"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			filter.EndTime = &endTime
		}
	}

	groupBy := c.DefaultQuery("groupBy", "day")
	if groupBy != "hour" && groupBy != "day" && groupBy != "week" && groupBy != "month" {
		groupBy = "day"
	}

	trends, err := h.traceService.GetScoreTrends(c.Request.Context(), filter, groupBy)
	if err != nil {
		h.logger.Error("failed to get score trends", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve score trends",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"trends":  trends,
		"groupBy": groupBy,
	})
}

// GetScoreComparisons retrieves score comparisons across dimensions
func (h *TraceHandler) GetScoreComparisons(c *gin.Context) {
	projectIDStr := c.Query("projectId")
	if projectIDStr == "" {
		h.logger.Warn("projectId not found in query parameters")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "projectId is required",
		})
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		h.logger.Error("invalid projectId format", zap.String("projectId", projectIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid projectId format",
		})
		return
	}

	dimension := c.Query("dimension")
	if dimension == "" {
		h.logger.Warn("dimension not found in query parameters")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "dimension is required (model, user, session, prompt)",
		})
		return
	}

	if dimension != "model" && dimension != "user" && dimension != "session" && dimension != "prompt" {
		h.logger.Error("invalid dimension", zap.String("dimension", dimension))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "dimension must be one of: model, user, session, prompt",
		})
		return
	}

	filter := &clickhouse.ScoreFilter{
		ProjectID: projectID,
	}

	// Optional filters
	if name := c.Query("name"); name != "" {
		filter.Name = &name
	}

	if source := c.Query("source"); source != "" {
		filter.Source = &source
	}

	if startTimeStr := c.Query("startTime"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			filter.StartTime = &startTime
		}
	}

	if endTimeStr := c.Query("endTime"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			filter.EndTime = &endTime
		}
	}

	comparisons, err := h.traceService.GetScoreComparisons(c.Request.Context(), filter, dimension)
	if err != nil {
		h.logger.Error("failed to get score comparisons", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve score comparisons",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"comparisons": comparisons,
		"dimension":   dimension,
	})
}

// ListTraces returns paginated traces
func (h *TraceHandler) ListTraces(c *gin.Context) {
	// Basic filters
	projectID := c.Query("projectId")
	if projectID == "" {
		h.logger.Warn("projectId not found in query parameters")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "projectId is required",
		})
		return
	}

	// Validate UUID format
	if _, err := uuid.Parse(projectID); err != nil {
		h.logger.Error("invalid projectId format", zap.String("projectId", projectID), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid projectId format",
		})
		return
	}

	// Pagination
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 50
	}
	sessionID := c.Query("sessionId")
	userID := c.Query("userId")
	model := c.Query("model")
	name := c.Query("name")
	status := c.Query("status")

	// Tags filter (comma-separated)
	var tags []string
	if tagsParam := c.Query("tags"); tagsParam != "" {
		for _, tag := range splitAndTrim(tagsParam, ",") {
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}

	// Time filters
	startTime := c.Query("startTime")
	endTime := c.Query("endTime")

	// Numeric filters
	minLatency, _ := strconv.Atoi(c.DefaultQuery("minLatency", "0"))
	maxLatency, _ := strconv.Atoi(c.DefaultQuery("maxLatency", "0"))
	minCost, _ := strconv.ParseFloat(c.DefaultQuery("minCost", "0"), 64)
	maxCost, _ := strconv.ParseFloat(c.DefaultQuery("maxCost", "0"), 64)

	// Sorting
	sortBy := c.DefaultQuery("sortBy", "start_time")
	sortOrder := c.DefaultQuery("sortOrder", "DESC")

	// Prompt filtering
	promptID := c.Query("promptId")
	promptVersion := c.Query("promptVersion")

	opts := &service.ListTracesOptions{
		ProjectID:     projectID,
		SessionID:     sessionID,
		UserID:        userID,
		Model:         model,
		Name:          name,
		Status:        status,
		Tags:          tags,
		StartTime:     startTime,
		EndTime:       endTime,
		MinLatency:    minLatency,
		MaxLatency:    maxLatency,
		MinCost:       minCost,
		MaxCost:       maxCost,
		PromptID:      promptID,
		PromptVersion: promptVersion,
		SortBy:        sortBy,
		SortOrder:     sortOrder,
		Limit:         limit,
		Offset:        offset,
	}

	traces, total, err := h.traceService.ListTraces(c.Request.Context(), opts)
	if err != nil {
		h.logger.Error("failed to list traces",
			zap.String("projectId", projectID),
			zap.Int("limit", limit),
			zap.Int("offset", offset),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve traces",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   traces,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// splitAndTrim splits a string by delimiter and trims each part
func splitAndTrim(s, sep string) []string {
	parts := make([]string, 0)
	for _, part := range stringsplit(s, sep) {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func stringsplit(s, sep string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, sep)
}

// GetTrace returns a single trace
func (h *TraceHandler) GetTrace(c *gin.Context) {
	id := c.Param("id")

	trace, err := h.traceService.GetTrace(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Trace not found",
		})
		return
	}

	c.JSON(http.StatusOK, trace)
}

// GetSpans returns spans for a trace
func (h *TraceHandler) GetSpans(c *gin.Context) {
	traceID := c.Param("id")

	spans, err := h.traceService.GetSpans(c.Request.Context(), traceID)
	if err != nil {
		h.logger.Error("failed to get spans", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve spans",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": spans,
	})
}

// DeleteTrace deletes a trace
func (h *TraceHandler) DeleteTrace(c *gin.Context) {
	id := c.Param("id")

	if err := h.traceService.DeleteTrace(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to delete trace",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Trace deleted",
	})
}

// ListSessions returns sessions with aggregated metrics
func (h *TraceHandler) ListSessions(c *gin.Context) {
	// Pagination
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 50
	}

	// Filters
	projectID := c.Query("projectId")
	userID := c.Query("userId")
	startTime := c.Query("startTime")
	endTime := c.Query("endTime")

	opts := &service.ListSessionsOptions{
		ProjectID: projectID,
		UserID:    userID,
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     limit,
		Offset:    offset,
	}

	sessions, total, err := h.traceService.ListSessions(c.Request.Context(), opts)
	if err != nil {
		h.logger.Error("failed to list sessions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve sessions",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   sessions,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetSession returns a single session with traces
func (h *TraceHandler) GetSession(c *gin.Context) {
	sessionID := c.Param("id")

	// Get session summary
	session, err := h.traceService.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Session not found",
		})
		return
	}

	// Get traces for this session
	limit, _ := strconv.Atoi(c.DefaultQuery("traceLimit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("traceOffset", "0"))

	traces, traceTotal, err := h.traceService.GetSessionTraces(c.Request.Context(), sessionID, limit, offset)
	if err != nil {
		h.logger.Error("failed to get session traces", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve session traces",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session": session,
		"traces": gin.H{
			"data":   traces,
			"total":  traceTotal,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetOverview returns analytics overview
func (h *TraceHandler) GetOverview(c *gin.Context) {
	projectID := c.Query("projectId")
	startTime := c.Query("startTime")
	endTime := c.Query("endTime")

	opts := &service.AnalyticsOptions{
		ProjectID: projectID,
		StartTime: startTime,
		EndTime:   endTime,
	}

	metrics, err := h.traceService.GetOverviewMetrics(c.Request.Context(), opts)
	if err != nil {
		h.logger.Error("failed to get overview metrics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve overview metrics",
		})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// GetCostAnalytics returns cost analytics
func (h *TraceHandler) GetCostAnalytics(c *gin.Context) {
	projectID := c.Query("projectId")
	startTime := c.Query("startTime")
	endTime := c.Query("endTime")
	granularity := c.DefaultQuery("granularity", "day")

	opts := &service.AnalyticsOptions{
		ProjectID:   projectID,
		StartTime:   startTime,
		EndTime:     endTime,
		Granularity: granularity,
	}

	timeSeries, totalCost, byModel, err := h.traceService.GetCostAnalytics(c.Request.Context(), opts)
	if err != nil {
		h.logger.Error("failed to get cost analytics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve cost analytics",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      timeSeries,
		"totalCost": totalCost,
		"byModel":   byModel,
	})
}

// GetUsageAnalytics returns usage analytics
func (h *TraceHandler) GetUsageAnalytics(c *gin.Context) {
	projectID := c.Query("projectId")
	startTime := c.Query("startTime")
	endTime := c.Query("endTime")
	granularity := c.DefaultQuery("granularity", "day")

	opts := &service.AnalyticsOptions{
		ProjectID:   projectID,
		StartTime:   startTime,
		EndTime:     endTime,
		Granularity: granularity,
	}

	timeSeries, totalTokens, err := h.traceService.GetUsageAnalytics(c.Request.Context(), opts)
	if err != nil {
		h.logger.Error("failed to get usage analytics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve usage analytics",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        timeSeries,
		"totalTokens": totalTokens,
	})
}

// ListUsers returns paginated users with aggregated metrics
func (h *TraceHandler) ListUsers(c *gin.Context) {
	// Pagination
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 50
	}

	// Filters
	projectID := c.Query("projectId")
	startTime := c.Query("startTime")
	endTime := c.Query("endTime")

	opts := &service.ListUsersOptions{
		ProjectID: projectID,
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     limit,
		Offset:    offset,
	}

	users, total, err := h.traceService.ListUsers(c.Request.Context(), opts)
	if err != nil {
		h.logger.Error("failed to list users", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve users",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   users,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetUser returns a single user with aggregated metrics and traces
func (h *TraceHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")

	// Get user summary
	user, err := h.traceService.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "User not found",
		})
		return
	}

	// Get traces for this user
	traceLimit, _ := strconv.Atoi(c.DefaultQuery("traceLimit", "50"))
	traceOffset, _ := strconv.Atoi(c.DefaultQuery("traceOffset", "0"))

	traces, traceTotal, err := h.traceService.GetUserTraces(c.Request.Context(), userID, traceLimit, traceOffset)
	if err != nil {
		h.logger.Error("failed to get user traces", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve user traces",
		})
		return
	}

	// Get sessions for this user
	sessionLimit, _ := strconv.Atoi(c.DefaultQuery("sessionLimit", "20"))
	sessionOffset, _ := strconv.Atoi(c.DefaultQuery("sessionOffset", "0"))

	sessions, sessionTotal, err := h.traceService.GetUserSessions(c.Request.Context(), userID, sessionLimit, sessionOffset)
	if err != nil {
		h.logger.Error("failed to get user sessions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve user sessions",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
		"traces": gin.H{
			"data":   traces,
			"total":  traceTotal,
			"limit":  traceLimit,
			"offset": traceOffset,
		},
		"sessions": gin.H{
			"data":   sessions,
			"total":  sessionTotal,
			"limit":  sessionLimit,
			"offset": sessionOffset,
		},
	})
}

// SearchTraces performs full-text search on trace content
func (h *TraceHandler) SearchTraces(c *gin.Context) {
	// Pagination
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 50
	}

	// Search query
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_query",
			"message": "Search query (q) is required",
		})
		return
	}

	// Filters
	projectID := c.Query("projectId")
	startTime := c.Query("startTime")
	endTime := c.Query("endTime")

	opts := &service.SearchTracesOptions{
		ProjectID: projectID,
		Query:     query,
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     limit,
		Offset:    offset,
	}

	traces, total, err := h.traceService.SearchTraces(c.Request.Context(), opts)
	if err != nil {
		h.logger.Error("failed to search traces", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to search traces",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   traces,
		"total":  total,
		"limit":  limit,
		"offset": offset,
		"query":  query,
	})
}

// GetIngestionStats returns ingestion statistics (batch writer and sampler)
func (h *TraceHandler) GetIngestionStats(c *gin.Context) {
	response := gin.H{}

	// Get batch writer metrics
	batchMetrics := h.traceService.GetBatchWriterMetrics()
	if batchMetrics != nil {
		response["batchWriter"] = batchMetrics
	}

	// Get sampler metrics
	samplerStats := h.traceService.GetSamplerStats()
	if samplerStats != nil {
		response["sampler"] = samplerStats
	}

	if len(response) == 0 {
		response["message"] = "No ingestion features enabled"
	}

	c.JSON(http.StatusOK, response)
}
