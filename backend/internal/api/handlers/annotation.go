package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/api/middleware"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/service"
	"go.uber.org/zap"
)

// AnnotationHandler handles annotation endpoints
type AnnotationHandler struct {
	annotationService *service.AnnotationService
	logger            *zap.Logger
}

// NewAnnotationHandler creates a new annotation handler
func NewAnnotationHandler(annotationService *service.AnnotationService, logger *zap.Logger) *AnnotationHandler {
	return &AnnotationHandler{
		annotationService: annotationService,
		logger:            logger,
	}
}

// Queue Response Types

// AnnotationQueueResponse represents an annotation queue in API responses
type AnnotationQueueResponse struct {
	ID                    string                 `json:"id"`
	ProjectID             string                 `json:"projectId"`
	Name                  string                 `json:"name"`
	Description           string                 `json:"description,omitempty"`
	ScoreConfigs          []domain.ScoreConfig   `json:"scoreConfigs"`
	Config                map[string]interface{} `json:"config"`
	ItemSource            string                 `json:"itemSource"`
	ItemSourceConfig      map[string]interface{} `json:"itemSourceConfig"`
	AssignmentStrategy    string                 `json:"assignmentStrategy"`
	MaxAnnotationsPerItem int                    `json:"maxAnnotationsPerItem"`
	Instructions          string                 `json:"instructions,omitempty"`
	IsActive              bool                   `json:"isActive"`
	CreatedAt             string                 `json:"createdAt"`
	UpdatedAt             string                 `json:"updatedAt"`
}

// AnnotationQueueItemResponse represents a queue item in API responses
type AnnotationQueueItemResponse struct {
	ID             string                 `json:"id"`
	QueueID        string                 `json:"queueId"`
	ItemType       string                 `json:"itemType"`
	ItemID         string                 `json:"itemId"`
	ItemData       map[string]interface{} `json:"itemData,omitempty"`
	Metadata       map[string]interface{} `json:"metadata"`
	Priority       int                    `json:"priority"`
	MaxAnnotations int                    `json:"maxAnnotations"`
	CreatedAt      string                 `json:"createdAt"`
	UpdatedAt      string                 `json:"updatedAt"`
}

