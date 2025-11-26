package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/repository/postgres"
	"github.com/otelguard/otelguard/internal/service"
	"go.uber.org/zap"
)

// ExperimentHandler handles experiment-related endpoints
type ExperimentHandler struct {
	experimentService *service.ExperimentService
	scheduler         *service.ExperimentScheduler
	logger            *zap.Logger
}

// NewExperimentHandler creates a new experiment handler
func NewExperimentHandler(experimentService *service.ExperimentService, experimentRepo *postgres.ExperimentRepository, logger *zap.Logger) *ExperimentHandler {
	scheduler := service.NewExperimentScheduler(experimentService, experimentRepo, logger)
	scheduler.Start()

	return &ExperimentHandler{
		experimentService: experimentService,
		scheduler:         scheduler,
		logger:            logger,
	}
}

// List returns all experiments for a project
func (h *ExperimentHandler) List(c *gin.Context) {
	projectID := c.Query("projectId")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "projectId is required",
		})
		return
	}

	if _, err := uuid.Parse(projectID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid projectId format",
		})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	experiments, total, err := h.experimentService.List(c.Request.Context(), projectID, &service.ListOptions{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		h.logger.Error("failed to list experiments", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve experiments",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   experiments,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// ListByDataset returns all experiments for a specific dataset
func (h *ExperimentHandler) ListByDataset(c *gin.Context) {
	datasetID := c.Param("datasetId")
	if _, err := uuid.Parse(datasetID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid dataset ID",
		})
		return
	}

	experiments, err := h.experimentService.ListByDataset(c.Request.Context(), datasetID)
	if err != nil {
		h.logger.Error("failed to list experiments by dataset", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve experiments",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": experiments,
	})
}

// Get retrieves an experiment by ID
func (h *ExperimentHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid experiment ID",
		})
		return
	}

	experiment, err := h.experimentService.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Experiment not found",
			})
			return
		}
		h.logger.Error("failed to get experiment", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve experiment",
		})
		return
	}

	c.JSON(http.StatusOK, experiment)
}

// Create creates a new experiment
func (h *ExperimentHandler) Create(c *gin.Context) {
	var input domain.ExperimentCreate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	experiment, err := h.experimentService.Create(c.Request.Context(), &input)
	if err != nil {
		h.logger.Error("failed to create experiment", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to create experiment",
		})
		return
	}

	c.JSON(http.StatusCreated, experiment)
}

// Execute executes an experiment
func (h *ExperimentHandler) Execute(c *gin.Context) {
	var input domain.ExperimentExecute
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	run, err := h.experimentService.Execute(c.Request.Context(), &input)
	if err != nil {
		h.logger.Error("failed to execute experiment", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Experiment execution started",
		"run":     run,
	})
}

// GetRun retrieves an experiment run by ID
func (h *ExperimentHandler) GetRun(c *gin.Context) {
	runID := c.Param("runId")
	if _, err := uuid.Parse(runID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid run ID",
		})
		return
	}

	run, err := h.experimentService.GetRun(c.Request.Context(), runID)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Run not found",
			})
			return
		}
		h.logger.Error("failed to get run", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve run",
		})
		return
	}

	c.JSON(http.StatusOK, run)
}

// ListRuns returns all runs for an experiment
func (h *ExperimentHandler) ListRuns(c *gin.Context) {
	experimentID := c.Param("id")
	if _, err := uuid.Parse(experimentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid experiment ID",
		})
		return
	}

	runs, err := h.experimentService.ListRuns(c.Request.Context(), experimentID)
	if err != nil {
		h.logger.Error("failed to list runs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve runs",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": runs,
	})
}

// GetResults returns results for an experiment run
func (h *ExperimentHandler) GetResults(c *gin.Context) {
	runID := c.Param("runId")
	if _, err := uuid.Parse(runID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid run ID",
		})
		return
	}

	results, err := h.experimentService.GetResults(c.Request.Context(), runID)
	if err != nil {
		h.logger.Error("failed to get results", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve results",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": results,
	})
}

// CompareRuns compares multiple experiment runs
func (h *ExperimentHandler) CompareRuns(c *gin.Context) {
	var req struct {
		RunIDs []string `json:"runIds" binding:"required,min=2"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// Parse run IDs
	runIDs := make([]uuid.UUID, 0, len(req.RunIDs))
	for _, idStr := range req.RunIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_request",
				"message": "invalid run ID format",
			})
			return
		}
		runIDs = append(runIDs, id)
	}

	comparison, err := h.experimentService.CompareRuns(c.Request.Context(), runIDs)
	if err != nil {
		h.logger.Error("failed to compare runs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, comparison)
}

// StatisticalComparison performs statistical significance testing on experiment runs
func (h *ExperimentHandler) StatisticalComparison(c *gin.Context) {
	var req struct {
		RunIDs []string `json:"runIds" binding:"required,min=2"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// Parse run IDs
	runIDs := make([]uuid.UUID, 0, len(req.RunIDs))
	for _, idStr := range req.RunIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_request",
				"message": "invalid run ID format",
			})
			return
		}
		runIDs = append(runIDs, id)
	}

	comparison, err := h.experimentService.PerformStatisticalComparison(c.Request.Context(), runIDs)
	if err != nil {
		h.logger.Error("failed to perform statistical comparison", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, comparison)
}

