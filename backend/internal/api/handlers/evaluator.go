package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/service"
)

// EvaluatorHandler handles HTTP requests for LLM-as-a-Judge evaluations
type EvaluatorHandler struct {
	evaluatorService *service.EvaluatorService
	logger           *zap.Logger
}

// NewEvaluatorHandler creates a new EvaluatorHandler
func NewEvaluatorHandler(evaluatorService *service.EvaluatorService, logger *zap.Logger) *EvaluatorHandler {
	return &EvaluatorHandler{
		evaluatorService: evaluatorService,
		logger:           logger,
	}
}

// CreateEvaluator creates a new evaluator configuration
func (h *EvaluatorHandler) CreateEvaluator(c *gin.Context) {
	var req domain.EvaluatorCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// Get project ID from context
	projectIDStr := c.GetString("project_id")
	if projectIDStr == "" {
		projectIDStr = c.Query("project_id")
	}
	if projectIDStr == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "missing_project_id",
			Message: "project_id is required",
		})
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_project_id",
			Message: "invalid project_id format",
		})
		return
	}
	req.ProjectID = projectID

	evaluator, err := h.evaluatorService.CreateEvaluator(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("failed to create evaluator", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "failed to create evaluator",
		})
		return
	}

	c.JSON(http.StatusCreated, evaluator)
}

// GetEvaluator retrieves an evaluator by ID
func (h *EvaluatorHandler) GetEvaluator(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "invalid evaluator ID format",
		})
		return
	}

	evaluator, err := h.evaluatorService.GetEvaluator(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "evaluator not found",
			})
			return
		}
		h.logger.Error("failed to get evaluator", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "failed to retrieve evaluator",
		})
		return
	}

	c.JSON(http.StatusOK, evaluator)
}

// UpdateEvaluator updates an evaluator
func (h *EvaluatorHandler) UpdateEvaluator(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "invalid evaluator ID format",
		})
		return
	}

	var req domain.EvaluatorUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	evaluator, err := h.evaluatorService.UpdateEvaluator(c.Request.Context(), id, &req)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "evaluator not found",
			})
			return
		}
		h.logger.Error("failed to update evaluator", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "failed to update evaluator",
		})
		return
	}

	c.JSON(http.StatusOK, evaluator)
}

// DeleteEvaluator deletes an evaluator
func (h *EvaluatorHandler) DeleteEvaluator(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "invalid evaluator ID format",
		})
		return
	}

	if err := h.evaluatorService.DeleteEvaluator(c.Request.Context(), id); err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "evaluator not found",
			})
			return
		}
		h.logger.Error("failed to delete evaluator", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "failed to delete evaluator",
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ListEvaluators lists evaluators based on filter criteria
func (h *EvaluatorHandler) ListEvaluators(c *gin.Context) {
	filter := &domain.EvaluatorFilter{
		ProjectID:  c.Query("project_id"),
		Type:       c.Query("type"),
		Provider:   c.Query("provider"),
		OutputType: c.Query("output_type"),
		Search:     c.Query("search"),
	}

	if enabledStr := c.Query("enabled"); enabledStr != "" {
		enabled := enabledStr == "true"
		filter.Enabled = &enabled
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}
	if filter.Limit == 0 {
		filter.Limit = 50
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}

	evaluators, total, err := h.evaluatorService.ListEvaluators(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("failed to list evaluators", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "failed to list evaluators",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   evaluators,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// GetTemplates returns all built-in evaluation templates
func (h *EvaluatorHandler) GetTemplates(c *gin.Context) {
	templates := h.evaluatorService.GetTemplates()
	c.JSON(http.StatusOK, gin.H{
		"data": templates,
	})
}

// GetTemplate returns a specific template by ID
func (h *EvaluatorHandler) GetTemplate(c *gin.Context) {
	templateID := c.Param("templateId")
	template := h.evaluatorService.GetTemplateByID(templateID)
	if template == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "template not found",
		})
		return
	}
	c.JSON(http.StatusOK, template)
}

// RunEvaluation runs a single evaluation synchronously
func (h *EvaluatorHandler) RunEvaluation(c *gin.Context) {
	var req domain.RunEvaluationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	if req.Async {
		// Create async job instead
		jobCreate := &domain.EvaluationJobCreate{
			EvaluatorID: req.EvaluatorID,
			TargetType:  "trace",
			TargetIDs:   []uuid.UUID{req.TraceID},
		}

		// Get project ID
		projectIDStr := c.GetString("project_id")
		if projectIDStr == "" {
			projectIDStr = c.Query("project_id")
		}
		if projectIDStr != "" {
			projectID, _ := uuid.Parse(projectIDStr)
			jobCreate.ProjectID = projectID
		}

		job, err := h.evaluatorService.CreateJob(c.Request.Context(), jobCreate)
		if err != nil {
			h.logger.Error("failed to create evaluation job", zap.Error(err))
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: err.Error(),
			})
			return
		}

		c.JSON(http.StatusAccepted, gin.H{
			"job_id": job.ID,
			"status": job.Status,
		})
		return
	}

	result, err := h.evaluatorService.RunEvaluation(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("evaluation failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "evaluation_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// BatchEvaluation creates a batch evaluation job
func (h *EvaluatorHandler) BatchEvaluation(c *gin.Context) {
	var req domain.BatchEvaluationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// Get project ID
	projectIDStr := c.GetString("project_id")
	if projectIDStr == "" {
		projectIDStr = c.Query("project_id")
	}
	if projectIDStr == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "missing_project_id",
			Message: "project_id is required",
		})
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_project_id",
			Message: "invalid project_id format",
		})
		return
	}

	jobCreate := &domain.EvaluationJobCreate{
		ProjectID:   projectID,
		EvaluatorID: req.EvaluatorID,
		TargetType:  "batch",
		TargetIDs:   req.TraceIDs,
	}

	job, err := h.evaluatorService.CreateJob(c.Request.Context(), jobCreate)
	if err != nil {
		h.logger.Error("failed to create batch evaluation job", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"job_id":      job.ID,
		"status":      job.Status,
		"total_items": job.TotalItems,
	})
}

