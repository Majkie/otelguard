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
	"github.com/otelguard/otelguard/internal/service"
	"github.com/otelguard/otelguard/pkg/validator"
	"go.uber.org/zap"
)

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

	trace := &domain.Trace{
		ID:               traceUUID,
		ProjectID:        projectUUID,
		Name:             req.Name,
		Input:            req.Input,
		Output:           req.Output,
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

		trace := &domain.Trace{
			ID:               traceUUID,
			ProjectID:        projectUUID,
			Name:             r.Name,
			Input:            r.Input,
			Output:           r.Output,
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

	span := &domain.Span{
		ID:        spanUUID,
		TraceID:   traceUUID,
		ProjectID: projectUUID,
		Name:      req.Name,
		Type:      req.Type,
		Input:     req.Input,
		Output:    req.Output,
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

// ListTraces returns paginated traces
func (h *TraceHandler) ListTraces(c *gin.Context) {
	// Pagination
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 50
	}

	// Basic filters
	projectID := c.Query("projectId")
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

	opts := &service.ListTracesOptions{
		ProjectID:  projectID,
		SessionID:  sessionID,
		UserID:     userID,
		Model:      model,
		Name:       name,
		Status:     status,
		Tags:       tags,
		StartTime:  startTime,
		EndTime:    endTime,
		MinLatency: minLatency,
		MaxLatency: maxLatency,
		MinCost:    minCost,
		MaxCost:    maxCost,
		SortBy:     sortBy,
		SortOrder:  sortOrder,
		Limit:      limit,
		Offset:     offset,
	}

	traces, total, err := h.traceService.ListTraces(c.Request.Context(), opts)
	if err != nil {
		h.logger.Error("failed to list traces", zap.Error(err))
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
	c.JSON(http.StatusOK, gin.H{
		"totalTraces":    0,
		"totalTokens":    0,
		"totalCost":      0,
		"avgLatencyMs":   0,
		"errorRate":      0,
	})
}

// GetCostAnalytics returns cost analytics
func (h *TraceHandler) GetCostAnalytics(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"data":      []interface{}{},
		"totalCost": 0,
	})
}

// GetUsageAnalytics returns usage analytics
func (h *TraceHandler) GetUsageAnalytics(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"data":        []interface{}{},
		"totalTokens": 0,
	})
}