// AnnotationAssignmentResponse represents an assignment in API responses
type AnnotationAssignmentResponse struct {
	ID          string  `json:"id"`
	QueueItemID string  `json:"queueItemId"`
	UserID      string  `json:"userId"`
	Status      string  `json:"status"`
	AssignedAt  string  `json:"assignedAt"`
	StartedAt   *string `json:"startedAt,omitempty"`
	CompletedAt *string `json:"completedAt,omitempty"`
	SkippedAt   *string `json:"skippedAt,omitempty"`
	Notes       string  `json:"notes,omitempty"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
}

// AnnotationResponse represents an annotation in API responses
type AnnotationResponse struct {
	ID              string                 `json:"id"`
	AssignmentID    string                 `json:"assignmentId"`
	QueueID         string                 `json:"queueId"`
	QueueItemID     string                 `json:"queueItemId"`
	UserID          string                 `json:"userId"`
	Scores          map[string]interface{} `json:"scores"`
	Labels          []string               `json:"labels"`
	Notes           string                 `json:"notes,omitempty"`
	ConfidenceScore *float64               `json:"confidenceScore,omitempty"`
	AnnotationTime  *string                `json:"annotationTime,omitempty"`
	CreatedAt       string                 `json:"createdAt"`
	UpdatedAt       string                 `json:"updatedAt"`
}

// InterAnnotatorAgreementResponse represents agreement metrics in API responses
type InterAnnotatorAgreementResponse struct {
	ID              string   `json:"id"`
	QueueID         string   `json:"queueId"`
	QueueItemID     string   `json:"queueItemId"`
	ScoreConfigName string   `json:"scoreConfigName"`
	AgreementType   string   `json:"agreementType"`
	AgreementValue  *float64 `json:"agreementValue,omitempty"`
	AnnotatorCount  int      `json:"annotatorCount"`
	CalculatedAt    string   `json:"calculatedAt"`
}

// Queue Management Endpoints

// CreateQueue creates a new annotation queue
// @Summary Create annotation queue
// @Tags annotation
// @Accept json
// @Produce json
// @Param queue body domain.AnnotationQueueCreate true "Queue creation data"
// @Success 201 {object} AnnotationQueueResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/projects/{projectId}/annotation-queues [post]
func (h *AnnotationHandler) CreateQueue(c *gin.Context) {
	projectID := c.Param("projectId")

	var req domain.AnnotationQueueCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	req.ProjectID = uuid.MustParse(projectID)

	queue, err := h.annotationService.CreateQueue(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("failed to create annotation queue", zap.Error(err))
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create annotation queue",
		})
		return
	}

	response := h.convertQueueToResponse(queue)
	c.JSON(http.StatusCreated, response)
}

// GetQueue retrieves an annotation queue by ID
// @Summary Get annotation queue
// @Tags annotation
// @Produce json
// @Param queueId path string true "Queue ID"
// @Success 200 {object} AnnotationQueueResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/annotation-queues/{queueId} [get]
func (h *AnnotationHandler) GetQueue(c *gin.Context) {
	queueID := c.Param("queueId")

	queue, err := h.annotationService.GetQueue(c.Request.Context(), queueID)
	if err != nil {
		c.JSON(http.StatusNotFound, middleware.ErrorResponse{
			Error:   "not_found",
			Message: "Annotation queue not found",
		})
		return
	}

	response := h.convertQueueToResponse(queue)
	c.JSON(http.StatusOK, response)
}

// ListQueuesByProject retrieves annotation queues for a project
// @Summary List annotation queues
// @Tags annotation
// @Produce json
// @Param projectId path string true "Project ID"
// @Success 200 {array} AnnotationQueueResponse
// @Router /api/v1/projects/{projectId}/annotation-queues [get]
func (h *AnnotationHandler) ListQueuesByProject(c *gin.Context) {
	projectID := c.Param("projectId")

	queues, err := h.annotationService.ListQueuesByProject(c.Request.Context(), projectID)
	if err != nil {
		h.logger.Error("failed to list annotation queues", zap.Error(err))
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list annotation queues",
		})
		return
	}

	var responses []AnnotationQueueResponse
	for _, queue := range queues {
		responses = append(responses, h.convertQueueToResponse(&queue))
	}

	c.JSON(http.StatusOK, responses)
}

// UpdateQueue updates an annotation queue
// @Summary Update annotation queue
// @Tags annotation
// @Accept json
// @Produce json
// @Param queueId path string true "Queue ID"
// @Param queue body domain.AnnotationQueueUpdate true "Queue update data"
// @Success 200 {object} AnnotationQueueResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/annotation-queues/{queueId} [put]
func (h *AnnotationHandler) UpdateQueue(c *gin.Context) {
	queueID := c.Param("queueId")

	var req domain.AnnotationQueueUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	queue, err := h.annotationService.UpdateQueue(c.Request.Context(), queueID, &req)
	if err != nil {
		h.logger.Error("failed to update annotation queue", zap.Error(err))
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update annotation queue",
		})
		return
	}

	response := h.convertQueueToResponse(queue)
	c.JSON(http.StatusOK, response)
}

// DeleteQueue deletes an annotation queue
// @Summary Delete annotation queue
// @Tags annotation
// @Param queueId path string true "Queue ID"
// @Success 204
// @Router /api/v1/annotation-queues/{queueId} [delete]
func (h *AnnotationHandler) DeleteQueue(c *gin.Context) {
	queueID := c.Param("queueId")

	if err := h.annotationService.DeleteQueue(c.Request.Context(), queueID); err != nil {
		h.logger.Error("failed to delete annotation queue", zap.Error(err))
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete annotation queue",
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// Queue Item Management

// CreateQueueItem creates a new queue item
// @Summary Create queue item
// @Tags annotation
// @Accept json
// @Produce json
// @Param item body domain.AnnotationQueueItemCreate true "Queue item creation data"
// @Success 201 {object} AnnotationQueueItemResponse
// @Router /api/v1/annotation-queues/{queueId}/items [post]
func (h *AnnotationHandler) CreateQueueItem(c *gin.Context) {
	queueID := c.Param("queueId")

	var req domain.AnnotationQueueItemCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	req.QueueID = uuid.MustParse(queueID)

	item, err := h.annotationService.CreateQueueItem(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("failed to create queue item", zap.Error(err))
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create queue item",
		})
		return
	}

	response := h.convertQueueItemToResponse(item)
	c.JSON(http.StatusCreated, response)
}

// ListQueueItems retrieves queue items for a queue
// @Summary List queue items
// @Tags annotation
// @Produce json
// @Param queueId path string true "Queue ID"
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} AnnotationQueueItemResponse
// @Router /api/v1/annotation-queues/{queueId}/items [get]
func (h *AnnotationHandler) ListQueueItems(c *gin.Context) {
	queueID := c.Param("queueId")

	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	items, err := h.annotationService.ListQueueItems(c.Request.Context(), queueID, limit, offset)
	if err != nil {
		h.logger.Error("failed to list queue items", zap.Error(err))
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list queue items",
		})
		return
	}

	var responses []AnnotationQueueItemResponse
	for _, item := range items {
		responses = append(responses, h.convertQueueItemToResponse(&item))
	}

	c.JSON(http.StatusOK, responses)
}

// Assignment Management

// AssignNextItem assigns the next available item to the current user
// @Summary Assign next item
// @Tags annotation
// @Produce json
// @Param queueId path string true "Queue ID"
// @Success 200 {object} AnnotationAssignmentResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/annotation-queues/{queueId}/assign [post]
func (h *AnnotationHandler) AssignNextItem(c *gin.Context) {
	queueID := c.Param("queueId")
	userID := c.GetString(string(middleware.ContextUserID))

	assignment, err := h.annotationService.AssignNextItem(c.Request.Context(), queueID, userID)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, middleware.ErrorResponse{
				Error:   "no_items_available",
				Message: "No items available for assignment",
			})
			return
		}
		h.logger.Error("failed to assign next item", zap.Error(err))
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to assign next item",
		})
		return
	}

	response := h.convertAssignmentToResponse(assignment)
	c.JSON(http.StatusOK, response)
}

// StartAssignment marks an assignment as in progress
// @Summary Start assignment
// @Tags annotation
// @Param assignmentId path string true "Assignment ID"
// @Success 204
// @Router /api/v1/annotation-assignments/{assignmentId}/start [post]
func (h *AnnotationHandler) StartAssignment(c *gin.Context) {
	assignmentID := c.Param("assignmentId")
	userID := c.GetString(string(middleware.ContextUserID))

	if err := h.annotationService.StartAssignment(c.Request.Context(), assignmentID, userID); err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, middleware.ErrorResponse{
				Error:   "not_found",
				Message: "Assignment not found",
			})
			return
		}
		if err == domain.ErrForbidden {
			c.JSON(http.StatusForbidden, middleware.ErrorResponse{
				Error:   "forbidden",
				Message: "You don't have permission to modify this assignment",
			})
			return
		}
		h.logger.Error("failed to start assignment", zap.Error(err))
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to start assignment",
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// SkipAssignment marks an assignment as skipped
// @Summary Skip assignment
// @Tags annotation
// @Accept json
// @Param assignmentId path string true "Assignment ID"
// @Param notes body object false "Skip notes"
// @Success 204
// @Router /api/v1/annotation-assignments/{assignmentId}/skip [post]
func (h *AnnotationHandler) SkipAssignment(c *gin.Context) {
	assignmentID := c.Param("assignmentId")
	userID := c.GetString(string(middleware.ContextUserID))

	var req struct {
		Notes string `json:"notes,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	if err := h.annotationService.SkipAssignment(c.Request.Context(), assignmentID, userID, req.Notes); err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, middleware.ErrorResponse{
				Error:   "not_found",
				Message: "Assignment not found",
			})
			return
		}
		if err == domain.ErrForbidden {
			c.JSON(http.StatusForbidden, middleware.ErrorResponse{
				Error:   "forbidden",
				Message: "You don't have permission to modify this assignment",
			})
			return
		}
		h.logger.Error("failed to skip assignment", zap.Error(err))
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to skip assignment",
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// Annotation Management