// GetJob retrieves an evaluation job status
func (h *EvaluatorHandler) GetJob(c *gin.Context) {
	idStr := c.Param("jobId")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "invalid job ID format",
		})
		return
	}

	job, err := h.evaluatorService.GetJob(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "job not found",
			})
			return
		}
		h.logger.Error("failed to get job", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "failed to retrieve job",
		})
		return
	}

	c.JSON(http.StatusOK, job)
}

// ListJobs lists evaluation jobs
func (h *EvaluatorHandler) ListJobs(c *gin.Context) {
	filter := &domain.EvaluationJobFilter{
		ProjectID:   c.Query("project_id"),
		EvaluatorID: c.Query("evaluator_id"),
		Status:      c.Query("status"),
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}
	if filter.Limit == 0 {
		filter.Limit = 50
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}

	jobs, total, err := h.evaluatorService.ListJobs(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("failed to list jobs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "failed to list jobs",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   jobs,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// GetResults retrieves evaluation results
func (h *EvaluatorHandler) GetResults(c *gin.Context) {
	filter := &domain.EvaluationResultFilter{
		ProjectID:   c.Query("project_id"),
		EvaluatorID: c.Query("evaluator_id"),
		JobID:       c.Query("job_id"),
		TraceID:     c.Query("trace_id"),
		SpanID:      c.Query("span_id"),
		Status:      c.Query("status"),
	}

	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if startDate, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			filter.StartDate = startDate
		}
	}

	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if endDate, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			filter.EndDate = endDate
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}
	if filter.Limit == 0 {
		filter.Limit = 50
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}

	results, total, err := h.evaluatorService.GetResults(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error("failed to get results", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "failed to retrieve results",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   results,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// GetStats retrieves evaluation statistics
func (h *EvaluatorHandler) GetStats(c *gin.Context) {
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "missing_project_id",
			Message: "project_id is required",
		})
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_project_id",
			Message: "invalid project_id format",
		})
		return
	}

	var evaluatorID *uuid.UUID
	if evalIDStr := c.Query("evaluator_id"); evalIDStr != "" {
		evalID, err := uuid.Parse(evalIDStr)
		if err == nil {
			evaluatorID = &evalID
		}
	}

	var startDate, endDate time.Time
	if startDateStr := c.Query("start_date"); startDateStr != "" {
		startDate, _ = time.Parse(time.RFC3339, startDateStr)
	}
	if endDateStr := c.Query("end_date"); endDateStr != "" {
		endDate, _ = time.Parse(time.RFC3339, endDateStr)
	}

	stats, err := h.evaluatorService.GetStats(c.Request.Context(), projectID, evaluatorID, startDate, endDate)
	if err != nil {
		h.logger.Error("failed to get stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "failed to retrieve stats",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetCostSummary retrieves cost summary by evaluator
func (h *EvaluatorHandler) GetCostSummary(c *gin.Context) {
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "missing_project_id",
			Message: "project_id is required",
		})
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_project_id",
			Message: "invalid project_id format",
		})
		return
	}

	var startDate, endDate time.Time
	if startDateStr := c.Query("start_date"); startDateStr != "" {
		startDate, _ = time.Parse(time.RFC3339, startDateStr)
	}
	if endDateStr := c.Query("end_date"); endDateStr != "" {
		endDate, _ = time.Parse(time.RFC3339, endDateStr)
	}

	summaries, err := h.evaluatorService.GetCostSummary(c.Request.Context(), projectID, startDate, endDate)
	if err != nil {
		h.logger.Error("failed to get cost summary", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "failed to retrieve cost summary",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": summaries,
	})
}
