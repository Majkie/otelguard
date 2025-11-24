package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/service"
	"go.uber.org/zap"
)

// ExperimentHandler handles experiment-related endpoints
type ExperimentHandler struct {
	experimentService *service.ExperimentService
	logger            *zap.Logger
}

// NewExperimentHandler creates a new experiment handler
func NewExperimentHandler(experimentService *service.ExperimentService, logger *zap.Logger) *ExperimentHandler {
	return &ExperimentHandler{
		experimentService: experimentService,
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