// CreateAnnotation creates a new annotation
// @Summary Create annotation
// @Tags annotation
// @Accept json
// @Produce json
// @Param annotation body domain.AnnotationCreate true "Annotation creation data"
// @Success 201 {object} AnnotationResponse
// @Router /api/v1/annotations [post]
func (h *AnnotationHandler) CreateAnnotation(c *gin.Context) {
	var req domain.AnnotationCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	annotation, err := h.annotationService.CreateAnnotation(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("failed to create annotation", zap.Error(err))
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create annotation",
		})
		return
	}

	response := h.convertAnnotationToResponse(annotation)
	c.JSON(http.StatusCreated, response)
}

// GetAnnotation retrieves an annotation by ID
// @Summary Get annotation
// @Tags annotation
// @Produce json
// @Param annotationId path string true "Annotation ID"
// @Success 200 {object} AnnotationResponse
// @Router /api/v1/annotations/{annotationId} [get]
func (h *AnnotationHandler) GetAnnotation(c *gin.Context) {
	annotationID := c.Param("annotationId")

	annotation, err := h.annotationService.GetAnnotation(c.Request.Context(), annotationID)
	if err != nil {
		c.JSON(http.StatusNotFound, middleware.ErrorResponse{
			Error:   "not_found",
			Message: "Annotation not found",
		})
		return
	}

	response := h.convertAnnotationToResponse(annotation)
	c.JSON(http.StatusOK, response)
}

