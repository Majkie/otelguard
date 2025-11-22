package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/service"
	"go.uber.org/zap"
)

// PromptHandler handles prompt-related endpoints
type PromptHandler struct {
	promptService *service.PromptService
	logger        *zap.Logger
}

// NewPromptHandler creates a new prompt handler
func NewPromptHandler(promptService *service.PromptService, logger *zap.Logger) *PromptHandler {
	return &PromptHandler{
		promptService: promptService,
		logger:        logger,
	}
}

// List returns all prompts for a project
func (h *PromptHandler) List(c *gin.Context) {
	projectID := c.GetString("project_id")
	if projectID == "" {
		projectID = "default"
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}

	prompts, total, err := h.promptService.List(c.Request.Context(), projectID, &service.ListOptions{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		h.logger.Error("failed to list prompts", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve prompts",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   prompts,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// Create creates a new prompt
func (h *PromptHandler) Create(c *gin.Context) {
	var req struct {
		Name        string   `json:"name" binding:"required"`
		Description string   `json:"description"`
		Content     string   `json:"content"`
		Tags        []string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	projectID := c.GetString("project_id")
	if projectID == "" {
		projectID = "default"
	}

	projectUUID, err := uuid.Parse(projectID)
	if err != nil {
		projectUUID = uuid.New()
	}

	now := time.Now()
	prompt := &domain.Prompt{
		ID:          uuid.New(),
		ProjectID:   projectUUID,
		Name:        req.Name,
		Description: req.Description,
		Tags:        req.Tags,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.promptService.Create(c.Request.Context(), prompt); err != nil {
		h.logger.Error("failed to create prompt", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to create prompt",
		})
		return
	}

	c.JSON(http.StatusCreated, prompt)
}

// Get returns a single prompt
func (h *PromptHandler) Get(c *gin.Context) {
	id := c.Param("id")
	prompt, err := h.promptService.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Prompt not found",
			})
			return
		}
		h.logger.Error("failed to get prompt", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve prompt",
		})
		return
	}
	c.JSON(http.StatusOK, prompt)
}

// Update updates a prompt
func (h *PromptHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	prompt, err := h.promptService.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Prompt not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve prompt",
		})
		return
	}

	if req.Name != "" {
		prompt.Name = req.Name
	}
	prompt.Description = req.Description
	if req.Tags != nil {
		prompt.Tags = req.Tags
	}
	prompt.UpdatedAt = time.Now()

	if err := h.promptService.Update(c.Request.Context(), prompt); err != nil {
		h.logger.Error("failed to update prompt", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to update prompt",
		})
		return
	}

	c.JSON(http.StatusOK, prompt)
}

// Delete deletes a prompt
func (h *PromptHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.promptService.Delete(c.Request.Context(), id); err != nil {
		h.logger.Error("failed to delete prompt", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to delete prompt",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Prompt deleted"})
}

// ListVersions returns all versions of a prompt
func (h *PromptHandler) ListVersions(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": []interface{}{}, "total": 0})
}

// CreateVersion creates a new version of a prompt
func (h *PromptHandler) CreateVersion(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not_implemented"})
}

// GetVersion returns a specific version of a prompt
func (h *PromptHandler) GetVersion(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not_implemented"})
}

// Compile compiles a prompt template with variables
func (h *PromptHandler) Compile(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Variables map[string]interface{} `json:"variables"`
		Version   int                    `json:"version,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	// TODO: Implement template compilation
	c.JSON(http.StatusOK, gin.H{
		"id":       id,
		"compiled": "Compiled prompt content would go here",
	})
}