// CreateScheduleRequest represents a request to schedule an experiment
type CreateScheduleRequest struct {
	ExperimentID string `json:"experimentId" binding:"required"`
	ScheduleType string `json:"scheduleType" binding:"required"` // once, daily, weekly, monthly
	ScheduleTime string `json:"scheduleTime,omitempty"`          // RFC3339 format for "once"
	Enabled      bool   `json:"enabled"`
}

// CreateSchedule creates a new experiment schedule
func (h *ExperimentHandler) CreateSchedule(c *gin.Context) {
	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

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

	experimentID, err := uuid.Parse(req.ExperimentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid experimentId format",
		})
		return
	}

	// Verify experiment exists
	_, err = h.experimentService.Get(c.Request.Context(), experimentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Experiment not found",
		})
		return
	}

	// Parse schedule time if provided
	var scheduleTime time.Time
	if req.ScheduleTime != "" {
		scheduleTime, err = time.Parse(time.RFC3339, req.ScheduleTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_request",
				"message": "invalid scheduleTime format, use RFC3339",
			})
			return
		}
	} else if req.ScheduleType == "once" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "scheduleTime is required for 'once' type",
		})
		return
	}

	// Create schedule
	schedule := &service.ScheduledExperiment{
		ID:           uuid.New(),
		ExperimentID: experimentID,
		ProjectID:    projectUUID,
		ScheduleType: req.ScheduleType,
		ScheduleTime: scheduleTime,
		Enabled:      req.Enabled,
		CreatedBy:    uuid.MustParse(c.GetString("user_id")), // From auth context
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	h.scheduler.AddSchedule(schedule)

	c.JSON(http.StatusCreated, gin.H{
		"id":            schedule.ID,
		"experiment_id": schedule.ExperimentID,
		"schedule_type": schedule.ScheduleType,
		"schedule_time": schedule.ScheduleTime,
		"enabled":       schedule.Enabled,
		"next_run_at":   schedule.NextRunAt,
		"created_at":    schedule.CreatedAt,
	})
}

// ListSchedules lists all schedules for a project
func (h *ExperimentHandler) ListSchedules(c *gin.Context) {
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

	schedules := h.scheduler.ListSchedules(projectUUID)

	// Convert to response format
	result := make([]gin.H, len(schedules))
	for i, schedule := range schedules {
		result[i] = gin.H{
			"id":            schedule.ID,
			"experiment_id": schedule.ExperimentID,
			"schedule_type": schedule.ScheduleType,
			"schedule_time": schedule.ScheduleTime,
			"enabled":       schedule.Enabled,
			"last_run_at":   schedule.LastRunAt,
			"next_run_at":   schedule.NextRunAt,
			"created_at":    schedule.CreatedAt,
			"updated_at":    schedule.UpdatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  result,
		"total": len(result),
	})
}

// GetSchedule retrieves a specific schedule
func (h *ExperimentHandler) GetSchedule(c *gin.Context) {
	scheduleID := c.Param("scheduleId")
	if scheduleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "scheduleId is required",
		})
		return
	}

	scheduleUUID, err := uuid.Parse(scheduleID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid scheduleId format",
		})
		return
	}

	schedule, exists := h.scheduler.GetSchedule(scheduleUUID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "not_found",
			"message": "Schedule not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":            schedule.ID,
		"experiment_id": schedule.ExperimentID,
		"schedule_type": schedule.ScheduleType,
		"schedule_time": schedule.ScheduleTime,
		"enabled":       schedule.Enabled,
		"last_run_at":   schedule.LastRunAt,
		"next_run_at":   schedule.NextRunAt,
		"created_at":    schedule.CreatedAt,
		"updated_at":    schedule.UpdatedAt,
	})
}

// UpdateScheduleRequest represents an update to a schedule
type UpdateScheduleRequest struct {
	Enabled *bool `json:"enabled"`
}

// UpdateSchedule updates a schedule
func (h *ExperimentHandler) UpdateSchedule(c *gin.Context) {
	scheduleID := c.Param("scheduleId")
	if scheduleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "scheduleId is required",
		})
		return
	}

	scheduleUUID, err := uuid.Parse(scheduleID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid scheduleId format",
		})
		return
	}

	var req UpdateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	err = h.scheduler.UpdateSchedule(scheduleUUID, func(schedule *service.ScheduledExperiment) {
		if req.Enabled != nil {
			schedule.Enabled = *req.Enabled
		}
	})

	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Schedule not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Schedule updated",
	})
}

// DeleteSchedule deletes a schedule
func (h *ExperimentHandler) DeleteSchedule(c *gin.Context) {
	scheduleID := c.Param("scheduleId")
	if scheduleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "scheduleId is required",
		})
		return
	}

	scheduleUUID, err := uuid.Parse(scheduleID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid scheduleId format",
		})
		return
	}

	h.scheduler.RemoveSchedule(scheduleUUID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Schedule deleted",
	})
}