// ListAnnotationsByQueueItem retrieves annotations for a queue item
// @Summary List annotations for queue item
// @Tags annotation
// @Produce json
// @Param queueItemId path string true "Queue Item ID"
// @Success 200 {array} AnnotationResponse
// @Router /api/v1/annotation-queue-items/{queueItemId}/annotations [get]
func (h *AnnotationHandler) ListAnnotationsByQueueItem(c *gin.Context) {
	queueItemID := c.Param("queueItemId")

	annotations, err := h.annotationService.ListAnnotationsByQueueItem(c.Request.Context(), queueItemID)
	if err != nil {
		h.logger.Error("failed to list annotations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list annotations",
		})
		return
	}

	var responses []AnnotationResponse
	for _, annotation := range annotations {
		responses = append(responses, h.convertAnnotationToResponse(&annotation))
	}

	c.JSON(http.StatusOK, responses)
}

// User Assignments

// ListUserAssignments retrieves assignments for the current user
// @Summary List user assignments
// @Tags annotation
// @Produce json
// @Param status query string false "Assignment status filter"
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} AnnotationAssignmentResponse
// @Router /api/v1/user/annotation-assignments [get]
func (h *AnnotationHandler) ListUserAssignments(c *gin.Context) {
	userID := c.GetString(string(middleware.ContextUserID))
	status := c.Query("status")

	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	assignments, err := h.annotationService.ListUserAssignments(c.Request.Context(), userID, &status, limit, offset)
	if err != nil {
		h.logger.Error("failed to list user assignments", zap.Error(err))
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list user assignments",
		})
		return
	}

	var responses []AnnotationAssignmentResponse
	for _, assignment := range assignments {
		responses = append(responses, h.convertAssignmentToResponse(&assignment))
	}

	c.JSON(http.StatusOK, responses)
}

// Statistics

// GetQueueStats gets statistics for a queue
// @Summary Get queue statistics
// @Tags annotation
// @Produce json
// @Param queueId path string true "Queue ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/annotation-queues/{queueId}/stats [get]
func (h *AnnotationHandler) GetQueueStats(c *gin.Context) {
	queueID := c.Param("queueId")

	stats, err := h.annotationService.GetQueueStats(c.Request.Context(), queueID)
	if err != nil {
		h.logger.Error("failed to get queue stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get queue statistics",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetUserStats gets annotation statistics for the current user
// @Summary Get user annotation statistics
// @Tags annotation
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/user/annotation-stats [get]
func (h *AnnotationHandler) GetUserStats(c *gin.Context) {
	userID := c.GetString(string(middleware.ContextUserID))

	stats, err := h.annotationService.GetUserStats(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get user stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get user statistics",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// CalculateAgreement calculates inter-annotator agreement for a queue item
// @Summary Calculate inter-annotator agreement
// @Tags annotation
// @Produce json
// @Param queueId path string true "Queue ID"
// @Param queueItemId path string true "Queue Item ID"
// @Param scoreConfigName query string true "Score Config Name"
// @Success 200 {object} InterAnnotatorAgreementResponse
// @Router /api/v1/annotation-queues/{queueId}/items/{queueItemId}/agreement [post]
func (h *AnnotationHandler) CalculateAgreement(c *gin.Context) {
	queueID := c.Param("queueId")
	queueItemID := c.Param("queueItemId")
	scoreConfigName := c.Query("scoreConfigName")

	if scoreConfigName == "" {
		c.JSON(http.StatusBadRequest, middleware.ErrorResponse{
			Error:   "invalid_request",
			Message: "scoreConfigName query parameter is required",
		})
		return
	}

	agreement, err := h.annotationService.CalculateInterAnnotatorAgreement(c.Request.Context(), queueID, queueItemID, scoreConfigName)
	if err != nil {
		h.logger.Error("failed to calculate agreement", zap.Error(err))
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to calculate inter-annotator agreement",
		})
		return
	}

	response := h.convertAgreementToResponse(agreement)
	c.JSON(http.StatusOK, response)
}

// GetQueueAgreements gets inter-annotator agreements for a queue
// @Summary Get queue agreements
// @Tags annotation
// @Produce json
// @Param queueId path string true "Queue ID"
// @Param limit query int false "Limit" default(50)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} InterAnnotatorAgreementResponse
// @Router /api/v1/annotation-queues/{queueId}/agreements [get]
func (h *AnnotationHandler) GetQueueAgreements(c *gin.Context) {
	queueID := c.Param("queueId")

	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	agreements, err := h.annotationService.GetInterAnnotatorAgreements(c.Request.Context(), queueID, limit, offset)
	if err != nil {
		h.logger.Error("failed to get queue agreements", zap.Error(err))
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get queue agreements",
		})
		return
	}

	var responses []InterAnnotatorAgreementResponse
	for _, agreement := range agreements {
		responses = append(responses, h.convertAgreementToResponse(&agreement))
	}

	c.JSON(http.StatusOK, responses)
}

// GetQueueAgreementStats gets overall agreement statistics for a queue
// @Summary Get queue agreement statistics
// @Tags annotation
// @Produce json
// @Param queueId path string true "Queue ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/annotation-queues/{queueId}/agreement-stats [get]
func (h *AnnotationHandler) GetQueueAgreementStats(c *gin.Context) {
	queueID := c.Param("queueId")

	stats, err := h.annotationService.GetQueueAgreementStats(c.Request.Context(), queueID)
	if err != nil {
		h.logger.Error("failed to get queue agreement stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get queue agreement statistics",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ExportAnnotations exports annotations for a queue in the specified format
// @Summary Export annotations
// @Tags annotation
// @Produce json,csv
// @Param queueId path string true "Queue ID"
// @Param format query string false "Export format (json or csv)" default(json)
// @Success 200 {file} file
// @Router /api/v1/annotation-queues/{queueId}/export [get]
func (h *AnnotationHandler) ExportAnnotations(c *gin.Context) {
	queueID := c.Param("queueId")
	format := c.DefaultQuery("format", "json")

	// Get all annotations for the queue
	annotations, err := h.annotationService.ListAnnotationsByQueue(c.Request.Context(), queueID, 10000, 0) // Large limit for export
	if err != nil {
		h.logger.Error("failed to get annotations for export", zap.Error(err))
		c.JSON(http.StatusInternalServerError, middleware.ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to export annotations",
		})
		return
	}

	if format == "csv" {
		h.exportAnnotationsAsCSV(c, annotations)
	} else {
		h.exportAnnotationsAsJSON(c, annotations)
	}
}

func (h *AnnotationHandler) exportAnnotationsAsJSON(c *gin.Context, annotations []domain.Annotation) {
	var responses []AnnotationResponse
	for _, annotation := range annotations {
		responses = append(responses, h.convertAnnotationToResponse(&annotation))
	}

	c.Header("Content-Disposition", "attachment; filename=annotations.json")
	c.Header("Content-Type", "application/json")
	c.JSON(http.StatusOK, responses)
}

func (h *AnnotationHandler) exportAnnotationsAsCSV(c *gin.Context, annotations []domain.Annotation) {
	c.Header("Content-Disposition", "attachment; filename=annotations.csv")
	c.Header("Content-Type", "text/csv")

	// Write CSV header
	c.Writer.Write([]byte("ID,Queue ID,Queue Item ID,User ID,Scores,Labels,Notes,Confidence Score,Annotation Time,Created At\n"))

	// Write data rows
	for _, annotation := range annotations {
		scoresStr := "N/A"
		if len(annotation.Scores) > 0 {
			scoresStr = string(annotation.Scores)
		}

		labelsStr := strings.Join(annotation.Labels, ";")

		confidenceStr := ""
		if annotation.ConfidenceScore.Valid {
			confidenceStr = fmt.Sprintf("%.2f", annotation.ConfidenceScore.Float64)
		}

		annotationTimeStr := ""
		if annotation.AnnotationTime.Valid {
			annotationTimeStr = annotation.AnnotationTime.String
		}

		row := fmt.Sprintf("%s,%s,%s,%s,%q,%q,%q,%s,%s,%s\n",
			annotation.ID.String(),
			annotation.QueueID.String(),
			annotation.QueueItemID.String(),
			annotation.UserID.String(),
			scoresStr,
			labelsStr,
			annotation.Notes,
			confidenceStr,
			annotationTimeStr,
			annotation.CreatedAt.Format(time.RFC3339),
		)

		c.Writer.Write([]byte(row))
	}
}

// Helper methods for converting domain objects to responses

func (h *AnnotationHandler) convertQueueToResponse(queue *domain.AnnotationQueue) AnnotationQueueResponse {
	var scoreConfigs []domain.ScoreConfig
	if len(queue.ScoreConfigs) > 0 {
		if err := json.Unmarshal(queue.ScoreConfigs, &scoreConfigs); err != nil {
			h.logger.Warn("failed to unmarshal score configs", zap.Error(err))
		}
	}

	var config map[string]interface{}
	if len(queue.Config) > 0 {
		if err := json.Unmarshal(queue.Config, &config); err != nil {
			h.logger.Warn("failed to unmarshal config", zap.Error(err))
		}
	}

	var itemSourceConfig map[string]interface{}
	if len(queue.ItemSourceConfig) > 0 {
		if err := json.Unmarshal(queue.ItemSourceConfig, &itemSourceConfig); err != nil {
			h.logger.Warn("failed to unmarshal item source config", zap.Error(err))
		}
	}

	return AnnotationQueueResponse{
		ID:                    queue.ID.String(),
		ProjectID:             queue.ProjectID.String(),
		Name:                  queue.Name,
		Description:           queue.Description,
		ScoreConfigs:          scoreConfigs,
		Config:                config,
		ItemSource:            queue.ItemSource,
		ItemSourceConfig:      itemSourceConfig,
		AssignmentStrategy:    queue.AssignmentStrategy,
		MaxAnnotationsPerItem: queue.MaxAnnotationsPerItem,
		Instructions:          queue.Instructions,
		IsActive:              queue.IsActive,
		CreatedAt:             queue.CreatedAt.Format(time.RFC3339),
		UpdatedAt:             queue.UpdatedAt.Format(time.RFC3339),
	}
}

func (h *AnnotationHandler) convertQueueItemToResponse(item *domain.AnnotationQueueItem) AnnotationQueueItemResponse {
	var itemData map[string]interface{}
	if len(item.ItemData) > 0 {
		if err := json.Unmarshal(item.ItemData, &itemData); err != nil {
			h.logger.Warn("failed to unmarshal item data", zap.Error(err))
		}
	}

	var metadata map[string]interface{}
	if len(item.Metadata) > 0 {
		if err := json.Unmarshal(item.Metadata, &metadata); err != nil {
			h.logger.Warn("failed to unmarshal metadata", zap.Error(err))
		}
	}

	return AnnotationQueueItemResponse{
		ID:             item.ID.String(),
		QueueID:        item.QueueID.String(),
		ItemType:       item.ItemType,
		ItemID:         item.ItemID,
		ItemData:       itemData,
		Metadata:       metadata,
		Priority:       item.Priority,
		MaxAnnotations: item.MaxAnnotations,
		CreatedAt:      item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      item.UpdatedAt.Format(time.RFC3339),
	}
}

func (h *AnnotationHandler) convertAssignmentToResponse(assignment *domain.AnnotationAssignment) AnnotationAssignmentResponse {
	var startedAt, completedAt, skippedAt *string
	if assignment.StartedAt.Valid {
		t := assignment.StartedAt.Time.Format(time.RFC3339)
		startedAt = &t
	}
	if assignment.CompletedAt.Valid {
		t := assignment.CompletedAt.Time.Format(time.RFC3339)
		completedAt = &t
	}
	if assignment.SkippedAt.Valid {
		t := assignment.SkippedAt.Time.Format(time.RFC3339)
		skippedAt = &t
	}

	return AnnotationAssignmentResponse{
		ID:          assignment.ID.String(),
		QueueItemID: assignment.QueueItemID.String(),
		UserID:      assignment.UserID.String(),
		Status:      assignment.Status,
		AssignedAt:  assignment.AssignedAt.Format(time.RFC3339),
		StartedAt:   startedAt,
		CompletedAt: completedAt,
		SkippedAt:   skippedAt,
		Notes:       assignment.Notes,
		CreatedAt:   assignment.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   assignment.UpdatedAt.Format(time.RFC3339),
	}
}

func (h *AnnotationHandler) convertAnnotationToResponse(annotation *domain.Annotation) AnnotationResponse {
	var scores map[string]interface{}
	if len(annotation.Scores) > 0 {
		if err := json.Unmarshal(annotation.Scores, &scores); err != nil {
			h.logger.Warn("failed to unmarshal scores", zap.Error(err))
		}
	}

	var confidenceScore *float64
	if annotation.ConfidenceScore.Valid {
		confidenceScore = &annotation.ConfidenceScore.Float64
	}

	var annotationTime *string
	if annotation.AnnotationTime.Valid {
		annotationTime = &annotation.AnnotationTime.String
	}

	return AnnotationResponse{
		ID:              annotation.ID.String(),
		AssignmentID:    annotation.AssignmentID.String(),
		QueueID:         annotation.QueueID.String(),
		QueueItemID:     annotation.QueueItemID.String(),
		UserID:          annotation.UserID.String(),
		Scores:          scores,
		Labels:          annotation.Labels,
		Notes:           annotation.Notes,
		ConfidenceScore: confidenceScore,
		AnnotationTime:  annotationTime,
		CreatedAt:       annotation.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       annotation.UpdatedAt.Format(time.RFC3339),
	}
}

func (h *AnnotationHandler) convertAgreementToResponse(agreement *domain.InterAnnotatorAgreement) InterAnnotatorAgreementResponse {
	var agreementValue *float64
	if agreement.AgreementValue.Valid {
		agreementValue = &agreement.AgreementValue.Float64
	}

	return InterAnnotatorAgreementResponse{
		ID:              agreement.ID.String(),
		QueueID:         agreement.QueueID.String(),
		QueueItemID:     agreement.QueueItemID.String(),
		ScoreConfigName: agreement.ScoreConfigName,
		AgreementType:   agreement.AgreementType,
		AgreementValue:  agreementValue,
		AnnotatorCount:  agreement.AnnotatorCount,
		CalculatedAt:    agreement.CalculatedAt.Format(time.RFC3339),
	}
}
